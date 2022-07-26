package controllers

import (
	"net/http"

	"github.com/hnabbasi/gowxapi/services"

	"github.com/gin-gonic/gin"
)

// Welcome message from the server
func Home(c *gin.Context) {
	c.JSON(http.StatusOK, "\u26c5 Welcome to Hussain's Weather API")
}

// Get all alerts for state by state code e.g. TX
func GetAlertsForState(c *gin.Context) {
	if alerts, err := services.GetAlertsForState(c.Param("state")); err != nil {
		c.JSON(http.StatusBadRequest, "Could not fetch alerts")
	} else {
		c.JSON(http.StatusOK, alerts)
	}
}

// Get complete weather information for a give city name e.g. Houston
// including:
// - Current conditions
// - Active alerts
// - Hourly conditions for next 24 hours
// - Weekly conditions for next 7 days
// - Hourly rain chances for next 24 hours
// - Daily rain chances for next 7 days
// - Area forecast discussion
func GetWeather(c *gin.Context) {
	if resp, err := services.GetWeather(c.Param("cityState")); err != nil {
		c.IndentedJSON(http.StatusBadRequest, err)
	} else {
		c.IndentedJSON(http.StatusOK, resp)
	}
}
