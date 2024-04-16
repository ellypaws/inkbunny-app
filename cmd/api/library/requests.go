// Package library provides a set of functions to interact with the API.
// It assumes that the API is running separately and can be accessed via HTTP.
package library

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"image"
	"image/jpeg"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
)

var ErrDeadAPI = errors.New("API is not running")
var ErrNilHost = errors.New("host is nil")

type Host url.URL

type Request struct {
	Host      *Host
	Client    *http.Client
	Method    string
	Data      any
	MarshalTo any
	opts      []func(http.Header)
}

var DefaultHost = (*Host)(&url.URL{
	Scheme: "http",
	Host:   "localhost:1323",
})

func (h *Host) String() string {
	return (*url.URL)(h).String()
}

func (h *Host) Base() string {
	return fmt.Sprintf("%s://%s", h.Scheme, h.Host)
}

func FromString(s string) *Host {
	u, err := url.Parse(s)
	if err != nil {
		return nil
	}
	return (*Host)(u)
}

func (h *Host) WithPath(path string) *Host {
	if h == nil {
		return nil
	}
	p := *h
	p.Path = path
	return &p
}

func (h *Host) WithQuery(q url.Values) *Host {
	if h == nil {
		return nil
	}
	p := *h
	p.RawQuery = q.Encode()
	return &p
}

func (h *Host) Alive() bool {
	req, err := http.NewRequest(http.MethodHead, h.Base(), nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func (h *Host) GET(getURL string) ([]byte, error) {
	return h.WithPath(getURL).Request(http.MethodGet, nil).Do()
}

func (h *Host) POST(postURL string, jsonData []byte) ([]byte, error) {
	return h.WithPath(postURL).Request(http.MethodPost, jsonData).Do()
}

func (h *Host) WithMethod(method, path string, jsonData []byte) ([]byte, error) {
	return h.WithPath(path).Request(method, jsonData).Do()
}

func (h *Host) Request(method string, from []byte) *Request {
	return &Request{
		Host:      h,
		Method:    method,
		Data:      from,
		MarshalTo: nil,
		Client:    nil,
	}
}

func (r *Request) Do() ([]byte, error) {
	if r.Host == nil {
		return nil, ErrNilHost
	}

	if !r.Host.Alive() {
		return nil, ErrDeadAPI
	}

	var buffer io.Reader
	switch d := r.Data.(type) {
	case []byte:
		buffer = bytes.NewBuffer(d)
	case io.Reader:
		buffer = d
	default:
		b, err := json.Marshal(r.Data)
		if err != nil {
			return nil, err
		}
		buffer = bytes.NewBuffer(b)
	}

	request, err := http.NewRequest(r.Method, r.Host.String(), buffer)
	if err != nil {
		return nil, err
	}

	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	request.Header.Set(echo.HeaderAccept, echo.MIMEApplicationJSON)

	for _, opt := range r.opts {
		opt(request.Header)
	}

	if r.Client == nil {
		r.Client = http.DefaultClient
	}

	response, err := r.Client.Do(request)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(response)

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		errorString := "(unknown error)"
		if len(body) > 0 {
			errorString = fmt.Sprintf("\n```json\n%v\n```", string(body))
		}
		return nil, fmt.Errorf("unexpected status code: `%v` %v", response.Status, errorString)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if r.MarshalTo != nil {
		if err := json.Unmarshal(body, r.MarshalTo); err != nil {
			return nil, err
		}
	}
	return body, nil
}

func (r *Request) WithDest(dest any) *Request {
	r.MarshalTo = dest
	return r
}

func WithDest(dest any) func(*Request) {
	return func(r *Request) {
		r.MarshalTo = dest
	}
}

func WithBytes(b []byte) func(*Request) {
	return func(r *Request) {
		r.Data = b
	}
}

func WithMethod(method string) func(*Request) {
	return func(r *Request) {
		r.Method = method
	}
}

func WithClient(c *http.Client) func(*Request) {
	return func(r *Request) {
		r.Client = c
	}
}

func WithAuthorizationBearer(token string) func(*Request) {
	return func(r *Request) {
		r.opts = append(r.opts, func(h http.Header) {
			h.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		})
	}
}

func WithStruct(s any) func(*Request) {
	if s == nil {
		return func(r *Request) {}
	}
	switch s.(type) {
	case []byte, io.Reader:
		return func(r *Request) {
			r.Data = s
		}
	default:
		b, err := json.Marshal(s)
		if err != nil {
			log.Fatalf("error marshalling struct: %v", err)
		}
		return func(r *Request) {
			r.Data = b
		}
	}
}

func WithImage(img *image.Image) func(*Request) {
	return func(r *Request) {
		buf := new(bytes.Buffer)
		err := jpeg.Encode(buf, *img, nil)
		if err != nil {
			log.Fatalf("error encoding image: %v", err)
		}
		imgBytes := buf.Bytes()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("image", "image.jpg")
		if err != nil {
			log.Fatalf("error creating form file: %v", err)
		}
		part.Write(imgBytes)

		err = writer.Close()
		if err != nil {
			log.Fatalf("error closing writer: %v", err)
		}

		r.Data = body
		r.opts = append(r.opts, func(h http.Header) {
			h.Set(echo.HeaderContentType, writer.FormDataContentType())
		})
	}
}

// WithImageAndFields modifies the request to include a multipart form with an image and additional fields.
func WithImageAndFields(img *image.Image, fields map[string]string) func(*Request) {
	return func(r *Request) {
		buf := new(bytes.Buffer)
		// Encode the image into JPEG format.
		err := jpeg.Encode(buf, *img, nil)
		if err != nil {
			log.Fatalf("error encoding image: %v", err)
		}
		imgBytes := buf.Bytes()

		// Create a new multipart writer.
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		// Create a form file part for the image.
		part, err := writer.CreateFormFile("image", "image.jpg")
		if err != nil {
			log.Fatalf("error creating form file: %v", err)
		}
		_, err = part.Write(imgBytes)
		if err != nil {
			log.Fatalf("error writing image bytes to form file: %v", err)
		}

		// Iterate over the fields map and add each as a part of the form.
		for key, val := range fields {
			err := writer.WriteField(key, val)
			if err != nil {
				log.Fatalf("error adding field %s to form: %v", key, err)
			}
		}

		// Close the multipart writer to finalize the form body.
		err = writer.Close()
		if err != nil {
			log.Fatalf("error closing writer: %v", err)
		}

		// Set the request body and content type.
		r.Data = body
		r.opts = append(r.opts, func(h http.Header) {
			h.Set(echo.HeaderContentType, writer.FormDataContentType())
		})
	}
}

func closeResponseBody(response *http.Response) {
	if response != nil {
		if err := response.Body.Close(); err != nil {
			fmt.Println("Error closing response body:", err)
		}
	}
}
