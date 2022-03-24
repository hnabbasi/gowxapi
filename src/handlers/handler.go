package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaz-yamam0t0/go-timeparser/timeparser"
)

const (
	baseURL                        = "https://api.weather.gov"
	geocodeURL                     = "http://api.openweathermap.org/geo/1.0/direct?q="
	stateAlerts                    = baseURL + "/alerts/active/area"
	getLatestObservationsByStation = baseURL + "/stations/%v/observations/latest?require_qc=true"
	getLocationByPoints            = baseURL + "/points/%v"
	getAfdByLocation0              = baseURL + "/products/types/AFD/locations/%v"
)

var (
	wg = sync.WaitGroup{}
)

// Welcome message from the server
func Home(c *gin.Context) {
	c.JSON(http.StatusOK, "\u26c5 Welcome to Hussain's Weather API")
}

// Get all alerts for state by state code e.g. TX
func GetAlertsForState(c *gin.Context) {
	state := c.Param("state")

	alerts, err := getAlerts(state)

	if err != nil {
		c.JSON(http.StatusBadRequest, "Could not fetch alerts")
	}

	c.JSON(http.StatusOK, alerts)
}

// Get complete weather information for a give city name e.g. Houston
// including:
// - Current conditions
// - Active alerts
// - Hourly conditions for next 24 hours
// - Weekly conditions for next 7 days
// - Hourly rain chances for next 24 hours
// - Daily rain chances for next 7 days
// - Area forecast discussion
func GetWeather(c *gin.Context) {
	var weatherResponse WeatherResponse
	coordsChannel := make(chan string)
	locationChannel := make(chan LocationResponse)
	alertsChannel := make(chan AlertResponse)
	observationChannel := make(chan Observation)
	periodsChannel := make(chan []Period)
	weeklyChannel := make(chan []DailyForecast)
	rainChannel := make(chan map[string][]int)
	productChannel := make(chan Product)

	wg.Add(8)

	go func(coordsChannel chan string) {
		cityCoords, err := getCity(c.Param("cityState"))
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
		} else {
			coordsChannel <- cityCoords
		}
		wg.Done()
	}(coordsChannel)

	go func(locationChannel chan LocationResponse) {
		locationChannel <- getLocation(<-coordsChannel)
		wg.Done()
	}(locationChannel)
	weatherResponse.LocationResponse = <-locationChannel

	go func(alertsChannel chan AlertResponse) {
		alerts, err := getAlerts(weatherResponse.LocationResponse.State)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
		}
		alertsChannel <- alerts
		wg.Done()
	}(alertsChannel)
	weatherResponse.AlertResponse = <-alertsChannel

	go func(observationChannel chan Observation) {
		url := fmt.Sprintf(getLatestObservationsByStation, weatherResponse.LocationResponse.ObservationStation)
		observations, _ := getCurrentConditions(url)
		observationChannel <- observations
		wg.Done()
	}(observationChannel)
	weatherResponse.Observation = <-observationChannel

	go func(periodsChannel chan []Period) {
		hourly, _ := getPeriods(weatherResponse.LocationResponse.HourlyForecastUrl, 24)
		periodsChannel <- hourly
		wg.Done()
	}(periodsChannel)
	weatherResponse.Hourly = <-periodsChannel

	go func(weeklyChannel chan []DailyForecast) {
		weekly, _ := getWeekly(weatherResponse.LocationResponse.ForecastUrl)
		weeklyChannel <- weekly
		wg.Done()
	}(weeklyChannel)
	weatherResponse.Weekly = <-weeklyChannel

	go func(rainChannel chan map[string][]int) {
		rainChances, _ := getRainChancesMap(weatherResponse.LocationResponse.ForecastGridDataUrl)
		rainChannel <- rainChances
		wg.Done()
	}(rainChannel)
	weatherResponse.RainChances.UnitCode = "wmoUnit:percent"
	weatherResponse.RainChances.Values = <-rainChannel

	go func(productChannel chan Product) {
		product, _ := getAfdProduct(fmt.Sprintf(getAfdByLocation0, weatherResponse.CountyWarningArea))
		productChannel <- product
		wg.Done()
	}(productChannel)
	weatherResponse.AreaForecastDiscussion = <-productChannel

	wg.Wait()
	c.JSON(http.StatusOK, weatherResponse)
}

func getAfdProduct(url string) (Product, error) {
	response, err := getHttpResponse(url)
	if err != nil {
		log.Fatal(err)
		return Product{}, err
	}

	var allProductsResponse struct {
		Graph [1]struct {
			Id string `json:"@id"`
		} `json:"@graph"`
	}
	if er := json.Unmarshal(response, &allProductsResponse); er != nil {
		return Product{}, er
	}

	productResponse, err := getHttpResponse(string(allProductsResponse.Graph[0].Id))
	if err != nil {
		return Product{}, err
	}

	var product Product
	if err = json.Unmarshal(productResponse, &product); err != nil {
		return Product{}, err
	}

	return product, nil
}

