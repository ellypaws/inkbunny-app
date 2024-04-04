package sd

import (
	"encoding/json"
	"fmt"
	stick "github.com/76creates/stickers/flexbox"
	"github.com/TheZoraiz/ascii-image-converter/aic_package"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	utils "github.com/ellypaws/inkbunny-app/cmd/cli/components"
	"github.com/ellypaws/inkbunny-app/cmd/cli/entle"
	"github.com/ellypaws/inkbunny-sd/entities"
	sd "github.com/ellypaws/inkbunny-sd/stable_diffusion"
	zone "github.com/lrstanley/bubblezone"
	"os"
	"strings"
	"time"
)

type Model struct {
	width     int
	height    int
	spinner   *spinner.Model
	t2i       *entities.TextToImageResponse
	image     string
	progress  progress.Model
	threshold uint8
	cache     *string

	Config *Config
}

type Config struct {
	host         *sd.Host
	Queue        chan IO
	IsProcessing bool
}

type IO struct {
	Request  *entities.TextToImageRequest
	Response chan *entities.TextToImageResponse
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

const (
	ButtonStartGeneration = "start"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = entle.Width()
		m.height = entle.Height()
	case tea.MouseMsg:
		if zone.Get(ButtonStartGeneration).InBounds(msg) {
			return StartGeneration(m)
		}
	case *entities.TextToImageResponse:
		ProcessImage(&m, msg)
		return m, utils.ForceRender()
	case *ProgressResponse:
		var cmd tea.Cmd
		*m.spinner, cmd = m.spinner.Update(m.spinner.Tick())
		return m, tea.Batch(m.progress.SetPercent(msg.Progress), utils.ForceRender(), cmd)
	case tea.KeyMsg:
		var cmd tea.Cmd
		switch msg.String() {
		case "s":
			return StartGeneration(m)
		case tea.KeyUp.String():
			m.threshold += 1
			if m.t2i != nil {
				ProcessImage(&m, m.t2i)
				cmd = utils.ForceRender()
			}
		case tea.KeyDown.String():
			m.threshold -= 1
			if m.t2i != nil {
				ProcessImage(&m, m.t2i)
				cmd = utils.ForceRender()
			}
		}
		return m, cmd
	}

	return m.propagate(msg)
}

func (m Model) propagate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.(type) {
	case spinner.TickMsg:
		*m.spinner, cmd = m.spinner.Update(msg)
		cmd = tea.Batch(cmd, utils.ForceRender())
	case progress.FrameMsg:
		m.progress, cmd = utils.Propagate(m.progress, msg)
		cmd = tea.Batch(cmd, utils.ForceRender())
	default:
		return m, nil
	}
	return m, cmd
}

func (m Model) View() string {
	if m.cache != nil {
		return *m.cache
	}
	var s strings.Builder
	if m.Config.IsProcessing {
		s.WriteString(
			lipgloss.JoinHorizontal(
				lipgloss.Center,
				m.spinner.View(),
				"	",
				m.progress.View(),
			))
	} else {
		s.WriteString(m.render())
	}
	return s.String()
}

func (m Model) Render(s entle.Screen) func() string {
	return func() string {
		sdContent := stick.New(s.Width, s.Height)
		sdContent.SetRows(
			[]*stick.Row{
				sdContent.NewRow().AddCells(
					stick.NewCell(1, 1).SetContent(zone.Mark(ButtonStartGeneration, "Press 's' to start processing")),
				),
				sdContent.NewRow().AddCells(
					stick.NewCell(1, 12).SetContent(m.View()),
				),
			})
		return sdContent.Render()
	}
}

func (m Model) render() string {
	var s strings.Builder
	s.WriteString(lipgloss.JoinVertical(
		lipgloss.Center,
		utils.IF(len(m.image) == 0, "", fmt.Sprintf("Threshold: %d", m.threshold)),
		m.image,
		utils.IF(len(m.image) == 0, "", fmt.Sprintf("Threshold: %d", m.threshold)),
	))
	return s.String()
}

func New(host *sd.Host, spinner *spinner.Model) Model {
	return Model{
		spinner:   spinner,
		progress:  progress.New(progress.WithDefaultGradient()),
		threshold: 128 / 2,

		Config: &Config{
			host:  host,
			Queue: make(chan IO),
		},
	}
}

func StartGeneration(m Model) (tea.Model, tea.Cmd) {
	m.cache = nil
	_ = m.Config.AddToQueue(&entities.TextToImageRequest{
		Prompt:      "A cat with rainbow background",
		Steps:       20,
		SamplerName: "DDIM",
	})
	return m, tea.Batch(m.progress.SetPercent(0), utils.ForceRender())
}

func ProcessImage(m *Model, response *entities.TextToImageResponse) {
	m.t2i = response
	images, err := utils.ToImages(response)
	if err != nil {
		fmt.Println(err)
	}

	for _, img := range images {
		m.image = m.imageAscii(img)
		m.cache = nil
		v := m.View()
		m.cache = &v
		break
	}
}

func (m Model) imageAscii(image []byte) string {
	if image == nil {
		return ""
	}
	f, err := os.CreateTemp("", "image_*.png")
	if err != nil {
		return ""
	}
	//defer os.Remove(f.Name())

	_, _ = f.Write(image)

	flags := aic_package.DefaultFlags()

	size := utils.ImageSize(image)
	scaled := utils.Scale([2]int{m.width, m.height}, size)
	flags.Dimensions = []int{max(5, int(float64(scaled[0])*1.35)), max(5, int(float64(scaled[1])*0.65))}

	flags.Colored = true
	flags.Braille = true
	flags.Threshold = int(m.threshold)
	//flags.Dither = true

	// Conversion for an image
	asciiArt, err := aic_package.Convert(f.Name(), flags)
	if err != nil {
		tea.Println(err)
	}
	_ = f.Close()
	return asciiArt
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
	if c.host == nil {
		req.Response <- nil
		return
	}
	go updateProgress(c, program, req.Response)
	response, err := c.host.TextToImageRequest(req.Request)
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
			p, err := GetCurrentProgress(c.host)
			if err == nil {
				program.Send(p)
			}
		}
	}
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
