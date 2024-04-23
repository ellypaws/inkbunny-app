package entities

type Scripts struct {
	ADetailer  *ADetailer  `json:"ADetailer,omitempty"`
	ControlNet *ControlNet `json:"ControlNet,omitempty"`
	CFGRescale *CFGRescale `json:"CFG Rescale Extension,omitempty"`
}
