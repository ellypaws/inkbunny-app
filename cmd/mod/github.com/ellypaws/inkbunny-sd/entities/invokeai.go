package entities

import "encoding/json"

func UnmarshalInvokeAI(data []byte) (InvokeAI, error) {
	var r InvokeAI
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *InvokeAI) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type InvokeAI struct {
	GenerationMode       string  `json:"generation_mode"`
	PositivePrompt       string  `json:"positive_prompt"`
	NegativePrompt       string  `json:"negative_prompt"`
	Width                int64   `json:"width"`
	Height               int64   `json:"height"`
	Seed                 int64   `json:"seed"`
	RandDevice           string  `json:"rand_device"`
	CFGScale             float64 `json:"cfg_scale"`
	CFGRescaleMultiplier float64 `json:"cfg_rescale_multiplier"`
	Steps                int64   `json:"steps"`
	Scheduler            string  `json:"scheduler"`
	Model                Model   `json:"model"`
	PositiveStylePrompt  string  `json:"positive_style_prompt"`
	NegativeStylePrompt  string  `json:"negative_style_prompt"`
	AppVersion           string  `json:"app_version"`
}

type Model struct {
	Key  string `json:"key"`
	Hash string `json:"hash"`
	Name string `json:"name"`
	Base string `json:"base"`
	Type string `json:"type"`
}

func (r *InvokeAI) Convert() TextToImageRequest {
	return TextToImageRequest{
		Prompt:         r.PositivePrompt,
		NegativePrompt: r.NegativePrompt,
		Width:          int(r.Width),
		Height:         int(r.Height),
		Seed:           r.Seed,
		CFGScale:       r.CFGScale,
		Steps:          int(r.Steps),
		OverrideSettings: Config{
			RandnSource:       r.RandDevice,
			SDModelCheckpoint: &r.Model.Name,
			SDCheckpointHash:  r.Model.Hash,
		},
		Scheduler: &r.Scheduler,
	}
}
