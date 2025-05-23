package api

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	units "github.com/labstack/gommon/bytes"
	"github.com/redis/go-redis/v9"

	"github.com/ellypaws/inkbunny-app/pkg/api/cache"
	. "github.com/ellypaws/inkbunny-app/pkg/api/entities"
	"github.com/ellypaws/inkbunny-app/pkg/api/service"
	"github.com/ellypaws/inkbunny-app/pkg/app"
	"github.com/ellypaws/inkbunny-app/pkg/crashy"
	"github.com/ellypaws/inkbunny-app/pkg/db"
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/utils"
)

var getHandlers = pathHandler{
	"/inkbunny/description":     handler{GetInkbunnyDescription, withCache},
	"/inkbunny/submission":      handler{GetInkbunnySubmission, withCache},
	"/inkbunny/submission/:ids": handler{GetInkbunnySubmission, withCache},
	"/inkbunny/sorter":          handler{GetSorterHandler, append(reducedMiddleware, WithRedis...)},
	"/inkbunny/search":          handler{GetInkbunnySearch, append(loggedInMiddleware, WithRedis...)},
	"/image":                    handler{GetImageHandler, append(StaticMiddleware, SIDMiddleware)},
	"/review/:id":               handler{GetReviewHandler, append(reducedMiddleware, WithRedis...)},
	"/report/:id":               handler{GetReportHandler, append(reportMiddleware, WithRedis...)},
	"/report/:id/:key":          handler{GetReportKeyHandler, append(StaticMiddleware, SIDMiddleware)},
	"/heuristics/:id":           handler{GetHeuristicsHandler, append(reducedMiddleware, WithRedis...)},
	"/audits":                   handler{GetAuditHandler, staffMiddleware},
	"/tickets":                  handler{GetTicketsHandler, staffMiddleware},
	"/auditors":                 handler{GetAllAuditorsJHandler, staffMiddleware},
	"/username/:username":       handler{GetUsernameHandler, append(loggedInMiddleware, WithRedis...)},
	"/avatar/:username":         handler{GetAvatarHandler, StaticMiddleware},
	"/artists":                  handler{GetArtistsHandler, append(loggedInMiddleware, WithRedis...)},
	"/models":                   handler{GetModelsHandler, withCache},
	"/models/:hash":             handler{GetModelsHandler, WithRedis},
	"/files/:file":              handler{GetFileHandler, StaticMiddleware},
}

// Deprecated: use registerAs((*echo.Echo).GET, getHandlers) instead
func registerGetRoutes(e *echo.Echo) {
	registerAs(e.GET, getHandlers)
}

