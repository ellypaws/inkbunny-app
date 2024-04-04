package main

import (
	"fmt"
	stick "github.com/76creates/stickers/flexbox"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ellypaws/inkbunny-app/cmd/cli/apis"
	utils "github.com/ellypaws/inkbunny-app/cmd/cli/components"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/list"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/sd"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/settings"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/tabs"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/tickets"
	"github.com/ellypaws/inkbunny-app/cmd/cli/entle"
	zone "github.com/lrstanley/bubblezone"
	"log"
	"time"
)

type model struct {
	window      entle.Screen
	sd          sd.Model
	tickets     tickets.Model
	submissions list.List
	settings    settings.Model
	tabs        tabs.Tabs
	flexbox     *stick.FlexBox

	config *apis.Config

	viewport viewport.Model
	render   *string
}

// Zone names
const (
	buttonStart = sd.ButtonStartGeneration
	buttonView  = list.ButtonViewSubmissions
)

const (
	RESIZE_TICK = 250

	fastTick = RESIZE_TICK * time.Millisecond
	slowTick = fastTick * 4
)

func (m model) Init() tea.Cmd {
	return tea.Batch(resizeTick(fastTick), m.sd.Init())
}

func resizeTick(duration time.Duration) tea.Cmd {
	return tea.Tick(duration, func(time.Time) tea.Msg {
		return tea.WindowSizeMsg{
			Width:  entle.Width(),
			Height: entle.Height(),
		}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case utils.RerenderMsg:
		m.render = nil
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}

		return m.propagate(msg, nil)
	case tea.WindowSizeMsg:
		if m.window.Width != msg.Width || m.window.Height != msg.Height {
			m.window.Width = msg.Width
			m.window.Height = msg.Height
			cmd = resizeTick(fastTick)
			m.render = nil
			return m.propagate(msg, cmd)
		}
		cmd = resizeTick(slowTick)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case tea.KeyCtrlA.String():
			utils.AlwaysRender = !utils.AlwaysRender
			return m, utils.ForceRender()
		}
		return m.propagate(msg, nil)
	}
	return m.propagate(msg, cmd)
}

func (m model) propagate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	switch m.tabs.Index() {
	case 0:
		m.submissions, cmd = utils.Propagate(m.submissions, msg)
	case 1:
		m.tickets, cmd = utils.Propagate(m.tickets, msg)
	//case 2:
	case 3:
		m.sd, cmd = utils.Propagate(m.sd, msg)
	case 4:
		m.settings, cmd = utils.Propagate(m.settings, msg)
	default:
		cmd = nil
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	m.tabs, cmd = utils.Propagate(m.tabs, msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if !utils.AlwaysRender && cmds == nil {
		r := m.Render()
		m.render = &r
		return m, nil
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
	if !utils.AlwaysRender && m.render != nil {
		return *m.render
	}
	return m.Render()
}

func (m model) Render() string {
	top := stick.New(m.window.Width, 3)
	top.SetRows(
		[]*stick.Row{top.NewRow().AddCells(
			stick.NewCell(1, 1).SetContent(m.tabs.View()),
		)})

	s := entle.Screen{
		Width:  m.window.Width - 2,
		Height: m.window.Height - top.GetHeight() - 2,
	}

	var renderers = []Renderer{
		m.submissions.Render(s),
		m.tickets.View,
		empty,
		m.sd.Render(s),
		m.settings.View,
	}

	return zone.Scan(lipgloss.JoinVertical(
		lipgloss.Center,
		top.Render(), fmt.Sprintf("Always re-render: %v", utils.AlwaysRender),
		lipgloss.PlaceHorizontal(
			m.window.Width, lipgloss.Center,
			lipgloss.PlaceVertical(
				m.window.Height-6, lipgloss.Top,
				renderers[m.tabs.Index()](),
			)),
	))
}

type Renderer = func() string

func render(i uint8, renderers []Renderer) string {
	return renderers[i]()
}

func empty() string {
	return "empty"
}

func main() {
	config := apis.New()
	stable := sd.New(config.SD)
	m := model{
		sd:          stable,
		submissions: list.New(),
		tabs: tabs.New([]string{
			"Submissions",
			"Tickets",
			"Audit",
			"Generation",
			"Settings",
		}),
		tickets:  tickets.New(config.User()),
		settings: settings.New(config),
	}

	m.window.Width = entle.Width()
	m.window.Height = entle.Height()

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	zone.NewGlobal()
	defer zone.Close()
	go stable.Config.Run(p)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
