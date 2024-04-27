package entities

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"strings"
)
import "errors"

import "encoding/json"

func UnmarshalComfyUI(data []byte) (ComfyUI, error) {
	var r ComfyUI
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ComfyUI) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type ComfyUI struct {
	LastNodeID int64    `json:"last_node_id"`
	LastLinkID int64    `json:"last_link_id"`
	Nodes      []Node   `json:"nodes"`
	Links      [][]Link `json:"links"`
	Groups     []Group  `json:"groups"`
	Config     Empty    `json:"config"`
	Extra      Empty    `json:"extra"`
	Version    float64  `json:"version"`
}

type Empty struct{}

type Group struct {
	Title    string  `json:"title"`
	Bounding []int64 `json:"bounding"`
	Color    string  `json:"color"`
	FontSize int64   `json:"font_size"`
	Locked   bool    `json:"locked"`
}

type Node struct {
	ID            int64          `json:"id"`
	Type          NodeType       `json:"type"`
	Pos           *Pos           `json:"pos"`
	Size          *Pos           `json:"size"`
	Flags         Flags          `json:"flags"`
	Order         int64          `json:"order"`
	Mode          int64          `json:"mode"`
	Inputs        []Input        `json:"inputs,omitempty"`
	Outputs       []Output       `json:"outputs,omitempty"`
	Properties    Properties     `json:"properties"`
	WidgetsValues []WidgetsValue `json:"widgets_values,omitempty"`
	Color         *string        `json:"color,omitempty"`
	BGColor       *string        `json:"bgcolor,omitempty"`
}

type Flags struct {
	Collapsed *bool `json:"collapsed,omitempty"`
}

type Input struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Link      *int64  `json:"link"`
	SlotIndex *int64  `json:"slot_index,omitempty"`
	Widget    *Widget `json:"widget,omitempty"`
}

type Widget struct {
	Name string `json:"name"`
}

type Output struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Links     []int64 `json:"links"`
	SlotIndex *int64  `json:"slot_index,omitempty"`
	Shape     *int64  `json:"shape,omitempty"`
	Dir       *int64  `json:"dir,omitempty"`
	Label     *string `json:"label,omitempty"`
}

type Properties struct {
	NodeNameForSR      *string `json:"Node name for S&R,omitempty"`
	Text               *string `json:"text,omitempty"`
	MatchColors        *string `json:"matchColors,omitempty"`
	MatchTitle         *string `json:"matchTitle,omitempty"`
	ShowNav            *bool   `json:"showNav,omitempty"`
	Sort               *string `json:"sort,omitempty"`
	CustomSortAlphabet *string `json:"customSortAlphabet,omitempty"`
	ToggleRestriction  *string `json:"toggleRestriction,omitempty"`
	ShowOutputText     *bool   `json:"showOutputText,omitempty"`
	Horizontal         *bool   `json:"horizontal,omitempty"`
}

type Link struct {
	Integer *int64
	String  *string
}

func (x *Link) UnmarshalJSON(data []byte) error {
	object, err := unmarshalUnion(data, &x.Integer, nil, nil, &x.String, false, nil, false, nil, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *Link) MarshalJSON() ([]byte, error) {
	return marshalUnion(x.Integer, nil, nil, x.String, false, nil, false, nil, false, nil, false, nil, false)
}

type Pos struct {
	DoubleArray []float64
	DoubleMap   map[string]float64
}

func (x *Pos) UnmarshalJSON(data []byte) error {
	x.DoubleArray = nil
	x.DoubleMap = nil
	object, err := unmarshalUnion(data, nil, nil, nil, nil, true, &x.DoubleArray, false, nil, true, &x.DoubleMap, false, nil, false)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *Pos) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, nil, x.DoubleArray != nil, x.DoubleArray, false, nil, x.DoubleMap != nil, x.DoubleMap, false, nil, false)
}

type WidgetsValue struct {
	Bool   *bool
	Double *float64
	String *string
}

func (x *WidgetsValue) UnmarshalJSON(data []byte) error {
	object, err := unmarshalUnion(data, nil, &x.Double, &x.Bool, &x.String, false, nil, false, nil, false, nil, false, nil, true)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *WidgetsValue) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, x.Double, x.Bool, x.String, false, nil, false, nil, false, nil, false, nil, true)
}

