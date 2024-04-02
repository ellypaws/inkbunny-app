package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	api "github.com/ellypaws/inkbunny-app/cmd/cli/requests"
	"github.com/ellypaws/inkbunny-sd/entities"
	"log"
	"os"
	"strings"
)

type model struct {
	width   int
	height  int
	config  *api.Config
	spinner spinner.Model
	program *tea.Program
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	//var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case *entities.TextToImageResponse:
		images, err := api.ToImages(msg)
		if err != nil {
			fmt.Println(err)
		}

		for i, img := range images {
			_ = os.WriteFile(fmt.Sprintf("image_%d.png", i), img, 0644)
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			_ = m.config.AddToQueue(&entities.TextToImageRequest{
				Prompt: "A cat",
				Steps:  20,
			})
		}
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
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var s strings.Builder
	if m.config.IsProcessing {
		s.WriteString(m.spinner.View())
	} else {
		s.WriteString("Press 's' to start processing")
	}
	return s.String()
}

func main() {
	config := api.New()
	model := model{
		config:  config,
		spinner: spinner.New(),
	}

	p := tea.NewProgram(model)

	go config.Run(p)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
