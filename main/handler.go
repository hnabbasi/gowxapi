package main

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
	"github.com/joho/godotenv"
	"github.com/kaz-yamam0t0/go-timeparser/timeparser"
)

const (
	baseURL                        = "https://api.weather.gov"
	geocodeURL                     = "http://api.openweathermap.org/geo/1.0/direct?q="
	stateAlerts                    = baseURL + "/alerts/active/area"
	getLatestObservationsByStation = baseURL + "/stations/%v/observations/latest?require_qc=true"
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
		log.Fatal(err.Error())
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
	c.IndentedJSON(http.StatusOK, "\u26c5 Welcome to Hussain's Weather API")
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
	weeklyChannel := make(chan []DailyForecast)
	rainChannel := make(chan map[int][]int)

	wg.Add(7)

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

	go func(rainChannel chan map[int][]int) {
		rainChances, _ := getRainChancesMap(weatherResponse.LocationResponse.ForecastGridDataUrl)
		rainChannel <- rainChances
		wg.Done()
	}(rainChannel)
	weatherResponse.RainChances = <-rainChannel

	// TODO: get rain chances

	wg.Wait()
	c.IndentedJSON(http.StatusOK, weatherResponse)
}

func getRainChancesMap(url string) (map[int][]int, error) {
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

func fillPeriods(periods []ValueItem) (map[int][]int, error) {
	retVal := make(map[int][]int)

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
			if _, e := retVal[current.Day()]; !e {
				retVal[current.Day()] = make([]int, 24)
			}
			retVal[current.Day()][current.Hour()] = int(v.Value)
			current = current.Add(time.Hour)
		}
	}
	return retVal, nil
}

func periodToDate(period string) time.Time {
	return time.Now()
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

func getHttpResponse(url string) ([]byte, error) {
	resp, err := http.Get(url)
	log.Println(url)

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
		Event         string    `json:"event,omitempty"`
		Status        string    `json:"status,omitempty"`
		Effective     time.Time `json:"effective,omitempty"`
		Expires       time.Time `json:"expires,omitempty"`
		Severity      string    `json:"severity,omitempty"`
		Headline      string    `json:"headline,omitempty"`
		AffectedAreas string    `json:"areaDesc,omitempty"`
	} `json:"properties,omitempty"`
}

type AlertResponse struct {
	Updated time.Time `json:"updated"`
	Alerts  []Alert   `json:"alerts"`
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
	Number           int       `json:"number,omitempty"`
	Name             string    `json:"name,omitempty"`
	StartTime        time.Time `json:"startTime,omitempty"`
	EndTime          time.Time `json:"endTime,omitempty"`
	IsDaytime        bool      `json:"isDayTime,omitempty"`
	Temperature      int       `json:"temperature,omitempty"`
	TemperatureUnit  string    `json:"temperatureUnit,omitempty"`
	TemperatureTrend string    `json:"temperatureTrend,omitempty"`
	WindSpeed        string    `json:"windSpeed,omitempty"`
	WindDirection    string    `json:"windDirection,omitempty"`
	Icon             string    `json:"icon,omitempty"`
	ShortForecast    string    `json:"shortForecast,omitempty"`
	DetailedForecast string    `json:"detailedForecast,omitempty"`
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
	Date            time.Time
	TemperatureUnit string
	DayTemp         int
	DayIcon         string
	NightTemp       int
	NightIcon       string
}

// end: DailyForecast

type Observation struct {
	TextDescription  string    `json:"textDescription,omitempty"`
	Icon             string    `json:"icon,omitempty"`
	Temperature      ValueItem `json:"temperature,omitempty"`
	Dewpoint         ValueItem `json:"dewpoint,omitempty"`
	WindDirection    ValueItem `json:"windDirection,omitempty"`
	WindSpeed        ValueItem `json:"windSpeed,omitempty"`
	WindGust         ValueItem `json:"windGust,omitempty"`
	Visibility       ValueItem `json:"visibility,omitempty"`
	RelativeHumidity ValueItem `json:"relativeHumidity,omitempty"`
	WindChill        ValueItem `json:"windChill,omitempty"`
	HeatIndex        ValueItem `json:"heatIndex,omitempty"`
}

type ValueItem struct {
	UnitCode  string  `json:"unitCode,omitempty"`
	ValidTime string  `json:"validTime,omitempty"`
	Value     float64 `json:"value,omitempty"`
}

type WeatherResponse struct {
	AlertResponse
	LocationResponse
	Hourly      []Period        `json:"hourly"`
	Weekly      []DailyForecast `json:"weekly"`
	Observation `json:"latest_observations"`
	RainChances map[int][]int `json:rainChances`
}