// GetInkbunnyDescription returns the description of a submission using a DescriptionRequest
// It requires a valid SID to be passed in the request.
// It returns a slice of DescriptionResponse as JSON.
// Example:
//
//	  DescriptionRequest{
//				SID:           "session_id",
//				SubmissionIDs: "14576",
//	  }
func GetInkbunnyDescription(c echo.Context) error {
	var request DescriptionRequest
	_ = c.Bind(&request)

	if request.SID == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing SID"})
	}

	if submissionIDs := c.QueryParam("submission_ids"); submissionIDs != "" {
		if request.SubmissionIDs != "" {
			request.SubmissionIDs += ","
		}
		request.SubmissionIDs += submissionIDs
	}

	if request.SubmissionIDs == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing submission ID"})
	}

	if cookie, err := c.Cookie("sid"); request.SID == "" && err == nil {
		request.SID = cookie.Value
	}

	if request.SID == "guest" {
		user, err := api.Guest().Login()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
		request.SID = user.Sid
	}

	details, err := service.RetrieveSubmission(c, api.SubmissionDetailsRequest{
		SID:             request.SID,
		SubmissionIDs:   request.SubmissionIDs,
		ShowDescription: api.Yes,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if len(details.Submissions) == 0 {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no submissions found"})
	}

	var descriptions []DescriptionResponse
	for _, submission := range details.Submissions {
		descriptions = append(descriptions, DescriptionResponse{
			SubmissionID: request.SubmissionIDs,
			Title:        submission.Title,
			Username:     submission.Username,
			Description:  submission.Description,
		})
	}

	return c.JSON(http.StatusOK, descriptions)
}

// GetInkbunnySubmission returns the details of a submission using api.SubmissionDetailsRequest
// It requires a valid SID to be passed in the request.
// It returns api.SubmissionDetailsResponse as JSON.
// The order of preference for the SID is (where the rightmost value takes precedence):
//
//	request body -> cookie -> query parameter
//
// Similarly, the order of preference for the submission ID is:
//
//	request body -> query parameter -> path parameter
func GetInkbunnySubmission(c echo.Context) error {
	var bind struct {
		api.SubmissionDetailsRequest
		SubmissionIDs *string `query:"submission_ids" param:"ids"`
		SessionID     *string `query:"sid"`
	}

	err := c.Bind(&bind)
	if err != nil {
		return c.JSON(http.StatusBadRequest, crashy.Wrap(err))
	}

	request := bind.SubmissionDetailsRequest

	if bind.SubmissionIDs != nil {
		request.SubmissionIDs = *bind.SubmissionIDs
	}

	if cookie, err := c.Cookie("sid"); request.SID == "" && err == nil {
		request.SID = cookie.Value
	}

	if bind.SessionID != nil {
		request.SID = *bind.SessionID
	}

	details, err := service.RetrieveSubmission(c, request)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if len(details.Submissions) == 0 {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no submissions found"})
	}

	return c.JSON(http.StatusOK, details)
}

var submissionIDs = regexp.MustCompile(`https://inkbunny.net/s/(\d+)`)

// GetSorterHandler sorts arbitrary submission links by its artist usernames
func GetSorterHandler(c echo.Context) error {
	var request struct {
		Text      string `json:"text" query:"text"`
		SessionID string `json:"sid" query:"sid"`
	}

	err := c.Bind(&request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, crashy.Wrap(err))
	}

	request.SessionID, err = GetSID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
	}

	if request.Text == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing text"})
	}

	matches := submissionIDs.FindAllStringSubmatch(request.Text, -1)
	if matches == nil {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "no submission IDs found"})
	}

	var ids []string
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		ids = append(ids, match[1])
	}

	req := api.SubmissionDetailsRequest{
		SID:           request.SessionID,
		SubmissionIDs: strings.Join(ids, ","),
	}

	var details api.SubmissionDetailsResponse
	if len(ids) > 100 {
		details, err = service.BatchRetrieveSubmission(c, req, ids)
	} else {
		details, err = service.RetrieveSubmission(c, req)
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if len(details.Submissions) == 0 {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no submissions found"})
	}

	submissions := make(map[string][]string)
	for _, submission := range details.Submissions {
		submissions[submission.Username] = append(submissions[submission.Username], fmt.Sprintf("https://inkbunny.net/s/%s", submission.SubmissionID))
	}

	return c.JSON(http.StatusOK, submissions)
}

// GetInkbunnySearch returns the search results of a submission using api.SubmissionSearchRequest
// It requires a valid SID to be passed in the request.
// It returns api.SubmissionSearchResponse as JSON.
func GetInkbunnySearch(c echo.Context) error {
	var request = api.SubmissionSearchRequest{
		Text:               "ai_generated",
		SubmissionsPerPage: 10,
		Random:             true,
		GetRID:             true,
		Type:               api.SubmissionTypes{api.SubmissionTypePicturePinup},
	}
	var bind = struct {
		*api.SubmissionSearchRequest
		Types *string `json:"types,omitempty" query:"types"`
	}{
		SubmissionSearchRequest: &request,
	}
	err := c.Bind(&bind)
	if err != nil {
		return c.JSON(http.StatusBadRequest, crashy.Wrap(err))
	}

	if temp := c.QueryParam("temp"); temp == "true" {
		return c.JSONBlob(http.StatusOK, app.Temp())
	}

	request.SID, _, err = GetSIDandID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
	}

	if bind.Types != nil {
		*bind.Types = strings.Trim(*bind.Types, "[]")
		*bind.Types = strings.ReplaceAll(*bind.Types, `"`, "")
		for _, t := range strings.Split(*bind.Types, ",") {
			i, err := strconv.Atoi(t)
			if err != nil {
				return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{
					ErrorString: "invalid submission type",
					Debug:       err,
				})
			}
			request.Type = append(request.Type, api.SubmissionType(i))
		}
	}

	searchResponse, err := service.RetrieveSearch(c, request)
	if err != nil {
		return err
	}

	if output := c.QueryParam("output"); output != "" {
		switch output {
		case "json":
			return c.JSON(http.StatusOK, searchResponse)
		case "xml":
			return c.XML(http.StatusOK, searchResponse)
		case "mail":
			return mail(c, &api.Credentials{Sid: request.SID}, searchResponse)
		default:
			return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "invalid output format"})
		}
	}

	return c.JSON(http.StatusOK, searchResponse)
}

