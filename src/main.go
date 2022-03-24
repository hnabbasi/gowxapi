package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hnabbasi/gowxapi/handlers"
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

	port, e := strconv.Atoi(os.Getenv("PORT"))
	if e != nil {
		port = 8080
	}

	server := os.Getenv("SERVER")
	if server == "" {
		server = "localhost"
	}

	router.Run(fmt.Sprintf("%v:%v", server, port))
}

func setupRoutes(router *gin.Engine) {
	router.GET("/", handlers.Home)
	router.GET("/weather/:cityState", handlers.GetWeather)
	router.GET("/alerts/:state", handlers.GetAlertsForState)
}
