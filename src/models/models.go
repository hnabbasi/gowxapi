package models

import (
	"time"
)

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
	Count   int       `json:"count"`
	Alerts  []Alert   `json:"alerts"`
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
	LatestObservations Observation
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

func MakeLocationResponse(locationDTO LocationDTO) LocationResponse {
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
		ObservationStation:     locationDTO.ObservationStation,
	}
}

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

type DailyForecast struct {
	Date            time.Time
	TemperatureUnit string
	DayTemp         int
	DayIcon         string
	NightTemp       int
	NightIcon       string
}

type Observation struct {
	Timestamp        time.Time `json:"timestamp"`
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

type Product struct {
	Id           string `json:"@id"`
	IssuanceTime string `json:"issuanceTime"`
	Text         string `json:"productText"`
}

type WeatherResponse struct {
	Updated time.Time `json:"updated"`
	LocationResponse
	Alerts struct {
		AlertResponse
	} `json:"alerts"`
	Observation `json:"latestObservations"`
	Hourly      []Period        `json:"hourly"`
	Daily       []DailyForecast `json:"daily"`
	RainChances struct {
		UnitCode string           `json:"unitCode"`
		Values   map[string][]int `json:"values"`
	} `json:"rainChances"`
	AreaForecastDiscussion Product `json:"areaForecastDiscussion"`
}
