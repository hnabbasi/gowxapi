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
	"time"
)

const (
	baseURL                        = "https://api.weather.gov"
	stateAlerts                    = baseURL + "/alerts/active/area"
	getLatestObservationsByStation = baseURL + "/stations/%v/observations/latest"
	getLocationByPoints            = baseURL + "/points/%v"
)

func main() {
	getWeatherAlertsSample()
	// loc := getLocation("29.7608,-95.3695")
	// fmt.Println(loc.toString())
}

func getLocation(coords string) Location {
	url := fmt.Sprintf(getLocationByPoints, coords)
	log.Println(url)

	response, err := getHttpResponse(url)

	if err != nil {
		log.Fatal(err)
		return Location{}
	}

	var location Location

	if jsonErr := json.Unmarshal(response, &location); jsonErr != nil {
		log.Fatal(jsonErr)
		return Location{}
	}
	return location
}

func getWeatherAlertsSample() {
	log.SetFlags(0)

	fmt.Println("Welcome to Weather Conditions API")

	if len(os.Args) < 2 {
		log.Fatal(errors.New("state code missing"))
		os.Exit(1)
	}

	result, err := getAlertsForState(strings.ToUpper(os.Args[1]))

	if err != nil {
		log.Fatal(err)
	}
	n := len(result)
	hasDivider := n > 1
	log.Printf("\u250F")
	for i, alert := range result {
		log.Printf(" (%v) Areas affected: %v\n", i+1, alert.Properties.AffectedAreas)
		log.Printf(" Event: %v\n", alert.Properties.Event)
		log.Printf(" Headline: %v\n", alert.Properties.Headline)
		log.Printf(" Id: %v\n", alert.Id)
		if hasDivider && i != n-1 {
			log.Println("\u2523")
		}
	}
	log.Println("\u2517")
}

func getAlertsForState(state string) ([]Alert, error) {
	url := fmt.Sprintf("%v/%v", stateAlerts, state)
	log.Println(url)

	response, err := getHttpResponse(url)

	if err != nil {
		log.Fatal(err)
		return nil, errors.New(err.Error())
	}

	var alertResponse struct {
		Updated string  `json:"updated"`
		Alerts  []Alert `json:"features"`
	}

	if jsonErr := json.Unmarshal(response, &alertResponse); jsonErr != nil {
		log.Fatal(jsonErr)
		log.Println("Can not unmarshal JSON")
	}

	count := len(alertResponse.Alerts)
	if count == 0 {
		log.Printf("No alerts found for the state of %v\n\n", state)
	} else if count == 1 {
		log.Println("Found 1 alert")
	} else {
		log.Printf("Found %v alerts\n\n", count)
	}

	return alertResponse.Alerts, nil
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

	str += fmt.Sprintf("Id:\t%v\n", location.Id)
	str += fmt.Sprintf("CountyWarningArea:\t%v\n", location.Properties.CountyWarningArea)
	str += fmt.Sprintf("GridId:\t%v\n", location.Properties.GridId)
	str += fmt.Sprintf("GridX:\t%v\n", location.Properties.GridX)
	str += fmt.Sprintf("GridY:\t%v\n", location.Properties.GridY)
	str += fmt.Sprintf("ObservationStationsUrl:\t%v\n", location.Properties.ObservationStationsUrl)
	str += fmt.Sprintf("Coordinates:\t%v\n", location.Properties.RelativeLocation.Geometry.Coordinates)
	str += fmt.Sprintf("City:\t%v\n", location.Properties.RelativeLocation.Properties.City)
	str += fmt.Sprintf("State:\t%v\n", location.Properties.RelativeLocation.Properties.State)
	str += fmt.Sprintf("ForecastGridDataUrl:\t%v\n", location.Properties.ForecastGridDataUrl)
	str += fmt.Sprintf("ForecastUrl:\t%v\n", location.Properties.ForecastUrl)
	str += fmt.Sprintf("HourlyForecastUrl:\t%v\n", location.Properties.HourlyForecastUrl)
	str += fmt.Sprintf("TimeZone:\t%v\n", location.Properties.TimeZone)
	str += fmt.Sprintf("County:\t%v\n", location.Properties.County)
	str += fmt.Sprintf("ZoneForecast:\t%v\n", location.Properties.ZoneForecast)
	str += fmt.Sprintf("FireWeatherZone:\t%v\n", location.Properties.FireWeatherZone)
	str += fmt.Sprintf("RadarStationUrl:\t%v\n", location.Properties.RadarStationUrl)
	str += fmt.Sprintf("ObservationStation:\t%v\n", location.getObservationStation())

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
