package api

import (
	"bytes"
	"encoding/base64"
	"github.com/disintegration/imaging"
	. "github.com/ellypaws/inkbunny-app/cmd/api/entities"
	"github.com/ellypaws/inkbunny-app/cmd/api/service"
	"github.com/ellypaws/inkbunny-app/cmd/crashy"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	sd "github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/llm"
	"github.com/ellypaws/inkbunny-sd/utils"
	"github.com/ellypaws/inkbunny/api"
	"github.com/go-errors/errors"
	"github.com/labstack/echo/v4"
	"image"
	"io"
	"net/http"
	"strconv"
	"time"
)

var postHandlers = pathHandler{
	"/login":              handler{login, nil},
	"/guest":              handler{guest, nil},
	"/logout":             handler{logout, loggedInMiddleware},
	"/validate":           handler{validate, loggedInMiddleware},
	"/llm":                handler{inference, nil},
	"/llm/json":           handler{stable, nil},
	"/prefill":            handler{prefill, nil},
	"/interrogate":        handler{interrogate, nil},
	"/interrogate/upload": handler{interrogateImage, nil},
	"/review/:id":         handler{GetReviewHandler, append(reducedMiddleware, WithRedis...)},
	"/heuristics":         handler{heuristics, nil},
	"/heuristics/:id":     handler{GetHeuristicsHandler, append(reducedMiddleware, WithRedis...)},
	"/sd/:path":           handler{HandlePath, nil},
	"/artists":            handler{upsertArtist, staffMiddleware},
	"/inkbunny/search":    handler{GetInkbunnySearch, append(loggedInMiddleware, WithRedis...)},
	"/generate":           handler{generate, append(staffMiddleware, WithRedis...)},
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

	if err := db.Error(Database); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	err = Database.InsertSIDHash(db.HashCredentials(*user))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	const twoYears = 2 * 365 * 24 * 60 * 60

	c.SetCookie(&http.Cookie{
		Name:     "sid",
		Value:    user.Sid,
		Expires:  time.Now().UTC().Add(24 * time.Hour),
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   twoYears,
	})

	c.SetCookie(&http.Cookie{
		Name:     "username",
		Value:    user.Username,
		Expires:  time.Now().UTC().Add(24 * time.Hour),
		Secure:   true,
		SameSite: http.SameSiteDefaultMode,
		MaxAge:   twoYears,
	})

	return c.JSON(http.StatusOK, user)
}

func guest(c echo.Context) error {
	sid, ok := c.Get("sid").(string)
	if !ok || sid == "" {
		xsid := c.Request().Header.Get("X-SID")
		if xsid != "" {
			sid = xsid
		}
	}
	if sid == "" {
		sidCookie, err := c.Cookie("sid")
		if err == nil && sidCookie.Value != "" {
			sid = sidCookie.Value
		}
	}

	if Database.ValidSID(api.Credentials{Sid: sid}) {
		return c.JSON(http.StatusOK, crashy.ErrorResponse{ErrorString: "already logged in"})
	}

	user := &api.Credentials{
		Username: "guest",
	}
	user, err := user.Login()
	if err != nil {
		return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
	}

	if err := db.Error(Database); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	err = Database.InsertSIDHash(db.HashCredentials(*user))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	const twoYears = 2 * 365 * 24 * 60 * 60

	c.SetCookie(&http.Cookie{
		Name:     "sid",
		Value:    user.Sid,
		Expires:  time.Now().UTC().Add(24 * time.Hour),
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   twoYears,
	})

	c.SetCookie(&http.Cookie{
		Name:     "username",
		Value:    user.Username,
		Expires:  time.Now().UTC().Add(24 * time.Hour),
		Secure:   true,
		SameSite: http.SameSiteDefaultMode,
		MaxAge:   twoYears,
	})

	err = user.ChangeRating(api.Ratings{
		General:        true,
		Nudity:         true,
		MildViolence:   true,
		Sexual:         true,
		StrongViolence: true,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	return c.JSON(http.StatusOK, user)
}

func logout(c echo.Context) error {
	sid, id, err := GetSIDandID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
	}

	user := &api.Credentials{Sid: sid, UserID: api.IntString(id)}

	err = user.Logout()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	if err := db.Error(Database); err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	sidHash := db.HashCredentials(api.Credentials{Sid: sid, UserID: api.IntString(id)})
	err = Database.RemoveSIDHash(sidHash)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: err.Error(), Debug: db.HashCredentials(*user)})
	}

	c.SetCookie(&http.Cookie{
		Name:   "sid",
		Value:  "",
		MaxAge: -1,
	})

	c.SetCookie(&http.Cookie{
		Name:   "username",
		Value:  "",
		MaxAge: -1,
	})

	return c.String(http.StatusOK, "logged out")
}

