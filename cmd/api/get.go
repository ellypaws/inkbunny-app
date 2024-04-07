package main

import (
	"fmt"
	. "github.com/ellypaws/inkbunny-app/api/entities"
	"github.com/ellypaws/inkbunny-app/cmd/app"
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-app/cmd/db"

	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"slices"
	"strings"
)

var getHandlers = pathHandler{
	"/":                         handler{Hello, nil},
	"/inkbunny/description":     handler{GetInkbunnyDescription, nil},
	"/inkbunny/submission":      handler{GetInkbunnySubmission, nil},
	"/inkbunny/submission/:ids": handler{GetInkbunnySubmission, nil},
	"/inkbunny/search":          handler{GetInkbunnySearch, nil},
	"/image":                    handler{GetImageHandler, nil},
	"/tickets/audits":           handler{GetAuditHandler, staffMiddleware},
	"/tickets/get":              handler{GetTicketsHandler, staffMiddleware},
	"/auditors":                 handler{GetAuditors, staffMiddleware},
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
	auditor, err := GetAuditor(c)
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
	auditor, err := GetAuditor(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
	}

	tickets, err := database.GetTicketsByAuditor(auditor.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	return c.JSON(http.StatusOK, tickets)
}

func validAuditor(user api.Credentials) bool {
	if err := db.Error(database); err != nil {
		log.Printf("warning: validAuditor was called with a nil database: %v", err)
		return false
	}
	return database.IsAuditorRole(int64(user.UserID.Int()))
}

func GetAuditors(c echo.Context) error {
	auditors := database.AllAuditors()
	if auditors == nil {
		return c.JSON(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: "no auditors found"})
	}
	return c.JSON(http.StatusOK, auditors)
}
