package main

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

var getRoutes = map[string]func(c echo.Context) error{
	"/": Hello,
}

func registerGetRoutes(e *echo.Echo) {
	for path, handler := range postRoutes {
		e.GET(path, handler)
	}
}

func Hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