func validate(c echo.Context) error {
	sid, _, err := GetSIDandID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, crashy.Wrap(err))
	}

	if !Database.ValidSID(api.Credentials{Sid: sid}) {
		return c.JSON(http.StatusUnauthorized, crashy.ErrorResponse{ErrorString: "invalid SID"})
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
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "config is required"})
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
		textToImage, err := sd.UnmarshalTextToImageRequest(message)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
		if textToImage.Prompt == "" {
			return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "prompt is empty"})
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
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "config is required"})
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

	details, err := service.RetrieveSubmission(c, api.SubmissionDetailsRequest{
		SID:             user.Sid,
		SubmissionIDs:   subRequest.SubmissionID,
		ShowDescription: api.Yes,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}
	if len(details.Submissions) == 0 {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no submissions found"})
	}
	if details.Submissions[0].Description == "" {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "no description found"})
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
	textToImage, err := sd.UnmarshalTextToImageRequest(message)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}
	if textToImage.Prompt == "" {
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "prompt is empty"})
	}

	return c.JSON(http.StatusOK, textToImage)
}

func prefill(c echo.Context) error {
	var prefillRequest PrefillRequest
	if err := c.Bind(&prefillRequest); err != nil {
		return err
	}

	if prefillRequest.Description == "" {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "description is required"})
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
		return c.JSON(http.StatusNotFound, crashy.ErrorResponse{ErrorString: "system message is empty"})
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

var defaultThreshold = 0.3

var defaultTaggerRequest = sd.TaggerRequest{
	Model:     sd.TaggerZ3DE621Convnext,
	Threshold: &defaultThreshold,
}

func interrogate(c echo.Context) error {
	var request = defaultTaggerRequest
	if err := c.Bind(&request); err != nil {
		return err
	}

	if request.Image == nil {
		return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "image is required"})
	}

	if sorted := c.QueryParam("sorted"); sorted == "false" {
		response, err := SDHost.Interrogate(&request)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}

		return c.JSON(http.StatusOK, response)
	}

	response, err := SDHost.InterrogateRaw(&request)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	return c.JSONBlob(http.StatusOK, response)
}

func interrogateImage(c echo.Context) error {
	file, err := c.FormFile("image")
	if err != nil {
		return c.JSON(http.StatusBadRequest, crashy.Wrap(err))
	}

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	msize := [2]int{512, 512}

	var b64 string
	if compare(dimensions(img), msize) > 0 {
		b64 = resizeImage(img, msize)
		if b64 == "" {
			return c.JSON(http.StatusInternalServerError, crashy.ErrorResponse{ErrorString: "error resizing image"})
		}
	} else {
		b64 = base64.StdEncoding.EncodeToString(data)
	}

	request := defaultTaggerRequest
	request.Image = &b64

	if threshold := c.FormValue("threshold"); threshold != "" {
		f, err := strconv.ParseFloat(threshold, 64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, crashy.Wrap(err))
		}
		request.SetThreshold(f)
	}

	if sorted := c.FormValue("sorted"); sorted == "false" {
		response, err := SDHost.Interrogate(&request)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}

		return c.JSON(http.StatusOK, response)
	}

	response, err := SDHost.InterrogateRaw(&request)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
	}

	return c.JSONBlob(http.StatusOK, response)
}

func compare(a, b [2]int) int {
	if a[0] == b[0] && a[1] == b[1] {
		return 0
	}
	if a[0] > b[0] || a[1] > b[1] {
		return 1
	}
	return -1
}

func dimensions(src image.Image) [2]int {
	bounds := src.Bounds()
	return [2]int{bounds.Dx(), bounds.Dy()}
}

func resizeImage(src image.Image, max [2]int) string {
	i := imaging.Fit(src, max[0], max[1], imaging.Lanczos)
	writer := new(bytes.Buffer)
	err := imaging.Encode(writer, i, imaging.PNG)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(writer.Bytes())
}

func heuristics(c echo.Context) error {
	var request struct {
		Parameters  string `json:"parameters"`
		Description string `json:"description"`
	}
	if err := c.Bind(&request); err != nil {
		return err
	}

	if request.Parameters != "" {
		object, err := utils.ParameterHeuristics(request.Parameters)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
		return c.JSON(http.StatusOK, object)
	}

	if request.Description != "" {
		object, err := utils.DescriptionHeuristics(request.Description)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, crashy.Wrap(err))
		}
		return c.JSON(http.StatusOK, object)
	}

	return c.JSON(http.StatusBadRequest, crashy.ErrorResponse{ErrorString: "parameters or description is required", Debug: request})
}

// Set query "image" to "true" to return just the (first) image
// Set query "regenerate" to "true" to regenerate the image
// Set query "simple" to "true" to return slices
func generate(c echo.Context) error {
	var object map[string]sd.TextToImageRequest
	if err := c.Bind(&object); err != nil {
		return err
	}

	if !SDHost.Alive() {
		return c.JSON(http.StatusServiceUnavailable, crashy.ErrorResponse{ErrorString: "host is not available"})
	}

	responses, err := service.Inference(c, object, SDHost, Database)
	if err != nil {
		return err
	}

	if c.QueryParam("simple") == "true" {
		type out struct {
			Key    *string `json:"key"`
			Prompt *string `json:"prompt"`
			Image  *string `json:"image"`
		}

		var slices []out
		for key, response := range responses {
			for i := range response.Images {
				slices = append(slices, out{
					Key:    &key,
					Image:  &response.Images[i],
					Prompt: &response.Info.Prompt,
				})
			}
		}
		return c.JSON(http.StatusOK, slices)
	}
	return c.JSON(http.StatusOK, responses)
}
