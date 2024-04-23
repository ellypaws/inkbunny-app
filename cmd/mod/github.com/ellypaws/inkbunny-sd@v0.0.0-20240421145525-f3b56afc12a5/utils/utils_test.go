package utils

import (
	_ "embed"
	"encoding/json"
	"testing"
)

const sample = `"See me after class, young lady."

|| Technical Information ||

[i](golden retriever, in a classroom, class, classroom, indoors, inside building, background blur)[/i]

[b]Negative prompt[/b]: deformityv6, bwu, dfc, ubbp, updn, ribcage

Steps: 50, Sampler: DPM++ 2M Karras, CFG scale: 12, Seed: 581623237, Size: 768x1024, Model hash: 70b33002f4, Model: furryrock_V70, Denoising strength: 0.45, Hires upscale: 2, Hires steps: 15, Hires upscaler: 4x-UltraMix_Smooth, Version: v1.6.0-2-g4afaaf8a`

func TestExtractPositivePrompt(t *testing.T) {
	t.Log(ExtractPositivePrompt(sample))
}

func TestExtractNegativePrompt(t *testing.T) {
	t.Log(ExtractNegativePrompt(sample))
}

func TestExtractPositiveBackwards(t *testing.T) {
	t.Log(ExtractPositiveBackwards(sample))
}

func TestExtractPositiveForward(t *testing.T) {
	t.Log(ExtractPositiveForward(sample))
}

func TestExtractNegativeBackwards(t *testing.T) {
	t.Log(ExtractNegativeBackwards(sample))
}

func TestExtractNegativeForward(t *testing.T) {
	t.Log(ExtractNegativeForward(sample))
}

//go:embed samples/description.txt
var description []byte

func TestDescriptionHeuristics(t *testing.T) {
	t2i, err := DescriptionHeuristics(sample)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	t2i, err = DescriptionHeuristics(string(description))
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	bytes, _ := json.MarshalIndent(t2i, "", "  ")
	t.Logf("TextToImageRequest: %v", string(bytes))
}

const testParameters = `(golden retriever, in a classroom, class, classroom, indoors, inside building, background blur)
Negative prompt: deformityv6, bwu, dfc, ubbp, updn, ribcage
Steps: 50, Sampler: DPM++ 2M Karras, CFG scale: 12, Seed: 581623237, Size: 768x1024, Model hash: 70b33002f4, Model: furryrock_V70, Denoising strength: 0.45, Hires upscale: 2, Hires steps: 15, Hires upscaler: 4x-UltraMix_Smooth, Version: v1.6.0-2-g4afaaf8a`

//go:embed samples/parameters.txt
var testFile []byte

func TestParameterHeuristics(t *testing.T) {
	t2i, err := ParameterHeuristics(testParameters)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	bytes, _ := json.MarshalIndent(t2i, "", "  ")
	t.Logf("PNGInfo: %v", string(bytes))
}

func TestParameterFileHeuristics(t *testing.T) {
	t2i, err := ParameterHeuristics(string(testFile))
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	bytes, _ := json.MarshalIndent(t2i, "", "  ")
	t.Logf("PNGInfo: %v", string(bytes))
}

func TestExtractAll(t *testing.T) {
	result := ExtractAll(sample, Patterns)
	if len(result) == 0 {
		t.Error("No results")
	}
	t.Logf("%+v", result)

	expected := map[string]string{
		"steps":      "50",
		"sampler":    "DPM++ 2M Karras",
		"cfg":        "12",
		"seed":       "581623237",
		"width":      "768",
		"height":     "1024",
		"hash":       "70b33002f4",
		"model":      "furryrock_V70",
		"denoising":  "0.45",
		"loraHashes": "",
		"tiHashes":   "",
		"version":    "v1.6.0-2-g4afaaf8a",
	}

	assert(t, result, expected)

	sampleWithHashes := sample + ` Lora hashes: "sizeslideroffset: 1d5a77d6b141, lora_Furry_female: 578e3efedb64, Furtastic_Detailer: 7aa86566f5ee", TI hashes: "AS-YoungestV2: c71427a287b5, AS-YoungV2: 714bba6525de", Version: 1.7.3`

	expected["loraHashes"] = "sizeslideroffset: 1d5a77d6b141, lora_Furry_female: 578e3efedb64, Furtastic_Detailer: 7aa86566f5ee"
	expected["tiHashes"] = "AS-YoungestV2: c71427a287b5, AS-YoungV2: 714bba6525de"

	result = ExtractAll(sampleWithHashes, Patterns)
	assert(t, result, expected)
}

func assert(t *testing.T, result map[string]string, expected map[string]string) {
	for key, value := range result {
		if val, ok := expected[key]; ok {
			if val != value {
				t.Errorf("Expected [%s] to be [%s], got [%s]", key, val, value)
			}
		} else {
			t.Errorf("Unexpected key %s", key)
		}
	}
}
