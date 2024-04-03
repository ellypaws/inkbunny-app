package main

import (
	stick "github.com/76creates/stickers/flexbox"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	utils "github.com/ellypaws/inkbunny-app/cmd/cli/components"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/list"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/sd"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/tabs"
	"github.com/ellypaws/inkbunny-app/cmd/cli/entle"
	api "github.com/ellypaws/inkbunny-app/cmd/cli/requests"
	zone "github.com/lrstanley/bubblezone"
	"log"
	"time"
)

type model struct {
	activeIndex uint8
	window      struct {
		width  int
		height int
	}
	sd          sd.Model
	submissions list.List
	tabs        tabs.Tabs
	flexbox     *stick.FlexBox

	viewport viewport.Model
}

const (
	start       = sd.Start
	submissions = "submissions"

	RESIZE_TICK = 250
)

func (m model) Init() tea.Cmd {
	return tea.Batch(resizeTick(fastTick), m.sd.Init())
}

const fastTick = RESIZE_TICK * time.Millisecond
const slowTick = fastTick * 2

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
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}

		if zone.Get(submissions).InBounds(msg) {
			m.activeIndex = 1
		}
		return m.propagate(msg, nil)
	case tea.WindowSizeMsg:
		if m.window.width != msg.Width || m.window.height != msg.Height {
			m.window.width = msg.Width
			m.window.height = msg.Height
			cmd = resizeTick(fastTick)
			return m.propagate(msg, cmd)
		}
		cmd = resizeTick(slowTick)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.submissions.Active = !m.submissions.Active
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
	top := stick.New(m.window.width, 3)
	top.SetRows(
		[]*stick.Row{top.NewRow().AddCells(
			stick.NewCell(1, 1).SetContent(m.tabs.View()),
		)})

	sdContent := stick.New(m.window.width-2, m.window.height-top.GetHeight())
	sdContent.SetRows(
		[]*stick.Row{
			sdContent.NewRow().AddCells(
				stick.NewCell(1, 1).SetContent(zone.Mark(start, "Press 's' to start processing")),
			),
			sdContent.NewRow().AddCells(
				stick.NewCell(1, 12).SetContent(m.sd.View()),
			),
		})

	submissionList := stick.New(m.window.width-2, m.window.height-top.GetHeight())
	submissionList.SetRows(
		[]*stick.Row{submissionList.NewRow().AddCells(
			stick.NewCell(1, 1).SetContent(zone.Mark(submissions, "Press '1' to view submissions")),
			stick.NewCell(3, 1).SetContent(m.submissions.View()),
		)})

	content := stick.New(m.window.width-2, m.window.height-top.GetHeight()-2)
	content.SetRows(
		[]*stick.Row{content.NewRow().AddCells(
			stick.NewCell(1, 1).SetContent(submissionList.Render()),
			stick.NewCell(3, 1).SetContent(sdContent.Render()),
		)})

	if m.activeIndex == 1 {
		//s.WriteString(m.submissions.View())
	}
	return zone.Scan(lipgloss.JoinVertical(
		lipgloss.Left,
		top.Render(), "",
		lipgloss.PlaceHorizontal(
			m.window.width, lipgloss.Center,
			lipgloss.PlaceVertical(
				m.window.height-6, lipgloss.Top,
				content.Render(),
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

	model.window.width = entle.Width()
	model.window.height = entle.Height()

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	zone.NewGlobal()
	defer zone.Close()
	go config.Run(p)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
