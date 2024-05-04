package utils

import (
	"errors"
	"fmt"
	"github.com/ellypaws/inkbunny-sd/entities"
	"math"
	"strconv"
	"strings"
)

func DescriptionHeuristics(description string) (entities.TextToImageRequest, error) {
	description = RemoveBBCode(description)

	if description := parametersStart.FindString(description); description != "" {
		params, err := Common(
			WithString(description),
			WithKeyCondition(func(line string) bool { return strings.HasPrefix(line, "parameters") }))
		if err != nil {
			return entities.TextToImageRequest{}, err
		}

		for _, param := range params {
			if p, ok := param[Parameters]; ok {
				heuristics, err := ParameterHeuristics(p)
				if err != nil {
					return entities.TextToImageRequest{}, err
				}
				return heuristics, nil
			}
		}
	}

	results := ExtractAll(description, Patterns)

	var request entities.TextToImageRequest

	fieldsToSet := map[string]any{
		"steps":     &request.Steps,
		"sampler":   &request.SamplerName,
		"cfg":       &request.CFGScale,
		"seed":      &request.Seed,
		"width":     &request.Width,
		"height":    &request.Height,
		"hash":      &request.OverrideSettings.SDCheckpointHash,
		"model":     &request.OverrideSettings.SDModelCheckpoint,
		"denoising": &request.DenoisingStrength,
	}

	err := ResultsToFields(results, fieldsToSet)
	if err != nil {
		return request, err
	}

	request.Prompt = ExtractPositivePrompt(description)
	request.NegativePrompt = ExtractNegativePrompt(description)
	return request, nil
}

func RNSDAIHeuristics(description string) (entities.TextToImageRequest, error) {
	results := ExtractAll(description, RNSDAIPatterns)

	var request entities.TextToImageRequest

	fieldsToSet := map[string]any{
		"model":    &request.OverrideSettings.SDModelCheckpoint,
		"seed":     &request.Seed,
		"prompt":   &request.Prompt,
		"negative": &request.NegativePrompt,
	}

	err := ResultsToFields(results, fieldsToSet)
	if err != nil {
		return request, err
	}

	return request, nil
}

// IncompleteParameters is returned when the parameters potentially are not enough to create a request.
var IncompleteParameters = errors.New("incomplete parameters")

