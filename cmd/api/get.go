package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ellypaws/inkbunny-app/api/cache"
	. "github.com/ellypaws/inkbunny-app/api/entities"
	"github.com/ellypaws/inkbunny-app/api/service"
	"github.com/ellypaws/inkbunny-app/cmd/app"
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/utils"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	units "github.com/labstack/gommon/bytes"
	"github.com/redis/go-redis/v9"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

var getHandlers = pathHandler{
	"/":                         handler{Hello, withCache},
	"/inkbunny/description":     handler{GetInkbunnyDescription, withCache},
	"/inkbunny/submission":      handler{GetInkbunnySubmission, withCache},
	"/inkbunny/submission/:ids": handler{GetInkbunnySubmission, withCache},
	"/inkbunny/search":          handler{GetInkbunnySearch, append(loggedInMiddleware, withRedis...)},
	"/image":                    handler{GetImageHandler, append(staticMiddleware, SIDMiddleware)},
	"/review/:id":               handler{GetReviewHandler, append(staffMiddleware, withRedis...)},
	"/heuristics/:id":           handler{GetHeuristicsHandler, append(loggedInMiddleware, withRedis...)},
	"/audits":                   handler{GetAuditHandler, staffMiddleware},
	"/tickets":                  handler{GetTicketsHandler, staffMiddleware},
	"/auditors":                 handler{GetAllAuditorsJHandler, staffMiddleware},
	"/robots.txt":               handler{robots, staticMiddleware},
	"/favicon.ico":              handler{favicon, staticMiddleware},
	"/username/:username":       handler{GetUsernameHandler, append(loggedInMiddleware, withRedis...)},
	"/avatar/:username":         handler{GetAvatarHandler, staticMiddleware},
	"/artists":                  handler{GetArtistsHandler, append(loggedInMiddleware, withRedis...)},
	"/models":                   handler{GetModelsHandler, withCache},
	"/models/:hash":             handler{GetModelsHandler, withRedis},
}

func robots(c echo.Context) error {
	return c.File("public/robots.txt")
}

func favicon(c echo.Context) error {
	return c.File("public/16930_inkbunny_inkbunnylogo_trans_rev_outline.ico")
}

// Deprecated: use registerAs((*echo.Echo).GET, getHandlers) instead
func registerGetRoutes(e *echo.Echo) {
	registerAs(e.GET, getHandlers)
}

func Hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
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

	audits, err := database.GetAuditsByAuditor(auditor.UserID)
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

		tickets, err := database.GetTicketsByAuditor(p)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}

		return c.JSON(http.StatusOK, tickets)
	}

	tickets, err := database.GetAllTickets()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	return c.JSON(http.StatusOK, tickets)
}

func validAuditor(c echo.Context, user api.Credentials) bool {
	if err := db.Error(database); err != nil {
		c.Logger().Warnf("warning: validAuditor was called with a nil database: %v", err)
		return false
	}
	return database.IsAuditorRole(int64(user.UserID.Int()))
}

func GetAllAuditorsJHandler(c echo.Context) error {
	auditors := database.AllAuditors()
	if auditors == nil {
		return c.JSON(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: "no auditors found"})
	}
	return c.JSON(http.StatusOK, auditors)
}

