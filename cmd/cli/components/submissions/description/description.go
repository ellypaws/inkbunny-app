package description

import (
	"cli/entle"
	"fmt"
	stick "github.com/76creates/stickers/flexbox"
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
	api.Submission
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() string {
	return m.Render(entle.Screen{
		Width: entle.Width(), Height: entle.Height(),
	})
}

func (m Model) Render(s entle.Screen) string {
	title := fmt.Sprintf("%s by %s", titleStyle.Render(m.Title), m.Username)
	url := gray.Render(fmt.Sprintf("https://inkbunny.net/s/%s", m.SubmissionID))

	body := bodyStyle.Render(m.Description)

	top := stick.NewCell(1, 1).SetContent(lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		m.UpdateDateUser,
		url))

	content := stick.New(s.Width-100, s.Height)
	content.SetRows(
		[]*stick.Row{
			content.NewRow().AddCells(
				top,
			),
			content.NewRow().AddCells(
				stick.NewCell(1, 6).SetContent("image"),
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

func New() Model {
	return Model{
		Submission: api.Submission{
			SubmissionBasic: api.SubmissionBasic{
				Title:          "Title",
				Username:       "Username",
				SubmissionID:   "123456",
				UpdateDateUser: "2021-01-01 12:00:00",
			},
			Description: "Description",
		},
	}
}
