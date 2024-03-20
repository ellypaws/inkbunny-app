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
	registerGetRoutes(e)
	registerPostRoutes(e)

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}
