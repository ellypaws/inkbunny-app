package api

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/ellypaws/inkbunny-app/pkg/crashy"
	"github.com/ellypaws/inkbunny-app/pkg/db"
)

var putHandlers = pathHandler{
	"/ticket":  handler{newTicket, staffMiddleware},
	"/artist":  handler{newArtist, staffMiddleware},
	"/auditor": handler{newAuditor, staffMiddleware},
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

	if err := db.Error(Database); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	id, err := Database.InsertTicket(ticket)
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

	if err := db.Error(Database); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if len(artists) == 0 {
		return c.JSON(http.StatusLengthRequired, crashy.ErrorResponse{ErrorString: "no artists to upsert"})
	}

	known := Database.AllArtists()
	for _, artist := range artists {
		if artist.Username == "" {
			return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing username", Debug: artists})
		}
		for _, k := range known {
			if k.Username == artist.Username {
				return c.JSON(http.StatusConflict, crashy.ErrorResponse{ErrorString: "artist already exists", Debug: artists})
			}
		}
		err := Database.UpsertArtist(artist)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
	}

	return c.JSON(http.StatusOK, artists)
}

func newAuditor(c echo.Context) error {
	var auditors []db.Auditor
	if err := c.Bind(&auditors); err != nil {
		return err
	}

	if err := db.Error(Database); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if len(auditors) == 0 {
		return c.JSON(http.StatusLengthRequired, crashy.ErrorResponse{ErrorString: "no auditors to upsert"})
	}

	known := Database.AllAuditors()
	for _, auditor := range auditors {
		if auditor.Username == "" {
			return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing username", Debug: auditors})
		}
		for _, k := range known {
			if k.Username == auditor.Username {
				return c.JSON(http.StatusConflict, crashy.ErrorResponse{ErrorString: "auditor already exists", Debug: auditors})
			}
		}
		err := Database.InsertAuditor(auditor)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
	}

	return c.JSON(http.StatusOK, auditors)
}
