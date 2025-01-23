package handlers

import (
	"github.com/gin-gonic/gin"
	services "github.com/hnabbasi/gowxapi/services/locations"
	"net/http"
)

// GetLocations Get all available locations for the app
func GetLocations() gin.HandlerFunc {
	return func(c *gin.Context) {
		if locations, err := services.GetLocations(); err != nil {
			c.JSON(http.StatusBadRequest, "Could not fetch locations")
		} else {
			c.JSON(http.StatusOK, locations)
		}
	}
}
