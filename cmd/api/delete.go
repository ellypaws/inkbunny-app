package main

import (
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

var deleteHandlers = pathHandler{
	"/ticket/delete/:id":       handler{deleteTicket, staffMiddleware},
	"/artist/delete/:username": handler{deleteArtist, staffMiddleware},
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
	username := c.Param("username")
	if username == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing username"})
	}

	if err := database.DeleteArtist(username); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	return c.JSON(http.StatusOK, db.Artist{Username: username})
}
