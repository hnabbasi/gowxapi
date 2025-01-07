package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	baseURL     = "https://api.weather.gov"
	stateAlerts = baseURL + "/alerts/active/area"
)

func GetAlerts(state string) (AlertResponse, error) {
	url := fmt.Sprintf("%v/%v", stateAlerts, strings.ToUpper(state))
	response, err := getHttpResponse(url)

	var alertResponse struct {
		Updated time.Time `json:"updated"`
		Alerts  []Alert   `json:"features"`
	}

	if jsonErr := json.Unmarshal(response, &alertResponse); jsonErr != nil {
		log.Println(jsonErr)
		return AlertResponse{}, errors.New(jsonErr.Error())
	}
	return AlertResponse{Updated: alertResponse.Updated, Count: len(alertResponse.Alerts), Alerts: alertResponse.Alerts}, err
}

func getHttpResponse(url string) ([]byte, error) {
	resp, err := http.Get(url)

	if err != nil {
		log.Println(err)
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
