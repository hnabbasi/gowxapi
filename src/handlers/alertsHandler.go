package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	services "github.com/hnabbasi/gowxapi/services/alerts"
)

// Get all alerts for state by state code e.g. TX
func GetAlertsForState() gin.HandlerFunc {
	return func(c *gin.Context) {
		if alerts, err := services.GetAlerts(c.Param("state")); err != nil {
			c.JSON(http.StatusBadRequest, "Could not fetch alerts")
		} else {
			c.JSON(http.StatusOK, alerts)
		}
	}
}
