package main

import (
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/llm"
	"github.com/ellypaws/inkbunny-sd/utils"
	"github.com/ellypaws/inkbunny/api"
	"github.com/go-errors/errors"
	"github.com/labstack/echo/v4"
	"net/http"
)

var postRoutes = map[string]func(c echo.Context) error{
	"/login":    login,
	"/llm":      inference,
	"/llm/json": stable,
	"/prefill":  prefill,
}

func registerPostRoutes(e *echo.Echo) {
	for path, handler := range postRoutes {
		e.POST(path, handler)
	}
}

func login(c echo.Context) error {
	var loginRequest LoginRequest
	if err := c.Bind(&loginRequest); err != nil {
		return err
	}
	user := &api.Credentials{
		Username: loginRequest.Username,
		Password: loginRequest.Password,
	}
	user, err := user.Login()
	if err != nil {
		return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
	}
	return c.JSON(http.StatusOK, user)
}

func hostOnline(c llm.Config) error {
	endpointURL := c.Endpoint
	resp, err := http.Get(endpointURL.String())
	if err != nil {
		return errors.New(err)
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("endpoint is not available")
	}
	return nil
}

func inference(c echo.Context) error {
	var llmRequest InferenceRequest
	if err := c.Bind(&llmRequest); err != nil {
		return err
	}
	config := llmRequest.Config

	if localhost := c.QueryParam("localhost"); localhost == "true" {
		config = llm.Localhost()
	}

	if config.Endpoint.String() == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{Error: "config is required"})
	}

	err := hostOnline(config)
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, crashy.Wrap(err))
	}

	request := llmRequest.Request
	response, err := config.Infer(&request)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if output := c.QueryParams().Get("output"); output == "json" {
		message := utils.ExtractJson([]byte(response.Choices[0].Message.Content))
		textToImage, err := entities.UnmarshalTextToImageRequest(message)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
		if textToImage.Prompt == "" {
			return c.JSON(http.StatusNotFound, crashy.ErrorResponse{Error: "prompt is empty"})
		}
		if desc, ok := textToImage.Comments["description"]; ok && desc == "<|description|>" {
			textToImage.Comments["description"] = request.Messages[1].Content
		}
		return c.JSON(http.StatusOK, textToImage)
	}

	return c.JSON(http.StatusOK, response)
}

func stable(c echo.Context) error {
	var subRequest InferenceSubmissionRequest
	if err := c.Bind(&subRequest); err != nil {
		return err
	}
	config := subRequest.Config

	if localhost := c.QueryParam("localhost"); localhost == "true" {
		config = llm.Localhost()
	}

	if config.Endpoint.String() == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{Error: "config is required"})
	}

	err := hostOnline(config)
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, crashy.Wrap(err))
	}

	user := &subRequest.User
	if user.Sid == "" {
		user, err = user.Login()
		if err != nil {
			return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
		}
	}

	details, err := user.SubmissionDetails(
		api.SubmissionDetailsRequest{
			SubmissionIDs:   subRequest.SubmissionID,
			ShowDescription: api.Yes,
		})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}
	if len(details.Submissions) == 0 {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{Error: "no submissions found"})
	}
	if details.Submissions[0].Description == "" {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{Error: "no description found"})
	}

	request, err := descriptionHeuristics(details.Submissions[0].Description)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	system, err := llm.PrefillSystemDump(request)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	inferenceRequest := llm.Request{
		Messages: []llm.Message{
			system,
			llm.UserMessage(details.Submissions[0].Description),
		},
		Temperature:   1.0,
		MaxTokens:     1024,
		Stream:        false,
		StreamChannel: nil,
	}
	response, err := config.Infer(&inferenceRequest)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	message := utils.ExtractJson([]byte(response.Choices[0].Message.Content))
	textToImage, err := entities.UnmarshalTextToImageRequest(message)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}
	if textToImage.Prompt == "" {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{Error: "prompt is empty"})
	}

	return c.JSON(http.StatusOK, textToImage)
}

func descriptionHeuristics(description string) (entities.TextToImageRequest, error) {
	results := utils.ExtractAll(description, utils.Patterns)

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

	err := utils.ResultsToFields(results, fieldsToSet)
	if err != nil {
		return request, err
	}

	request.Prompt = utils.ExtractPositivePrompt(description)
	request.NegativePrompt = utils.ExtractNegativePrompt(description)
	return request, nil
}

func prefill(c echo.Context) error {
	var prefillRequest PrefillRequest
	if err := c.Bind(&prefillRequest); err != nil {
		return err
	}

	if prefillRequest.Description == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{Error: "description is required"})
	}

	request, err := descriptionHeuristics(prefillRequest.Description)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if output := c.QueryParams().Get("output"); output == "json" {
		return c.JSON(http.StatusOK, request)
	}

	system, err := llm.PrefillSystemDump(request)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if output := c.QueryParams().Get("output"); output != "complete" {
		return c.JSON(http.StatusOK, system)
	}

	var messages []llm.Message
	if system.Content != "" {
		messages = append(messages, system)
	} else {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{Error: "system message is empty"})
	}

	messages = append(messages, llm.UserMessage(request.Prompt))
	return c.JSON(http.StatusOK, llm.Request{
		Messages:      messages,
		Temperature:   1.0,
		MaxTokens:     1024,
		Stream:        false,
		StreamChannel: nil,
	})
}
