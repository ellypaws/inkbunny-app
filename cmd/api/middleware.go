package main

import (
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	"net/http"
)

// LoggedInMiddleware is a middleware for echo that checks if the "sid" cookie is set.
// Then it checks if the session ID is valid.
func LoggedInMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sid, err := c.Cookie("sid")
		if err != nil {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{Error: "missing sid cookie"})
		}
		if sid.Value == "" {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{Error: "empty sid cookie"})
		}
		if !database.ValidSID(api.Credentials{Sid: sid.Value}) {
			return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
		}
		return next(c)
	}
}

var loggedInMiddleware = []echo.MiddlewareFunc{LoggedInMiddleware}
