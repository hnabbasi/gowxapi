package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	baseURL                        = "https://api.weather.gov"
	geocodeURL                     = "http://api.openweathermap.org/geo/1.0/direct?q="
	stateAlerts                    = baseURL + "/alerts/active/area"
	getLatestObservationsByStation = baseURL + "/stations/%v/observations/latest"
	getLocationByPoints            = baseURL + "/points/%v"
	apiKey                         = "297532c551bed3f6704e6d6db1ff7b64"
)

var wg = sync.WaitGroup{}

func main() {
	setupServer()
}

func setupServer() {
	router := gin.Default()
	setupRoutes(router)
	router.Run("localhost:8080")
}

func setupRoutes(router *gin.Engine) {
	router.GET("/", home)
	router.GET("/weather/:cityState", getWeather)
	router.GET("/alerts/:state", getAlertsForState)
}

func home(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, "Welcome to Hussain's Weather API")
}

func getWeather(c *gin.Context) {
	var weatherResponse WeatherResponse
	wg.Add(1)

	coords, err := getCity(c.Param("cityState"))
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, err.Error())
	}
	// get location details
	go getLocation(coords, &weatherResponse)
	// if err != nil {
	// 	c.IndentedJSON(http.StatusBadRequest, fmt.Sprintf("Location not found for %v", c.Param("cityState")))
	// }

	// get current conditions / observations
	// get hourly forecast
	// get weekly forecast
	// get rain chances

	wg.Wait()
	c.IndentedJSON(http.StatusOK, weatherResponse)
}

func getCity(c string) (string, error) {
	url := fmt.Sprintf("%v%v&limit=1&appid=%v", geocodeURL, c, apiKey)
	log.Printf("Get city: %v", url)
	response, err := getHttpResponse(url)
	if err != nil {
		log.Fatal(err)
		return "", errors.New(err.Error())
	}

	var cityResponse []struct {
		Lat  float64 `json:"lat"`
		Long float64 `json:"lon"`
	}

	if e := json.Unmarshal(response, &cityResponse); e != nil {
		log.Fatal(e)
		return "", errors.New(err.Error())
	}
	return fmt.Sprintf("%v,%v", cityResponse[0].Lat, cityResponse[0].Long), nil
}

func getLocation(coords string, weatherResponse *WeatherResponse) {
	url := fmt.Sprintf(getLocationByPoints, coords)
	log.Printf("Get location url: %v", url)

	response, err := getHttpResponse(url)
	if err != nil {
		log.Fatal(err)
		// return LocationResponse{}, errors.New(err.Error())
	}

	var location LocationDTO
	if jsonErr := json.Unmarshal(response, &location); jsonErr != nil {
		log.Fatal(jsonErr)
		// return LocationResponse{}, errors.New(err.Error())
	}

	// return makeLocationResponse(location), nil
	weatherResponse.LocationResponse = makeLocationResponse(location)
	wg.Done()
}

func getAlertsForState(c *gin.Context) {
	state := c.Param("state")
	url := fmt.Sprintf("%v/%v", stateAlerts, strings.ToUpper(state))
	log.Println(url)

	response, err := getHttpResponse(url)

	if err != nil {
		log.Fatal(err)
		c.IndentedJSON(http.StatusInternalServerError, "Could not get response")
	}

	var alertResponse struct {
		Updated string  `json:"updated"`
		Alerts  []Alert `json:"features"`
	}

	if jsonErr := json.Unmarshal(response, &alertResponse); jsonErr != nil {
		log.Fatal(jsonErr)
		c.IndentedJSON(http.StatusInternalServerError, "Could not get alerts")
	}

	c.IndentedJSON(http.StatusOK, AlertResponse{
		Updated: alertResponse.Updated,
		Alerts:  alertResponse.Alerts})
}

