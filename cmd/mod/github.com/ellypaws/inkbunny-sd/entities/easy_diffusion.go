package entities

import "encoding/json"

func UnmarshalEasyDiffusion(data []byte) (EasyDiffusion, error) {
	var r EasyDiffusion
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *EasyDiffusion) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type EasyDiffusion struct {
	Prompt                  string   `json:"prompt"`
	Seed                    int64    `json:"seed"`
	UsedRandomSeed          bool     `json:"used_random_seed"`
	NegativePrompt          string   `json:"negative_prompt"`
	NumOutputs              int64    `json:"num_outputs"`
	NumInferenceSteps       int64    `json:"num_inference_steps"`
	GuidanceScale           float64  `json:"guidance_scale"`
	Width                   int64    `json:"width"`
	Height                  int64    `json:"height"`
	VRAMUsageLevel          string   `json:"vram_usage_level"`
	SamplerName             string   `json:"sampler_name"`
	UseStableDiffusionModel string   `json:"use_stable_diffusion_model"`
	ClipSkip                bool     `json:"clip_skip"`
	UseVaeModel             string   `json:"use_vae_model"`
	StreamProgressUpdates   bool     `json:"stream_progress_updates"`
	StreamImageProgress     bool     `json:"stream_image_progress"`
	ShowOnlyFilteredImage   bool     `json:"show_only_filtered_image"`
	BlockNsfw               bool     `json:"block_nsfw"`
	OutputFormat            string   `json:"output_format"`
	OutputQuality           int64    `json:"output_quality"`
	OutputLossless          bool     `json:"output_lossless"`
	MetadataOutputFormat    string   `json:"metadata_output_format"`
	OriginalPrompt          string   `json:"original_prompt"`
	ActiveTags              []string `json:"active_tags"`
	InactiveTags            []string `json:"inactive_tags"`
	EnableVaeTiling         bool     `json:"enable_vae_tiling"`
	UseEmbeddingsModel      []string `json:"use_embeddings_model"`
}

func (r *EasyDiffusion) Convert() *TextToImageRequest {
	if r == nil {
		return nil
	}
	if r.Prompt == "" {
		r.Prompt = r.OriginalPrompt
	}
	var config Config
	if r.UseStableDiffusionModel != "" {
		config.SDModelCheckpoint = &r.UseStableDiffusionModel
	}
	if r.UseVaeModel != "" {
		config.SDVae = &r.UseVaeModel
	}
	return &TextToImageRequest{
		BatchSize:        int(r.NumOutputs),
		Steps:            int(r.NumInferenceSteps),
		CFGScale:         r.GuidanceScale,
		Prompt:           r.Prompt,
		NegativePrompt:   r.NegativePrompt,
		Width:            int(r.Width),
		Height:           int(r.Height),
		SamplerName:      r.SamplerName,
		Seed:             r.Seed,
		OverrideSettings: config,
	}
}
