package services

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

	models "github.com/hnabbasi/gowxapi/models"

	"github.com/kaz-yamam0t0/go-timeparser/timeparser"
)

const (
	baseURL                        = "https://api.weather.gov"
	geocodeURL01                   = "https://geocode-api.arcgis.com/arcgis/rest/services/World/GeocodeServer/findAddressCandidates?f=pjson&singleLine=%v&token=%v"
	stateAlerts                    = baseURL + "/alerts/active/area"
	getLatestObservationsByStation = baseURL + "/stations/%v/observations/latest?require_qc=true"
	getLocationByPoints            = baseURL + "/points/%v"
	getAfdByLocation0              = baseURL + "/products/types/AFD/locations/%v"
)

// Get all alerts for state by state code e.g. TX
func GetAlertsForState(state string) (models.AlertResponse, error) {
	alerts, err := getAlerts(state)

	if err != nil {
		return models.AlertResponse{}, errors.New("failed to get alerts")
	}
	return alerts, nil
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
func GetWeather(cityState string) (models.WeatherResponse, error) {
	var weatherResponse models.WeatherResponse
	errorsChannel := make(chan error)
	doneChannel := make(chan bool)
	coordsChannel := make(chan string)
	locationChannel := make(chan models.LocationResponse)
	alertsChannel := make(chan models.AlertResponse)
	observationChannel := make(chan models.Observation)
	periodsChannel := make(chan []models.Period)
	weeklyChannel := make(chan []models.DailyForecast)
	rainChannel := make(chan map[string][]int)
	productChannel := make(chan models.Product)

	wg := sync.WaitGroup{}
	wg.Add(8)

	go func(coordsChannel chan string, errorsChannel chan error) {
		cityCoords, err := getCity(cityState)
		if err != nil {
			log.Fatal(err)
			errorsChannel <- errors.New(fmt.Sprintf("Could not find city for coords %v.", cityCoords))
		} else {
			coordsChannel <- cityCoords
			close(coordsChannel)
		}
		wg.Done()
	}(coordsChannel, errorsChannel)

	go func(locationChannel chan models.LocationResponse, errorsChannel chan error) {
		location, err := getLocation(<-coordsChannel)
		if err != nil {
			log.Fatal(err)
			errorsChannel <- errors.New("Could not find location.")
		} else {
			locationChannel <- location
			close(locationChannel)
		}
		wg.Done()
	}(locationChannel, errorsChannel)
	weatherResponse.LocationResponse = <-locationChannel

	go func(alertsChannel chan models.AlertResponse) {
		alerts, err := getAlerts(weatherResponse.LocationResponse.State)
		if err != nil {
			errorsChannel <- errors.New(fmt.Sprintf("Could not pull alerts for %v. Error:%v", weatherResponse.LocationResponse.State, err.Error()))
		} else {
			alertsChannel <- alerts
			close(alertsChannel)
		}
		wg.Done()
	}(alertsChannel)
	weatherResponse.AlertResponse = <-alertsChannel

	go func(observationChannel chan models.Observation) {
		url := fmt.Sprintf(getLatestObservationsByStation, weatherResponse.LocationResponse.ObservationStation)
		observations, err := getCurrentConditions(url)
		if err != nil {
			errorsChannel <- errors.New(fmt.Sprintf("Could not get latest conditions. Error:%v", err.Error()))
		} else {
			observationChannel <- observations
			close(observationChannel)
		}
		wg.Done()
	}(observationChannel)
	weatherResponse.Observation = <-observationChannel

	go func(periodsChannel chan []models.Period) {
		hourly, err := getPeriods(weatherResponse.LocationResponse.HourlyForecastUrl, 24)
		if err != nil {
			errorsChannel <- errors.New(fmt.Sprintf("Could not get hourly conditions. Error:%v", err.Error()))
		} else {
			periodsChannel <- hourly
			close(periodsChannel)
		}
		wg.Done()
	}(periodsChannel)
	weatherResponse.Hourly = <-periodsChannel

	go func(weeklyChannel chan []models.DailyForecast) {
		weekly, err := getWeekly(weatherResponse.LocationResponse.ForecastUrl)
		if err != nil {
			errorsChannel <- errors.New(fmt.Sprintf("Could not get weekly conditions. Error:%v", err.Error()))
		} else {
			weeklyChannel <- weekly
			close(weeklyChannel)
		}
		wg.Done()
	}(weeklyChannel)
	weatherResponse.Weekly = <-weeklyChannel

	go func(rainChannel chan map[string][]int) {
		rainChances, err := getRainChancesMap(weatherResponse.LocationResponse.ForecastGridDataUrl)
		if err != nil {
			errorsChannel <- errors.New(fmt.Sprintf("Could not get rain chances. Error:%v", err.Error()))
		} else {
			rainChannel <- rainChances
			close(rainChannel)
		}
		wg.Done()
	}(rainChannel)
	weatherResponse.RainChances.UnitCode = "wmoUnit:percent"
	weatherResponse.RainChances.Values = <-rainChannel

	go func(productChannel chan models.Product) {
		product, err := getAfdProduct(fmt.Sprintf(getAfdByLocation0, weatherResponse.CountyWarningArea))
		if err != nil {
			errorsChannel <- errors.New(fmt.Sprintf("Could not get forecast discussion. Error:%v", err.Error()))
		} else {
			productChannel <- product
			close(productChannel)
		}
		wg.Done()
	}(productChannel)
	weatherResponse.AreaForecastDiscussion = <-productChannel

	go func() {
		wg.Wait()
		close(doneChannel)
	}()

	select {
	case <-doneChannel:
		break
	case err := <-errorsChannel:
		close(errorsChannel)
		return models.WeatherResponse{}, err
	}
	return weatherResponse, nil
}

func getAfdProduct(url string) (models.Product, error) {
	response, err := getHttpResponse(url)
	if err != nil {
		log.Fatal(err)
		return models.Product{}, err
	}

	var allProductsResponse struct {
		Graph [1]struct {
			Id string `json:"@id"`
		} `json:"@graph"`
	}
	if er := json.Unmarshal(response, &allProductsResponse); er != nil {
		return models.Product{}, er
	}

	productResponse, err := getHttpResponse(string(allProductsResponse.Graph[0].Id))
	if err != nil {
		return models.Product{}, err
	}

	var product models.Product
	if err = json.Unmarshal(productResponse, &product); err != nil {
		return models.Product{}, err
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
				Values []models.ValueItem `json:"values"`
			} `json:"probabilityOfPrecipitation"`
		} `json:"properties"`
	}

	if e := json.Unmarshal(response, &rainChancesResponse); e != nil {
		log.Fatal(e.Error())
	}

	return fillPeriods(rainChancesResponse.Properties.Chances.Values)
}

