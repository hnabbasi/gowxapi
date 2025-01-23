package services

import "time"

type Token struct {
	AccessToken string        `json:"access_token"`
	ExpiresIn   time.Duration `json:"expires_in"`
}

type UserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