// ParameterHeuristics returns a TextToImageRequest from the given parameters.
// It uses the standard parameters embedded in an image from Stable Diffusion.
// The function emulates the behavior of parse_generation_parameters in
// https://github.com/AUTOMATIC1111/stable-diffusion-webui/blob/master/modules/infotext_utils.py#L233
func ParameterHeuristics(parameters string) (entities.TextToImageRequest, error) {
	parameters = strings.TrimSpace(parameters)

	lines := strings.Split(parameters, "\n")

	if len(lines) < 2 {
		return entities.TextToImageRequest{
			Prompt: parameters,
		}, IncompleteParameters
	}

	// Get positive and negative prompts excluding the last line, which is the extra parameters.
	positive, negative := GetPrompts(lines[:len(lines)-1])

	// Extract the parameters from the last line.
	results := ExtractDefaultKeys(lines[len(lines)-1], DefaultResults())

	if sizes, ok := results["Size"]; ok {
		for i, size := range strings.Split(sizes, "x") {
			switch i {
			case 0:
				results["Width"] = size
			case 1:
				results["Height"] = size
			}
		}
	}

	restoreOldHiresFixParams(results, false)

	if version, ok := results["Version"]; ok {
		if !strings.HasPrefix(version, "v") {
			results["Version"] = "v" + version
		}
		// Not used for now and should manually add a "Schedule type" if version of local Stable Diffusion is >= 1.7.0
		// semver is still not in standard go library, avoid using it for now
		// Check if the version is less than 1.7.0 then set "Downcast to fp16" to true'
		//if semver.Compare(version, "v1.7.0-225") < 0 {
		//	results["Downcast to fp16"] = "True"
		//} else {
		//	if sampler, ok := results["Sampler"]; ok {
		//		for _, schedulers := range []string{
		//			"Uniform",
		//			"Karras",
		//			"Exponential",
		//			"Polyexponential",
		//			"SGM Uniform",
		//		} {
		//			if strings.HasSuffix(sampler, schedulers) {
		//				results["Sampler"] = strings.TrimSuffix(sampler, schedulers)
		//				results["Schedule type"] = schedulers
		//				break
		//			}
		//		}
		//	}
		//}
	}

	var request entities.TextToImageRequest
	err := ResultsToFields(results, TextToImageFields(&request))
	if err != nil {
		return request, err
	}

	if hypernet, ok := results["Hypernet"]; ok {
		positive.WriteString(fmt.Sprintf("<hypernet:%s:%s>", hypernet, results["Hypernet strength"]))
	}

	if loras, ok := results["Lora hashes"]; ok {
		loras = strings.Trim(loras, `"`)
		for _, lora := range strings.Split(loras, ", ") {
			hashName := strings.SplitN(lora, ": ", 2)
			if len(hashName) == 2 {
				if request.LoraHashes == nil {
					request.LoraHashes = make(map[string]string)
				}
				request.LoraHashes[hashName[1]] = hashName[0]
			}
		}
	}

	if tis, ok := results["TI hashes"]; ok {
		tis = strings.Trim(tis, `"`)
		for _, ti := range strings.Split(tis, ", ") {
			nameHash := strings.SplitN(ti, ":", 2)
			if len(nameHash) == 2 {
				if request.TIHashes == nil {
					request.TIHashes = make(map[string]string)
				}
				request.TIHashes[nameHash[0]] = nameHash[1]
			}
		}
	}

	request.Prompt = positive.String()
	request.NegativePrompt = negative.String()

	// Fallback
	if request.Prompt == "" {
		request.Prompt = ExtractPositivePrompt(parameters)
	}

	if request.NegativePrompt == "" {
		request.NegativePrompt = ExtractNegativePrompt(parameters)
	}

	return request, nil
}

// DefaultResults returns the default key-value pairs for the parameters.
func DefaultResults() ExtractResult {
	var results = ExtractResult{
		"Clip skip": "1",
		//"Hires resize-1":                   "0",
		//"Hires resize-2":                   "0",
		"Hires sampler":    "Use same sampler",
		"Hires checkpoint": "Use same checkpoint",
		//"Hires prompt":                     "",
		//"Hires negative prompt":            "",
		"Mask mode": "Inpaint masked",
		//"Masked content":                   "original",
		//"Inpaint area":                     "Whole picture",
		//"Masked area padding":              "32",
		"RNG":           "GPU",
		"Schedule type": "Automatic",
		//"Schedule max sigma":               "0",
		//"Schedule min sigma":               "0",
		//"Schedule rho":                     "0",
		//"VAE Encoder":                      "Full",
		//"VAE Decoder":                      "Full",
		//"FP8 weight":                       "Disable",
		//"Cache FP16 weight for LoRA":       "False",
		//"Refiner switch by sampling steps": "False",
	}

	//if hypernet, ok := results["Hypernet"]; ok {
	//	positive.WriteString(fmt.Sprintf("<hypernet:%s:%s>", hypernet, results["Hypernet strength"]))
	//}

	//promptAttention := parsePromptAttention(request.Prompt) + parsePromptAttention(request.NegativePrompt)

	//var promptUsesEmphasis [][]string
	//for _, p := range promptAttention {
	//	if p[1] == 1.0 || p[0] == "BREAK" {
	//		promptUsesEmphasis = append(promptUsesEmphasis, p)
	//	}
	//}

	//if _, ok := results["Emphasis"]; !ok && promptUsesEmphasis {
	//	results["Emphasis"] = "Original"
	//}

	return results
}

