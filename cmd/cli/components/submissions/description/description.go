package description

import (
	"cli/entle"
	"fmt"
	stick "github.com/76creates/stickers/flexbox"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ellypaws/inkbunny/api"
)

const (
	hotPink     = lipgloss.Color("#FF06B7")
	pastelGreen = lipgloss.Color("#6A994E")
	darkGray    = lipgloss.Color("#767676")
)

var (
	pink  = lipgloss.NewStyle().Foreground(hotPink)
	green = lipgloss.NewStyle().Foreground(pastelGreen)
	gray  = lipgloss.NewStyle().Foreground(darkGray)

	normalStyle = lipgloss.NewStyle()

	titleStyle = pink.Bold(true)

	bodyStyle = normalStyle

	buttonStyle = green.Border(lipgloss.RoundedBorder()).Padding(0, 1)
)

type Model struct {
	submissions map[string]api.Submission

	Active  string
	loading bool
	spinner spinner.Model
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case []api.Submission:
		m.loading = false
		if m.submissions == nil {
			m.submissions = make(map[string]api.Submission)
		}
		for _, submission := range msg {
			m.submissions[submission.SubmissionID] = submission
		}
	case api.Submission:
		m.loading = false
		if m.submissions == nil {
			m.submissions = make(map[string]api.Submission)
		}
		m.submissions[msg.SubmissionID] = msg
	}
	if m.loading {
		if _, ok := msg.(spinner.TickMsg); ok {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(m.spinner.Tick())
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.loading {
		return m.spinner.View()
	}
	return m.Render(entle.Screen{
		Width: entle.Width(), Height: entle.Height(),
	})
}

func (m Model) Render(s entle.Screen) string {
	if m.loading {
		return lipgloss.Place(s.Width-100, s.Height, lipgloss.Center, lipgloss.Center, m.spinner.View())
	}
	sub, ok := m.submissions[m.Active]
	if !ok {
		return lipgloss.Place(s.Width-100, s.Height, lipgloss.Center, lipgloss.Center, "select a submission")
	}
	title := fmt.Sprintf("%s by %s", titleStyle.Render(sub.Title), sub.Username)
	url := gray.Render(fmt.Sprintf("https://inkbunny.net/s/%s", sub.SubmissionID))

	body := bodyStyle.Render(sub.Description)

	top := stick.NewCell(1, 1).SetContent(lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		sub.UpdateDateUser,
		url))

	content := stick.New(s.Width-100, s.Height)
	content.SetRows(
		[]*stick.Row{
			content.NewRow().AddCells(
				top,
			),
			content.NewRow().AddCells(
				stick.NewCell(1, 6).SetContent(sub.Files[0].FileURLFull),
			),
			content.NewRow().AddCells(
				stick.NewCell(1, 3).SetContent(body),
			),
			content.NewRow().AddCells(
				stick.NewCell(1, 1).SetContent(buttonStyle.Render("Reply...")),
				stick.NewCell(1, 1).SetContent(buttonStyle.Render("Mute this thread")),
			),
		})

	return content.Render()
}

var basic = api.Submission{
	SubmissionBasic: api.SubmissionBasic{
		Title:          "Title",
		Username:       "Username",
		SubmissionID:   "123456",
		UpdateDateUser: "2021-01-01 12:00:00",
	},
	Description: "Description",
}

func New() Model {
	return Model{
		submissions: nil,
		loading:     true,
		spinner:     spinner.New(spinner.WithSpinner(spinner.Moon)),
	}
}
