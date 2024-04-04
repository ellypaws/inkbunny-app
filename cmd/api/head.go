package main

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

var headHandlers = pathHandler{
	"/": head,
}

func head(c echo.Context) error {
	return c.String(http.StatusOK, "200")
}