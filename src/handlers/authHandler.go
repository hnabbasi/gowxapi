package handlers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	services "github.com/hnabbasi/gowxapi/services/auth"
	"net/http"
)

func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var userRequest services.UserRequest
		err := json.NewDecoder(c.Request.Body).Decode(&userRequest)

		if err != nil {
			c.JSON(http.StatusBadRequest, "invalid request")
			return
		}

		if token, err := services.Login(userRequest); err != nil {
			c.JSON(http.StatusUnauthorized, err.Error())
		} else {
			c.JSON(http.StatusOK, token)
		}
	}
}
