package main

import (
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/llm"
	"github.com/ellypaws/inkbunny-sd/utils"
	"github.com/ellypaws/inkbunny/api"
	"github.com/go-errors/errors"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
	"time"
)

var postHandlers = pathHandler{
	"/login":    login,
	"/logout":   logout,
	"/validate": validate,
	"/llm":      inference,
	"/llm/json": stable,
	"/prefill":  prefill,
}

// Deprecated: use registerAs((*echo.Echo).POST, postHandlers) instead
func registerPostRoutes(e *echo.Echo) {
	registerAs(e.POST, postHandlers)
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

	if err := db.Error(database); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	err = database.InsertSIDHash(db.HashCredentials(*user))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	const twoYears = 2 * 365 * 24 * 60 * 60

	c.SetCookie(&http.Cookie{
		Name:     "sid",
		Value:    user.Sid,
		Expires:  time.Now().Add(24 * time.Hour),
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   twoYears,
	})

	c.SetCookie(&http.Cookie{
		Name:     "username",
		Value:    user.Username,
		Expires:  time.Now().Add(24 * time.Hour),
		Secure:   true,
		SameSite: http.SameSiteDefaultMode,
		MaxAge:   twoYears,
	})

	return c.JSON(http.StatusOK, user)
}

func logout(c echo.Context) error {
	var user *api.Credentials
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, crashy.Wrap(err))
	}

	if sid, err := c.Cookie("sid"); err == nil {
		if user == nil {
			user = &api.Credentials{Sid: sid.Value}
		}
		user.Sid = sid.Value
	}

	if username, err := c.Cookie("username"); err == nil {
		if user == nil {
			user = &api.Credentials{Username: username.Value}
		}
		user.Username = username.Value
	}

	if user.Sid == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{Error: "SID is required"})
	}

	err := user.Logout()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if err := db.Error(database); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	err = database.RemoveSIDHash(db.HashCredentials(*user))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	return c.String(http.StatusOK, "logged out")
}

func validate(c echo.Context) error {
	var user api.Credentials
	if err := c.Bind(user); err != nil {
		return err
	}

	if cookie, err := c.Cookie("sid"); err == nil {
		user.Sid = cookie.Value
	}

	if user.Sid == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{Error: "SID is required"})
	}

	if err := db.Error(database); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if !database.ValidSID(user) {
		return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{Error: "invalid SID"})
	}

	return c.String(http.StatusOK, strconv.Itoa(http.StatusOK))
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

	if cookie, err := c.Cookie("sid"); err == nil {
		if user == nil {
			user = &api.Credentials{Sid: cookie.Value}
		}
		user.Sid = cookie.Value
	}

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

	request, err := utils.DescriptionHeuristics(details.Submissions[0].Description)
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

func prefill(c echo.Context) error {
	var prefillRequest PrefillRequest
	if err := c.Bind(&prefillRequest); err != nil {
		return err
	}

	if prefillRequest.Description == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{Error: "description is required"})
	}

	request, err := utils.DescriptionHeuristics(prefillRequest.Description)
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

	if prefillRequest.Description != "" {
		prefillRequest.Description = "Return the JSON without the // comments"
	}
	messages = append(messages, llm.UserMessage(prefillRequest.Description))

	return c.JSON(http.StatusOK, llm.Request{
		Messages:      messages,
		Temperature:   1.0,
		MaxTokens:     1024,
		Stream:        false,
		StreamChannel: nil,
	})
}
