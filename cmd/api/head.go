package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

var headHandlers = pathHandler{
	"/": handler{head, withCache},
}

func head(c echo.Context) error {
	return c.String(http.StatusOK, "200")
}
