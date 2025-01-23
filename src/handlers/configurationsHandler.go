package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	services "github.com/hnabbasi/gowxapi/services/configs"
	"net/http"
)

// GetConfigurations Get all configurations for the app
func GetConfigurations() gin.HandlerFunc {
	return func(c *gin.Context) {
		if configurations, err := services.GetConfigurations(); err != nil {
			c.JSON(http.StatusBadRequest, "Could not fetch configurations")
		} else {
			c.JSON(http.StatusOK, configurations)
		}
	}
}

func GetConfiguration() gin.HandlerFunc {
	return func(c *gin.Context) {
		if configurations, err := services.GetConfig(c.Param("key")); err != nil {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Could not fetch configurations. %s", err))
		} else {
			c.JSON(http.StatusOK, configurations)
		}
	}
}
