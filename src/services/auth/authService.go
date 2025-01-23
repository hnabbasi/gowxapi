package services

import "errors"

func Login(request UserRequest) (Token, error) {
	return login(request.Username, request.Password)
}

func login(username, password string) (Token, error) {
	if username != "admin" || password != "password1" {
		return Token{}, errors.New("invalid credentials")
	}

	return Token{
		AccessToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IlNwYWNlIENpdHkgV2VhdGhlciIsImlhdCI6MTUxNjIzOTAyMn0.WM3GWd9c03OwVMq__WLpiT0p1kexZyHF5eFtJAgYyrA",
		ExpiresIn:   3600,
	}, nil
}