func unmarshalUnion(data []byte, pi **int64, pf **float64, pb **bool, ps **string, haveArray bool, pa interface{}, haveObject bool, pc interface{}, haveMap bool, pm interface{}, haveEnum bool, pe interface{}, nullable bool) (bool, error) {
	if pi != nil {
		*pi = nil
	}
	if pf != nil {
		*pf = nil
	}
	if pb != nil {
		*pb = nil
	}
	if ps != nil {
		*ps = nil
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return false, err
	}

	switch v := tok.(type) {
	case json.Number:
		if pi != nil {
			i, err := v.Int64()
			if err == nil {
				*pi = &i
				return false, nil
			}
		}
		if pf != nil {
			f, err := v.Float64()
			if err == nil {
				*pf = &f
				return false, nil
			}
			return false, errors.New("Unparsable number")
		}
		return false, errors.New("Union does not contain number")
	case float64:
		return false, errors.New("Decoder should not return float64")
	case bool:
		if pb != nil {
			*pb = &v
			return false, nil
		}
		return false, errors.New("Union does not contain bool")
	case string:
		if haveEnum {
			return false, json.Unmarshal(data, pe)
		}
		if ps != nil {
			*ps = &v
			return false, nil
		}
		return false, errors.New("Union does not contain string")
	case nil:
		if nullable {
			return false, nil
		}
		return false, errors.New("Union does not contain null")
	case json.Delim:
		if v == '{' {
			if haveObject {
				return true, json.Unmarshal(data, pc)
			}
			if haveMap {
				return false, json.Unmarshal(data, pm)
			}
			return false, errors.New("Union does not contain object")
		}
		if v == '[' {
			if haveArray {
				return false, json.Unmarshal(data, pa)
			}
			return false, errors.New("Union does not contain array")
		}
		return false, errors.New("Cannot handle delimiter")
	}
	return false, errors.New("Cannot unmarshal union")

}

func marshalUnion(pi *int64, pf *float64, pb *bool, ps *string, haveArray bool, pa interface{}, haveObject bool, pc interface{}, haveMap bool, pm interface{}, haveEnum bool, pe interface{}, nullable bool) ([]byte, error) {
	if pi != nil {
		return json.Marshal(*pi)
	}
	if pf != nil {
		return json.Marshal(*pf)
	}
	if pb != nil {
		return json.Marshal(*pb)
	}
	if ps != nil {
		return json.Marshal(*ps)
	}
	if haveArray {
		return json.Marshal(pa)
	}
	if haveObject {
		return json.Marshal(pc)
	}
	if haveMap {
		return json.Marshal(pm)
	}
	if haveEnum {
		return json.Marshal(pe)
	}
	if nullable {
		return json.Marshal(nil)
	}
	return nil, errors.New("Union must not be null")
}

type NodeType string

const (
	VAEDecode                   NodeType = "VAEDecode"
	VAEEncode                   NodeType = "VAEEncode"
	UpscaleModelLoader          NodeType = "Upscale Model Loader"
	CRModuleInput               NodeType = "CR Module Input"
	ImageUpscaleWithModel       NodeType = "ImageUpscaleWithModel"
	SeedNode                    NodeType = "Seed (rgthree)"
	PreviewImage                NodeType = "PreviewImage"
	VAELoader                   NodeType = "VAELoader"
	CRModulePipeLoader          NodeType = "CR Module Pipe Loader"
	ConditioningConcat          NodeType = "ConditioningConcat"
	SaveImage                   NodeType = "SaveImage"
	CLIPTextEncode              NodeType = "CLIPTextEncode"
	ModelMergeSimple            NodeType = "ModelMergeSimple"
	Note                        NodeType = "Note"
	FreeU_V2                    NodeType = "FreeU_V2"
	CheckpointLoaderSimple      NodeType = "CheckpointLoaderSimple"
	KSamplerCycle               NodeType = "KSampler Cycle"
	CRApplyLoRAStack            NodeType = "CR Apply LoRA Stack"
	CLIPSetLastLayer            NodeType = "CLIPSetLastLayer"
	CRApplyModelMerge           NodeType = "CR Apply Model Merge"
	CRLoRAStack                 NodeType = "CR LoRA Stack"
	FastGroupsBypasser          NodeType = "Fast Groups Bypasser (rgthree)"
	EmptyLatentImage            NodeType = "EmptyLatentImage"
	CRModelMergeStack           NodeType = "CR Model Merge Stack"
	FastGroupsMuter             NodeType = "Fast Groups Muter (rgthree)"
	SimpleCounter               NodeType = "Simple Counter"
	KSampler                    NodeType = "KSampler"
	UltralyticsDetectorProvider NodeType = "UltralyticsDetectorProvider"
	FaceDetailer                NodeType = "FaceDetailer"
	Reroute                     NodeType = "Reroute"
	UltimateSDUpscale           NodeType = "UltimateSDUpscale"
)

