package main

import (
	"context"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var database *db.Sqlite

func main() {
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Database
	var err error
	database, err = db.New(context.Background())
	if err != nil {
		e.Logger.Fatal(err)
	}

	// Routes
	registerAs(e.GET, getHandlers)
	registerAs(e.POST, postHandlers)

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}

type route = func(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
type pathHandler = map[string]func(c echo.Context) error

func registerAs(route route, pathHandler pathHandler) {
	for path, handler := range pathHandler {
		route(path, handler)
	}
}
