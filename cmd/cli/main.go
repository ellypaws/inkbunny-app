package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	utils "github.com/ellypaws/inkbunny-app/cmd/cli/components"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/list"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/sd"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/tabs"
	api "github.com/ellypaws/inkbunny-app/cmd/cli/requests"
	zone "github.com/lrstanley/bubblezone"
	"log"
	"strings"
)

type model struct {
	width       int
	height      int
	activeIndex uint8
	sd          sd.Model
	submissions list.List
	tabs        tabs.Tabs
}

const (
	start       = sd.Start
	submissions = "submissions"
)

func (m model) Init() tea.Cmd {
	return m.sd.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}

		if zone.Get(submissions).InBounds(msg) {
			m.activeIndex = 1
		}
		return m.propagate(msg)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.activeIndex = 1
		}
		return m.propagate(msg)
	}
	return m.propagate(msg)
}

func (m model) propagate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.sd, cmd = utils.Propagate(m.sd, msg)
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

func safeDereference(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (m model) View() string {
	var s strings.Builder
	if m.sd.Config.IsProcessing {
		s.WriteString(m.sd.View())
	} else {
		s.WriteString(lipgloss.JoinVertical(
			lipgloss.Center,
			zone.Mark(start, "Press 's' to start processing"),
			zone.Mark(submissions, "Press '1' to view submissions"),
			m.sd.View(),
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
		sd:          sd.New(config),
		submissions: list.New(),
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
