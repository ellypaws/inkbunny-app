package main

import (
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

var putHandlers = pathHandler{
	"/tickets": handler{newTicket, staffMiddleware},
	"/artist":  handler{newArtist, staffMiddleware},
}

func newTicket(c echo.Context) error {
	var ticket db.Ticket = db.Ticket{
		DateOpened: time.Now().UTC(),
		Priority:   "low",
		Closed:     false,
	}
	if err := c.Bind(&ticket); err != nil {
		return err
	}

	if err := db.Error(database); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	id, err := database.InsertTicket(ticket)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	ticket.ID = id
	return c.JSON(http.StatusOK, ticket)
}

func newArtist(c echo.Context) error {
	var artists []db.Artist
	if err := c.Bind(&artists); err != nil {
		return err
	}

	if err := db.Error(database); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if len(artists) == 0 {
		return c.JSON(http.StatusLengthRequired, crashy.ErrorResponse{ErrorString: "no artists to upsert"})
	}

	known := database.AllArtists()
	for _, artist := range artists {
		for _, k := range known {
			if k.Username == artist.Username {
				return c.JSON(http.StatusConflict, crashy.ErrorResponse{ErrorString: "artist already exists", Debug: artist})
			}
		}
		err := database.UpsertArtist(artist)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
	}

	return c.JSON(http.StatusOK, artists)
}
