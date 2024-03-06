package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	registerGetRoutes(e)
	registerPostRoutes(e)

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}

type handler struct {
	login LoginRequest
}
