package api

import (
	"bytes"
	"encoding/json"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/stable_diffusion"
	"image"
	"net/url"
	"os"
	"time"
)

type Config struct {
	Host         *sd.Host
	SID          *string
	Queue        chan IO
	IsProcessing bool
}

type IO struct {
	Request  *entities.TextToImageRequest
	Response chan *entities.TextToImageResponse
}

func New() *Config {
	var h *sd.Host
	if env := os.Getenv("SD_URL"); env != "" {
		u, err := url.Parse(env)
		if err != nil {
			h = sd.DefaultHost
		}
		h = (*sd.Host)(u)
	}
	sid := os.Getenv("SD_SID")
	var session *string
	if sid != "" {
		session = &sid
	}
	return &Config{
		Host:  h,
		SID:   session,
		Queue: make(chan IO),
	}
}

func (c *Config) AddToQueue(req *entities.TextToImageRequest) <-chan *entities.TextToImageResponse {
	response := make(chan *entities.TextToImageResponse, 1)
	c.Queue <- IO{Request: req, Response: response}
	return response
}

func (c *Config) Run(program *tea.Program) {
	for {
		select {
		case req := <-c.Queue:
			c.IsProcessing = true
			processRequest(c, req, program)
			if len(c.Queue) == 0 {
				c.IsProcessing = false
			}
		}
	}
}

func processRequest(c *Config, req IO, program *tea.Program) {
	if c == nil || req.Request == nil || req.Response == nil {
		return
	}
	if c.Host == nil {
		req.Response <- nil
		return
	}
	go updateProgress(c, program, req.Response)
	response, err := c.Host.TextToImageRequest(req.Request)
	if err != nil {
		req.Response <- nil
		return
	}
	req.Response <- response
}

func updateProgress(c *Config, program *tea.Program, response chan *entities.TextToImageResponse) {
	for {
		select {
		case r := <-response:
			program.Send(r)
			return
		case <-time.After(1 * time.Second):
			progress, err := GetCurrentProgress(c.Host)
			if err == nil {
				program.Send(progress)
			}
		}
	}
}

func ToImages(response *entities.TextToImageResponse) ([][]byte, error) {
	return sd.ToImages(response)
}

func ImageSize(b []byte) [2]int {
	if len(b) == 0 {
		return [2]int{-1, -1}
	}

	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return [2]int{-1, -1}
	}

	boundSize := img.Bounds().Size()
	return [2]int{boundSize.X, boundSize.Y}
}

func Scale(max, dimensions [2]int) [2]int {
	var maxW = max[0]
	var maxH = max[1]

	originalRatio := float64(dimensions[0]) / float64(dimensions[1])
	maxRatio := float64(maxW) / float64(maxH)

	if originalRatio > maxRatio {
		dimensions[0] = maxW
		dimensions[1] = int(float64(maxW) / originalRatio)
	} else {
		dimensions[1] = maxH
		dimensions[0] = int(float64(maxH) * originalRatio)
	}

	if dimensions[0] > maxW {
		dimensions[0] = maxW
		dimensions[1] = int(float64(maxW) / originalRatio)
	}
	if dimensions[1] > maxH {
		dimensions[1] = maxH
		dimensions[0] = int(float64(maxH) * originalRatio)
	}

	return dimensions
}

type ProgressResponse struct {
	Progress    float64 `json:"progress"`
	EtaRelative float64 `json:"eta_relative"`
}

func GetCurrentProgress(h *sd.Host) (*ProgressResponse, error) {
	const path = "/sdapi/v1/progress"
	body, err := h.GET(path)
	if err != nil {
		return nil, err
	}
	respStruct := &ProgressResponse{}
	err = json.Unmarshal(body, respStruct)
	if err != nil {
		return nil, err
	}
	return respStruct, nil
}
