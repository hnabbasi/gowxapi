package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	handlers "github.com/hnabbasi/gowxapi/controllers"
	"github.com/joho/godotenv"
)

func main() {
	loadEnv()
	setupServer()
}

func loadEnv() {
	if err := godotenv.Load("../local.env"); err != nil {
		log.Fatal(err.Error())
	}
}

func setupServer() {
	router := gin.Default()
	setupRoutes(router)
	router.Run(getPort())
}

func setupRoutes(router *gin.Engine) {
	router.GET("/api", handlers.Home)
	router.GET("/weather/:cityState", handlers.GetWeather)
	router.GET("/alerts/:state", handlers.GetAlertsForState)
}

func getPort() string {
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		return fmt.Sprintf(":%v", val)
	}
	return ":8080"
}
