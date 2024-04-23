package sd

import (
	"errors"
	"github.com/ellypaws/inkbunny-sd/entities"
)

func (h *Host) GetLoras() ([]entities.Lora, error) {
	const loraPath = "/sdapi/v1/loras"
	body, err := h.GET(loraPath)
	if err != nil {
		return nil, err
	}

	var loras []entities.Lora
	loras, err = entities.UnmarshalLoras(body)
	if err != nil {
		return nil, err
	}

	return loras, nil
}

// GetLoraHash returns the hash of an entities.Lora
// If the metadata contains a hash, it returns the first 12 bytes.
// Otherwise, it calculates [LoraSafetensorHash] and returns the LoraHash.AutoV3
func GetLoraHash(lora entities.Lora) (string, error) {
	if lora.Metadata.SshsModelHash != nil {
		return (*lora.Metadata.SshsModelHash)[:12], nil
	}
	hash, err := LoraSafetensorHash(lora.Path)
	if err != nil {
		return "", err
	}
	return hash.AutoV3, nil
}

var ErrMissingHash = errors.New("missing hash")

var ErrWrongHashLength = errors.New("wrong hash length")

// DownloadModel downloads the model from a hash.
// It first checks if the model already exists in the host system.
// It requires the [Stable-Diffusion-Webui-Civitai-Helper] extension.
// If the hash is 10 bytes, it is a checkpoint using CheckpointHash.AutoV2.
// If the hash is 12 bytes, it is a lora using LoraHash.AutoV3.
//
//	Deprecated: TODO: To be implemented. Endpoint seems to be missing.
//
// [Stable-Diffusion-Webui-Civitai-Helper]: https://github.com/zixaphir/Stable-Diffusion-Webui-Civitai-Helper
func DownloadModel(hash string) error {
	switch len(hash) {
	case 0:
		return ErrMissingHash
	case 10:
		// model is a checkpoint using CheckpointHash.AutoV2
	case 12:
		// model is a lora
	case 64:
		hash = hash[:12]
	default:
		return ErrWrongHashLength
	}
	return nil
}
