package main

import (
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	"net/http"
)

var postRoutes = map[string]func(c echo.Context) error{
	"/login": login,
}

func registerPostRoutes(e *echo.Echo) {
	for path, handler := range postRoutes {
		e.POST(path, handler)
	}
}

func login(c echo.Context) error {
	var loginRequest LoginRequest
	if err := c.Bind(&loginRequest); err != nil {
		return err
	}
	user := &api.Credentials{
		Username: loginRequest.Username,
		Password: loginRequest.Password,
	}
	user, err := user.Login()
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, user)
}
