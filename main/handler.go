package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	baseURL                        = "https://api.weather.gov"
	stateAlerts                    = baseURL + "/alerts/active/area"
	getLatestObservationsByStation = baseURL + "/stations/%v/observations/latest"
	getLocationByPoints            = baseURL + "/points/%v"
)

func main() {
	router := gin.Default()
	router.GET("/location/:coords", getLocation)
	router.GET("/alerts/:state", getAlertsForState)
	router.Run("localhost:8080")
}

func getLocation(c *gin.Context) {
	coords := c.Param("coords")
	url := fmt.Sprintf(getLocationByPoints, coords)
	log.Println(url)

	response, err := getHttpResponse(url)
	if err != nil {
		log.Fatal(err)
		c.IndentedJSON(http.StatusInternalServerError, "Could not get response")
	}

	var location Location
	if jsonErr := json.Unmarshal(response, &location); jsonErr != nil {
		log.Fatal(jsonErr)
		c.IndentedJSON(http.StatusInternalServerError, "Could not find location")
	}

	c.IndentedJSON(http.StatusOK, location)
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
	var resp struct {
		Updated string  `json:"updated"`
		Alerts  []Alert `json:"alerts"`
	}

	resp.Updated = alertResponse.Updated
	resp.Alerts = alertResponse.Alerts

	c.IndentedJSON(http.StatusOK, resp)
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

type Location struct {
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

func (location *Location) getObservationStation() string {
	url := location.Properties.ObservationStationsUrl
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
	location.ObservationStation = station
	return station
}

func (location *Location) toString() string {
	str := ""

	str += fmt.Sprintf("Id: %v\n", location.Id)
	str += fmt.Sprintf("CountyWarningArea: %v\n", location.Properties.CountyWarningArea)
	str += fmt.Sprintf("GridId: %v\n", location.Properties.GridId)
	str += fmt.Sprintf("GridX: %v\n", location.Properties.GridX)
	str += fmt.Sprintf("GridY: %v\n", location.Properties.GridY)
	str += fmt.Sprintf("ObservationStationsUrl: %v\n", location.Properties.ObservationStationsUrl)
	str += fmt.Sprintf("Coordinates: %v\n", location.Properties.RelativeLocation.Geometry.Coordinates)
	str += fmt.Sprintf("City: %v\n", location.Properties.RelativeLocation.Properties.City)
	str += fmt.Sprintf("State: %v\n", location.Properties.RelativeLocation.Properties.State)
	str += fmt.Sprintf("ForecastGridDataUrl: %v\n", location.Properties.ForecastGridDataUrl)
	str += fmt.Sprintf("ForecastUrl: %v\n", location.Properties.ForecastUrl)
	str += fmt.Sprintf("HourlyForecastUrl: %v\n", location.Properties.HourlyForecastUrl)
	str += fmt.Sprintf("TimeZone: %v\n", location.Properties.TimeZone)
	str += fmt.Sprintf("County: %v\n", location.Properties.County)
	str += fmt.Sprintf("ZoneForecast: %v\n", location.Properties.ZoneForecast)
	str += fmt.Sprintf("FireWeatherZone: %v\n", location.Properties.FireWeatherZone)
	str += fmt.Sprintf("RadarStationUrl: %v\n", location.Properties.RadarStationUrl)
	str += fmt.Sprintf("ObservationStation: %v\n", location.getObservationStation())

	return str
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
