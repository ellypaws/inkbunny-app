package sd

import (
	"fmt"
	"github.com/TheZoraiz/ascii-image-converter/aic_package"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	utils "github.com/ellypaws/inkbunny-app/cmd/cli/components"
	api "github.com/ellypaws/inkbunny-app/cmd/cli/requests"
	"github.com/ellypaws/inkbunny-sd/entities"
	zone "github.com/lrstanley/bubblezone"
	"os"
	"strings"
)

type Model struct {
	width     int
	height    int
	Config    *api.Config
	spinner   spinner.Model
	t2i       *entities.TextToImageResponse
	image     []byte
	progress  progress.Model
	threshold uint8
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

const (
	Start = "start"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.MouseMsg:
		if zone.Get(Start).InBounds(msg) {
			return StartGeneration(m)
		}
	case *entities.TextToImageResponse:
		ProcessImage(&m, msg)
	case *api.ProgressResponse:
		return m, m.progress.SetPercent(msg.Progress)
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			return StartGeneration(m)
		case tea.KeyUp.String():
			m.threshold += 1
		case tea.KeyDown.String():
			m.threshold -= 1
		}
	}

	return m.propagate(msg)
}

func (m Model) propagate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.(type) {
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
	case progress.FrameMsg:
		m.progress, cmd = utils.Propagate(m.progress, msg)
	default:
		return m, nil
	}
	return m, cmd
}

func (m Model) View() string {
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
		asciiArt := m.imageAscii()
		s.WriteString(lipgloss.JoinVertical(
			lipgloss.Center,
			utils.IF(len(asciiArt) == 0, "", fmt.Sprintf("Threshold: %d", m.threshold)),
			asciiArt,
		))
	}
	return s.String()
}

func New(config *api.Config) Model {
	return Model{
		Config:    config,
		spinner:   spinner.New(spinner.WithSpinner(spinner.Moon)),
		progress:  progress.New(progress.WithDefaultGradient()),
		threshold: 128 / 2,
	}
}

func StartGeneration(m Model) (tea.Model, tea.Cmd) {
	_ = m.Config.AddToQueue(&entities.TextToImageRequest{
		Prompt:      "A cat with rainbow background",
		Steps:       20,
		SamplerName: "DDIM",
	})
	return m, m.progress.SetPercent(0)
}

func ProcessImage(m *Model, response *entities.TextToImageResponse) {
	m.t2i = response
	images, err := api.ToImages(response)
	if err != nil {
		fmt.Println(err)
	}

	for _, img := range images {
		m.image = img
		break
	}
}

func (m Model) imageAscii() string {
	if m.image == nil {
		return ""
	}
	f, _ := os.CreateTemp("", "image_*.png")
	defer os.Remove(f.Name())

	_, _ = f.Write(m.image)

	flags := aic_package.DefaultFlags()

	size := api.ImageSize(m.image)
	scaled := api.Scale([2]int{m.width - 3, m.height - 3}, size)
	flags.Dimensions = []int{scaled[0] + 45, scaled[1]}

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
