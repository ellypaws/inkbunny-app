package main

import (
	"fmt"
	"github.com/TheZoraiz/ascii-image-converter/aic_package"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	api "github.com/ellypaws/inkbunny-app/cmd/cli/requests"
	"github.com/ellypaws/inkbunny-sd/entities"
	zone "github.com/lrstanley/bubblezone"
	"log"
	"os"
	"strings"
)

type model struct {
	width       int
	height      int
	config      *api.Config
	spinner     spinner.Model
	t2i         *entities.TextToImageResponse
	image       *string
	progress    progress.Model
	activeIndex uint8
	//submissions subModel
}

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

		if zone.Get("start").InBounds(msg) {
			_ = m.config.AddToQueue(&entities.TextToImageRequest{
				Prompt:      "A cat with rainbow background",
				Steps:       20,
				SamplerName: "DDIM",
			})
			return m, m.progress.SetPercent(0)
		}
		return m, nil
	case *entities.TextToImageResponse:
		m.t2i = msg
		images, err := api.ToImages(msg)
		if err != nil {
			fmt.Println(err)
		}

		for _, img := range images {
			f, _ := os.CreateTemp("", "image_*.png")
			defer os.Remove(f.Name())

			_, _ = f.Write(img)

			flags := aic_package.DefaultFlags()

			flags.Dimensions = []int{50, 25}
			flags.Colored = true
			flags.Braille = true

			// Conversion for an image
			asciiArt, err := aic_package.Convert(f.Name(), flags)
			if err != nil {
				tea.Println(err)
			}
			_ = f.Close()

			m.image = &asciiArt
		}
	case *api.ProgressResponse:
		return m, m.progress.SetPercent(msg.Progress)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "s":
			_ = m.config.AddToQueue(&entities.TextToImageRequest{
				Prompt:      "A cat with rainbow background",
				Steps:       20,
				SamplerName: "DDIM",
			})
			return m, m.progress.SetPercent(0)
		}
	}
	//return m, nil
	return m.propagate(msg)
}

func (m model) propagate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	model, cmd := m.progress.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	m.progress = model.(progress.Model)
	return m, tea.Batch(cmds...)
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
		s.WriteString(lipgloss.JoinVertical(
			lipgloss.Center,
			zone.Mark("start", "Press 's' to start processing"),
			safeDereference(m.image),
		))
	}
	//if m.activeIndex == 1 {
	//	s.WriteString(m.submissions.View())
	//}
	return zone.Scan(lipgloss.PlaceHorizontal(
		m.width, lipgloss.Center,
		lipgloss.PlaceVertical(
			m.height, lipgloss.Center,
			s.String(),
		)),
	)
}

func main() {

	config := api.New()
	model := model{
		config:   config,
		spinner:  spinner.New(spinner.WithSpinner(spinner.Moon)),
		progress: progress.New(progress.WithDefaultGradient()),
	}

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	zone.NewGlobal()
	defer zone.Close()
	go config.Run(p)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
