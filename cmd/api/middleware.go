package main

import (
	"fmt"
	"github.com/coocood/freecache"
	"github.com/ellypaws/inkbunny-app/api/cache"
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/ellypaws/inkbunny/api"
	gocache "github.com/gitsight/go-echo-cache"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/bytes"
	"net/http"
	"strings"
	"time"
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
			return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "invalid sid"})
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

func GetCurrentAuditor(c echo.Context) (auditor *db.Auditor, err error) {
	auditor, ok := c.Get("auditor").(*db.Auditor)
	if !ok || auditor == nil {
		return nil, crashy.ErrorResponse{ErrorString: "empty auditor"}
	}

	return auditor, nil
}

var staffMiddleware = []echo.MiddlewareFunc{LoggedInMiddleware, RequireAuditor}

const timeToLive = 5 * time.Minute

var timeToLiveString = fmt.Sprintf("max-age=%v", timeToLive.Seconds())

var defaultCacheConfig = &gocache.Config{
	Methods: []string{echo.GET, echo.HEAD},
	TTL:     timeToLive,
	Refresh: func(r *http.Request) bool {
		return r.Header.Get("Cache-Control") == "no-cache"
	},
}

var globalCache = func() echo.MiddlewareFunc {
	c := freecache.NewCache(256 * bytes.MiB)
	return gocache.New(defaultCacheConfig, c)
}()

func CacheMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return globalCache(next)
}

func SetCacheHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", timeToLiveString)
		return next(c)
	}
}

var withCache = []echo.MiddlewareFunc{SetCacheHeaders, CacheMiddleware}

func Static(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		s := strings.Split(c.Request().RequestURI, "/")
		etag := s[len(s)-1]

		c.Response().Header().Set("Etag", etag)
		c.Response().Header().Set("Cache-Control", "public, max-age=86400") // 24 hours
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

var withOriginalResponseWriter = []echo.MiddlewareFunc{LoggedInMiddleware, RequireAuditor, OriginalResponseWriter, CacheMiddleware}

var staticMiddleware = []echo.MiddlewareFunc{Static, RedisMiddleware, CacheMiddleware}

var withRedis = []echo.MiddlewareFunc{OriginalResponseWriter, RedisMiddleware, CacheMiddleware}

func RedisMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !cache.Initialized {
			return next(c)
		}
		c.Set("redis", cache.RedisClient())
		return next(c)
	}
}