func getHttpResponse(url string) ([]byte, error) {
	resp, err := http.Get(url)

	if err != nil {
		log.Fatal(err)
		return nil, errors.New(err.Error())
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	return body, err
}

type Alert struct {
	Id         string `json:"id,omitempty"`
	Properties struct {
		Event         string `json:"event,omitempty"`
		Status        string `json:"status,omitempty"`
		Effective     string `json:"effective,omitempty"`
		Expires       string `json:"expires,omitempty"`
		Severity      string `json:"severity,omitempty"`
		Headline      string `json:"headline,omitempty"`
		AffectedAreas string `json:"areaDesc,omitempty"`
	} `json:"properties,omitempty"`
}

type AlertResponse struct {
	Updated string  `json:"updated"`
	Alerts  []Alert `json:"alerts"`
}

type LocationDTO struct {
	Id         string `json:"id,omitempty"`
	Properties struct {
		CountyWarningArea      string `json:"cwa,omitempty"`
		GridId                 string `json:"gridId,omitempty"`
		GridX                  int    `json:"gridX,omitempty"`
		GridY                  int    `json:"gridY,omitempty"`
		ObservationStationsUrl string `json:"observationStations,omitempty"`
		RelativeLocation       struct {
			Geometry struct {
				Point       string    `json:"type,omitempty"`
				Coordinates []float64 `json:"coordinates,omitempty"`
			} `json:"geometry,omitempty"`
			Properties struct {
				City  string `json:"city,omitempty"`
				State string `json:"state,omitempty"`
			} `json:"properties,omitempty"`
		} `json:"relativeLocation,omitempty"`
		ForecastGridDataUrl string `json:"forecastGridData,omitempty"`
		ForecastUrl         string `json:"forecast,omitempty"`
		HourlyForecastUrl   string `json:"forecastHourly,omitempty"`
		TimeZone            string `json:"timeZone,omitempty"`
		County              string `json:"county,omitempty"`
		ZoneForecast        string `json:"forecastZone,omitempty"`
		FireWeatherZone     string `json:"fireWeatherZone,omitempty"`
		RadarStationUrl     string `json:"radarStation,omitempty"`
	} `json:"properties,omitempty"`

	ObservationStation string
}

func (locationDTO *LocationDTO) getObservationStation() string {
	url := locationDTO.Properties.ObservationStationsUrl
	stations, err := getHttpResponse(url)

	if err != nil {
		log.Fatal(err)
		return "Get observation station failed."
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
	return station
}

func (locationDTO *LocationDTO) toString() string {
	str := ""

	str += fmt.Sprintf("Id: %v\n", locationDTO.Id)
	str += fmt.Sprintf("CountyWarningArea: %v\n", locationDTO.Properties.CountyWarningArea)
	str += fmt.Sprintf("GridId: %v\n", locationDTO.Properties.GridId)
	str += fmt.Sprintf("GridX: %v\n", locationDTO.Properties.GridX)
	str += fmt.Sprintf("GridY: %v\n", locationDTO.Properties.GridY)
	str += fmt.Sprintf("ObservationStationsUrl: %v\n", locationDTO.Properties.ObservationStationsUrl)
	str += fmt.Sprintf("Coordinates: %v\n", locationDTO.Properties.RelativeLocation.Geometry.Coordinates)
	str += fmt.Sprintf("City: %v\n", locationDTO.Properties.RelativeLocation.Properties.City)
	str += fmt.Sprintf("State: %v\n", locationDTO.Properties.RelativeLocation.Properties.State)
	str += fmt.Sprintf("ForecastGridDataUrl: %v\n", locationDTO.Properties.ForecastGridDataUrl)
	str += fmt.Sprintf("ForecastUrl: %v\n", locationDTO.Properties.ForecastUrl)
	str += fmt.Sprintf("HourlyForecastUrl: %v\n", locationDTO.Properties.HourlyForecastUrl)
	str += fmt.Sprintf("TimeZone: %v\n", locationDTO.Properties.TimeZone)
	str += fmt.Sprintf("County: %v\n", locationDTO.Properties.County)
	str += fmt.Sprintf("ZoneForecast: %v\n", locationDTO.Properties.ZoneForecast)
	str += fmt.Sprintf("FireWeatherZone: %v\n", locationDTO.Properties.FireWeatherZone)
	str += fmt.Sprintf("RadarStationUrl: %v\n", locationDTO.Properties.RadarStationUrl)
	str += fmt.Sprintf("ObservationStation: %v\n", locationDTO.getObservationStation())

	return str
}

type LocationResponse struct {
	Id                     string    `json:"id"`
	City                   string    `json:"city"`
	State                  string    `json:"state"`
	Coordinates            []float64 `json:"coordinates"`
	CountyWarningArea      string    `json:"cwa"`
	GridId                 string    `json:"gridId"`
	GridX                  int       `json:"gridX"`
	GridY                  int       `json:"gridY"`
	ObservationStationsUrl string    `json:"observationStations"`
	ForecastGridDataUrl    string    `json:"forecastGridData"`
	ForecastUrl            string    `json:"forecast"`
	HourlyForecastUrl      string    `json:"forecastHourly"`
	TimeZone               string    `json:"timeZone"`
	County                 string    `json:"county"`
	ZoneForecast           string    `json:"forecastZone"`
	FireWeatherZone        string    `json:"fireWeatherZone"`
	RadarStationUrl        string    `json:"radarStation"`
	ObservationStation     string    `json:"observationStation"`
}

func makeLocationResponse(locationDTO LocationDTO) LocationResponse {
	return LocationResponse{
		Id:                     locationDTO.Id,
		City:                   locationDTO.Properties.RelativeLocation.Properties.City,
		State:                  locationDTO.Properties.RelativeLocation.Properties.State,
		Coordinates:            locationDTO.Properties.RelativeLocation.Geometry.Coordinates,
		CountyWarningArea:      locationDTO.Properties.CountyWarningArea,
		GridId:                 locationDTO.Properties.GridId,
		GridX:                  locationDTO.Properties.GridX,
		GridY:                  locationDTO.Properties.GridY,
		ObservationStationsUrl: locationDTO.Properties.ObservationStationsUrl,
		ForecastGridDataUrl:    locationDTO.Properties.ForecastGridDataUrl,
		ForecastUrl:            locationDTO.Properties.ForecastUrl,
		HourlyForecastUrl:      locationDTO.Properties.HourlyForecastUrl,
		TimeZone:               locationDTO.Properties.TimeZone,
		County:                 locationDTO.Properties.County,
		ZoneForecast:           locationDTO.Properties.ZoneForecast,
		FireWeatherZone:        locationDTO.Properties.FireWeatherZone,
		RadarStationUrl:        locationDTO.Properties.RadarStationUrl,
		ObservationStation:     locationDTO.getObservationStation()}
}

// begin: Period

type Period struct {
	Number           int
	Name             string
	StartTime        string
	EndTime          string
	IsDaytime        bool
	Temperature      string
	TemperatureUnit  string
	TemperatureTrend string
	WindSpeed        string
	WindDirection    string
	Icon             string
	ShortForecast    string
	DetailedForecast string
}

func (p Period) GetWeatherString() string {
	switch p.ShortForecast {
	case "Slight Chance Rain Showers":
		return "Mostly Cloudy"
	case "Slight Chance Showers And Thunderstorms":
		return "Thunderstorms"
	default:
		return p.ShortForecast
	}
}

// end: Period

// begin: DailyForecast

type DailyForecast struct {
	Date      time.Time
	DayTemp   int
	DayIcon   string
	NightTemp int
	NightIcon string
}

func (f *DailyForecast) getName() string {
	if f.Date.Day() == time.Now().Day() {
		return "Today"
	} else {
		return f.Date.Format("dddd")
	}

}

func (f *DailyForecast) SetDay(temp int, icon string) {
	f.DayTemp = temp
	f.DayIcon = icon
}

func (f *DailyForecast) SetNight(temp int, icon string) {
	f.NightTemp = temp
	f.NightIcon = icon
}

// end: DailyForecast

type WeatherResponse struct {
	AlertResponse
	LocationResponse
	Hourly []Period
	Weekly []DailyForecast
}
