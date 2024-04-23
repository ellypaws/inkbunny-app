package entities

import "encoding/json"

type Hypernetworks []Hypernetwork

func UnmarshalHypernetworks(data []byte) (Hypernetworks, error) {
	var r Hypernetworks
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Hypernetworks) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Hypernetwork struct {
	Name string `json:"name"`
	Path string `json:"path"`
}