func getRainChancesMap(url string) (map[string][]int, error) {
	response, err := getHttpResponse(url)
	if err != nil {
		log.Fatal(err.Error())
	}

	var rainChancesResponse struct {
		Properties struct {
			Chances struct {
				Values []ValueItem `json:"values"`
			} `json:"probabilityOfPrecipitation"`
		} `json:"properties"`
	}

	if e := json.Unmarshal(response, &rainChancesResponse); e != nil {
		log.Fatal(e.Error())
	}

	return fillPeriods(rainChancesResponse.Properties.Chances.Values)
}

func fillPeriods(periods []ValueItem) (map[string][]int, error) {
	retVal := make(map[string][]int)

	for _, v := range periods {
		timestampArray := strings.Split(v.ValidTime, "/")

		current, err := time.Parse(time.RFC3339, timestampArray[0])
		if err != nil {
			return nil, errors.New(err.Error())
		}

		end, err := timeparser.ParseTimeStr(timestampArray[1], &current)
		if err != nil {
			return nil, errors.New(err.Error())
		}

		for current.Before(*end) {
			key := strings.Split(current.String(), " ")[0]
			if _, e := retVal[key]; !e {
				retVal[key] = make([]int, 24)
			}
			retVal[key][current.Hour()] = int(v.Value)
			current = current.Add(time.Hour)
		}
	}
	return retVal, nil
}

func getWeekly(url string) ([]DailyForecast, error) {
	response, err := getPeriods(url, 0)

	dailyMap := make(map[int]DailyForecast)

	for _, period := range response {
		day, exists := dailyMap[period.StartTime.Day()]

		if !exists {
			day = DailyForecast{Date: period.StartTime, TemperatureUnit: "F"}
		}

		if period.IsDaytime {
			day.DayTemp = period.Temperature
			day.DayIcon = period.Icon
		} else {
			day.NightTemp = period.Temperature
			day.NightIcon = period.Icon
		}

		dailyMap[period.StartTime.Day()] = day
	}

	weekly := []DailyForecast{}
	for _, forecast := range dailyMap {
		weekly = append(weekly, forecast)
	}
	sort.SliceStable(weekly, func(i, j int) bool {
		return weekly[i].Date.Day() < weekly[j].Date.Day()
	})
	return weekly, err
}

func getPeriods(url string, count int) ([]Period, error) {
	response, err := getHttpResponse(url)
	var periodsResponse struct {
		Properties struct {
			Period []Period `json:"periods"`
		} `json:"properties"`
	}
	e := json.Unmarshal(response, &periodsResponse)
	if e != nil {
		log.Fatal(e.Error())
	}
	if count > 0 {
		return periodsResponse.Properties.Period[:count], err
	} else {
		return periodsResponse.Properties.Period, err
	}
}

func getCurrentConditions(url string) (Observation, error) {
	response, err := getHttpResponse(url)
	var observationResponse struct {
		Properties struct {
			Observation
		} `json:"properties"`
	}
	e := json.Unmarshal(response, &observationResponse)
	if e != nil {
		log.Fatal(e.Error())
	}
	return observationResponse.Properties.Observation, err
}

func getCity(c string) (string, error) {
	url := fmt.Sprintf("%v%v&limit=1&appid=%v", geocodeURL, c, os.Getenv("API_KEY"))
	response, err := getHttpResponse(url)

	if err != nil {
		return "", err
	}

	var cityResponse []struct {
		Lat  float64 `json:"lat"`
		Long float64 `json:"lon"`
	}
	json.Unmarshal(response, &cityResponse)
	return fmt.Sprintf("%v,%v", cityResponse[0].Lat, cityResponse[0].Long), nil
}

func getLocation(coords string) LocationResponse {
	url := fmt.Sprintf(getLocationByPoints, coords)
	response, _ := getHttpResponse(url)
	var location LocationDTO
	json.Unmarshal(response, &location)
	location.getObservationStation()
	lr := makeLocationResponse(location)
	return lr
}

func getAlerts(state string) (AlertResponse, error) {
	url := fmt.Sprintf("%v/%v", stateAlerts, strings.ToUpper(state))
	response, err := getHttpResponse(url)

	var alertResponse struct {
		Updated time.Time `json:"updated"`
		Alerts  []Alert   `json:"features"`
	}

	if jsonErr := json.Unmarshal(response, &alertResponse); jsonErr != nil {
		log.Fatal(jsonErr)
		return AlertResponse{}, errors.New(jsonErr.Error())
	}
	return AlertResponse{Updated: alertResponse.Updated, Alerts: alertResponse.Alerts}, err
}

func (locationDTO *LocationDTO) getObservationStation() {
	url := locationDTO.Properties.ObservationStationsUrl
	stations, err := getHttpResponse(url)

	if err != nil {
		log.Fatal(err)
	}

	var response struct {
		Features []struct {
			Properties struct {
				StationId string `json:"stationIdentifier"`
			} `json:"properties"`
		} `json:"features"`
	}
	if jsonErr := json.Unmarshal(stations, &response); jsonErr != nil {
		log.Fatal(jsonErr)
	}

	station := response.Features[0].Properties.StationId
	locationDTO.ObservationStation = station
}

func getHttpResponse(url string) ([]byte, error) {
	resp, err := http.Get(url)

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(body))
	} else {
		return body, nil
	}
}
