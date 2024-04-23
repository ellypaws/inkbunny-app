package sd

import "github.com/ellypaws/inkbunny-sd/entities"

func (h *Host) GetVAEs() ([]entities.VAE, error) {
	const vaePath = "/sdapi/v1/sd-vae"

	body, err := h.GET(vaePath)
	if err != nil {
		return nil, err
	}

	v, err := entities.UnmarshalVAEs(body)
	if err != nil {
		return nil, err
	}

	return v, nil
}