func fillPeriods(periods []models.ValueItem) (map[string][]int, error) {
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

func getWeekly(url string) ([]models.DailyForecast, error) {
	response, err := getPeriods(url, 0)

	dailyMap := make(map[int]models.DailyForecast)

	for _, period := range response {
		day, exists := dailyMap[period.StartTime.Day()]

		if !exists {
			day = models.DailyForecast{Date: period.StartTime, TemperatureUnit: "F"}
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

	weekly := []models.DailyForecast{}
	for _, forecast := range dailyMap {
		weekly = append(weekly, forecast)
	}
	sort.SliceStable(weekly, func(i, j int) bool {
		return weekly[i].Date.Day() < weekly[j].Date.Day()
	})
	return weekly, err
}

func getPeriods(url string, count int) ([]models.Period, error) {
	response, err := getHttpResponse(url)

	if err != nil {
		return nil, err
	}

	var periodsResponse struct {
		Properties struct {
			Period []models.Period `json:"periods"`
		} `json:"properties"`
	}
	err = json.Unmarshal(response, &periodsResponse)
	if err != nil {
		return nil, err
	}

	if count > 0 {
		return periodsResponse.Properties.Period[:count], nil
	} else {
		return periodsResponse.Properties.Period, nil
	}
}

func getCurrentConditions(url string) (models.Observation, error) {
	response, err := getHttpResponse(url)
	if err != nil {
		return models.Observation{}, err
	}

	var observationResponse struct {
		Properties struct {
			models.Observation
		} `json:"properties"`
	}
	err = json.Unmarshal(response, &observationResponse)
	if err != nil {
		return models.Observation{}, err
	}
	return observationResponse.Properties.Observation, nil
}

func getCity(c string) (string, error) {
	key, ok := os.LookupEnv("API_KEY")

	if !ok {
		return "", errors.New("API Key for weather service not found")
	}

	url := fmt.Sprintf(geocodeURL01, c, key)
	response, err := getHttpResponse(url)

	if err != nil {
		return "", err
	}
	var cityResponse struct {
		Candidates [1]struct {
			Location struct {
				Lat  float64 `json:"x"`
				Long float64 `json:"y"`
			} `json:"location"`
		} `json:"candidates"`
	}
	err = json.Unmarshal(response, &cityResponse)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v,%v", cityResponse.Candidates[0].Location.Long, cityResponse.Candidates[0].Location.Lat), nil
}

func getLocation(coords string) (models.LocationResponse, error) {
	url := fmt.Sprintf(getLocationByPoints, coords)
	response, err := getHttpResponse(url)

	if err != nil {
		return models.LocationResponse{}, err
	}

	var location models.LocationDTO
	err = json.Unmarshal(response, &location)
	if err != nil {
		return models.LocationResponse{}, err
	}
	getObservationStation(location.Properties.ObservationStationsUrl, &location)
	lr := models.MakeLocationResponse(location)
	return lr, nil
}

func getAlerts(state string) (models.AlertResponse, error) {
	url := fmt.Sprintf("%v/%v", stateAlerts, strings.ToUpper(state))
	response, err := getHttpResponse(url)

	var alertResponse struct {
		Updated time.Time      `json:"updated"`
		Alerts  []models.Alert `json:"features"`
	}

	if jsonErr := json.Unmarshal(response, &alertResponse); jsonErr != nil {
		log.Fatal(jsonErr)
		return models.AlertResponse{}, errors.New(jsonErr.Error())
	}
	return models.AlertResponse{Updated: alertResponse.Updated, Alerts: alertResponse.Alerts}, err
}

func getObservationStation(stationUrl string, locationDTO *models.LocationDTO) {
	stations, err := getHttpResponse(stationUrl)

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
	}

	return body, nil
}