func logoutGuest(c echo.Context, user *api.Credentials) {
	if user == nil {
		return
	}
	if user.Username == "guest" {
		err := user.Logout()
		if err != nil {
			c.Logger().Errorf("error logging out guest: %v", err)
		} else {
			c.Logger().Info("logged out guest")
		}
	}
}

func mail(c echo.Context, user *api.Credentials, response api.SubmissionSearchResponse) error {
	if user == nil {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing user"})
	}

	submissionIDs := make([]string, len(response.Submissions))
	for i, s := range response.Submissions {
		submissionIDs[i] = s.SubmissionID
	}

	details, err := service.RetrieveSubmission(c, api.SubmissionDetailsRequest{
		SID:                         user.Sid,
		SubmissionIDs:               strings.Join(submissionIDs, ","),
		ShowDescription:             api.Yes,
		ShowDescriptionBbcodeParsed: api.Yes,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if len(details.Submissions) == 0 {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no submissions found"})
	}

	var validLabels = []app.Label{
		app.Generated,
		app.Assisted,
		app.StableDiffusion,
		app.YiffyMix,
		app.YiffyMix3,
	}

	var mails app.Mails
	for _, submission := range details.Submissions {
		var keywords []app.Label
		for _, keyword := range submission.Keywords {
			if slices.Contains(validLabels, app.Label(strings.ReplaceAll(keyword.KeywordName, " ", "_"))) {
				keywords = append(keywords, app.Label(keyword.KeywordName))
			}
		}

		keywords = append(keywords, app.Label(submission.TypeName))

		var files []app.File
		for _, file := range submission.Files {
			files = append(files, app.File{
				FileID:             file.FileID,
				FileName:           file.FileName,
				FilePreview:        file.FileURLPreview,
				NonCustomThumb:     file.ThumbnailURLLargeNonCustom,
				FileURL:            file.FileURLFull,
				UserID:             file.UserID,
				CreateDateTime:     file.CreateDateTime,
				CreateDateTimeUser: file.CreateDateTimeUser,
			})
		}

		mails = append(mails, app.Mail{
			SubmissionID:   submission.SubmissionID,
			Username:       submission.Username,
			ProfilePicture: submission.UserIconURLs.Large,
			Files:          files,
			Link:           fmt.Sprintf("https://inkbunny.net/s/%s", submission.SubmissionID),
			Title:          submission.Title,
			Description:    submission.Description,
			Html:           submission.DescriptionBBCodeParsed,
			Date:           submission.CreateDateSystem,
			Read:           false,
			Labels:         keywords,
		})
	}
	return c.JSON(http.StatusOK, mails)
}

func GetAuditHandler(c echo.Context) error {
	auditor, err := GetCurrentAuditor(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
	}

	audits, err := Database.GetAuditsByAuditor(auditor.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	return c.JSON(http.StatusOK, audits)
}

func GetTicketsHandler(c echo.Context) error {
	assignedID := c.QueryParam("assigned_id")

	if assignedID != "" {
		p, err := strconv.ParseInt(assignedID, 10, 64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, crashy.Wrap(err))
		}

		tickets, err := Database.GetTicketsByAuditor(p)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}

		return c.JSON(http.StatusOK, tickets)
	}

	tickets, err := Database.GetAllTickets()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	return c.JSON(http.StatusOK, tickets)
}

func validAuditor(c echo.Context, user api.Credentials) bool {
	if err := db.Error(Database); err != nil {
		c.Logger().Warnf("warning: validAuditor was called with a nil database: %v", err)
		return false
	}
	return Database.IsAuditorRole(int64(user.UserID.Int()))
}

func GetAllAuditorsJHandler(c echo.Context) error {
	auditors := Database.AllAuditors()
	if auditors == nil {
		return c.JSON(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: "no auditors found"})
	}
	return c.JSON(http.StatusOK, auditors)
}

