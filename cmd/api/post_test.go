package main

import (
	"bytes"
	"context"
	"encoding/json"
	. "github.com/ellypaws/inkbunny-app/api/entities"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	sd "github.com/ellypaws/inkbunny-sd/entities"
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

func tempDB() *db.Sqlite {
	database, err := db.New(context.WithValue(context.Background(), ":memory:", true))
	if err != nil {
		return nil
	}
	return database
}

func TestLogin(t *testing.T) {
	// Setup
	userJSON := `{"username":"guest","password":""}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(userJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	database = tempDB()
	if !assert.NoError(t, db.Error(database)) {
		return
	}

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

func TestPrefill(t *testing.T) {
	// Setup
	prefillRequest := PrefillRequest{
		Description: `
Generated with Stable Diffusion, seeded with a photo and Photoshopped a bit.

male, fennec, solo, (laying in a bed in the shape of a heart, heart-shaped bed, bed shaped like a heart), five fingers,
four toes, dark toes, dark pawpads, blonde hair, short hair, blue eyes, masterpiece, 8k quality, (detailed fur:1.2),
(cinematic lighting)+, backlighting, (shaded)+, photorealistic, hyperrealistic, (view from above),

Negative Prompt: AS-YoungV2-neg, bad-hands-5, boring_e621_v4, bwu, deformityv6, dfc, ubbp, updn, deformed feet,
deformed hands, fur markings, stripes, long torso, (out of proportion), (disproportional)

Steps: 42, Sampler: DPM++ 2M Karras, CFG scale: 7, Seed: 2938221969, Size: 1220x690, Model hash: 593395568a,
Model: indigoFurryMix_v90Hybrid, Denoising strength: 0.5,
TI hashes: "bad-hands-5: aa7651be154c, boring_e621_v4: f9b806505bc2, bwu: 70e376c5cf1d, deformityv6: 8455ec9b3d31, dfc: 21c6ae158a7e, ubbp: 047acf26d29c, updn: b4ae8ca1b247",
Version: v1.6.0`,
	}
	prefillRequestJSON, err := json.Marshal(prefillRequest)
	assert.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/prefill", bytes.NewReader(prefillRequestJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Assertions
	if assert.NoError(t, prefill(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var systemMessage llm.Message
		err := json.Unmarshal(rec.Body.Bytes(), &systemMessage)
		assert.NoError(t, err)
		assert.NotEmpty(t, systemMessage.Content)
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

	searchResponse := searchAIGenerated(user)
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

	var request sd.TextToImageRequest

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

func searchAIGenerated(user *api.Credentials) api.SubmissionSearchResponse {
	searchResponse, _ := user.SearchSubmissions(api.SubmissionSearchRequest{
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
