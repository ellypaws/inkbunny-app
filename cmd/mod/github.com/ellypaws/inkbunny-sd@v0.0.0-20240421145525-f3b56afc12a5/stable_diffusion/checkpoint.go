package sd

import "github.com/ellypaws/inkbunny-sd/entities"

func (h *Host) GetCheckpoints() ([]entities.Checkpoint, error) {
	const checkpointPath = "/sdapi/v1/sd-models"

	body, err := h.GET(checkpointPath)
	if err != nil {
		return nil, err
	}

	v, err := entities.UnmarshalCheckpoints(body)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func GetCheckpointHash(checkpoint entities.Checkpoint) (string, error) {
	if checkpoint.Hash != "" {
		return checkpoint.Hash, nil
	}
	hash, err := CheckpointSafeTensorHash(checkpoint.Filename)
	if err != nil {
		return "", err
	}
	return hash.AutoV2, nil
}