// GetReviewHandler returns heuristic analysis of a submission
//   - Set query "output" to "single_ticket", "multiple_tickets", "submissions", "full".
//
// - single_ticket: returns a single combined db.Ticket of all submissions
//
// - multiple_tickets: returns each db.Ticket for each submission
//
// - submissions: returns a []details of each submission
//
// - full: returns a []details db.Ticket with the original api.Submission and db.Ticket
// Note: "parameters" and "interrogate" won't automatically be set on full output
//
//   - Set query "parameters" to "true" to parse the utils.Params from json/text files
//   - Set query "interrogate" to "true" to parse entities.TaggerResponse from image files using (*sd.Host).Interrogate
//   - Set query "stream" to "true" to receive multiple JSON objects
func GetReviewHandler(c echo.Context) error {
	sid, _, err := GetSIDandID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
	}

	auditor, err := GetCurrentAuditor(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
	}

	submissionIDs := c.Param("id")
	if submissionIDs == "" || submissionIDs == "null" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing submission ID"})
	}

	req := api.SubmissionDetailsRequest{
		SID:                         sid,
		SubmissionIDs:               submissionIDs,
		OutputMode:                  "json",
		ShowDescription:             true,
		ShowDescriptionBbcodeParsed: true,
	}

	output := c.QueryParam("output")
	parameters := c.QueryParam("parameters")
	interrogate := c.QueryParam("interrogate")
	stream := c.QueryParam("stream")

	const (
		outputSingleTicket    = "single_ticket"
		outputMultipleTickets = "multiple_tickets"
		outputSubmissions     = "submissions"
		outputFull            = "full"
	)

	validOutputs := []string{
		outputSingleTicket,
		outputMultipleTickets,
		outputSubmissions,
		outputFull,
	}

	if output == "" {
		output = outputSingleTicket
	} else if !slices.Contains(validOutputs, output) {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{
			ErrorString: fmt.Sprintf("invalid output format %s. valid options are: %v", output, validOutputs),
			Debug:       output,
		})
	}

	cacheToUse := cache.SwitchCache(c)
	query := url.Values{
		"sid":         {sid},
		"parameters":  {parameters},
		"interrogate": {interrogate},
	}
	reviewKey := fmt.Sprintf(
		"%s:review:%s:%s?%s",
		echo.MIMEApplicationJSON,
		output,
		submissionIDs,
		query.Encode(),
	)
	item, errFunc := cacheToUse.Get(reviewKey)
	if errFunc == nil {
		c.Logger().Infof("Cache hit for %s", reviewKey)
		return c.Blob(http.StatusOK, item.MimeType, item.Blob)
	}
	c.Logger().Debugf("Cache miss for %s retrieving review...", reviewKey)
	var store any
	defer func() {
		if store == nil {
			return
		}
		bin, err := json.Marshal(store)
		if err != nil {
			c.Logger().Errorf("error marshaling review: %v", err)
			return
		}

		err = cacheToUse.Set(reviewKey, &cache.Item{
			Blob:     bin,
			MimeType: echo.MIMEApplicationJSON,
		}, cache.Week)
		if err != nil {
			c.Logger().Errorf("error caching review: %v", err)
			return
		}
		c.Logger().Infof("Cached %s %s %dKiB", reviewKey, echo.MIMEApplicationJSON, len(bin)/units.KiB)
	}()

	submissionDetails, err := service.RetrieveSubmission(c, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if len(submissionDetails.Submissions) == 0 {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no submissions found"})
	}

	if interrogate == "true" && !host.Alive() {
		c.Logger().Warn("interrogate was set to true but host is offline, only using cached captions...")
	}

	type details struct {
		URL        string          `json:"url"`
		ID         api.IntString   `json:"id"`
		User       api.UsernameID  `json:"user"`
		Submission *db.Submission  `json:"submission,omitempty"`
		Inkbunny   *api.Submission `json:"inkbunny,omitempty"`
		Ticket     *db.Ticket      `json:"ticket,omitempty"`
		Images     []*db.File      `json:"images,omitempty"`
	}

	var submissions = make([]details, len(submissionDetails.Submissions))

	auditorAsUser := auditorAsUsernameID(auditor)

	var eachSubmission sync.WaitGroup
	var dbMutex sync.Mutex
	for i, sub := range submissionDetails.Submissions {
		eachSubmission.Add(1)

		submission := db.InkbunnySubmissionToDBSubmission(sub)
		go processSubmission(c, &eachSubmission, &dbMutex, &submission)

		user := api.UsernameID{UserID: sub.UserID, Username: sub.Username}

		submissions[i] = details{
			URL:        submission.URL,
			ID:         api.IntString(submission.ID),
			User:       user,
			Submission: &submission,
		}

		switch output {
		case outputFull:
			submissions[i].Inkbunny = &sub
			for f, file := range sub.Files {
				if !strings.Contains(file.MimeType, "image") {
					continue
				}
				submissions[i].Images = append(submissions[i].Images, &submission.Files[f])
			}
			fallthrough
		case outputMultipleTickets:
			submissions[i].Ticket = &db.Ticket{
				ID:         submission.ID,
				Subject:    fmt.Sprintf("Review for %v", submission.URL),
				DateOpened: time.Now().UTC(),
				Status:     "triage",
				Labels:     nil,
				Priority:   "low",
				Closed:     false,
				Responses: []db.Response{
					{
						SupportTeam: false,
						User:        auditorAsUser,
						Date:        time.Now().UTC(),
						Message:     "",
					},
				},
				SubmissionIDs: []int64{int64(submission.ID)},
				AssignedID:    &auditor.UserID,
				UsersInvolved: db.Involved{
					Reporter: auditorAsUser,
					ReportedIDs: []api.UsernameID{
						user,
					},
				},
			}
		}
	}
	eachSubmission.Wait()
	if stream == "true" {
		return nil
	}

	for i, sub := range submissions {
		submissions[i].Ticket.Labels = db.TicketLabels(*sub.Submission)
		submissions[i].Ticket.Responses[0].Message = submissionMessage(sub.Submission)
	}

	switch output {
	case outputSubmissions, outputFull:
		store = submissions
		return c.JSON(http.StatusOK, submissions)
	case outputMultipleTickets:
		var tickets []db.Ticket
		for _, sub := range submissions {
			tickets = append(tickets, *sub.Ticket)
		}
		store = tickets
		return c.JSON(http.StatusOK, tickets)
	case outputSingleTicket:
		fallthrough
	default:
		var ticketLabels []db.TicketLabel
		for _, sub := range submissions {
			for _, label := range sub.Ticket.Labels {
				if !slices.Contains(ticketLabels, label) {
					ticketLabels = append(ticketLabels, label)
				}
			}
		}
		ticket := db.Ticket{
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
						for _, sub := range submissions {
							if sb.Len() > 0 {
								sb.WriteString("\n\n[s]                    [s]")
							}
							sb.WriteString(fmt.Sprintf("#M%s\n", sub.ID))
							sb.WriteString(submissionMessage(sub.Submission))
						}
						return sb.String()
					}(),
				},
			},
			SubmissionIDs: func() []int64 {
				var ids []int64
				for _, sub := range submissions {
					ids = append(ids, int64(sub.ID))
				}
				return ids
			}(),
			AssignedID: &auditor.UserID,
			UsersInvolved: db.Involved{
				Reporter: auditorAsUser,
				ReportedIDs: func() []api.UsernameID {
					var ids []api.UsernameID
					for _, sub := range submissions {
						ids = append(ids, sub.User)
					}
					return ids
				}(),
			},
		}
		store = ticket
		return c.JSON(http.StatusOK, ticket)
	}
}

