package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

const (
	baseURL                        = "https://api.weather.gov"
	geocodeURL                     = "http://api.openweathermap.org/geo/1.0/direct?q="
	stateAlerts                    = baseURL + "/alerts/active/area"
	getLatestObservationsByStation = baseURL + "/stations/%v/observations/latest"
	getLocationByPoints            = baseURL + "/points/%v"
)

var (
	wg = sync.WaitGroup{}
)

func main() {
	loadEnv()
	setupServer()
}

func loadEnv() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Fatal(err.Error)
	}
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

func getAlertsForState(c *gin.Context) {
	state := c.Param("state")

	alerts, err := getAlerts(state)

	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, "Could not fetch alerts")
	}

	c.IndentedJSON(http.StatusOK, alerts)
}

func getWeather(c *gin.Context) {
	var weatherResponse WeatherResponse
	coordsChannel := make(chan string)
	locationChannel := make(chan LocationResponse)
	alertsChannel := make(chan AlertResponse)
	observationChannel := make(chan Observation)
	periodsChannel := make(chan []Period)

	wg.Add(5)

	go func(coordsChannel chan string) {
		coordsChannel <- getCity(c.Param("cityState"))
		wg.Done()
	}(coordsChannel)

	go func(locationChannel chan LocationResponse) {
		locationChannel <- getLocation(<-coordsChannel)
		wg.Done()
	}(locationChannel)
	weatherResponse.LocationResponse = <-locationChannel

	go func(alertsChannel chan AlertResponse) {
		alerts, _ := getAlerts(weatherResponse.LocationResponse.State)
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

	// get hourly forecast

	go func(periodsChannel chan []Period) {
		hourly, _ := getPeriods(weatherResponse.LocationResponse.HourlyForecastUrl)
		periodsChannel <- hourly
		wg.Done()
	}(periodsChannel)
	weatherResponse.Hourly = <-periodsChannel
	// get weekly forecast
	// go func(periodsChannel chan []Period) {
	// 	weekly, _ := getPeriods(weatherResponse.LocationResponse.ForecastUrl)
	// 	periodsChannel <- weekly
	// 	wg.Done()
	// }(periodsChannel)
	// weatherResponse.Weekly = <-periodsChannel
	// get rain chances

	wg.Wait()
	c.IndentedJSON(http.StatusOK, weatherResponse)
}

func getPeriods(url string) ([]Period, error) {
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
	return periodsResponse.Properties.Period, err
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

func getCity(c string) string {
	url := fmt.Sprintf("%v%v&limit=1&appid=%v", geocodeURL, c, os.Getenv("API_KEY"))
	response, _ := getHttpResponse(url)
	var cityResponse []struct {
		Lat  float64 `json:"lat"`
		Long float64 `json:"lon"`
	}
	json.Unmarshal(response, &cityResponse)
	return fmt.Sprintf("%v,%v", cityResponse[0].Lat, cityResponse[0].Long)
}

func getLocation(coords string) LocationResponse {
	url := fmt.Sprintf(getLocationByPoints, coords)
	response, _ := getHttpResponse(url)
	var location LocationDTO
	json.Unmarshal(response, &location)
	return makeLocationResponse(location)
}

func getAlerts(state string) (AlertResponse, error) {
	url := fmt.Sprintf("%v/%v", stateAlerts, strings.ToUpper(state))
	log.Println(url)
	response, err := getHttpResponse(url)

	var alertResponse struct {
		Updated string  `json:"updated"`
		Alerts  []Alert `json:"features"`
	}

	if jsonErr := json.Unmarshal(response, &alertResponse); jsonErr != nil {
		log.Fatal(jsonErr)
		return AlertResponse{}, errors.New(jsonErr.Error())
	}
	return AlertResponse{Updated: alertResponse.Updated, Alerts: alertResponse.Alerts}, err
}

func getHttpResponse(url string) ([]byte, error) {
	resp, err := http.Get(url)
	fmt.Println(url)

	if err != nil {
		log.Fatal(err)
		return nil, errors.New(err.Error())
	}

	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// Models

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

// begin: LocationDTO

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
	LatestObservations Observation
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

// end : LocationDTO
// begin: LocationResponse

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

// end: LocationResponse
// begin: Period

type Period struct {
	Number           int    `json:"number,omitempty"`
	Name             string `json:"name,omitempty"`
	StartTime        string `json:"startTime,omitempty"`
	EndTime          string `json:"endTime,omitempty"`
	IsDaytime        bool   `json:"isDayTime,omitempty"`
	Temperature      int    `json:"temperature,omitempty"`
	TemperatureUnit  string `json:"temperatureUnit,omitempty"`
	TemperatureTrend string `json:"temperatureTrend,omitempty"`
	WindSpeed        string `json:"windSpeed,omitempty"`
	WindDirection    string `json:"windDirection,omitempty"`
	Icon             string `json:"icon,omitempty"`
	ShortForecast    string `json:"shortForecast,omitempty"`
	DetailedForecast string `json:"detailedForecast,omitempty"`
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

type Observation struct {
	TextDescription string `json:"textDescription,omitempty"`
	Icon            string `json:"icon,omitempty"`
	Temperature     struct {
		UnitCode string  `json:"unitCode"`
		Value    float64 `json:"value,omitempty"`
	} `json:"temperature,omitempty"`
	Dewpoint struct {
		UnitCode string  `json:"unitCode"`
		Value    float64 `json:"value,omitempty"`
	} `json:"dewpoint,omitempty"`
	WindDirection struct {
		UnitCode string  `json:"unitCode"`
		Value    float64 `json:"value,omitempty"`
	} `json:"windDirection,omitempty"`
	WindSpeed struct {
		UnitCode string  `json:"unitCode"`
		Value    float64 `json:"value,omitempty"`
	} `json:"windSpeed,omitempty"`
	WindGust struct {
		UnitCode string  `json:"unitCode"`
		Value    float64 `json:"value,omitempty"`
	} `json:"windGust,omitempty"`
	Visibility struct {
		UnitCode string  `json:"unitCode"`
		Value    float64 `json:"value,omitempty"`
	} `json:"visibility,omitempty"`
	RelativeHumidity struct {
		UnitCode string  `json:"unitCode"`
		Value    float64 `json:"value,omitempty"`
	} `json:"relativeHumidity,omitempty"`
	WindChill struct {
		UnitCode string  `json:"unitCode"`
		Value    float64 `json:"value,omitempty"`
	} `json:"windChill,omitempty"`
	HeatIndex struct {
		UnitCode string  `json:"unitCode"`
		Value    float64 `json:"value,omitempty"`
	} `json:"heatIndex,omitempty"`
}

type WeatherResponse struct {
	AlertResponse
	LocationResponse
	Hourly      []Period        `json:"hourly"`
	Weekly      []DailyForecast `json:"weekly"`
	Observation `json:"latest_observations"`
}
