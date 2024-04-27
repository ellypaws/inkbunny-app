package entities

import "encoding/json"

type ComfyUIAlternate map[string]ComfyUIAlternateValue

func UnmarshalComfyUIAlternate(data []byte) (ComfyUIAlternate, error) {
	var r ComfyUIAlternate
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ComfyUIAlternate) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type ComfyUIAlternateValue struct {
	Inputs    Inputs `json:"inputs"`
	ClassType string `json:"class_type"`
}

type Inputs struct {
	ModelName              *string     `json:"model_name,omitempty"`
	CkptName               *string     `json:"ckpt_name,omitempty"`
	Width                  *int64      `json:"width,omitempty"`
	Height                 *int64      `json:"height,omitempty"`
	BatchSize              *int64      `json:"batch_size,omitempty"`
	VaeName                *string     `json:"vae_name,omitempty"`
	Switch1                *string     `json:"switch_1,omitempty"`
	LoraName1              *string     `json:"lora_name_1,omitempty"`
	ModelWeight1           *float64    `json:"model_weight_1,omitempty"`
	ClipWeight1            *int64      `json:"clip_weight_1,omitempty"`
	Switch2                *string     `json:"switch_2,omitempty"`
	LoraName2              *string     `json:"lora_name_2,omitempty"`
	ModelWeight2           *float64    `json:"model_weight_2,omitempty"`
	ClipWeight2            *int64      `json:"clip_weight_2,omitempty"`
	Switch3                *string     `json:"switch_3,omitempty"`
	LoraName3              *string     `json:"lora_name_3,omitempty"`
	ModelWeight3           *float64    `json:"model_weight_3,omitempty"`
	ClipWeight3            *int64      `json:"clip_weight_3,omitempty"`
	LoraStack              []StringInt `json:"lora_stack,omitempty"`
	Model                  []StringInt `json:"model,omitempty"`
	Clip                   []StringInt `json:"clip,omitempty"`
	Seed                   *Seed       `json:"seed,omitempty"`
	Pos                    []StringInt `json:"pos,omitempty"`
	Neg                    []StringInt `json:"neg,omitempty"`
	Latent                 []StringInt `json:"latent,omitempty"`
	Vae                    []StringInt `json:"vae,omitempty"`
	Pipe                   []StringInt `json:"pipe,omitempty"`
	StopAtClipLayer        *int64      `json:"stop_at_clip_layer,omitempty"`
	Text                   *string     `json:"text,omitempty"`
	ConditioningTo         []StringInt `json:"conditioning_to,omitempty"`
	ConditioningFrom       []StringInt `json:"conditioning_from,omitempty"`
	Steps                  *int64      `json:"steps,omitempty"`
	CFG                    *int64      `json:"cfg,omitempty"`
	SamplerName            *string     `json:"sampler_name,omitempty"`
	Scheduler              *string     `json:"scheduler,omitempty"`
	TiledVae               *string     `json:"tiled_vae,omitempty"`
	LatentUpscale          *string     `json:"latent_upscale,omitempty"`
	UpscaleFactor          *float64    `json:"upscale_factor,omitempty"`
	UpscaleCycles          *int64      `json:"upscale_cycles,omitempty"`
	StartingDenoise        *int64      `json:"starting_denoise,omitempty"`
	CycleDenoise           *float64    `json:"cycle_denoise,omitempty"`
	ScaleDenoise           *string     `json:"scale_denoise,omitempty"`
	ScaleSampling          *string     `json:"scale_sampling,omitempty"`
	SecondaryStartCycle    *int64      `json:"secondary_start_cycle,omitempty"`
	PosAddMode             *string     `json:"pos_add_mode,omitempty"`
	PosAddStrength         *float64    `json:"pos_add_strength,omitempty"`
	PosAddStrengthScaling  *string     `json:"pos_add_strength_scaling,omitempty"`
	PosAddStrengthCutoff   *int64      `json:"pos_add_strength_cutoff,omitempty"`
	NegAddMode             *string     `json:"neg_add_mode,omitempty"`
	NegAddStrength         *float64    `json:"neg_add_strength,omitempty"`
	NegAddStrengthScaling  *string     `json:"neg_add_strength_scaling,omitempty"`
	NegAddStrengthCutoff   *int64      `json:"neg_add_strength_cutoff,omitempty"`
	SharpenStrength        *int64      `json:"sharpen_strength,omitempty"`
	SharpenRadius          *int64      `json:"sharpen_radius,omitempty"`
	StepsScaling           *string     `json:"steps_scaling,omitempty"`
	StepsControl           *string     `json:"steps_control,omitempty"`
	StepsScalingValue      *int64      `json:"steps_scaling_value,omitempty"`
	StepsCutoff            *int64      `json:"steps_cutoff,omitempty"`
	DenoiseCutoff          *float64    `json:"denoise_cutoff,omitempty"`
	Positive               []StringInt `json:"positive,omitempty"`
	Negative               []StringInt `json:"negative,omitempty"`
	LatentImage            []StringInt `json:"latent_image,omitempty"`
	UpscaleModel           []StringInt `json:"upscale_model,omitempty"`
	Samples                []StringInt `json:"samples,omitempty"`
	FilenamePrefix         *string     `json:"filename_prefix,omitempty"`
	Images                 []StringInt `json:"images,omitempty"`
	Image                  []StringInt `json:"image,omitempty"`
	GuideSize              *int64      `json:"guide_size,omitempty"`
	GuideSizeFor           *bool       `json:"guide_size_for,omitempty"`
	MaxSize                *int64      `json:"max_size,omitempty"`
	Denoise                *float64    `json:"denoise,omitempty"`
	Feather                *int64      `json:"feather,omitempty"`
	NoiseMask              *bool       `json:"noise_mask,omitempty"`
	ForceInpaint           *bool       `json:"force_inpaint,omitempty"`
	BboxThreshold          *float64    `json:"bbox_threshold,omitempty"`
	BboxDilation           *int64      `json:"bbox_dilation,omitempty"`
	BboxCropFactor         *int64      `json:"bbox_crop_factor,omitempty"`
	SamDetectionHint       *string     `json:"sam_detection_hint,omitempty"`
	SamDilation            *int64      `json:"sam_dilation,omitempty"`
	SamThreshold           *float64    `json:"sam_threshold,omitempty"`
	SamBboxExpansion       *int64      `json:"sam_bbox_expansion,omitempty"`
	SamMaskHintThreshold   *float64    `json:"sam_mask_hint_threshold,omitempty"`
	SamMaskHintUseNegative *string     `json:"sam_mask_hint_use_negative,omitempty"`
	DropSize               *int64      `json:"drop_size,omitempty"`
	Wildcard               *string     `json:"wildcard,omitempty"`
	Cycle                  *int64      `json:"cycle,omitempty"`
	InpaintModel           *bool       `json:"inpaint_model,omitempty"`
	NoiseMaskFeather       *int64      `json:"noise_mask_feather,omitempty"`
	BboxDetector           []StringInt `json:"bbox_detector,omitempty"`
	Pixels                 []StringInt `json:"pixels,omitempty"`
	Image2                 []StringInt `json:"image2,omitempty"`
	UpscaleBy              *int64      `json:"upscale_by,omitempty"`
	ModeType               *string     `json:"mode_type,omitempty"`
	TileWidth              *int64      `json:"tile_width,omitempty"`
	TileHeight             *int64      `json:"tile_height,omitempty"`
	MaskBlur               *int64      `json:"mask_blur,omitempty"`
	TilePadding            *int64      `json:"tile_padding,omitempty"`
	SeamFixMode            *string     `json:"seam_fix_mode,omitempty"`
	SeamFixDenoise         *int64      `json:"seam_fix_denoise,omitempty"`
	SeamFixWidth           *int64      `json:"seam_fix_width,omitempty"`
	SeamFixMaskBlur        *int64      `json:"seam_fix_mask_blur,omitempty"`
	SeamFixPadding         *int64      `json:"seam_fix_padding,omitempty"`
	ForceUniformTiles      *bool       `json:"force_uniform_tiles,omitempty"`
	TiledDecode            *bool       `json:"tiled_decode,omitempty"`
	Image1                 []StringInt `json:"image1,omitempty"`
}

type StringInt struct {
	Integer *int64
	String  *string
}

func (x *StringInt) UnmarshalJSON(data []byte) error {
	object, err := unmarshalUnion(data, &x.Integer, nil, nil, &x.String, false, nil, false, nil, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *StringInt) MarshalJSON() ([]byte, error) {
	return marshalUnion(x.Integer, nil, nil, x.String, false, nil, false, nil, false, nil, false, nil, false)
}

type Seed struct {
	Integer    *int64
	UnionArray []StringInt
}

func (x *Seed) UnmarshalJSON(data []byte) error {
	x.UnionArray = nil
	object, err := unmarshalUnion(data, &x.Integer, nil, nil, nil, true, &x.UnionArray, false, nil, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *Seed) MarshalJSON() ([]byte, error) {
	return marshalUnion(x.Integer, nil, nil, nil, x.UnionArray != nil, x.UnionArray, false, nil, false, nil, false, nil, false)
}