// GetReviewHandler returns heuristic analysis of a submission
//   - Set query "output" to "single_ticket", "multiple_tickets", "submissions", "full", "badges", "report", or "report_ids"
//
// - single_ticket: returns a single combined db.Ticket of all submissions
//
// - multiple_tickets: returns each db.Ticket for each submission
//
// - submissions: returns a []details of each submission
//
// - full: returns a []details db.Ticket with the original api.Submission and db.Ticket
//
// - badges: returns a simplified []details of each submission
//
// - report: returns a combined service.TicketReport of a specified user
//
// - report_ids: returns a combined service.TicketReport of a specified user with submission IDs
// Note: "parameters" and "interrogate" won't automatically be set on full output
//
// - badges: returns a simplified []details of each submission
//
//   - Set query "parameters" to "true" to parse the utils.Params from json/text files
//   - Set query "interrogate" to "true" to parse entities.TaggerResponse from image files using (*sd.Host).Interrogate
//   - Set query "stream" to "true" to receive multiple JSON objects
//   - Set the param ":id" to "search" to combine search to immediately review
func GetReviewHandler(c echo.Context) error {
	sid, err := GetSID(c)
	if err != nil {
		c.Logger().Errorf("error getting sid: %v", err)
		return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
	}

	hashed := db.Hash(sid)

	auditor, err := GetCurrentAuditor(c)
	if err != nil {
		c.Logger().Warnf("anonymous user %v", err)
	}

	output := c.QueryParam("output")
	parameters := c.QueryParam("parameters")
	interrogate := c.QueryParam("interrogate")
	stream := c.QueryParam("stream") == "true"

	validOutputs := []service.OutputType{
		service.OutputSingleTicket,
		service.OutputReport,
		service.OutputReportIDs,
		service.OutputMultipleTickets,
		service.OutputSubmissions,
		service.OutputFull,
		service.OutputBadges,
	}

	if output == "" {
		output = service.OutputSingleTicket
	} else if !slices.Contains(validOutputs, output) {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{
			ErrorString: fmt.Sprintf("invalid output format %s. valid options are: %v", output, validOutputs),
			Debug:       output,
		})
	}

	cacheToUse := cache.SwitchCache(c)
	query := url.Values{
		"interrogate": {interrogate},
		"parameters":  {parameters},
		"sid":         {hashed},
	}

	idParam := c.Param("id")
	if idParam == "" || idParam == "null" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing submission ID"})
	}

	skipCache := c.Request().Header.Get(echo.HeaderCacheControl) == "no-cache"

	var submissionIDSlice []string

	var searchStore service.SearchReview
	if idParam == "search" || output == service.OutputReport {
		var errFunc func(echo.Context) error
		searchStore.Search, errFunc = service.RetrieveReviewSearch(c, sid, output, query, cacheToUse)
		if errFunc != nil {
			return errFunc(c)
		}

		submissionIDSlice = make([]string, len(searchStore.Search.Submissions))

		for i, submission := range searchStore.Search.Submissions {
			submissionIDSlice[i] = submission.SubmissionID
		}

		if output != service.OutputReport {
			defer service.StoreSearchReview(c, query, &searchStore)
		}
	} else {
		submissionIDSlice = strings.Split(idParam, ",")
	}

	writer := c.Get("writer").(http.Flusher)

	var key string
	if output == service.OutputReport {
		// idParam should contain the artist name
		key = idParam
	} else {
		key = strings.Join(submissionIDSlice, ",")
	}
	reviewKey := fmt.Sprintf(
		"%s:review:%s:%s?%s",
		echo.MIMEApplicationJSON,
		output,
		key,
		query.Encode(),
	)

	var processed []service.Detail
	var missed = submissionIDSlice
	var store any
	if !skipCache {
		var errFunc func(echo.Context) error
		processed, missed, errFunc = service.RetrieveReview(c,
			&service.Review{
				Output:        output,
				Query:         query,
				Cache:         cacheToUse,
				Key:           reviewKey,
				Stream:        stream,
				Writer:        writer,
				SubmissionIDs: submissionIDSlice,
				Search:        &searchStore,
				Store:         &store,
				Database:      Database,
				ApiHost:       ServerHost,
				Auditor:       auditor,
			},
		)
		if errFunc != nil {
			return errFunc(c)
		}
	}

	if idParam != "search" && output != service.OutputReport && output != service.OutputReportIDs {
		defer service.StoreReview(c, reviewKey, &store, cache.Hour)
	}

	req := api.SubmissionDetailsRequest{
		SID:                         sid,
		SubmissionIDs:               strings.Join(missed, ","),
		OutputMode:                  "json",
		ShowDescription:             true,
		ShowDescriptionBbcodeParsed: true,
	}

	var submissionDetails api.SubmissionDetailsResponse
	if len(missed) > 100 {
		submissionDetails, err = service.BatchRetrieveSubmission(c, req, missed)
	} else {
		submissionDetails, err = service.RetrieveSubmission(c, req)
	}
	if err != nil {
		c.Logger().Errorf("error retrieving submission details: %v", err)
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if len(submissionDetails.Submissions) == 0 {
		c.Logger().Warnf("no submissions found for %s", missed)
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no submissions found"})
	}

	if interrogate == "true" && !SDHost.Alive() {
		c.Logger().Warn("interrogate was set to true but host is offline, only using cached captions...")
	}

	details := service.ProcessResponse(c, &service.Config{
		SubmissionDetails: submissionDetails,
		Artists:           Database.AllArtists(),
		Cache:             cacheToUse,
		Host:              SDHost,
		Output:            output,
		Parameters:        parameters == "true",
		Interrogate:       interrogate == "true",
		Auditor:           auditor,
		ApiHost:           ServerHost,
		Query:             query,
		Writer:            writer,
	})

	details = append(processed, details...)

	switch output {
	case service.OutputSubmissions, service.OutputFull, service.OutputBadges:
		store = details
	case service.OutputMultipleTickets:
		var tickets []db.Ticket
		for _, sub := range details {
			tickets = append(tickets, *sub.Ticket)
		}
		store = tickets
	case service.OutputReport, service.OutputReportIDs:
		report := service.CreateTicketReport(auditor, details, ServerHost)
		service.StoreReport(c, Database, report)
		store = report
	case service.OutputSingleTicket:
		stream = false
		fallthrough
	default:
		store = service.CreateSingleTicket(auditor, details)
	}

	if c.Param("id") == "search" {
		searchStore.Review = store
		if stream {
			return nil
		}
		return c.JSON(http.StatusOK, searchStore)
	}

	if stream {
		return nil
	}

	return c.JSON(http.StatusOK, store)
}

