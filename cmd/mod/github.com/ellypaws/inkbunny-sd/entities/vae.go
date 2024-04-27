package entities

import "encoding/json"

type VAEs []VAE

func UnmarshalVAEs(data []byte) (VAEs, error) {
	var r VAEs
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *VAEs) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type VAE struct {
	ModelName string `json:"model_name"`
	Filename  string `json:"filename"`
}
