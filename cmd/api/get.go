package main

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"fmt"
	. "github.com/ellypaws/inkbunny-app/api/entities"
	"github.com/ellypaws/inkbunny-app/cmd/app"
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/utils"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	"net/http"
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
	"/inkbunny/search":          handler{GetInkbunnySearch, withCache},
	"/image":                    handler{GetImageHandler, staticMiddleware},
	"/review/:id":               handler{GetReviewHandler, append(staffMiddleware, CacheMiddleware)},
	"/tickets/audits":           handler{GetAuditHandler, staffMiddleware},
	"/tickets/get":              handler{GetTicketsHandler, staffMiddleware},
	"/auditors":                 handler{GetAllAuditorsJHandler, staffMiddleware},
	"/robots.txt":               handler{robots, staticMiddleware},
}

func robots(c echo.Context) error {
	return c.File("public/robots.txt")
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
		Type:               api.SubmissionTypePicturePinup,
	}
	var bind = struct {
		*api.SubmissionSearchRequest
		SessionID  *string `query:"sid"`
		SearchTerm *string `query:"text"`
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
				NonCustomThumb:     file.ThumbURLLargeNonCustom,
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
//   - Set query "heuristics" to "true" to parse entities.TextToImageRequest using utils.ParameterHeuristics
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

	submissionDetails, err := api.Credentials{Sid: sid}.SubmissionDetails(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if len(submissionDetails.Submissions) == 0 {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no submissions found"})
	}

	var submissions = make(map[int64]*db.Submission)
	var userIDs []api.UsernameID
	var ticketLabels []db.TicketLabel
	var submissionIDsString []string
	var submissionIDsInt64 []int64
	var wg sync.WaitGroup
	for i := range submissionDetails.Submissions {
		submission := db.InkbunnySubmissionToDBSubmission(submissionDetails.Submissions[i])
		if len(submissionDetails.Submissions[i].Files) > 0 {
			c.Logger().Infof("processing files for %v", submission.ID)
			wg.Add(1)
			go parseFiles(c, &wg, &submission)
		}
		submissions[submission.ID] = &submission
		userIDs = append(userIDs, api.UsernameID{UserID: submissionDetails.Submissions[i].UserID, Username: submissionDetails.Submissions[i].Username})
		ticketLabels = append(ticketLabels, db.SubmissionLabels(submission)...)
		submissionIDsString = append(submissionIDsString, submission.URL)
		submissionIDsInt64 = append(submissionIDsInt64, submission.ID)
	}
	wg.Wait()

	var lastErr error
	for _, sub := range submissions {
		err := database.InsertSubmission(*sub)
		if err != nil {
			lastErr = err
		}
	}
	if lastErr != nil {
		return c.JSON(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: "some submissions failed to insert", Debug: err})
	}

	auditorAsUser := auditorAsUsernameID(auditor)

	ticket := db.Ticket{
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
				Message: fmt.Sprintf("The following submission doesn't include the prompts: \n%v",
					strings.Join(submissionIDsString, "\n")),
			},
		},
		SubmissionIDs: submissionIDsInt64,
		AssignedID:    &auditor.UserID,
		UsersInvolved: db.Involved{
			Reporter:    auditorAsUser,
			ReportedIDs: userIDs,
		},
	}

	switch c.QueryParam("output") {
	case "submissions":
		return c.JSON(http.StatusOK, submissions)
	default:
		return c.JSON(http.StatusOK, ticket)
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
	for i := range sub.Files {
		if c.QueryParam("parameters") == "true" {
			wg.Add(1)
			go processParams(c, wg, sub, c.QueryParam("heuristics") == "true")
		}
		if c.QueryParam("interrogate") == "true" {
			wg.Add(1)
			go processCaptions(c, wg, sub, i)
		}
	}
}

func processCaptions(c echo.Context, wg *sync.WaitGroup, sub *db.Submission, i int) {
	defer wg.Done()
	f := &sub.Files[i].File
	if !strings.HasPrefix(f.MimeType, "image") {
		return
	}
	c.Set("cache", fileCache)
	item, errorFunc := GetCache(c, f.FileName, f.FileURLScreen)
	if errorFunc != nil {
		return
	}
	req := defaultTaggerRequest

	base64String := base64.StdEncoding.EncodeToString(item.Blob)
	req.Image = &base64String
	*req.Threshold = 0.7

	c.Logger().Debugf("processing captions for %v", f.FileURLFull)
	t, err := host.Interrogate(&req)
	if err != nil {
		c.Logger().Errorf("error processing captions for %v: %v", f.FileURLFull, err)
		return
	}
	c.Logger().Debugf("finished captions for %v", f.FileURLFull)

	sub.Metadata.HumanConfidence = max(sub.Metadata.HumanConfidence, t.HumanPercent())
	if t.HumanPercent() > 0.5 {
		sub.Metadata.DetectedHuman = true
	}

	sub.Files[i].Caption = &t.Caption
}

func processParams(c echo.Context, wg *sync.WaitGroup, sub *db.Submission, heuristics bool) {
	defer wg.Done()

	if sub.Metadata.HasGenerationDetails {
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
	c.Logger().Debugf("getting file for %v", textFile.File.FileName)
	c.Request().Header.Set("Accept", "text/plain")
	b, errFunc := GetCache(c, textFile.File.FileName, textFile.File.FileURLFull)
	if errFunc != nil {
		return
	}

	if sub.Metadata.HasGenerationDetails {
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
	if params != nil {
		c.Logger().Debugf("finished params for %v", f.FileName)
		sub.Metadata.HasGenerationDetails = true
		sub.Metadata.Params = &params
		if heuristics {
			c.Logger().Debugf("processing object for %v", f.FileName)
			parseObjects(c, wg, sub)
		}
	}
}

func parseObjects(c echo.Context, wg *sync.WaitGroup, sub *db.Submission) {
	if sub.Metadata.Objects != nil {
		return
	}

	var mutex sync.Mutex
	for fileName, params := range *sub.Metadata.Params {
		if p, ok := params[utils.Parameters]; ok {
			c.Logger().Debugf("processing heuristics for %v", fileName)
			wg.Add(1)
			go func(name string, content string) {
				defer wg.Done()
				defer c.Logger().Debugf("finished heuristics for %v", name)
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
}

func auditorAsUsernameID(auditor *db.Auditor) api.UsernameID {
	return api.UsernameID{UserID: strconv.FormatInt(auditor.UserID, 10), Username: auditor.Username}
}
