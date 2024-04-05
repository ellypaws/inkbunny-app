package library

import (
	"github.com/disintegration/imaging"
	"github.com/ellypaws/inkbunny-app/api/entities"
	"github.com/ellypaws/inkbunny-app/api/paths"
	sd "github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/llm"
	"github.com/ellypaws/inkbunny/api"
	"image"
	"net/http"
	"strconv"
)

func Post(h *Host, opts ...func(*Request)) error {
	r := &Request{
		Host:   h,
		Method: http.MethodPost,
	}
	for _, opt := range opts {
		opt(r)
	}
	_, err := r.Do()
	return err
}

func (h *Host) PostLogin(u *api.Credentials) (*api.Credentials, error) {
	h = h.WithPath(paths.Login)
	err := Post(h,
		WithStruct(u),
		WithDest(u),
		WithClient(h.GetClient(u)),
	)
	return u, err
}

func (h *Host) PostLogout(u *api.Credentials) error {
	h = h.WithPath(paths.Logout)
	err := Post(h,
		WithStruct(u),
		WithClient(h.GetClient(u)),
	)
	return err
}

func (h *Host) PostValidate(u *api.Credentials) error {
	h = h.WithPath(paths.Validate)
	err := Post(h,
		WithStruct(u),
		WithClient(h.GetClient(u)),
	)
	return err
}

func (h *Host) PostLLM(u *api.Credentials, req entities.InferenceRequest, localhost bool) (llm.Response, error) {
	h = h.WithPath(paths.LLM)
	if localhost {
		h.WithQuery(map[string][]string{
			"localhost": {"true"},
		})
	}
	var response llm.Response
	err := Post(h,
		WithStruct(req),
		WithClient(h.GetClient(u)),
		WithDest(&response),
	)
	return response, err
}

func (h *Host) PostLLMToGeneration(u *api.Credentials, req entities.InferenceRequest, localhost bool) (sd.TextToImageRequest, error) {
	h = h.WithPath(paths.LLM)
	if localhost {
		h.WithQuery(map[string][]string{
			"localhost": {"true"},
			"output":    {"json"},
		})
	}
	var request sd.TextToImageRequest
	err := Post(h,
		WithStruct(req),
		WithClient(h.GetClient(u)),
		WithDest(&request),
	)
	return request, err
}

func (h *Host) PostPrefill(u *api.Credentials, req entities.PrefillRequest) (llm.Request, error) {
	h = h.WithPath(paths.Prefill)
	var response llm.Request
	err := Post(h,
		WithStruct(req),
		WithClient(h.GetClient(u)),
		WithDest(&response),
	)
	return response, err
}

func (h *Host) PostInterrogate(u *api.Credentials, req sd.TaggerRequest) (sd.TaggerResponse, error) {
	h = h.WithPath(paths.Interrogate).
		WithQuery(
			map[string][]string{
				"sorted": {"true"},
			})
	var response sd.TaggerResponse
	err := Post(h,
		WithStruct(req),
		WithClient(h.GetClient(u)),
		WithDest(&response),
	)
	return response, err
}

// PostInterrogateUpload sends as a POST multipart form file
func (h *Host) PostInterrogateUpload(u *api.Credentials, img *image.Image, req sd.TaggerRequest) (sd.TaggerResponse, error) {
	var response sd.TaggerResponse
	if img == nil {
		return response, nil
	}
	h = h.WithPath(paths.Interrogate)

	msize := [2]int{512, 512}
	if cmp(Dimensions(img), msize) > 0 {
		*img = ResizeImage(img, msize)
	}

	err := Post(h,
		WithImageAndFields(img, ReqToFields(req)),
		WithClient(h.GetClient(u)),
		WithDest(&response),
	)
	return response, err
}

func ReqToFields(req sd.TaggerRequest) map[string]string {
	fields := map[string]string{
		"model":  req.Model,
		"sorted": "true",
	}
	if req.Threshold != nil {
		fields["threshold"] = strconv.FormatFloat(*req.Threshold, 'f', -1, 64)
	}
	return fields
}

func cmp(a, b [2]int) int {
	if a[0] == b[0] && a[1] == b[1] {
		return 0
	}
	if a[0] > b[0] || a[1] > b[1] {
		return 1
	}
	return -1
}

func Dimensions(src *image.Image) [2]int {
	bounds := (*src).Bounds()
	return [2]int{bounds.Dx(), bounds.Dy()}
}

func ResizeImage(src *image.Image, max [2]int) image.Image {
	return imaging.Fit(*src, max[0], max[1], imaging.Lanczos)
}
