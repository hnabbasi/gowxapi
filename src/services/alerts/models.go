package services

import "time"

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
