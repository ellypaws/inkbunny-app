package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/ellypaws/inkbunny-app/pkg/crashy"
	"github.com/ellypaws/inkbunny-app/pkg/db"
)

var deleteHandlers = pathHandler{
	"/ticket/:id":       handler{deleteTicket, staffMiddleware},
	"/artist":           handler{deleteArtist, staffMiddleware},
	"/artist/:username": handler{deleteArtist, staffMiddleware},
	"/auditor":          handler{deleteAuditor, staffMiddleware},
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

	if err := Database.DeleteTicket(i); err != nil {
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
		if err := Database.DeleteArtist(artist.Username); err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
	}

	return c.JSON(http.StatusOK, artists)
}

func deleteAuditor(c echo.Context) error {
	var auditors []db.Auditor
	if err := c.Bind(&auditors); err != nil {
		return c.JSON(http.StatusBadRequest, crashy.Wrap(err))
	}

	if len(auditors) == 0 {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing auditors"})
	}

	for _, auditor := range auditors {
		if auditor.Username == "" {
			return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing username", Debug: auditors})
		}
		if err := Database.DeleteAuditor(auditor.UserID); err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
	}

	return c.JSON(http.StatusOK, auditors)
}
