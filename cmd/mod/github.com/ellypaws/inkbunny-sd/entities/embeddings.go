package entities

import "encoding/json"

func UnmarshalEmbeddings(data []byte) (Embeddings, error) {
	var r Embeddings
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Embeddings) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Embeddings struct {
	Loaded  map[string]Embedding `json:"loaded"`
	Skipped map[string]Embedding `json:"skipped"`
}

type Embedding struct {
	Step             *int64  `json:"step"`
	SDCheckpoint     *string `json:"sd_checkpoint"`
	SDCheckpointName *string `json:"sd_checkpoint_name"`
	Shape            int64   `json:"shape"`
	Vectors          int64   `json:"vectors"`
}
