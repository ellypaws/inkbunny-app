package main

import (
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	"net/http"
)

var getRoutes = map[string]func(c echo.Context) error{
	"/":                     Hello,
	"/inkbunny/description": GetInkbunnyDescription,
	"/inkbunny/submission":  GetInkbunnySubmission,
	"/inkbunny/search":      GetInkbunnySearch,
}

func registerGetRoutes(e *echo.Echo) {
	for path, handler := range getRoutes {
		e.GET(path, handler)
	}
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

	if sid := c.QueryParams().Get("sid"); sid != "" {
		request.SID = sid
	}

	if request.SID == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{Error: "missing SID"})
	}

	if submissionIDs := c.QueryParam("submission_ids"); submissionIDs != "" {
		if request.SubmissionIDs != "" {
			request.SubmissionIDs += ","
		}
		request.SubmissionIDs += submissionIDs
	}

	if request.SubmissionIDs == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{Error: "missing submission ID"})
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
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{Error: "no submissions found"})
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
func GetInkbunnySubmission(c echo.Context) error {
	var request api.SubmissionDetailsRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}

	details, err := api.Credentials{Sid: request.SID}.SubmissionDetails(request)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}
	if len(details.Submissions) == 0 {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{Error: "no submissions found"})
	}

	return c.JSON(http.StatusOK, details)
}

// GetInkbunnySearch returns the search results of a submission using api.SubmissionSearchRequest
// It requires a valid SID to be passed in the request.
// It returns api.SubmissionSearchResponse as JSON.
func GetInkbunnySearch(c echo.Context) error {
	var request api.SubmissionSearchRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}

	searchResponse, err := api.Credentials{Sid: request.SID}.SearchSubmissions(request)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}
	if len(searchResponse.Submissions) == 0 {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{Error: "no submissions found"})
	}

	return c.JSON(http.StatusOK, searchResponse)
}
