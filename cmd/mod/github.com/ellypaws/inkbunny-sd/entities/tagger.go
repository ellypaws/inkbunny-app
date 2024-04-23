// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    taggerInterrogate, err := UnmarshalTaggerInterrogate(bytes)
//    bytes, err = taggerInterrogate.Marshal()

package entities

import (
	"encoding/base64"
	"encoding/json"
)

func UnmarshalTaggerInterrogate(data []byte) (TaggerRequest, error) {
	var r TaggerRequest
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *TaggerRequest) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func (r *TaggerRequest) SetThreshold(f float64) *TaggerRequest {
	r.Threshold = &f
	return r
}

func (r *TaggerRequest) WithImageBytes(b []byte) *TaggerRequest {
	b64 := base64.StdEncoding.EncodeToString(b)
	r.Image = &b64
	return r
}

// TaggerRequest Interrogate request model
type TaggerRequest struct {
	// Image to work on, must be a Base64 string containing the image's data or a URL.
	Image *string `json:"image,omitempty"`
	// The interrogator model used.
	Model string `json:"model"`
	// name to queue image as or use <sha256>. leave empty to retrieve the final response
	NameInQueue *string `json:"name_in_queue,omitempty"`
	// name of queue; leave empty for single response
	Queue *string `json:"queue,omitempty"`
	// The threshold used for the interrogator model.
	Threshold *float64 `json:"threshold,omitempty"`
}

func UnmarshalTaggerResponse(data []byte) (TaggerResponse, error) {
	var r TaggerResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *TaggerResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type TagEnum = string

// TaggerResponse Interrogate response model
type TaggerResponse struct {
	// The generated captions for the image.
	Caption CaptionEnum `json:"caption"`
}

type CaptionEnum struct {
	Tag    map[string]float64 `json:"tag,omitempty"`
	Rating map[string]float64 `json:"rating,omitempty"`
}

func (r *TaggerResponse) Captions() map[string]float64 {
	return r.Caption.Tag
}

func (r *TaggerResponse) HumanPercent() float64 {
	if r.Caption.Tag == nil {
		return 0
	}
	if p, ok := r.Caption.Tag[TagEnumHuman]; ok {
		return p
	}
	return 0
}

func (r *TaggerResponse) CubPercent() float64 {
	if r.Caption.Tag == nil {
		return 0
	}
	if p, ok := r.Caption.Tag[TagEnumCub]; ok {
		return p
	}
	return 0
}

func (r *TaggerResponse) FurryPercent() float64 {
	if r.Caption.Tag == nil {
		return 0
	}
	var p float64
	for _, v := range []TagEnum{
		TagEnumFurry,
		TagEnumAnthro,
		TagEnumMammal,
	} {
		if f, ok := r.Caption.Tag[v]; ok {
			p = max(p, f)
		}
	}
	return p
}

func UnmarshalInterrogators(data []byte) (Interrogators, error) {
	var r Interrogators
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Interrogators) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Interrogators struct {
	Models []string `json:"models"`
}

const (
	TaggerWd14VitV1         = "wd14-vit.v1"
	TaggerWd14VitV2         = "wd14-vit.v2"
	TaggerWd14ConvnextV1    = "wd14-convnext.v1"
	TaggerWd14ConvnextV2    = "wd14-convnext.v2"
	TaggerWd14Convnextv2V1  = "wd14-convnextv2.v1"
	TaggerWd14Swinv2V1      = "wd14-swinv2-v1"
	TaggerWdV14MoatTaggerV2 = "wd-v1-4-moat-tagger.v2"
	TaggerMldCaformerDec    = "mld-caformer.dec-5-97527"
	TaggerMldTresnetd       = "mld-tresnetd.6-30000"
	TaggerZ3DE621Convnext   = "Z3D-E621-Convnext"
)

const (
	TagEnumHuman  TagEnum = "human"
	TagEnumFurry  TagEnum = "furry"
	TagEnumCub    TagEnum = "cub"
	TagEnumAnthro TagEnum = "anthro"
	TagEnumMammal TagEnum = "mammal"
)
