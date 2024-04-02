package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	api "github.com/ellypaws/inkbunny-app/cmd/cli/requests"
	"github.com/ellypaws/inkbunny-sd/entities"
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
	case *entities.TextToImageResponse:
		m.t2i = msg
		images, err := api.ToImages(msg)
		if err != nil {
			fmt.Println(err)
		}

		for i, img := range images {
			_ = os.WriteFile(fmt.Sprintf("image_%d.png", i), img, 0644)
		}
	case *api.ProgressResponse:
		return m, m.progress.SetPercent(msg.Progress)
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "s":
			_ = m.config.AddToQueue(&entities.TextToImageRequest{
				Prompt: "A cat",
				Steps:  50,
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

func (m model) View() string {
	var s strings.Builder
	if m.config.IsProcessing {
		s.WriteString(
			lipgloss.JoinHorizontal(
				lipgloss.Center,
				m.spinner.View(),
				m.progress.View(),
			))
	} else {
		s.WriteString("Press 's' to start processing")
	}
	//if m.activeIndex == 1 {
	//	s.WriteString(m.submissions.View())
	//}
	return lipgloss.PlaceVertical(
		m.height,
		lipgloss.Center,
		s.String(),
	)
}

func main() {
	config := api.New()
	model := model{
		config:   config,
		spinner:  spinner.New(),
		progress: progress.New(progress.WithDefaultGradient()),
	}

	p := tea.NewProgram(model)

	go config.Run(p)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
