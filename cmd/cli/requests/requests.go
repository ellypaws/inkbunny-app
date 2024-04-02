package api

import (
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/stable_diffusion"
	"net/url"
	"os"
)

type Config struct {
	Host  *sd.Host
	SID   string
	Queue chan IO
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
	response := make(chan *entities.TextToImageResponse)
	c.Queue <- IO{Request: req, Response: response}
	return response
}

func (c *Config) Run() {
	for {
		select {
		case req := <-c.Queue:
			processRequest(c, req)
		}
	}
}

func processRequest(c *Config, req IO) {
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
}

func ToImages(response *entities.TextToImageResponse) ([][]byte, error) {
	return sd.ToImages(response)
}
