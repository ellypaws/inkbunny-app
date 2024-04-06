package main

import (
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

// LoggedInMiddleware is a middleware for echo that checks if the "sid" cookie is set.
// Then it checks if the session ID is valid.
func LoggedInMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sid, err := c.Cookie("sid")
		if err != nil {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "missing sid cookie"})
		}
		if sid == nil || sid.Value == "" {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "empty sid cookie"})
		}
		if !database.ValidSID(api.Credentials{Sid: sid.Value}) {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "invalid sid"})
		}
		return next(c)
	}
}

var loggedInMiddleware = []echo.MiddlewareFunc{LoggedInMiddleware}

func StaffMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sid, err := c.Cookie("sid")
		if err != nil {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "missing sid cookie"})
		}
		if sid == nil || sid.Value == "" {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "empty sid cookie"})
		}
		id, err := database.GetUserIDFromSID(sid.Value)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "invalid sid"})
		}
		i, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "invalid sid"})
		}
		if !database.IsAuditorRole(i) {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "not authorized"})
		}
		return next(c)
	}
}

var staffMiddleware = []echo.MiddlewareFunc{LoggedInMiddleware, StaffMiddleware}
