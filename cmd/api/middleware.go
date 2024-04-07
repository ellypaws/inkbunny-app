package main

import (
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	"net/http"
)

// LoggedInMiddleware is a middleware for echo that checks if the "sid" cookie is set.
// Then it checks if the session ID is valid.
func LoggedInMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sid, ok := c.Get("sid").(string)
		if !ok || sid == "" {
			xsid := c.Request().Header.Get("X-SID")
			if xsid != "" {
				sid = xsid
			}
		}
		if sid == "" {
			sidCookie, err := c.Cookie("sid")
			if err == nil && sidCookie.Value != "" {
				sid = sidCookie.Value
			}
		}

		if sid == "" {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "empty sid"})
		}

		if !database.ValidSID(api.Credentials{Sid: sid}) {

		}

		id, err := database.GetUserIDFromSID(sid)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "invalid sid"})
		}

		c.Set("sid", sid)
		c.Set("id", id)
		return next(c)
	}
}

var loggedInMiddleware = []echo.MiddlewareFunc{LoggedInMiddleware}

func RequireAuditor(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		_, id, err := GetSIDandID(c)

		auditor, err := database.GetAuditorByID(id)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "invalid user"})
		}

		if auditor.Username == "" {
			return c.JSON(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: "invalid user"})
		}

		if !auditor.Role.IsAuditor() {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "unauthorized"})
		}

		c.Set("auditor", &auditor)
		return next(c)
	}
}

func GetSIDandID(c echo.Context) (string, int64, error) {
	sid, ok := c.Get("sid").(string)
	if !ok || sid == "" {
		return "", -1, crashy.ErrorResponse{ErrorString: "empty sid"}
	}
	id, ok := c.Get("id").(int64)
	if !ok {
		return "", -1, crashy.ErrorResponse{ErrorString: "empty id"}
	}

	return sid, id, nil
}

func GetAuditor(c echo.Context) (auditor *db.Auditor, err error) {
	auditor, ok := c.Get("auditor").(*db.Auditor)
	if !ok || auditor == nil {
		return nil, crashy.ErrorResponse{ErrorString: "empty auditor"}
	}

	return auditor, nil
}

var staffMiddleware = []echo.MiddlewareFunc{LoggedInMiddleware, RequireAuditor}
