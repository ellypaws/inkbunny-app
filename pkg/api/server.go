package api

import (
	"fmt"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	logger "github.com/labstack/gommon/log"

	"github.com/ellypaws/inkbunny-app/pkg/db"
	sd "github.com/ellypaws/inkbunny-sd/stable_diffusion"
)

var (
	Database   *db.Sqlite
	SDHost     = sd.DefaultHost
	ServerHost *url.URL
)

type RunConfig struct {
	Database    *db.Sqlite
	SDHost      *sd.Host
	ServerHost  *url.URL
	Port        uint
	LogLevel    logger.Lvl
	Middlewares []echo.MiddlewareFunc
	Extra       []func(e *echo.Echo)
}

func Run(config RunConfig) {
	Database = config.Database
	SDHost = config.SDHost
	ServerHost = config.ServerHost

	e := echo.New()

	e.Use(middleware.Recover())

	registerAs(e.GET, getHandlers)
	registerAs(e.POST, postHandlers)
	registerAs(e.HEAD, headHandlers)
	registerAs(e.DELETE, deleteHandlers)
	registerAs(e.PUT, putHandlers)
	registerAs(e.PATCH, patchHandlers)

	e.Logger.SetLevel(config.LogLevel)
	e.Logger.SetHeader(`${time_rfc3339} ${level}	${short_file}:${line}	`)

	e.Use(config.Middlewares...)

	for _, f := range config.Extra {
		f(e)
	}

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", config.Port)))
}

type route = func(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route

type handler struct {
	handler    func(c echo.Context) error
	middleware []echo.MiddlewareFunc
}

type pathHandler = map[string]handler

func registerAs(route route, pathHandler pathHandler) {
	for path, handler := range pathHandler {
		route(path, handler.handler, handler.middleware...)
	}
}