// GetReportHandler returns a report analysis of an artist
// Set query "limit" to limit the number of submissions returned
// Set query "text" to use a custom search term
func GetReportHandler(c echo.Context) error {
	artist := c.Param("id")
	cacheToUse := cache.SwitchCache(c)
	limitQuery := c.QueryParam("limit")

	var limit int
	if limitQuery == "" {
		limit = 10
	} else {
		var err error
		limit, err = strconv.Atoi(limitQuery)
		if err != nil {
			return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "invalid limit", Debug: err})
		}
		limit = max(limit, 1)
	}

	reportKey := fmt.Sprintf(
		"%s:report:%s?limit=%d",
		echo.MIMEApplicationJSON,
		artist,
		limit,
	)

	sid, err := GetSID(c)
	if err != nil {
		c.Logger().Errorf("error getting sid: %v", err)
		return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "cannot generate a report for logged out user"})
	}

	hashed := db.Hash(sid)

	submissions, err := service.RetrieveSearch(c, api.SubmissionSearchRequest{
		SID:                sid,
		Username:           artist,
		SubmissionsPerPage: api.IntString(limit),
		SubmissionIDsOnly:  true,
		GetRID:             true,
		KeywordID:          db.AIGeneratedID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if len(submissions.Submissions) == 0 {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no submissions found"})
	}

	var missed []string
	var processed []service.Detail

	skipCache := c.Request().Header.Get(echo.HeaderCacheControl) == "no-cache"
	for _, submission := range submissions.Submissions {
		if skipCache {
			missed = append(missed, submission.SubmissionID)
			continue
		}

		key := fmt.Sprintf(
			"%s:review:%s:%s?interrogate=&parameters=true&sid=%s",
			echo.MIMEApplicationJSON,
			service.OutputBadges,
			submission.SubmissionID,
			hashed,
		)
		item, errFunc := cacheToUse.Get(key)
		if errFunc == nil {
			var detail service.Detail
			if err := json.Unmarshal(item.Blob, &detail); err != nil {
				c.Logger().Errorf("error unmarshaling submission %v: %v", submission.SubmissionID, err)
				return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
			}
			processed = append(processed, detail)
			continue
		}

		missed = append(missed, submission.SubmissionID)
	}

	auditor, err := GetCurrentAuditor(c)
	if err != nil {
		c.Logger().Warnf("anonymous user %v", err)
	}
	if len(missed) > 0 {
		req := api.SubmissionDetailsRequest{
			SID:                         sid,
			SubmissionIDs:               strings.Join(missed, ","),
			OutputMode:                  "json",
			ShowDescription:             true,
			ShowDescriptionBbcodeParsed: true,
		}
		submissionDetails, err := service.RetrieveSubmission(c, req)
		if err != nil {
			c.Logger().Errorf("error retrieving submission details: %v", err)
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}

		if len(submissionDetails.Submissions) == 0 {
			c.Logger().Warnf("no submissions found for %s", artist)
			return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no submissions found"})
		}

		details := service.ProcessResponse(c, &service.Config{
			SubmissionDetails: submissionDetails,
			Artists:           Database.AllArtists(),
			Cache:             cacheToUse,
			Host:              SDHost,
			Output:            service.OutputBadges,
			Parameters:        true,
			Interrogate:       false,
			Auditor:           auditor,
			ApiHost:           ServerHost,
			Query: url.Values{
				"interrogate": {""},
				"parameters":  {"true"},
				"sid":         {hashed},
			},
			Writer: c.Get("writer").(http.Flusher),
		})

		processed = append(processed, details...)
	}

	out := service.CreateReport(processed, auditor, ServerHost)

	var store any
	date := out.ReportDate.Format("2006-01-02")
	reportKey = fmt.Sprintf(
		"%s:report:%s:%s",
		echo.MIMEApplicationJSON,
		artist,
		date,
	)
	store = out

	service.StoreReview(c, reportKey, &store, cache.Indefinite)
	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/report/%s/%s.json", artist, date))
}

func GetReportKeyHandler(c echo.Context) error {
	artist := c.Param("id")
	key := strings.TrimSuffix(c.Param("key"), ".json")
	cacheToUse := cache.SwitchCache(c)

	if key == "latest" {
		t, err := Database.GetLatestTicketReport(artist)
		if err == nil {
			reportKey := fmt.Sprintf(
				"%s:report:%s:%s",
				echo.MIMEApplicationJSON,
				t.Username,
				t.ReportDate.Format(db.TicketDateLayout),
			)

			service.StoreReview(c, reportKey, nil, cache.Indefinite, t.Report...)
			return c.Redirect(
				http.StatusFound,
				fmt.Sprintf("/report/%s/%s.json", artist, t.ReportDate.Format(db.TicketDateLayout)),
			)
		}
	}

	reportKey := fmt.Sprintf(
		"%s:report:%s:%s",
		echo.MIMEApplicationJSON,
		artist,
		key,
	)

	item, errFunc := cacheToUse.Get(reportKey)
	if errFunc == nil {
		return c.Blob(http.StatusOK, item.MimeType, item.Blob)
	}
	if !errors.Is(errFunc, redis.Nil) {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "an error occurred while retrieving the report", Debug: errFunc})
	}

	t, err := Database.GetTicketReportByKey(fmt.Sprintf("%s:%s", key, artist))
	if err == nil {
		go service.StoreReview(c, reportKey, nil, cache.Indefinite, t.Report...)
		return c.Blob(http.StatusOK, echo.MIMEApplicationJSON, t.Report)
	}

	return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no report found"})
}

