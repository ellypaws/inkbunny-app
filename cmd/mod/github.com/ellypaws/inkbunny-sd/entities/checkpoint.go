package entities

import "encoding/json"

type Checkpoints []Checkpoint

func UnmarshalCheckpoints(data []byte) (Checkpoints, error) {
	var r Checkpoints
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Checkpoints) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Checkpoint struct {
	Title     string `json:"title"`
	ModelName string `json:"model_name"`
	Hash      string `json:"hash"`   // AutoV2 short hash
	Sha256    string `json:"sha256"` // Full SHA256 hash
	Filename  string `json:"filename"`
}