// AllDefaultResults returns the default key-value pairs for the parameters.
// This is more faithful to the original implementation in the webui.
func AllDefaultResults() ExtractResult {
	var results = ExtractResult{
		"Clip skip":                        "1",
		"Hires resize-1":                   "0",
		"Hires resize-2":                   "0",
		"Hires sampler":                    "Use same sampler",
		"Hires checkpoint":                 "Use same checkpoint",
		"Hires prompt":                     "",
		"Hires negative prompt":            "",
		"Mask mode":                        "Inpaint masked",
		"Masked content":                   "original",
		"Inpaint area":                     "Whole picture",
		"Masked area padding":              "32",
		"RNG":                              "GPU",
		"Schedule type":                    "Automatic",
		"Schedule max sigma":               "0",
		"Schedule min sigma":               "0",
		"Schedule rho":                     "0",
		"VAE Encoder":                      "Full",
		"VAE Decoder":                      "Full",
		"FP8 weight":                       "Disable",
		"Cache FP16 weight for LoRA":       "False",
		"Refiner switch by sampling steps": "False",
	}

	//if hypernet, ok := results["Hypernet"]; ok {
	//	positive.WriteString(fmt.Sprintf("<hypernet:%s:%s>", hypernet, results["Hypernet strength"]))
	//}

	//promptAttention := parsePromptAttention(request.Prompt) + parsePromptAttention(request.NegativePrompt)

	//var promptUsesEmphasis [][]string
	//for _, p := range promptAttention {
	//	if p[1] == 1.0 || p[0] == "BREAK" {
	//		promptUsesEmphasis = append(promptUsesEmphasis, p)
	//	}
	//}

	//if _, ok := results["Emphasis"]; !ok && promptUsesEmphasis {
	//	results["Emphasis"] = "Original"
	//}

	return results
}

// GetPrompts returns the positive and negative prompts from the given lines.
// It goes line by line until it finds the negative prompt, then it returns the positive and negative prompts.
// Everything before the negative prompt is considered the positive prompt.
// The last line should not be included since it's the extra parameters.
// It returns a strings.Builder, use the String() method to get the result.
func GetPrompts(lines []string) (strings.Builder, strings.Builder) {
	var positive, negative strings.Builder
	var negativeFound bool
	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case len(line) == 0:
			continue
		case negativeStart.MatchString(line):
			negativeFound = true
			negative.WriteString(strings.TrimSpace(line[len(negativeStart.FindString(line)):]))
		case negativeFound:
			if negative.Len() > 0 {
				negative.WriteString("\n")
			}
			negative.WriteString(line)
		default:
			if positive.Len() > 0 {
				positive.WriteString("\n")
			}
			positive.WriteString(line)
		}
	}
	return positive, negative
}