func maxConfidence(old, new *entities.TaggerResponse) *entities.TaggerResponse {
	if old == nil {
		return new
	}

	var merged entities.TaggerResponse
	merged.Caption.Tag = make(map[string]float64)
	for label, oldConfidence := range old.Caption.Tag {
		if new.Caption.Tag == nil {
			merged.Caption.Tag = old.Caption.Tag
			continue
		}
		merged.Caption.Tag = make(map[string]float64)
		if newConfidence, ok := new.Caption.Tag[label]; ok {
			merged.Caption.Tag[label] = max(oldConfidence, newConfidence)
		} else {
			merged.Caption.Tag[label] = oldConfidence
		}
	}

	for label, newConfidence := range new.Caption.Tag {
		if _, ok := merged.Caption.Tag[label]; !ok {
			merged.Caption.Tag[label] = newConfidence
		}
	}

	return &merged
}

// GetHeuristicsHandler returns the heuristics of a submission
func GetHeuristicsHandler(c echo.Context) error {
	sid, err := GetSID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
	}

	submissionIDs := c.Param("id")
	if submissionIDs == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing submission ID"})
	}

	req := api.SubmissionDetailsRequest{
		SID:                         sid,
		SubmissionIDs:               submissionIDs,
		OutputMode:                  "json",
		ShowDescription:             true,
		ShowDescriptionBbcodeParsed: true,
	}

	submissionDetails, err := service.RetrieveSubmission(c, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if len(submissionDetails.Submissions) == 0 {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no submissions found"})
	}

	type details struct {
		URL        string
		ID         api.IntString
		User       api.UsernameID
		Submission *db.Submission
	}

	var submissions = make(map[int64]details)

	cacheToUse := cache.SwitchCache(c)
	artists := Database.AllArtists()

	writer := c.Get("writer").(http.Flusher)

	var waitGroup sync.WaitGroup
	var mutex sync.Locker
	for _, sub := range submissionDetails.Submissions {
		waitGroup.Add(1)

		submission := service.InkbunnySubmissionToDBSubmission(sub, true)
		go func(wg *sync.WaitGroup, sub *db.Submission) {
			service.RetrieveParams(c, wg, sub, cacheToUse, artists)

			if c.QueryParam("stream") == "true" {
				mutex.Lock()
				defer mutex.Unlock()
				c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

				enc := json.NewEncoder(c.Response())
				if err := enc.Encode(sub); err != nil {
					c.Logger().Errorf("error encoding submission %v: %v", sub.ID, err)
					c.Response().WriteHeader(http.StatusInternalServerError)
					return
				}
				c.Logger().Debugf("flushing %v", sub.ID)

				writer.Flush()
			}
		}(&waitGroup, &submission)

		submissions[submission.ID] = details{
			URL:        submission.URL,
			ID:         api.IntString(submission.ID),
			User:       api.UsernameID{UserID: sub.UserID, Username: sub.Username},
			Submission: &submission,
		}
	}
	waitGroup.Wait()
	if c.QueryParam("stream") == "true" {
		return nil
	}

	type p struct {
		URL     string                                 `json:"url"`
		Params  utils.Params                           `json:"params"`
		Objects map[string]entities.TextToImageRequest `json:"objects"`
	}
	var params []*p
	for _, sub := range submissions {
		params = append(params, &p{
			URL:     sub.URL,
			Params:  sub.Submission.Metadata.Params,
			Objects: sub.Submission.Metadata.Objects,
		})
	}

	return c.JSON(http.StatusOK, params)
}

