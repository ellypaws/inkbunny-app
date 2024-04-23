package sd

import (
	"fmt"
	"github.com/ellypaws/inkbunny-sd/entities"
)

func (h *Host) GetInterrogators() ([]string, error) {
	const interrogatorsPath = "/tagger/v1/interrogators"

	body, err := h.GET(interrogatorsPath)
	if err != nil {
		return nil, err
	}

	var interrogators entities.Interrogators
	interrogators, err = entities.UnmarshalInterrogators(body)
	if err != nil {
		return nil, err
	}

	return interrogators.Models, nil
}

func (h *Host) InterrogateRaw(req *entities.TaggerRequest) ([]byte, error) {
	const interrogatePath = "/tagger/v1/interrogate"

	jsonData, err := req.Marshal()
	if err != nil {
		return nil, err
	}

	return h.POST(interrogatePath, jsonData)
}

// Interrogate sends a POST request to the tagger API with the given request.
// It requires the [stable-diffusion-webui-wd14-tagger] extension with additional fixes from [dm18]
// The default model used is [Z3D-E621-Convnext]
//
// [stable-diffusion-webui-wd14-tagger]: https://github.com/picobyte/stable-diffusion-webui-wd14-tagger
// [dm18]: https://github.com/dm18/stable-diffusion-webui-wd14-tagger
// [Z3D-E621-Convnext]: https://huggingface.co/toynya/Z3D-E621-Convnext
func (h *Host) Interrogate(req *entities.TaggerRequest) (entities.TaggerResponse, error) {
	if req == nil {
		return entities.TaggerResponse{}, ErrMissingRequest
	}
	response, err := h.InterrogateRaw(req)
	if err != nil {
		return entities.TaggerResponse{}, fmt.Errorf("error with POST request: %w", err)
	}

	return entities.UnmarshalTaggerResponse(response)
}

func (h *Host) InterrogateWith(url string, model string) (entities.TaggerResponse, error) {
	if model == "" {
		model = entities.TaggerZ3DE621Convnext
	}
	req := entities.TaggerRequest{
		Image: &url,
		Model: model,
	}
	return h.Interrogate(&req)
}
