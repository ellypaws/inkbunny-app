package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

var headHandlers = pathHandler{
	"/": handler{head, withCache},
}

func head(c echo.Context) error {
	return c.String(http.StatusOK, "200")
}