// GetUsernameHandler returns a list of suggested users based on a username
// Set query "exact" to "true" to only return a single exact match
func GetUsernameHandler(c echo.Context) error {
	username := c.Param("username")
	if username == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing username"})
	}

	exact := c.QueryParam("exact") == "true"

	cacheToUse := cache.SwitchCache(c)
	key := fmt.Sprintf("%v:inkbunny:username_autosuggest:%v", echo.MIMEApplicationJSON, username)
	if exact {
		key = fmt.Sprintf("%v:inkbunny:username_autosuggest:exact:%v", echo.MIMEApplicationJSON, username)
	}

	item, err := cacheToUse.Get(key)
	if err == nil {
		return c.Blob(http.StatusOK, item.MimeType, item.Blob)
	}
	if !errors.Is(err, redis.Nil) {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "an error occurred while retrieving the username", Debug: err})
	}

	usernames, err := api.GetUserID(username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	var users []api.Autocomplete
	if exact {
		for i, user := range usernames.Results {
			if strings.EqualFold(user.Value, user.SearchTerm) {
				users = append(users, usernames.Results[i])
				break
			}
		}
	} else {
		users = usernames.Results
	}

	bin, err := json.Marshal(users)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	_ = cacheToUse.Set(key, &cache.Item{
		Blob:     bin,
		MimeType: echo.MIMEApplicationJSON,
	}, cache.Year)

	return c.JSON(http.StatusOK, users)
}

