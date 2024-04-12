package main

import (
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

var deleteHandlers = pathHandler{
	"/ticket/:id":       handler{deleteTicket, staffMiddleware},
	"/artist":           handler{deleteArtist, staffMiddleware},
	"/artist/:username": handler{deleteArtist, staffMiddleware},
}

func deleteTicket(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing id"})
	}

	i, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "invalid id"})
	}

	if err := database.DeleteTicket(i); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	return c.JSON(http.StatusOK, db.Ticket{ID: i})
}

func deleteArtist(c echo.Context) error {
	var artists []db.Artist
	if err := c.Bind(&artists); err != nil {
		return c.JSON(http.StatusBadRequest, crashy.Wrap(err))
	}

	if username := c.Param("username"); username != "" {
		artists = append(artists, db.Artist{Username: username})
	}

	if len(artists) == 0 {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing artists"})
	}

	for _, artist := range artists {
		if artist.Username == "" {
			return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing username", Debug: artists})
		}
		if err := database.DeleteArtist(artist.Username); err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
	}

	return c.JSON(http.StatusOK, artists)
}
