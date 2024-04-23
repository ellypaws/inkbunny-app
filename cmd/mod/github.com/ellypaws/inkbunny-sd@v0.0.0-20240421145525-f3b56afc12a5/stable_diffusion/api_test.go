package sd

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/labstack/gommon/bytes"
	"os"
	"strings"
	"testing"
)

var h = func() *Host {
	if h := FromString(os.Getenv("URL")); h != nil {
		return h
	}
	return DefaultHost
}()

var slow = func() bool {
	if os.Getenv("SLOW") == "true" {
		return true
	}
	return false
}()

func TestHost_GetConfig(t *testing.T) {
	config, err := h.GetConfig()
	if err != nil {
		t.Errorf("Failed to get config: %v", err)
	}

	if config == nil {
		t.Errorf("Config is nil")
	}

	t.Logf("Config: %v", config)
}

func TestToImages(t *testing.T) {
	if !slow {
		t.Skip("Skipping image generation, set SLOW=true to enable")
	}
	request := &entities.TextToImageRequest{
		Prompt:      "A cat",
		Steps:       20,
		SamplerName: "DDIM",
	}
	response, err := h.TextToImageRequest(request)
	if err != nil {
		t.Errorf("Failed to get response: %v", err)
	}

	images, err := ToImages(response)
	if err != nil {
		t.Errorf("Failed to get images: %v", err)
	}

	if os.Getenv("SAVE_IMAGES") != "true" {
		return
	}
	for i, img := range images {
		if _, err := os.Stat("images"); os.IsNotExist(err) {
			os.Mkdir("images", os.ModePerm)
		}

		file, err := os.Create(fmt.Sprintf("images/%d.png", i))
		if err != nil {
			t.Errorf("Failed to create file: %v", err)
		}

		_, err = file.Write(img)
	}
}

//go:embed images/0.png
var image []byte

func TestHost_Interrogate(t *testing.T) {
	if _, err := os.Stat("images/0.png"); err != nil {
		s := slow
		slow = true
		t.Run("TestToImages", TestToImages)
		slow = s
	}
	if len(image) == 0 {
		t.Fatalf("Image is empty")
	}
	b64 := base64.StdEncoding.EncodeToString(image)
	req := (&entities.TaggerRequest{
		Image: &b64,
		Model: entities.TaggerZ3DE621Convnext,
	}).SetThreshold(0.5)
	response, err := h.Interrogate(req)
	if err != nil {
		t.Fatalf("Failed to interrogate: %v", err)
	}

	if response.Caption.Tag == nil {
		t.Fatalf("Caption is nil")
	}

	var foundFeline bool
	for tag, confidence := range response.Captions() {
		t.Logf("Tag: %v, Confidence: %.2f", tag, confidence)
		if tag == "felid" {
			foundFeline = true
		}
	}

	if !foundFeline {
		t.Errorf("Failed to find felid")
	}
}

func TestHost_GetCheckpoints(t *testing.T) {
	checkpoints, err := h.GetCheckpoints()
	if err != nil {
		t.Errorf("Failed to get checkpoints: %v", err)
	}

	if len(checkpoints) == 0 {
		t.Errorf("Checkpoints are empty")
	}

	t.Logf("Checkpoints: %v", checkpoints)
}

func TestGetCheckpointHash(t *testing.T) {
	checkpoints, err := h.GetCheckpoints()
	if err != nil {
		t.Errorf("Failed to get checkpoints: %v", err)
	}

	if len(checkpoints) == 0 {
		t.Errorf("Checkpoints are empty")
	}

	hash, err := GetCheckpointHash(checkpoints[0])
	if err != nil {
		t.Errorf("Failed to get checkpoint hash: %v", err)
	}

	if hash == "" {
		t.Errorf("Hash is empty")
	}

	t.Logf("Hash: %+v", hash)
}

func TestHost_GetLoras(t *testing.T) {
	loras, err := h.GetLoras()
	if err != nil {
		t.Errorf("Failed to get loras: %v", err)
	}

	if len(loras) == 0 {
		t.Errorf("Loras are empty")
	}

	t.Logf("Loras: %v", len(loras))

	for _, lora := range loras {
		if lora.Metadata.SshsModelHash == nil {
			t.Logf("Lora %s has no hash", lora.Name)

			if !strings.HasSuffix(lora.Path, ".safetensors") {
				continue
			}

			f, err := os.Stat(lora.Path)
			if err != nil {
				t.Errorf("File %v not found in %v: %v", lora.Name, lora.Path, err)
				continue
			}
			if f.Size() >= 128*bytes.MiB {
				continue
			}
			hash, err := CheckpointSafeTensorHash(lora.Path)
			if err != nil {
				t.Errorf("Failed to get hash: %v", err)
				continue
			}
			t.Logf("Calculated hash: %v", hash.SHA256)
		}
	}
}
