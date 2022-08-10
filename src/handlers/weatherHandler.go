package handlers

import (
	"net/http"

	services "github.com/hnabbasi/gowxapi/services/weather"

	"github.com/gin-gonic/gin"
)

// Get complete weather information for a give city name e.g. Houston
// including:
// - Current conditions
// - Active alerts
// - Hourly conditions for next 24 hours
// - Weekly conditions for next 7 days
// - Hourly rain chances for next 24 hours
// - Daily rain chances for next 7 days
// - Area forecast discussion
func GetWeather() gin.HandlerFunc {
	return func(c *gin.Context) {
		if resp, err := services.GetWeather(c.Param("cityState")); err != nil {
			c.IndentedJSON(http.StatusBadRequest, err)
		} else {
			c.IndentedJSON(http.StatusOK, resp)
		}
	}
}
