package apis

import (
	"net/url"
	"os"

	"github.com/ellypaws/inkbunny/api"

	"github.com/ellypaws/inkbunny-app/pkg/api/cache"
	"github.com/ellypaws/inkbunny-sd/stable_diffusion"
)

type Config struct {
	SD   *sd.Host
	LLM  *url.URL
	API  *url.URL
	user *api.Credentials
}

func New() *Config {
	cache.Init()
	var sdURL *sd.Host
	if env := os.Getenv("SD_URL"); env != "" {
		u, err := url.Parse(env)
		if err != nil {
			sdURL = sd.DefaultHost
		}
		sdURL = (*sd.Host)(u)
	}
	var llmURL *url.URL
	if llm := os.Getenv("LLM_URL"); llm != "" {
		u, err := url.Parse(llm)
		if err != nil {
			return nil
		}
		llmURL = u
	}
	var apiURL *url.URL
	if apiString := os.Getenv("API_URL"); apiString != "" {
		u, err := url.Parse(apiString)
		if err != nil {
			return nil
		}
		apiURL = u
	}

	var user *api.Credentials
	if sid := os.Getenv("SID"); sid != "" {
		user = &api.Credentials{Sid: sid}
	}
	if username := os.Getenv("USERNAME"); username != "" {
		if user == nil {
			user = &api.Credentials{Username: username}
		}
		user.Username = username
	}
	return &Config{
		SD:   sdURL,
		LLM:  llmURL,
		API:  apiURL,
		user: user,
	}
}

func (c *Config) SetUser(user *api.Credentials) {
	c.user = user
}

func (c *Config) User() *api.Credentials {
	return c.user
}

func (c *Config) SetSD(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return err
	}
	c.SD = (*sd.Host)(u)
	return nil
}

func (c *Config) SetLLM(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return err
	}
	c.LLM = u
	return nil
}

func (c *Config) SetAPI(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return err
	}
	c.API = u
	return nil
}
