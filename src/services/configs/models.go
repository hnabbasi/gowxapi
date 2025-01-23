package services

type Configuration struct {
	Key   string      `json:"key,omitempty"`
	Value interface{} `json:"value,omitempty"`
}
