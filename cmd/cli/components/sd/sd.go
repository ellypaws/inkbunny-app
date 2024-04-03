package sd

import (
	"fmt"
	stick "github.com/76creates/stickers/flexbox"
	"github.com/TheZoraiz/ascii-image-converter/aic_package"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	utils "github.com/ellypaws/inkbunny-app/cmd/cli/components"
	"github.com/ellypaws/inkbunny-app/cmd/cli/entle"
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
	image     string
	progress  progress.Model
	threshold uint8
	cache     *string
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
		m.width = entle.Width()
		m.height = entle.Height()
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
					stick.NewCell(1, 1).SetContent(zone.Mark(Start, "Press 's' to start processing")),
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

func New(config *api.Config) Model {
	return Model{
		Config:    config,
		spinner:   spinner.New(spinner.WithSpinner(spinner.Moon)),
		progress:  progress.New(progress.WithDefaultGradient()),
		threshold: 128 / 2,
	}
}

func StartGeneration(m Model) (tea.Model, tea.Cmd) {
	m.cache = nil
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

	size := api.ImageSize(image)
	scaled := api.Scale([2]int{m.width, m.height}, size)
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
