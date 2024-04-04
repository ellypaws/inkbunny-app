// Package api provides a set of functions to interact with the API.
// It assumes that the API is running separately and can be accessed via HTTP.
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

var ErrDeadAPI = errors.New("API is not running")
var ErrNilHost = errors.New("host is nil")

type Host url.URL

type Request struct {
	Host      *Host
	Method    string
	Data      any
	MarshalTo any
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

func (h *Host) Request(method string, jsonData []byte) *Request {
	return &Request{
		Host:      h,
		Method:    method,
		Data:      jsonData,
		MarshalTo: nil,
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

	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	request.Header.Set("Accept", "application/json")

	response, err := http.DefaultClient.Do(request)
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

func closeResponseBody(response *http.Response) {
	if response != nil {
		if err := response.Body.Close(); err != nil {
			fmt.Println("Error closing response body:", err)
		}
	}
}
