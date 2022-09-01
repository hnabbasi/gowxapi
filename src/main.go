package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/hnabbasi/gowxapi/handlers"
)

func main() {
	if err := setupServer(); err != nil {
		log.Println(err.Error())
	}
}

func setupServer() error {
	router := gin.Default()
	setupRoutes(router)
	return router.Run(getPort())
}

func setupRoutes(router *gin.Engine) {
	router.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, "\u26c5 Welcome to Hussain's Weather API")
	})
	router.GET("/weather/:cityState", handlers.GetWeather())
	router.GET("/alerts/:state", handlers.GetAlertsForState())
}

func getPort() string {
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		return fmt.Sprintf(":%v", val)
	}
	return ":8080"
}
