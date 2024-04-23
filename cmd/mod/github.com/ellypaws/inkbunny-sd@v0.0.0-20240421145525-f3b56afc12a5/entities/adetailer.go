package entities

type ADetailerParameters struct {
	AdModel                    string  `json:"ad_model,omitempty"`
	AdPrompt                   string  `json:"ad_prompt,omitempty"`
	AdNegativePrompt           string  `json:"ad_negative_prompt,omitempty"`
	AdConfidence               float64 `json:"ad_confidence,omitempty"`
	AdMaskKLargest             int     `json:"ad_mask_k_largest,omitempty"`
	AdMaskMinRatio             float64 `json:"ad_mask_min_ratio,omitempty"`
	AdMaskMaxRatio             float64 `json:"ad_mask_max_ratio,omitempty"`
	AdDilateErode              int     `json:"ad_dilate_erode,omitempty"`
	AdXOffset                  int     `json:"ad_x_offset,omitempty"`
	AdYOffset                  int     `json:"ad_y_offset,omitempty"`
	AdMaskMergeInvert          string  `json:"ad_mask_merge_invert,omitempty"`
	AdMaskBlur                 int     `json:"ad_mask_blur,omitempty"`
	AdDenoisingStrength        float64 `json:"ad_denoising_strength,omitempty"`
	AdInpaintOnlyMasked        bool    `json:"ad_inpaint_only_masked,omitempty"`
	AdInpaintOnlyMaskedPadding int     `json:"ad_inpaint_only_masked_padding,omitempty"`
	AdUseInpaintWidthHeight    bool    `json:"ad_use_inpaint_width_height,omitempty"`
	AdInpaintWidth             int     `json:"ad_inpaint_width,omitempty"`
	AdInpaintHeight            int     `json:"ad_inpaint_height,omitempty"`
	AdUseSteps                 bool    `json:"ad_use_steps,omitempty"`
	AdSteps                    int     `json:"ad_steps,omitempty"`
	AdUseCfgScale              bool    `json:"ad_use_cfg_scale,omitempty"`
	AdCfgScale                 float64 `json:"ad_cfg_scale,omitempty"`
	AdUseSampler               bool    `json:"ad_use_sampler,omitempty"`
	AdSampler                  string  `json:"ad_sampler,omitempty"`
	AdUseNoiseMultiplier       bool    `json:"ad_use_noise_multiplier,omitempty"`
	AdNoiseMultiplier          float64 `json:"ad_noise_multiplier,omitempty"`
	AdUseClipSkip              bool    `json:"ad_use_clip_skip,omitempty"`
	AdClipSkip                 int     `json:"ad_clip_skip,omitempty"`
	AdRestoreFace              bool    `json:"ad_restore_face,omitempty"`
	AdControlnetModel          string  `json:"ad_controlnet_model,omitempty"`
	AdControlnetModule         *string `json:"ad_controlnet_module,omitempty"`
	AdControlnetWeight         float64 `json:"ad_controlnet_weight,omitempty"`
	AdControlnetGuidanceStart  float64 `json:"ad_controlnet_guidance_start,omitempty"`
	AdControlnetGuidanceEnd    float64 `json:"ad_controlnet_guidance_end,omitempty"`
}

type ADetailer struct {
	Args []*ADetailerParameters `json:"args,omitempty"`
}
