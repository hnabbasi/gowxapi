package services

import (
	"errors"
	"fmt"
)

var configs map[string]string

func getConfigs() map[string]string {
	if configs == nil {
		configs = make(map[string]string)
		configs["posts_count"] = "5"
		configs["wx_source"] = "NWS"
	}
	return configs
}

func GetConfigurations() (map[string]string, error) {
	return getConfigs(), nil
}

func GetConfig(key string) (Configuration, error) {
	configs := getConfigs()
	config, exists := configs[key]

	if !exists {
		return Configuration{}, errors.New(fmt.Sprintf("Config %s does not exist", key))
	}

	return Configuration{key, config}, nil
}