func TextToImageFields(request *entities.TextToImageRequest) map[string]any {
	if request == nil {
		return nil
	}
	return map[string]any{
		"Steps":                 &request.Steps,
		"Sampler":               &request.SamplerName,
		"CFG scale":             &request.CFGScale,
		"Seed":                  &request.Seed,
		"Denoising strength":    &request.DenoisingStrength,
		"Width":                 &request.Width,
		"Height":                &request.Height,
		"Model":                 &request.OverrideSettings.SDModelCheckpoint,
		"Model hash":            &request.OverrideSettings.SDCheckpointHash,
		"VAE":                   &request.OverrideSettings.SDVae,
		"VAE hash":              &request.OverrideSettings.SDVaeExplanation,
		"Hires upscale":         &request.HrScale,
		"Hires steps":           &request.HrSecondPassSteps,
		"Hires upscaler":        &request.HrUpscaler,
		"Clip skip":             &request.OverrideSettings.CLIPStopAtLastLayers,
		"Hires resize-1":        &request.HrResizeX,
		"Hires resize-2":        &request.HrResizeY,
		"Hires sampler":         &request.HrSamplerName,
		"Hires checkpoint":      &request.HrCheckpointName,
		"Hires prompt":          &request.HrPrompt,
		"Hires negative prompt": &request.HrNegativePrompt,
		"RNG":                   &request.OverrideSettings.RandnSource,
		"KScheduler":            &request.OverrideSettings.KSchedType,
		"Schedule type":         &request.Scheduler, // For 1.8.0 and above
		"Schedule max sigma":    &request.OverrideSettings.SigmaMax,
		"Schedule min sigma":    &request.OverrideSettings.SigmaMin,
		"Schedule rho":          &request.OverrideSettings.Rho,
		"VAE Encoder":           &request.OverrideSettings.SDVaeEncodeMethod,
		"VAE Decoder":           &request.OverrideSettings.SDVaeDecodeMethod,
		"Downcast to fp16":      &request.OverrideSettings.DowncastAlphasCumprodToFP16,
		//"FP8 weight":                       &request.OverrideSettings.DisableWeightsAutoSwap,       // TODO: this is a bool, but FP8 weight is a string e.g. "Disable"
		//"Cache FP16 weight for LoRA":       &request.OverrideSettings.SDVaeCheckpointCache,         // TODO: this is a float64, but Cache FP16 weight for LoRA is a bool e.g. False
		//"Emphasis":                         &request.OverrideSettings.EnableEmphasis,               // TODO: this is a string, but Emphasis is a bool e.g. "Original"
		//"Emphasis":                         &request.OverrideSettings.UseOldEmphasisImplementation, // TODO: this is a string, but Emphasis is a bool e.g. "Original"
		//"Refiner switch by sampling steps": &request.OverrideSettings.HiresFixRefinerPass,          // TODO: this is a string, but Refiner switch by sampling steps is a bool e.g. False
	}
}

// Deprecated: not yet implemented in callers
// restoreOldHiresFixParams restores the old hires fix parameters if the new hires fix parameters are not present.
// Set use is true if the new hires fix parameters should be used, false if the old hires fix parameters should be used.
func restoreOldHiresFixParams(results ExtractResult, use bool) {
	var firstpassWidth, firstpassHeight int

	fieldsToSet := map[string]any{
		"First pass size-1": &firstpassWidth,
		"First pass size-2": &firstpassHeight,
	}
	if err := ResultsToFields(results, fieldsToSet); err != nil {
		return
	}

	if use {
		var hiresWidth, hiresHeight int
		fieldsToSet = map[string]any{
			"Hires resize-1": &hiresWidth,
			"Hires resize-2": &hiresHeight,
		}
		if err := ResultsToFields(results, fieldsToSet); err != nil {
			return
		}

		if hiresWidth != 0 && hiresHeight != 0 {
			results["Size-1"] = strconv.Itoa(hiresWidth)
			results["Size-2"] = strconv.Itoa(hiresHeight)
			return
		}
	}

	if firstpassWidth == 0 || firstpassHeight == 0 {
		return
	}

	width, _ := strconv.Atoi(results["Size-1"])
	height, _ := strconv.Atoi(results["Size-2"])

	if firstpassWidth == 0 || firstpassHeight == 0 {
		firstpassWidth, firstpassHeight = oldHiresFixFirstPassDimensions(width, height)
	}

	results["Size-1"] = strconv.Itoa(firstpassWidth)
	results["Size-2"] = strconv.Itoa(firstpassHeight)
	results["Hires resize-1"] = strconv.Itoa(width)
	results["Hires resize-2"] = strconv.Itoa(height)
}

func oldHiresFixFirstPassDimensions(width int, height int) (int, int) {
	desiredPixelCount := 512 * 512
	actualPixelCount := width * height
	scale := math.Sqrt(float64(desiredPixelCount) / float64(actualPixelCount))
	width = int(math.Ceil(scale*float64(width/64)) * 64)
	height = int(math.Ceil(scale*float64(height/64)) * 64)
	return width, height
}
