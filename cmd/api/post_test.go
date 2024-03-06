package main

import (
	"bytes"
	"encoding/json"
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/llm"
	"github.com/ellypaws/inkbunny-sd/utils"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogin(t *testing.T) {
	// Setup
	userJSON := `{"username":"guest","password":""}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(userJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Assertions
	if assert.NoError(t, login(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var loginResponse api.Credentials
		err := json.Unmarshal(rec.Body.Bytes(), &loginResponse)
		assert.NoError(t, err)
		assert.NotEmpty(t, loginResponse.Sid)
	}
}

func TestInference(t *testing.T) {
	// Setup
	request := InferenceRequest{
		Config: llm.Localhost(),
		Request: llm.Request{
			Messages: []llm.Message{
				llm.DefaultSystem,
				llm.UserMessage("Say hello!"),
			},
			Temperature:   1.0,
			MaxTokens:     32,
			Stream:        false,
			StreamChannel: nil,
		},
	}
	llmRequestJSON, err := json.Marshal(request)
	assert.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/llm", bytes.NewReader(llmRequestJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Assertions
	if assert.NoError(t, inference(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var inferenceResponse llm.Response
		err := json.Unmarshal(rec.Body.Bytes(), &inferenceResponse)
		assert.NoError(t, err)
		assert.NotEmpty(t, inferenceResponse.Choices)
	}
}

func TestInferenceComplete(t *testing.T) {
	user, err := api.Guest().Login()
	if !assert.NoError(t, err) {
		return
	}

	err = user.ChangeRating(api.Ratings{
		General:        true,
		Nudity:         true,
		MildViolence:   true,
		Sexual:         true,
		StrongViolence: true,
	})
	if !assert.NoError(t, err) {
		return
	}

	searchResponse := searchAIGenerated(err, user)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, searchResponse.Submissions)
		assert.NotEmpty(t, searchResponse.Submissions[0].SubmissionID)
	}

	details, err := user.SubmissionDetails(
		api.SubmissionDetailsRequest{
			SubmissionIDs:   searchResponse.Submissions[0].SubmissionID,
			ShowDescription: api.Yes,
		})

	if assert.NoError(t, err) {
		assert.NotEmpty(t, details.Submissions)
		assert.NotEmpty(t, details.Submissions[0].Description)
	}

	result := utils.ExtractAll(details.Submissions[0].Description, utils.Patterns)

	var request entities.TextToImageRequest

	fieldsToSet := map[string]any{
		"steps":     &request.Steps,
		"sampler":   &request.SamplerName,
		"cfg":       &request.CFGScale,
		"seed":      &request.Seed,
		"width":     &request.Width,
		"height":    &request.Height,
		"hash":      &request.OverrideSettings.SDCheckpointHash,
		"model":     &request.OverrideSettings.SDModelCheckpoint,
		"denoising": &request.DenoisingStrength,
	}

	err = utils.ResultsToFields(result, fieldsToSet)
	if !assert.NoError(t, err) {
		return
	}

	request.Prompt = utils.ExtractPositivePrompt(details.Submissions[0].Description)
	request.NegativePrompt = utils.ExtractNegativePrompt(details.Submissions[0].Description)

	system, err := llm.PrefillSystemDump(request)
	if !assert.NoError(t, err) {
		return
	}

	// Setup
	inferenceRequest := InferenceRequest{
		Config: llm.Localhost(),
		Request: llm.Request{
			Messages: []llm.Message{
				system,
				llm.UserMessage(details.Submissions[0].Description),
			},
			Temperature:   1.0,
			MaxTokens:     1024,
			Stream:        false,
			StreamChannel: nil,
		},
	}

	llmRequestJSON, err := json.Marshal(inferenceRequest)
	if !assert.NoError(t, err) {
		return
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/llm", bytes.NewReader(llmRequestJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Assertions
	if assert.NoError(t, inference(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var inferenceResponse llm.Response
		err := json.Unmarshal(rec.Body.Bytes(), &inferenceResponse)
		assert.NoError(t, err)
		assert.NotEmpty(t, inferenceResponse.Choices)
	}
}

func searchAIGenerated(err error, user *api.Credentials) api.SubmissionSearchResponse {
	searchResponse, err := user.SearchSubmissions(api.SubmissionSearchRequest{
		SubmissionIDsOnly:  true,
		SubmissionsPerPage: 1,
		Page:               1,
		Text:               "ai_generated",
		Type:               api.SubmissionTypePicturePinup,
		OrderBy:            "views",
		Random:             true,
		Scraps:             "both",
	})
	return searchResponse
}