func fallback[T any](field *T, fallback T) {
	if field == nil {
		panic("fallback called with nil field")
	}
	if reflect.ValueOf(*field).IsZero() {
		*field = fallback
	}
}

var negatives = []string{
	"low quality",
	"easynegative",
	"blurry",
}

func (r *ComfyUI) Convert() *TextToImageRequest {
	if r == nil {
		return nil
	}
	var _ Config
	var req TextToImageRequest
	var prompt strings.Builder
	var loras = make(map[string]float64)
	for _, node := range r.Nodes {
		switch node.Type {
		case CheckpointLoaderSimple:
			for _, input := range node.WidgetsValues {
				if input.String != nil {
					req.OverrideSettings.SDModelCheckpoint = input.String
				}
			}
		case VAELoader:
			for _, input := range node.WidgetsValues {
				if input.String != nil {
					req.OverrideSettings.SDVae = input.String
				}
			}
		case CRLoRAStack:
			var lastLora *string
			var enabled bool
			for i, input := range node.WidgetsValues {
				switch i % 4 {
				case 0:
					if input.String != nil {
						enabled = *input.String == "On"
					}
				case 1:
					if input.String != nil {
						if enabled {
							lastLora = input.String
							loras[*lastLora] = 1
						}
					}
				case 2:
					if input.Double != nil {
						if enabled && lastLora != nil {
							loras[*lastLora] = *input.Double
							enabled = false
							lastLora = nil
						}
					}
				}
			}
		case CLIPTextEncode:
			for _, input := range node.WidgetsValues {
				if input.String != nil {
					if req.NegativePrompt != "" {
						prompt.WriteString(strings.TrimSpace(*input.String))
						continue
					}
					if slices.ContainsFunc(negatives, func(negative string) bool {
						return strings.Contains(*input.String, negative)
					}) {
						req.NegativePrompt = *input.String
						continue
					}
					prompt.WriteString(strings.TrimSpace(*input.String))
				}
			}
		case SeedNode:
			for _, input := range node.WidgetsValues {
				if input.Double != nil {
					req.Seed = int64(*input.Double)
				}
			}
		case KSamplerCycle:
			for i, input := range node.WidgetsValues {
				switch i {
				case 0:
					if input.Double != nil {
						fallback(&req.Seed, int64(*input.Double))
					}
				case 1:
				case 2:
					if input.Double != nil {
						req.Steps = int(*input.Double)
					}
				case 3:
					if input.Double != nil {
						req.CFGScale = *input.Double
					}
				case 4:
					if input.String != nil {
						req.SamplerName = *input.String
					}
				case 8:
					if input.Double != nil {
						req.HrScale = *input.Double
					}
				}
			}
		case KSampler:
			for i, input := range node.WidgetsValues {
				switch i {
				case 0:
					if input.Double != nil {
						fallback(&req.Seed, int64(*input.Double))
					}
				case 2:
					if input.Double != nil {
						fallback(&req.Steps, int(*input.Double))
					}
				case 3:
					if input.Double != nil {
						fallback(&req.CFGScale, *input.Double)
					}
				case 4:
					if input.String != nil {
						fallback(&req.SamplerName, *input.String)
					}
				case 5:
					if input.String != nil {
						fallback(&req.Scheduler, input.String)
					}
				case 6:
					if input.Double != nil {
						fallback(&req.DenoisingStrength, *input.Double)
					}
				}
			}
		}
	}

	for lora, weight := range loras {
		prompt.WriteString(fmt.Sprintf("<lora:%s:%.2f>", lora, weight))
	}

	req.Prompt = prompt.String()

	return &req
}