func processObjectMetadata(submission *db.Submission) {
	submission.Metadata.MissingPrompt = true
	submission.Metadata.MissingModel = true

	artists := database.AllArtists()
	for _, obj := range submission.Metadata.Objects {
		submission.Metadata.AISubmission = true
		meta := strings.ToLower(obj.Prompt + obj.NegativePrompt)
		for _, artist := range artists {
			re, err := regexp.Compile(fmt.Sprintf(`\b%s\b`, strings.ToLower(artist.Username)))
			if err != nil {
				continue
			}
			if re.MatchString(meta) {
				submission.Metadata.ArtistUsed = append(submission.Metadata.ArtistUsed, artist)
			}
		}

		privateTools := []string{
			"midjourney",
			"novelai",
		}

		for _, tool := range privateTools {
			if strings.Contains(meta, tool) {
				submission.Metadata.PrivateTool = true
				submission.Metadata.Generator = tool
				break
			}
		}

		if obj.Prompt != "" {
			submission.Metadata.MissingPrompt = false
		}

		if obj.OverrideSettings.SDModelCheckpoint != nil || obj.OverrideSettings.SDCheckpointHash != "" {
			submission.Metadata.MissingModel = false
		}
	}
}

func submissionMessage(sub *db.Submission) string {
	var sb strings.Builder
	sb.WriteString("The following submission is pending review: ")
	sb.WriteString(sub.URL)

	flags := db.TicketLabels(*sub)
	if len(flags) > 0 {
		sb.WriteString("\n")
		sb.WriteString("The following flags were detected: ")
		for i, flag := range flags {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(string(flag))
		}
	}

	if len(sub.Metadata.ArtistUsed) > 0 {
		sb.WriteString("\n")
		sb.WriteString("The prompt may have used these artists: ")
		for i, artist := range sub.Metadata.ArtistUsed {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("ib!")
			sb.WriteString(artist.Username)
		}
	}

	if sub.Metadata.HasJSON {
		sb.WriteString("\n")
		sb.WriteString("The submission has a JSON file")
	}

	if sub.Metadata.HasTxt {
		sb.WriteString("\n")
		sb.WriteString("The submission has a text file")
	}

	if sub.Metadata.MissingPrompt {
		sb.WriteString("\n")
		sb.WriteString("The submission is missing the prompt")
	}

	if len(sub.Metadata.AIKeywords) > 0 {
		sb.WriteString("\n")
		sb.WriteString("The submission has the following AI keywords: ")
		for i, keyword := range sub.Metadata.AIKeywords {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(keyword)
		}
		if sub.Metadata.Params == nil {
			sb.WriteString("\n")
			sb.WriteString("The submission is potentially missing parameters")
		}
	} else {
		if sub.Metadata.AISubmission {
			sb.WriteString("\n")
			sb.WriteString("The submission was detected to have AI content, but was not tagged as such")
		}
	}

	if sub.Metadata.DetectedHuman {
		sb.WriteString("\n")
		if !sub.Metadata.TaggedHuman {
			sb.WriteString("A human was detected in the submission but was not tagged\n")
		} else {
			sb.WriteString("A human was detected in the submission and was tagged\n")
		}
		sb.WriteString("The detection rate is: ")
		sb.WriteString(fmt.Sprintf("%.2f", sub.Metadata.HumanConfidence))
	}

	return sb.String()
}