// GetArtistsHandler returns a list of known artists
// Set query "avatar" to "true" to include the avatar URL
func GetArtistsHandler(c echo.Context) error {
	artists := Database.AllArtists()

	if c.QueryParam("avatar") != "true" {
		return c.JSON(http.StatusOK, artists)
	}

	type artistAvatarID struct {
		db.Artist
		Icon *string `json:"icon,omitempty"`
		ID   *int64  `json:"id,omitempty"`
	}

	var artistsWithIcon []artistAvatarID

	var add = func(artist db.Artist, icon *string) []artistAvatarID {
		return append(artistsWithIcon,
			artistAvatarID{
				Artist: artist,
				Icon:   icon,
				ID:     artist.UserID,
			})
	}

	cacheToUse := cache.SwitchCache(c)
	for _, artist := range artists {
		if artist.UserID == nil {
			artistsWithIcon = add(artist, nil)
			continue
		}

		key := fmt.Sprintf("%v:inkbunny:username_autosuggest:exact:%v", echo.MIMEApplicationJSON, artist.Username)

		item, err := cacheToUse.Get(key)
		if err == nil {
			c.Logger().Debugf("Cache hit for %s", key)
			var users []api.Autocomplete
			if err := json.Unmarshal(item.Blob, &users); err != nil {
				return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
			}
			if len(users) == 0 {
				continue
			}
			artistsWithIcon = add(artist, &users[0].Icon)
			continue
		}

		if !errors.Is(err, redis.Nil) {
			return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "an error occurred while retrieving the username", Debug: err})
		}

		c.Logger().Debugf("Cache miss for %s retrieving username...", key)
		usernames, err := api.GetUserID(artist.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}

		var users []api.Autocomplete

		for i, user := range usernames.Results {
			if strings.EqualFold(user.Value, user.SearchTerm) {
				users = append(users, usernames.Results[i])
				artistsWithIcon = add(artist, &user.Icon)
				break
			}
		}

		bin, err := json.Marshal(users)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}

		_ = cacheToUse.Set(key, &cache.Item{
			Blob:     bin,
			MimeType: echo.MIMEApplicationJSON,
		}, cache.Year)
		c.Logger().Infof("Cached %s %s %dKiB", key, echo.MIMEApplicationJSON, len(bin)/units.KiB)
	}

	return c.JSON(http.StatusOK, artistsWithIcon)
}

// GetModelsHandler returns a list of known models
// Set query "civitai" to "true" to return civitai.CivitAIModel
// Set query "recache" to "true" to force a recache (slow)
// Recache is only true if civitai is not true as that would skip querying host
//
// The order of operations is:
//
//	Database -> Redis:Host -> Host -> Redis:CivitAI -> CivitAI
func GetModelsHandler(c echo.Context) error {
	hash := c.Param("hash")
	if hash == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing model hash"})
	}

	if hash == "all" {
		models, err := Database.AllModels()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
		return c.JSON(http.StatusOK, models)
	}

	if len(hash) > 12 {
		c.Logger().Warnf("hash %s is the full SHA256", hash)
		hash = hash[:12]
	}

	cacheToUse := cache.SwitchCache(c)
	if c.QueryParam("civitai") != "true" {
		if c.QueryParam("recache") != "true" {
			if models := Database.ModelNamesFromHash(hash); models != nil {
				return c.JSON(http.StatusOK, db.ModelHashes{hash: models})
			} else {
				c.Logger().Warnf("model %s not found, attempting to find", hash)
			}
		} else {
			c.Logger().Infof("recache was set to true for %s", hash)
		}

		var match db.ModelHashes
		if len(hash) == 12 {
			var err error
			match, err = service.QueryHost(c, cacheToUse, SDHost, Database, hash)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
			}
		}

		if len(match) == 0 {
			c.Logger().Warnf("model %s not found in known models, querying CivitAI...", hash)
		} else {
			return c.JSON(http.StatusOK, match)
		}
	}

	match, civ, err := service.QueryCivitAI(c, cacheToUse, hash)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}
	if c.QueryParam("civitai") == "true" {
		return c.JSON(http.StatusOK, civ)
	}
	if err = Database.UpsertModel(match); err != nil {
		c.Logger().Errorf("error inserting model %s: %s", hash, err)
	}

	return c.JSON(http.StatusOK, match)
}
