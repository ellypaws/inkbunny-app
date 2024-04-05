package library

import (
	"github.com/ellypaws/inkbunny-app/api/entities"
	"github.com/ellypaws/inkbunny-app/api/paths"
	"github.com/ellypaws/inkbunny/api"
	"net/http"
	"strings"
)

func Get(h *Host, opts ...func(*Request)) error {
	r := &Request{
		Host:   h,
		Method: http.MethodGet,
	}
	for _, opt := range opts {
		opt(r)
	}
	_, err := r.Do()
	return err
}

// GetDescription method fetches description details.
func (h *Host) GetDescription(u *api.Credentials, id string) ([]entities.DescriptionResponse, error) {
	h = h.
		WithPath(paths.InkbunnyDescription).
		WithQuery(
			map[string][]string{
				"sid":            {u.Sid},
				"username":       {u.Username},
				"submission_ids": {id},
			})
	var responses []entities.DescriptionResponse
	err := Get(h,
		WithStruct(u),
		WithClient(h.GetClient(u)),
		WithDest(&responses),
	)
	return responses, err
}

// GetSubmission method fetches submission details.
func (h *Host) GetSubmission(u *api.Credentials, req api.SubmissionDetailsRequest) (api.SubmissionDetailsResponse, error) {
	h = h.
		WithPath(paths.InkbunnySubmission)
	var response api.SubmissionDetailsResponse
	err := Get(h,
		WithStruct(req),
		WithClient(h.GetClient(u)),
		WithDest(&response),
	)
	return response, err
}

// GetSubmissionIDs method fetches submission details by IDs.
func (h *Host) GetSubmissionIDs(u *api.Credentials, ids string) (api.SubmissionDetailsResponse, error) {
	h = h.
		WithPath(strings.ReplaceAll(paths.InkbunnySubmissionIDs, ":ids", ids))
	var response api.SubmissionDetailsResponse
	err := Get(h,
		WithStruct(u),
		WithClient(h.GetClient(u)),
		WithDest(&response),
	)
	return response, err
}

// GetSearch method fetches search results based on a query.
func (h *Host) GetSearch(u *api.Credentials, req api.SubmissionSearchRequest) (api.SubmissionSearchResponse, error) {
	h = h.
		WithPath(paths.InkbunnySearch).
		WithQuery(
			map[string][]string{
				"sid":    {u.Sid},
				"text":   {req.Text},
				"output": {"json"},
			})
	var response api.SubmissionSearchResponse
	err := Get(h,
		WithStruct(req),
		WithClient(h.GetClient(u)),
		WithDest(&response),
	)
	return response, err
}

// GetImage method retrieves an image.
func (h *Host) GetImage(u *api.Credentials, url string) ([]byte, error) {
	h = h.
		WithPath(paths.Image).
		WithQuery(
			map[string][]string{
				"sid": {u.Sid},
				"url": {url},
			})
	var imageData []byte
	err := Get(h,
		WithStruct(u),
		WithClient(h.GetClient(u)),
		WithDest(&imageData),
	)
	return imageData, err
}
