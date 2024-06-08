package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ellypaws/inkbunny-app/cmd/api/cache"
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	units "github.com/labstack/gommon/bytes"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

type Review struct {
	Output OutputType
	Query  url.Values

	Cache cache.Cache
	Key   string

	Stream bool
	Writer http.Flusher

	SubmissionIDs []string
	Search        *SearchReview
	Store         *any

	Database *db.Sqlite
	ApiHost  *url.URL

	Auditor *db.Auditor
}

func RetrieveReview(c echo.Context, review *Review) (processed []Detail, missed []string, errFunc func(c echo.Context) error) {
	if review.Output != OutputReport {
		item, err := review.Cache.Get(review.Key)
		if err == nil {
			c.Logger().Infof("Cache hit for %s", review.Key)
			if c.Param("id") != "search" {
				return nil, nil, func(c echo.Context) error { return c.Blob(http.StatusOK, item.MimeType, item.Blob) }
			} else {
				var store any
				if err := json.Unmarshal(item.Blob, &store); err != nil {
					return nil, nil, func(c echo.Context) error { return c.JSON(http.StatusInternalServerError, crashy.Wrap(err)) }
				}
				review.Search.Review = store
				return nil, nil, func(c echo.Context) error { return c.JSON(http.StatusOK, review.Search) }
			}
		}
	}

	for _, id := range review.SubmissionIDs {
		key := fmt.Sprintf(
			"%s:review:%s:%s?%s",
			echo.MIMEApplicationJSON,
			review.Output,
			id,
			review.Query.Encode(),
		)

		item, err := review.Cache.Get(key)
		if err != nil {
			missed = append(missed, id)
			continue
		}

		if review.Stream {
			err := streamBlob(c, item.Blob, err, review.Writer)
			if err != nil {
				c.Logger().Errorf("error flushing submission %v: %v", id, err)
				return nil, nil, func(c echo.Context) error { return c.JSON(http.StatusInternalServerError, crashy.Wrap(err)) }
			}
			c.Logger().Debugf("flushing %v", id)
		} else {
			c.Logger().Infof("Cache hit for %s", key)
		}

		var detail Detail
		if err := json.Unmarshal(bytes.Trim(item.Blob, "[]"), &detail); err != nil {
			c.Logger().Errorf("error unmarshaling submission %v: %v", id, err)
			return nil, nil, func(c echo.Context) error { return c.JSON(http.StatusInternalServerError, crashy.Wrap(err)) }
		}

		processed = append(processed, detail)
	}

	if len(missed) > 0 {
		c.Logger().Debugf("Cache miss for %s retrieving review...", review.Key)
		return processed, missed, nil
	}

	if len(processed) == 0 {
		return nil, nil, func(c echo.Context) error {
			return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no reviews found"})
		}
	}

	switch review.Output {
	case OutputSingleTicket:
		*review.Store = CreateSingleTicket(review.Auditor, processed)
	case OutputReport:
		report := CreateTicketReport(review.Auditor, processed, review.ApiHost)
		StoreReport(c, review.Database, report)
		*review.Store = report
	default:
		*review.Store = processed
	}

	if c.Param("id") == "search" {
		review.Search.Review = review.Store
		if review.Stream && review.Output != OutputSingleTicket {
			return nil, nil, func(c echo.Context) error { return nil }
		}
		return nil, nil, func(c echo.Context) error { return c.JSON(http.StatusOK, review.Search) }
	}

	if review.Stream && review.Output != OutputSingleTicket {
		return nil, nil, func(c echo.Context) error { return nil }
	}

	return nil, nil, func(c echo.Context) error { return c.JSON(http.StatusOK, review.Store) }
}

func streamBlob(c echo.Context, blob []byte, err error, writer http.Flusher) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	if _, err := c.Response().Write(blob); err != nil {
		c.Response().WriteHeader(http.StatusInternalServerError)
		return err
	}

	if _, err = c.Response().Write([]byte("\n")); err != nil {
		c.Response().WriteHeader(http.StatusInternalServerError)
		return err
	}

	writer.Flush()
	return nil
}

func CreateSingleTicket(auditor *db.Auditor, details []Detail) db.Ticket {
	auditorAsUser := AuditorAsUsernameID(auditor)

	var ticketLabels []db.TicketLabel
	for _, sub := range details {
		for _, label := range sub.Ticket.Labels {
			if !slices.Contains(ticketLabels, label) {
				ticketLabels = append(ticketLabels, label)
			}
		}
	}
	return db.Ticket{
		Subject:    "subject",
		DateOpened: time.Now().UTC(),
		Status:     "triage",
		Labels:     ticketLabels,
		Priority:   "low",
		Closed:     false,
		Responses: []db.Response{
			{
				SupportTeam: false,
				User:        auditorAsUser,
				Date:        time.Now().UTC(),
				Message: func() string {
					var sb strings.Builder
					for _, sub := range details {
						if sb.Len() > 0 {
							sb.WriteString("\n\n[s]                    [/s]\n\n")
						}
						sb.WriteString(sub.Ticket.Responses[0].Message)
					}
					return sb.String()
				}(),
			},
		},
		SubmissionIDs: func() []int64 {
			var ids []int64
			for _, sub := range details {
				ids = append(ids, int64(sub.ID))
			}
			return ids
		}(),
		AssignedID: &auditor.UserID,
		UsersInvolved: db.Involved{
			Reporter: auditorAsUser,
			ReportedIDs: func() []api.UsernameID {
				var ids []api.UsernameID
				for _, sub := range details {
					ids = append(ids, sub.User)
				}
				return ids
			}(),
		},
	}
}

func StoreReview(c echo.Context, key string, store *any, duration time.Duration, bin ...byte) {
	if bin == nil {
		if store == nil {
			c.Logger().Warnf("trying to cache nil review for %s", key)
			return
		}

		if *store == nil {
			return
		}

		var err error
		bin, err = json.Marshal(store)
		if err != nil {
			c.Logger().Errorf("error marshaling review: %v", err)
			return
		}
	}

	err := cache.SwitchCache(c).Set(key, &cache.Item{
		Blob:     bin,
		MimeType: echo.MIMEApplicationJSON,
	}, duration)
	if err != nil {
		c.Logger().Errorf("error caching review: %v", err)
		return
	}

	c.Logger().Infof("Cached %s %s %dKiB", key, echo.MIMEApplicationJSON, len(bin)/units.KiB)
}
