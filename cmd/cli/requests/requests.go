package api

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/stable_diffusion"
	"net/url"
	"os"
)

type Config struct {
	Host         *sd.Host
	SID          string
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
	return &Config{
		Host:  h,
		SID:   os.Getenv("SID"),
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
	response, err := c.Host.TextToImageRequest(req.Request)
	if err != nil {
		req.Response <- nil
		return
	}
	req.Response <- response
	program.Send(<-req.Response)
}

func ToImages(response *entities.TextToImageResponse) ([][]byte, error) {
	return sd.ToImages(response)
}