func processSubmission(c echo.Context, eachSubmission *sync.WaitGroup, mutex *sync.Mutex, sub *db.Submission) {
	defer eachSubmission.Done()
	var fileWaitGroup sync.WaitGroup
	if len(sub.Files) > 0 {
		fileWaitGroup.Add(1)
		c.Logger().Infof("processing files for %s %s", sub.URL, sub.Title)
		go parseFiles(c, &fileWaitGroup, sub)
	}
	fileWaitGroup.Wait()

	mutex.Lock()
	defer mutex.Unlock()

	for _, obj := range sub.Metadata.Objects {
		for hash, model := range obj.LoraHashes {
			database.Wait()
			err := database.UpsertModel(db.ModelHashes{
				hash: []string{model},
			})
			if err != nil {
				c.Logger().Errorf("error inserting model %s: %s", hash, err)
			}
		}
	}

	if c.QueryParam("stream") == "true" {
		enc := json.NewEncoder(c.Response())
		if err := enc.Encode(sub); err != nil {
			c.Logger().Errorf("error encoding submission %v: %v", sub.ID, err)
		}
		c.Logger().Debugf("flushing %v", sub.ID)
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)
		c.Get("writer").(http.Flusher).Flush()
		c.Logger().Infof("finished processing %v", sub.ID)
	}

	err := database.InsertSubmission(*sub)
	if err != nil {
		c.Logger().Errorf("error inserting submission %v: %v", sub.ID, err)
	}
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

func parseFiles(c echo.Context, wg *sync.WaitGroup, sub *db.Submission) {
	defer wg.Done()
	if c.QueryParam("parameters") == "true" {
		wg.Add(1)
		go processParams(c, wg, sub)
	}
	if c.QueryParam("interrogate") != "true" {
		return
	}
	for i := range sub.Files {
		wg.Add(1)
		go service.ProcessCaption(c, wg, sub, i, host)
	}
}

func processParams(c echo.Context, wg *sync.WaitGroup, sub *db.Submission) {
	defer wg.Done()

	if sub.Metadata.Params != nil {
		return
	}

	var textFile *db.File
	for i, f := range sub.Files {
		if strings.HasSuffix(f.File.MimeType, "json") {
			textFile = &sub.Files[i]
			break
		}
		if strings.HasPrefix(f.File.MimeType, "text") {
			textFile = &sub.Files[i]
			break
		}
	}

	defer processObjectMetadata(sub)

	if textFile == nil {
		c.Logger().Debugf("processing description heuristics for %v", sub.URL)
		heuristics, err := utils.DescriptionHeuristics(sub.Description)
		if err != nil {
			c.Logger().Errorf("error processing description heuristics for %v: %v", sub.URL, err)
		}
		if reflect.DeepEqual(heuristics, entities.TextToImageRequest{}) {
			c.Logger().Debugf("no heuristics found for %v", sub.URL)
			return
		}
		sub.Metadata.Objects = map[string]entities.TextToImageRequest{sub.Title: heuristics}
		return
	}

	cacheToUse := cache.SwitchCache(c)

	b, errFunc := cache.Retrieve(c, cacheToUse, cache.Fetch{
		Key:      fmt.Sprintf("%s:%s", textFile.File.MimeType, textFile.File.FileURLFull),
		URL:      textFile.File.FileURLFull,
		MimeType: textFile.File.MimeType,
	})
	if errFunc != nil {
		return
	}

	if sub.Metadata.Params != nil {
		return
	}

	// Because some artists already have standardized txt files, opt to split each file separately
	var params utils.Params
	var err error
	f := &textFile.File
	c.Logger().Debugf("processing params for %v", f.FileName)
	switch {
	case strings.Contains(f.FileName, "_AutoSnep_"):
		params, err = utils.AutoSnep(utils.WithBytes(b.Blob))
	case strings.Contains(f.FileName, "_druge_"):
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseDruge())
	case strings.Contains(f.FileName, "_AIBean_"):
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseAIBean())
	case strings.Contains(f.FileName, "_artiedragon_"):
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseArtie())
	case strings.Contains(f.FileName, "_picker52578_"):
		params, err = utils.Common(
			utils.WithBytes(b.Blob),
			utils.WithFilename("picker52578_"),
			utils.WithKeyCondition(func(line string) bool { return strings.HasPrefix(line, "File Name") }))
	case strings.Contains(f.FileName, "_fairygarden_"):
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseFairyGarden())
	case strings.Contains(f.FileName, "_Cirn0_"):
		params, err = utils.Common(utils.WithBytes(b.Blob), utils.UseCirn0())
	default:
		params, err = utils.Common(
			// prepend "photo 1" to the input in case it's missing
			utils.WithBytes(bytes.Join([][]byte{[]byte(f.FileName), b.Blob}, []byte("\n"))),
			utils.WithKeyCondition(func(line string) bool { return strings.HasPrefix(line, f.FileName) }))
	}
	if err != nil {
		c.Logger().Errorf("error processing params for %v: %v", f.FileName, err)
		return
	}
	if len(params) > 0 {
		c.Logger().Debugf("finished params for %v", f.FileName)
		sub.Metadata.Params = &params
		parseObjects(c, sub)
	}
	if len(sub.Metadata.Objects) == 0 {
		c.Logger().Debugf("processing description heuristics for %v", sub.URL)
		heuristics, err := utils.DescriptionHeuristics(sub.Description)
		if err != nil {
			c.Logger().Errorf("error processing description heuristics for %v: %v", sub.URL, err)
		}
		if reflect.DeepEqual(heuristics, entities.TextToImageRequest{}) {
			c.Logger().Debugf("no heuristics found for %v", sub.URL)
			return
		}
		sub.Metadata.Objects = map[string]entities.TextToImageRequest{sub.Title: heuristics}
		return
	}
}

