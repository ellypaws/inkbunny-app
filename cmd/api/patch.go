package main

import (
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/labstack/echo/v4"
	"net/http"
	"slices"
)

var patchHandlers = pathHandler{
	"/ticket":  handler{updateTicket, staffMiddleware},
	"/artist":  handler{upsertArtist, staffMiddleware},
	"/auditor": handler{upsertAuditor, staffMiddleware},
	"/models":  handler{upsertModel, staffMiddleware},
}

func updateTicket(c echo.Context) error {
	var ticket db.Ticket
	if err := c.Bind(&ticket); err != nil {
		return err
	}

	if err := db.Error(database); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if ticket.DateOpened.IsZero() {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing date opened"})
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
		if artist.Username == "" {
			return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing username", Debug: artists})
		}
		err := database.UpsertArtist(artist)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
	}

	return c.JSON(http.StatusOK, artists)
}

func upsertAuditor(c echo.Context) error {
	var auditors []db.Auditor
	if err := c.Bind(&auditors); err != nil {
		return c.JSON(http.StatusBadRequest, crashy.Wrap(err))
	}

	if err := db.Error(database); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if len(auditors) == 0 {
		return c.JSON(http.StatusLengthRequired, crashy.ErrorResponse{ErrorString: "no auditors to upsert"})
	}

	stored := database.AllAuditors()
	for _, auditor := range auditors {
		if auditor.Username == "" {
			return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing username", Debug: auditors})
		}

		var valid bool
		for _, known := range stored {
			if auditor.Username == known.Username {
				valid = true
				break
			}
		}
		if !valid {
			return c.JSON(http.StatusConflict, crashy.ErrorResponse{ErrorString: "auditor not found", Debug: auditors})
		}

		err := database.EditAuditorRole(auditor.UserID, auditor.Role)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
	}

	return c.JSON(http.StatusOK, auditors)
}

func upsertModel(c echo.Context) error {
	var modelHashes db.ModelHashes
	if err := c.Bind(&modelHashes); err != nil {
		return c.JSON(http.StatusBadRequest, crashy.Wrap(err))
	}

	if err := db.Error(database); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if len(modelHashes) == 0 {
		return c.JSON(http.StatusLengthRequired, crashy.ErrorResponse{ErrorString: "no models to upsert"})
	}

	stored, err := database.AllModels()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	for hash, models := range modelHashes {
		known, ok := stored[hash]
		if !ok {
			stored[hash] = models
			continue
		}

		for _, model := range models {
			if model == "" {
				return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{
					ErrorString: "missing model name",
					Debug:       model,
				})
			}
			if slices.Contains(known, model) {
				continue
			}
			known = append(known, model)
		}
		stored[hash] = known
	}

	err = database.InsertModel(stored)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	return c.JSON(http.StatusOK, modelHashes)
}
