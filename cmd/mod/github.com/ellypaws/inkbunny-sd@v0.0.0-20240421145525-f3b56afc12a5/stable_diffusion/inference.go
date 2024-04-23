package sd

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/ellypaws/inkbunny-sd/entities"
)

var ErrMissingRequest = errors.New("missing request")

func (h *Host) TextToImageRequest(req *entities.TextToImageRequest) (*entities.TextToImageResponse, error) {
	jsonData, err := req.Marshal()
	if err != nil {
		return nil, err
	}

	return h.TextToImageRaw(jsonData)
}

func (h *Host) TextToImageRaw(req []byte) (*entities.TextToImageResponse, error) {
	const text2imgPath = "/sdapi/v1/txt2img"

	response, err := h.POST(text2imgPath, req)
	if err != nil {
		return nil, fmt.Errorf("error with POST request: %w", err)
	}

	return entities.JSONToTextToImageResponse(response)
}

func ToImages(response *entities.TextToImageResponse) ([][]byte, error) {
	if response == nil {
		return nil, ErrMissingRequest
	}
	var images [][]byte
	for _, img := range response.Images {
		data, err := base64.StdEncoding.DecodeString(img)
		if err != nil {
			return nil, err
		}
		images = append(images, data)
	}
	return images, nil
}