func parseObjects(c echo.Context, sub *db.Submission) {
	if sub.Metadata.Objects != nil {
		return
	}

	var wg sync.WaitGroup
	var mutex sync.Mutex
	for fileName, params := range *sub.Metadata.Params {
		if p, ok := params[utils.Parameters]; ok {
			c.Logger().Debugf("processing heuristics for %v", fileName)
			wg.Add(1)
			go func(name string, content string) {
				defer wg.Done()
				heuristics, err := utils.ParameterHeuristics(content)
				if err != nil {
					c.Logger().Errorf("error processing heuristics for %v: %v", name, err)
					return
				}
				if sub.Metadata.Objects == nil {
					sub.Metadata.Objects = make(map[string]entities.TextToImageRequest)
				}
				mutex.Lock()
				sub.Metadata.Objects[name] = heuristics
				mutex.Unlock()
			}(fileName, p)
		}
	}
	wg.Wait()
}

func auditorAsUsernameID(auditor *db.Auditor) api.UsernameID {
	return api.UsernameID{UserID: strconv.FormatInt(auditor.UserID, 10), Username: auditor.Username}
}

// GetHeuristicsHandler returns the heuristics of a submission
func GetHeuristicsHandler(c echo.Context) error {
	sid, _, err := GetSIDandID(c)
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

	var waitGroup sync.WaitGroup
	var mutex sync.Locker
	for _, sub := range submissionDetails.Submissions {
		waitGroup.Add(1)

		submission := db.InkbunnySubmissionToDBSubmission(sub)
		go func(wg *sync.WaitGroup, sub *db.Submission) {
			defer wg.Done()
			var p sync.WaitGroup
			p.Add(1)
			go processParams(c, &p, sub)
			p.Wait()
			if c.QueryParam("stream") == "true" {
				mutex.Lock()
				defer mutex.Unlock()
				if c.QueryParam("stream") == "true" {
					enc := json.NewEncoder(c.Response())
					if err := enc.Encode(sub); err != nil {
						c.Logger().Errorf("error encoding submission %v: %v", sub.ID, err)
					}
					c.Logger().Debugf("flushing %v", sub.ID)
					c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
					c.Response().WriteHeader(http.StatusOK)
					c.Get("writer").(http.Flusher).Flush()
					c.Logger().Infof("finished processing %v", sub.ID)
				}
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
		Params  *utils.Params                          `json:"params"`
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
	artists := database.AllArtists()

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
		models, err := database.AllModels()
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
			if models := database.ModelNamesFromHash(hash); models != nil {
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
			match, err = service.QueryHost(c, cacheToUse, host, database, hash)
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
	if err = database.UpsertModel(match); err != nil {
		c.Logger().Errorf("error inserting model %s: %s", hash, err)
	}

	return c.JSON(http.StatusOK, match)
}
