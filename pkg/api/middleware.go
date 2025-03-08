package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"

	"github.com/ellypaws/inkbunny-app/pkg/api/cache"
	"github.com/ellypaws/inkbunny-app/pkg/crashy"
	"github.com/ellypaws/inkbunny-app/pkg/db"
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

		if !Database.ValidSID(api.Credentials{Sid: sid}) {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "invalid sid"})
		}

		id, err := Database.GetUserIDFromSID(sid)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "invalid sid"})
		}

		c.Set("sid", sid)
		c.Set("id", id)
		return next(c)
	}
}

var loggedInMiddleware = []echo.MiddlewareFunc{LoggedInMiddleware}

func SIDMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		_ = fetchSID(c)
		return next(c)
	}
}

func fetchSID(c echo.Context) error {
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
		sid = c.QueryParam("sid")
	}

	c.Set("sid", sid)
	id, err := Database.GetUserIDFromSID(sid)
	if err == nil {
		c.Set("id", id)
	}
	return nil
}

func RequireSID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		_ = fetchSID(c)
		sid, ok := c.Get("sid").(string)
		if !ok || len(sid) == 0 {
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "empty sid"})
		}
		return next(c)
	}
}

func RequireAuditor(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		_, id, err := GetSIDandID(c)

		auditor, err := Database.GetAuditorByID(id)
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

func TryAuditor(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		_, id, err := GetSIDandID(c)
		if err != nil {
			c.Set("auditor", &AnonymousAuditor)
			return next(c)
		}

		auditor, err := Database.GetAuditorByID(id)
		if err != nil {
			c.Set("auditor", &AnonymousAuditor)
			return next(c)
		}

		if auditor.Username == "" {
			c.Set("auditor", &AnonymousAuditor)
			return next(c)
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

func GetSID(c echo.Context) (string, error) {
	sid, ok := c.Get("sid").(string)
	if !ok || sid == "" {
		return "", crashy.ErrorResponse{ErrorString: "empty sid"}
	}

	return sid, nil
}

func GetCurrentAuditor(c echo.Context) (auditor *db.Auditor, err error) {
	auditor, ok := c.Get("auditor").(*db.Auditor)
	if !ok || auditor == nil {
		return nil, crashy.ErrorResponse{ErrorString: "empty auditor"}
	}

	return auditor, nil
}

var AnonymousAuditor = db.Auditor{
	Username: "anonymous",
	Role:     db.RoleAuditor,
}

func Anonymous(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set("auditor", &AnonymousAuditor)
		c.Set("id", int64(0))
		return next(c)
	}
}

var staffMiddleware = []echo.MiddlewareFunc{LoggedInMiddleware, RequireAuditor}

var reducedMiddleware = []echo.MiddlewareFunc{RequireSID, TryAuditor}

var reportMiddleware = []echo.MiddlewareFunc{SIDMiddleware, TryAuditor}

const timeToLive = 5 * time.Minute

var timeToLiveString = fmt.Sprintf("max-age=%v", timeToLive.Seconds())

func SetCacheHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set(echo.HeaderCacheControl, timeToLiveString)
		return next(c)
	}
}

var withCache = []echo.MiddlewareFunc{SetCacheHeaders}

func Static(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		s := strings.Split(c.Request().RequestURI, "/")
		etag := s[len(s)-1]

		c.Response().Header().Set("Etag", etag)
		c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=86400") // 24 hours
		if match := c.Request().Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, etag) {
				return c.NoContent(http.StatusNotModified)
			}
		}
		return next(c)
	}
}

func OriginalResponseWriter(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set("writer", c.Response().Writer)
		return next(c)
	}
}

var withOriginalResponseWriter = []echo.MiddlewareFunc{LoggedInMiddleware, RequireAuditor, OriginalResponseWriter}

var StaticMiddleware = []echo.MiddlewareFunc{Static, RedisMiddleware}

var WithRedis = []echo.MiddlewareFunc{OriginalResponseWriter, SetCacheHeaders, RedisMiddleware}

func RedisMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !cache.Initialized {
			return next(c)
		}
		c.Set("redis", cache.RedisClient())
		return next(c)
	}
}
