package sd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

var ErrDeadAPI = errors.New("API is not running")
var ErrNilHost = errors.New("host is nil")

type Host url.URL

var DefaultHost = (*Host)(&url.URL{
	Scheme: "http",
	Host:   "localhost:7860",
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
	return h.WithPath(getURL).Request(http.MethodGet, nil)
}

func (h *Host) POST(postURL string, jsonData []byte) ([]byte, error) {
	return h.WithPath(postURL).Request(http.MethodPost, jsonData)
}

func (h *Host) Request(method string, jsonData []byte) ([]byte, error) {
	if h == nil {
		return nil, ErrNilHost
	}

	if !h.Alive() {
		return nil, ErrDeadAPI
	}

	request, err := http.NewRequest(method, h.String(), bytes.NewBuffer(jsonData))
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
	return body, nil
}

func closeResponseBody(response *http.Response) {
	if response != nil {
		if err := response.Body.Close(); err != nil {
			fmt.Println("Error closing response body:", err)
		}
	}
}
