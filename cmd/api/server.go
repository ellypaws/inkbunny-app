package main

import (
	"github.com/ellypaws/inkbunny-app/cmd/db"
	sd "github.com/ellypaws/inkbunny-sd/stable_diffusion"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log"
	"net/url"
	"os"
)

var (
	database *db.Sqlite
	host     = sd.DefaultHost
	port     = "1323"
)

func init() {
	if h := os.Getenv("SD_HOST"); h != "" {
		u, err := url.Parse(h)
		if err != nil {
			log.Fatal(err)
		}
		host = (*sd.Host)(u)
	}

	if host == nil || !host.Alive() {
		log.Println("warning: host is not alive")
	}

	log.Printf("host: %s\n", host)

	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	// Database
	var err error
	database, err = db.New(nil)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	registerAs(e.GET, getHandlers)
	registerAs(e.POST, postHandlers)

	// Start server
	e.Logger.Fatal(e.Start(":" + port))
}

type route = func(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
type pathHandler = map[string]func(c echo.Context) error

func registerAs(route route, pathHandler pathHandler) {
	for path, handler := range pathHandler {
		route(path, handler)
	}
}
