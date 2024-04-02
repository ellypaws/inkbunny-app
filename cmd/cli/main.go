package main

import (
	"fmt"
	"github.com/TheZoraiz/ascii-image-converter/aic_package"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	utils "github.com/ellypaws/inkbunny-app/cmd/cli/components"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/list"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/tabs"
	api "github.com/ellypaws/inkbunny-app/cmd/cli/requests"
	"github.com/ellypaws/inkbunny-sd/entities"
	zone "github.com/lrstanley/bubblezone"
	"log"
	"os"
	"strconv"
	"strings"
)

type model struct {
	width       int
	height      int
	config      *api.Config
	spinner     spinner.Model
	t2i         *entities.TextToImageResponse
	image       []byte
	progress    progress.Model
	activeIndex uint8
	threshold   uint8
	submissions list.List
	tabs        tabs.Tabs
}

const (
	start       = "start"
	submissions = "submissions"
)

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	//var cmd tea.Cmd
	switch msg := msg.(type) {
	//case spinner.TickMsg:
	//	m.spinner, cmd = m.spinner.Update(msg)
	//	return m, cmd
	//case progress.FrameMsg:
	//	return m.progress.Update(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}

		if zone.Get(start).InBounds(msg) {
			return startGeneration(m)
		}
		if zone.Get(submissions).InBounds(msg) {
			m.activeIndex = 1
		}
		return m.propagate(msg)
	case *entities.TextToImageResponse:
		processImage(&m, msg)
	case *api.ProgressResponse:
		return m, m.progress.SetPercent(msg.Progress)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "s":
			return startGeneration(m)
		case "1":
			m.activeIndex = 1
		case tea.KeyUp.String():
			m.threshold += 1
		case tea.KeyDown.String():
			m.threshold -= 1
		}
		return m.propagate(msg)
	}
	return m.propagate(msg)
}

func (m model) propagate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	m.progress, cmd = utils.Propagate(m.progress, msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	m.submissions, cmd = utils.Propagate(m.submissions, msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	m.tabs, cmd = utils.Propagate(m.tabs, msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func processImage(m *model, response *entities.TextToImageResponse) {
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

func startGeneration(m model) (tea.Model, tea.Cmd) {
	_ = m.config.AddToQueue(&entities.TextToImageRequest{
		Prompt:      "A cat with rainbow background",
		Steps:       20,
		SamplerName: "DDIM",
	})
	return m, m.progress.SetPercent(0)
}

func IF[T any](condition bool, a, b T) T {
	if condition {
		return a
	}
	return b
}

func safeDereference(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (m model) View() string {
	var s strings.Builder
	if m.config.IsProcessing {
		s.WriteString(
			lipgloss.JoinHorizontal(
				lipgloss.Center,
				m.spinner.View(),
				"	",
				m.progress.View(),
			))
	} else {
		asciiArt := m.processImage()
		s.WriteString(lipgloss.JoinVertical(
			lipgloss.Center,
			zone.Mark(start, "Press 's' to start processing"),
			zone.Mark(submissions, "Press '1' to view submissions"),
			IF(len(asciiArt) == 0, "", strconv.Itoa(int(m.threshold))),
			asciiArt,
		))
	}
	if m.activeIndex == 1 {
		s.WriteString(m.submissions.View())
	}
	return zone.Scan(lipgloss.JoinVertical(
		lipgloss.Center,
		m.tabs.View(), "",
		lipgloss.PlaceHorizontal(
			m.width, lipgloss.Center,
			lipgloss.PlaceVertical(
				m.height-6, lipgloss.Center,
				s.String(),
			)),
	))
}

func main() {
	config := api.New()
	model := model{
		config:      config,
		spinner:     spinner.New(spinner.WithSpinner(spinner.Moon)),
		progress:    progress.New(progress.WithDefaultGradient()),
		submissions: list.New(),
		threshold:   128 / 2,
		tabs: tabs.New([]string{
			"Tickets",
			"Submissions",
			"Audit",
			"Generation",
		}),
	}

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	zone.NewGlobal()
	defer zone.Close()
	go config.Run(p)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func (m model) processImage() string {
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
