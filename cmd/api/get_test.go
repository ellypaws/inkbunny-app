package main

import (
	"bytes"
	"encoding/json"
	"github.com/ellypaws/inkbunny-app/cmd/app"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHelloWorld(t *testing.T) {
	// Setup
	var userJSON string
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(userJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Assertions
	if assert.NoError(t, Hello(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "Hello, World!", rec.Body.String())
	}
}

func TestGetInkbunnyDescription(t *testing.T) {
	// Setup
	request := DescriptionRequest{
		SID:           "session_id",
		SubmissionIDs: "14576",
	}

	user, err := api.Guest().Login()
	if !assert.NoError(t, err) {
		return
	}

	request.SID = user.Sid
	userJSON, err := json.Marshal(request)
	if !assert.NoError(t, err) {
		return
	}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/inkbunny/description", bytes.NewReader(userJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Assertions
	if assert.NoError(t, GetInkbunnyDescription(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var descriptionResponse []DescriptionResponse
		err := json.Unmarshal(rec.Body.Bytes(), &descriptionResponse)
		assert.NoError(t, err)
		assert.NotEmpty(t, descriptionResponse)
		err = user.Logout()
		assert.NoError(t, err)
	}
}

func TestGetInkbunnySubmission(t *testing.T) {
	// Setup
	request := api.SubmissionDetailsRequest{
		SID:           "session_id",
		SubmissionIDs: "14576",
	}

	user, err := api.Guest().Login()
	if !assert.NoError(t, err) {
		return
	}

	request.SID = user.Sid
	userJSON, err := json.Marshal(request)
	if !assert.NoError(t, err) {
		return
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/inkbunny/submission", bytes.NewReader(userJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Assertions
	if assert.NoError(t, GetInkbunnySubmission(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var submissionDetails api.SubmissionDetailsResponse
		err := json.Unmarshal(rec.Body.Bytes(), &submissionDetails)
		assert.NoError(t, err)
		assert.NotEmpty(t, submissionDetails.Submissions)
		err = user.Logout()
		assert.NoError(t, err)
	}
}

func TestGetInkbunnySearch(t *testing.T) {
	// Setup
	request := api.SubmissionSearchRequest{
		SID:   "session_id",
		Text:  "Inkbunny Logo (Mascot Only)",
		Title: api.Yes,
	}

	user, err := api.Guest().Login()
	if !assert.NoError(t, err) {
		return
	}

	request.SID = user.Sid
	userJSON, err := json.Marshal(request)
	if !assert.NoError(t, err) {
		return
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/inkbunny/search?output=json&temp=no", bytes.NewReader(userJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Assertions
	if assert.NoError(t, GetInkbunnySearch(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var searchResponse api.SubmissionSearchResponse
		err := json.Unmarshal(rec.Body.Bytes(), &searchResponse)
		assert.NoError(t, err)
		assert.NotEmpty(t, searchResponse.Submissions)
	}

	req = httptest.NewRequest(http.MethodGet, "/inkbunny/search?output=mail&temp=yes", bytes.NewReader(userJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	// Assertions
	if assert.NoError(t, GetInkbunnySearch(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var mails []app.Mail
		err := json.Unmarshal(rec.Body.Bytes(), &mails)
		assert.NoError(t, err)
		assert.NotEmpty(t, mails)
	}

	err = user.Logout()
	assert.NoError(t, err)
}
