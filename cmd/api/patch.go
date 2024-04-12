package main

import (
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/labstack/echo/v4"
	"net/http"
)

var patchHandlers = pathHandler{
	"/ticket": handler{updateTicket, staffMiddleware},
	"/artist": handler{upsertArtist, staffMiddleware},
}

func updateTicket(c echo.Context) error {
	var ticket db.Ticket
	if err := c.Bind(&ticket); err != nil {
		return err
	}

	if err := db.Error(database); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	id, err := database.UpsertTicket(ticket)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if ticket.ID != id {
		return c.JSON(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: "got the wrong ticket back from the database", Debug: ticket})
	}
	return c.JSON(http.StatusOK, ticket)
}

func upsertArtist(c echo.Context) error {
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

	for _, artist := range artists {
		err := database.UpsertArtist(artist)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
	}

	return c.JSON(http.StatusOK, artists)
}
