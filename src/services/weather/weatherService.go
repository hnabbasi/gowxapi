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

	"github.com/hnabbasi/gowxapi/models"
	alerts "github.com/hnabbasi/gowxapi/services/alerts"

	"github.com/kaz-yamam0t0/go-timeparser/timeparser"
)

const (
	baseURL                        = "https://api.weather.gov"
	geocodeURL01                   = "https://geocode-api.arcgis.com/arcgis/rest/services/World/GeocodeServer/findAddressCandidates?f=pjson&singleLine=%v&token=%v"
	getLatestObservationsByStation = baseURL + "/stations/%v/observations/latest?require_qc=true"
	getLocationByPoints            = baseURL + "/points/%v"
	getAfdByLocation0              = baseURL + "/products/types/AFD/locations/%v"
)

// GetWeather Get complete weather information for a give city name e.g. Houston
// including:
// - Current conditions
// - Active alerts
// - Hourly conditions for next 24 hours
// - Daily conditions for next 7 days
// - Hourly rain chances for next 24 hours
// - Daily rain chances for next 7 days
// - Area forecast discussion
func GetWeather(cityState string) (models.WeatherResponse, error) {

	weatherResponse := models.WeatherResponse{}
	wg := sync.WaitGroup{}

	cityCoords, err := getCity(cityState)
	if err != nil {
		log.Println(err)
		return weatherResponse, errors.New(fmt.Sprintf("could not find city for coords %v", cityCoords))
	}

	location, err := getLocation(cityCoords)
	if err != nil {
		log.Println(err)
		return weatherResponse, errors.New("could not find location")
	}
	weatherResponse.LocationResponse = location

	wg.Add(6)
	go startAlertsRoutine(&wg, &weatherResponse)
	go startObservationsRoutine(&wg, &weatherResponse)
	go startHourlyRoutine(&wg, &weatherResponse)
	go startDailyRoutine(&wg, &weatherResponse)
	go startRainRoutine(&wg, &weatherResponse)
	go startAfdProductRoutine(&wg, &weatherResponse)
	wg.Wait()
	return weatherResponse, nil
}

// Goroutines

func startAlertsRoutine(wg *sync.WaitGroup, weatherResponse *models.WeatherResponse) {
	response, err := alerts.GetAlerts(weatherResponse.LocationResponse.State)
	if err != nil {
		log.Printf(fmt.Sprintf("Could not pull alerts for %v. Error:%v", weatherResponse.LocationResponse.State, err.Error()))
	} else {
		weatherResponse.AlertResponse = response
	}
	wg.Done()
}

func startObservationsRoutine(wg *sync.WaitGroup, weatherResponse *models.WeatherResponse) {
	url := fmt.Sprintf(getLatestObservationsByStation, weatherResponse.LocationResponse.ObservationStation)
	observations, err := getCurrentConditions(url)
	if err != nil {
		log.Printf(fmt.Sprintf("Could not get latest conditions. Error:%v", err.Error()))
	} else {
		weatherResponse.Observation = observations
	}
	wg.Done()
}

func startHourlyRoutine(wg *sync.WaitGroup, weatherResponse *models.WeatherResponse) {
	hourly, err := getPeriods(weatherResponse.LocationResponse.HourlyForecastUrl, 24)
	if err != nil {
		log.Printf(fmt.Sprintf("Could not get hourly conditions. Error:%v", err.Error()))
	} else {
		weatherResponse.Hourly = hourly
	}
	wg.Done()
}

func startDailyRoutine(wg *sync.WaitGroup, weatherResponse *models.WeatherResponse) {
	daily, err := getDaily(weatherResponse.LocationResponse.ForecastUrl)
	if err != nil {
		log.Printf(fmt.Sprintf("Could not get daily conditions. Error:%v", err.Error()))
	} else {
		weatherResponse.Daily = daily
	}
	wg.Done()
}

func startRainRoutine(wg *sync.WaitGroup, weatherResponse *models.WeatherResponse) {
	rainChances, err := getRainChancesMap(weatherResponse.LocationResponse.ForecastGridDataUrl)
	if err != nil {
		log.Printf(fmt.Sprintf("Could not get rain chances. Error:%v", err.Error()))
	} else {
		weatherResponse.RainChances.UnitCode = "wmoUnit:percent"
		weatherResponse.RainChances.Values = rainChances
	}
	wg.Done()
}

func startAfdProductRoutine(wg *sync.WaitGroup, weatherResponse *models.WeatherResponse) {
	product, err := getAfdProduct(fmt.Sprintf(getAfdByLocation0, weatherResponse.CountyWarningArea))
	if err != nil {
		log.Printf(fmt.Sprintf("Could not get forecast discussion. Error:%v", err.Error()))
	} else {
		weatherResponse.AreaForecastDiscussion = product
	}
	wg.Done()
}

// Methods

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

func getDaily(url string) ([]models.DailyForecast, error) {
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

	var daily []models.DailyForecast
	for _, forecast := range dailyMap {
		daily = append(daily, forecast)
	}
	sort.SliceStable(daily, func(i, j int) bool {
		return daily[i].Date.Day() < daily[j].Date.Day()
	})
	return daily, err
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

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err.Error())
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(body))
	}

	return body, nil
}
