package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ellypaws/inkbunny-app/api/cache"
	"github.com/ellypaws/inkbunny-app/api/civitai"
	. "github.com/ellypaws/inkbunny-app/api/entities"
	"github.com/ellypaws/inkbunny-app/api/service"
	"github.com/ellypaws/inkbunny-app/cmd/app"
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/ellypaws/inkbunny-sd/entities"
	sd "github.com/ellypaws/inkbunny-sd/stable_diffusion"
	"github.com/ellypaws/inkbunny-sd/utils"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	units "github.com/labstack/gommon/bytes"
	"github.com/redis/go-redis/v9"
	"net/http"
	"path/filepath"
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
	"/inkbunny/search":          handler{GetInkbunnySearch, append(loggedInMiddleware, withCache...)},
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

	details, err := api.Credentials{Sid: request.SID}.SubmissionDetails(
		api.SubmissionDetailsRequest{
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

	details, err := api.Credentials{Sid: request.SID}.SubmissionDetails(request)
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
		Random:             api.Yes,
		Type:               api.SubmissionTypes{api.SubmissionTypePicturePinup},
	}
	var bind = struct {
		*api.SubmissionSearchRequest
		SessionID  *string `query:"sid"`
		SearchTerm *string `json:"text,omitempty" query:"text"`
		UserID     *string `json:"user_id,omitempty" query:"user_id"`
		Username   *string `json:"username,omitempty" query:"username"`
		Types      *string `json:"types,omitempty" query:"types"`
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

	if cookie, err := c.Cookie("sid"); request.SID == "" && err == nil {
		request.SID = cookie.Value
	}

	if bind.SessionID != nil {
		request.SID = *bind.SessionID
	}

	if bind.SearchTerm != nil {
		request.Text = *bind.SearchTerm
	}

	if bind.UserID != nil {
		request.UserID = *bind.UserID
	}

	if bind.Username != nil {
		request.Username = *bind.Username
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

	user := &api.Credentials{Sid: request.SID}

	if user.Sid == "guest" {
		user, err = api.Guest().Login()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
		defer logoutGuest(c, user)
	}

	request.SID = user.Sid
	searchResponse, err := user.SearchSubmissions(request)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}
	if len(searchResponse.Submissions) == 0 {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no submissions found"})
	}

	if output := c.QueryParam("output"); output != "" {
		switch output {
		case "json":
			return c.JSON(http.StatusOK, searchResponse)
		case "xml":
			return c.XML(http.StatusOK, searchResponse)
		case "mail":
			return mail(c, user, searchResponse)
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

	details, err := user.SubmissionDetails(api.SubmissionDetailsRequest{
		SID:                         user.Sid,
		SubmissionIDs:               strings.Join(submissionIDs, ","),
		ShowDescription:             api.Yes,
		ShowDescriptionBbcodeParsed: api.Yes,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
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
//   - Set query "output" to "ticket", "submissions"
//   - Set query "parameters" to "true" to parse the utils.Params from json/text files
//   - Set query "interrogate" to "true" to parse entities.TaggerResponse from image files using (*sd.Host).Interrogate
//   - Set query "multiple" to "true" to separate each db.Ticket by submission
//   - Set query "stream" to "true" to receive multiple JSON objects
//   - Set query "full" to "true" to include the original api.Submission
func GetReviewHandler(c echo.Context) error {
	sid, _, err := GetSIDandID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
	}

	auditor, err := GetCurrentAuditor(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
	}

	submissionID := c.Param("id")
	if submissionID == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing submission ID"})
	}

	req := api.SubmissionDetailsRequest{
		SID:                         sid,
		SubmissionIDs:               submissionID,
		OutputMode:                  "json",
		ShowDescription:             true,
		ShowDescriptionBbcodeParsed: true,
	}

	cacheToUse := cache.SwitchCache(c)
	key := fmt.Sprintf("%v:inkbunny:submissions:%v", echo.MIMEApplicationJSON, submissionID)

	var submissionDetails api.SubmissionDetailsResponse
	item, errFunc := cacheToUse.Get(key)
	if errFunc == nil {
		if err := json.Unmarshal(item.Blob, &submissionDetails); err == nil {
			c.Logger().Infof("Cache hit for %s", key)
		}
	} else {
		c.Logger().Infof("Cache miss for %s retrieving submission...", key)
		submissionDetails, err = api.Credentials{Sid: sid}.SubmissionDetails(req)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
		bin, err := json.Marshal(submissionDetails)
		if err != nil {
			c.Logger().Errorf("error marshaling submission details: %v", err)
		}
		err = cacheToUse.Set(key, &cache.Item{
			Blob:       bin,
			LastAccess: time.Now().UTC(),
			MimeType:   echo.MIMEApplicationJSON,
		})
		if err != nil {
			c.Logger().Errorf("error caching submission details: %v", err)
		} else {
			c.Logger().Infof("Cached %s %s %dKiB", key, echo.MIMEApplicationJSON, len(bin)/units.KiB)
		}
	}

	if len(submissionDetails.Submissions) == 0 {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no submissions found"})
	}

	if c.QueryParam("interrogate") == "true" && !host.Alive() {
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
		labels := db.SubmissionLabels(submission)

		submissions[i] = details{
			URL:        submission.URL,
			ID:         api.IntString(submission.ID),
			User:       user,
			Submission: &submission,
		}

		if c.QueryParam("full") == "true" || c.QueryParam("multiple") == "true" {
			submissions[i].Inkbunny = &sub
			submissions[i].Ticket = &db.Ticket{
				ID:         submission.ID,
				Subject:    fmt.Sprintf("Review for %v", submission.URL),
				DateOpened: time.Now().UTC(),
				Status:     "triage",
				Labels:     labels,
				Priority:   "low",
				Closed:     false,
				Responses: []db.Response{
					{
						SupportTeam: false,
						User:        auditorAsUser,
						Date:        time.Now().UTC(),
						Message:     submissionMessage(&submission),
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
			for f, file := range sub.Files {
				if !strings.Contains(file.MimeType, "image") {
					continue
				}
				submissions[i].Images = append(submissions[i].Images, &submission.Files[f])
			}
		}
	}
	eachSubmission.Wait()
	if c.QueryParam("stream") == "true" {
		return nil
	}

	if c.QueryParam("multiple") == "true" {
		var tickets []db.Ticket
		for _, sub := range submissions {
			tickets = append(tickets, *sub.Ticket)
		}
		return c.JSON(http.StatusOK, tickets)
	}

	var ticketLabels []db.TicketLabel
	for _, sub := range submissions {
		ticketLabels = append(ticketLabels, db.SubmissionLabels(*sub.Submission)...)
	}

	switch c.QueryParam("output") {
	case "submissions":
		return c.JSON(http.StatusOK, submissions)
	default:
		return c.JSON(http.StatusOK, db.Ticket{
			ID:         1,
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
						sb.WriteString("The following submissions don't include their prompts: ")
						for _, sub := range submissions {
							sb.WriteString("\n")
							sb.WriteString(sub.URL)
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
		})
	}
}

func processObjectMetadata(c echo.Context, submission *db.Submission) {
	for _, obj := range submission.Metadata.Objects {
		meta := strings.ToLower(obj.Prompt + obj.NegativePrompt)
		for _, artist := range database.AllArtists() {
			if strings.Contains(meta, strings.ToLower(artist.Username)) {
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
	}
}

func submissionMessage(sub *db.Submission) string {
	var sb strings.Builder
	sb.WriteString("The following submission is pending review: ")
	sb.WriteString(sub.URL)

	flags := db.SubmissionLabels(*sub)
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

	if sub.Metadata.HasJSON {
		sb.WriteString("\n")
		sb.WriteString("The submission has a JSON file")
	}

	if sub.Metadata.HasTxt {
		sb.WriteString("\n")
		sb.WriteString("The submission has a text file")
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
		if sub.Metadata.Params != nil {
			sb.WriteString("\n")
			sb.WriteString("Parameters were detected but no AI keywords were found")
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

	if sub.Metadata.Objects != nil {
		processObjectMetadata(c, sub)
	}

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

	if textFile == nil {
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
		params, err = utils.Common(
			// prepend "photo 1" to the input in case it's missing
			utils.WithBytes(bytes.Join([][]byte{[]byte("photo 1"), b.Blob}, []byte("\n"))),
			utils.UseFairyGarden())
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
		if err == nil {
			sub.Metadata.Objects = map[string]entities.TextToImageRequest{"description_heuristics": heuristics}
		}
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

	submissionID := c.Param("id")
	if submissionID == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "missing submission ID"})
	}

	req := api.SubmissionDetailsRequest{
		SID:                         sid,
		SubmissionIDs:               submissionID,
		OutputMode:                  "json",
		ShowDescription:             true,
		ShowDescriptionBbcodeParsed: true,
	}

	submissionDetails, err := api.Credentials{Sid: sid}.SubmissionDetails(req)
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
		Params  *utils.Params                          `json:"params"`
		Objects map[string]entities.TextToImageRequest `json:"objects"`
	}
	var params = make(map[string]*p)
	for id, sub := range submissions {
		params[fmt.Sprintf("https://inkbunny.net/s/%d", id)] = &p{
			Params:  sub.Submission.Metadata.Params,
			Objects: sub.Submission.Metadata.Objects,
		}
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
		Blob:       bin,
		LastAccess: time.Now().UTC(),
		MimeType:   echo.MIMEApplicationJSON,
	})

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
			Blob:       bin,
			LastAccess: time.Now().UTC(),
			MimeType:   echo.MIMEApplicationJSON,
		})
		c.Logger().Infof("Cached %s %s %dKiB", key, echo.MIMEApplicationJSON, len(bin)/units.KiB)
	}

	return c.JSON(http.StatusOK, artistsWithIcon)
}

// GetModelsHandler returns a list of known models
// Set query "civitai" to "true" to return civitai.CivitAIModel
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

	full := c.QueryParam("civitai") == "true"

	if !full {
		if models := database.ModelNamesFromHash(hash); models != nil {
			return c.JSON(http.StatusOK, db.ModelHashes{hash: models})
		}
	}

	cacheToUse := cache.SwitchCache(c)

	c.Logger().Warnf("model %s not found, attempting to find", hash)

	var knownModels []entities.Lora
	knownLorasKey := fmt.Sprintf("%v:loras", echo.MIMEApplicationJSON)
	item, err := cacheToUse.Get(knownLorasKey)
	if err == nil {
		c.Logger().Infof("Cache hit for %s", knownLorasKey)
		if err := json.Unmarshal(item.Blob, &knownModels); err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
	} else {
		c.Logger().Warnf("Cache miss for %s, retrieving known models...", knownLorasKey)

		knownModels, err = host.GetLoras()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}

		bin, err := json.Marshal(knownModels)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}

		err = cacheToUse.Set(knownLorasKey, &cache.Item{
			Blob:       bin,
			LastAccess: time.Now().UTC(),
			MimeType:   echo.MIMEApplicationJSON,
		})
		if err != nil {
			c.Logger().Errorf("error caching known models: %v", err)
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
	}

	var match db.ModelHashes
	for _, lora := range knownModels {
		if h := lora.Metadata.SshsModelHash; h == nil {
			c.Logger().Warnf("model %s does not have a hash, calculating...", lora.Name)
		} else if *h == "" {
			c.Logger().Warnf("model %s contained an empty hash, calculating...", lora.Name)
			lora.Metadata.SshsModelHash = nil
		}

		autov3, err := sd.GetLoraHash(lora)
		if err != nil {
			if !errors.Is(err, sd.ErrNotSafeTensor) {
				c.Logger().Errorf("error calculating hash for %s: %v", lora.Name, err)
				return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
			}
		}

		if autov3 == "" {
			c.Logger().Warnf("model %s does not have a hash, skipping...", lora.Name)
			continue
		}

		names := []string{lora.Name}
		if lora.Alias != lora.Name &&
			lora.Alias != "None" &&
			lora.Alias != "" {
			names = append(names, lora.Alias)
		}
		if lora.Path != "" {
			names = append(names, filepath.Base(lora.Path))
		}
		h := db.ModelHashes{autov3: names}
		if err := database.UpsertModel(db.ModelHashes{autov3: names}); err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
		if hash == autov3 {
			match = h
		}
	}

	if len(match) == 0 {
		c.Logger().Warnf("model %s not found in known models, querying CivitAI...", hash)
	}

	key := fmt.Sprintf("%s:civitai:%s", echo.MIMEApplicationJSON, hash)
	item, err = cacheToUse.Get(key)
	if err == nil {
		c.Logger().Infof("Cache hit for %s", key)
		if full {
			return c.Blob(http.StatusOK, item.MimeType, item.Blob)
		}
	}

	model, err := civitai.DefaultHost.GetByHash(hash)
	if err != nil {
		c.Logger().Warnf("model %s not found in CivitAI", hash)
		if full {
			return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "model not found"})
		}
	} else {
		// TODO: Not yet implemented. This is where we download the model if found in CivitAI.
		err = sd.DownloadModel(hash)

		bin, err := json.Marshal(model)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}

		if err = cacheToUse.Set(key, &cache.Item{
			Blob:       bin,
			LastAccess: time.Now().UTC(),
			MimeType:   echo.MIMEApplicationJSON,
		}); err != nil {
			c.Logger().Errorf("error caching CivitAI model %s: %v", hash, err)
		} else {
			c.Logger().Infof("Cached %s %s %dKiB", key, echo.MIMEApplicationJSON, len(bin)/units.KiB)
		}

		var name string
		for _, file := range model.Files {
			var hashToCheck string
			switch len(hash) {
			case 10:
				hashToCheck = file.Hashes.AutoV2
			case 12:
				hashToCheck = file.Hashes.AutoV3
			}
			if hash == hashToCheck {
				name = file.Name
				if !file.Primary {
					c.Logger().Warnf("model %s has a non-primary file: %s", model.Name, file.Name)
				}
				c.Logger().Infof("download url is %s", file.DownloadURL)
				break
			}
		}

		if name == "" {
			msg := fmt.Sprintf("hash %s not found in model %s", hash, model.Name)
			c.Logger().Error(msg)
			return c.JSON(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: msg, Debug: model})
		}

		if err = database.UpsertModel(db.ModelHashes{hash: []string{model.Name, name}}); err != nil {
			c.Logger().Errorf("error inserting model %s: %s", hash, err)
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}

		if full {
			return c.JSON(http.StatusOK, model)
		}
	}

	return c.JSON(http.StatusOK, match)
}
