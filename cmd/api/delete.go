package main

import (
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

var deleteHandlers = pathHandler{
	"/ticket/delete/:id": handler{deleteTicket, staffMiddleware},
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
