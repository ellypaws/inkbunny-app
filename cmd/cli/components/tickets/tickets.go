package tickets

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ellypaws/inkbunny-app/cmd/cli/apis"
	utils "github.com/ellypaws/inkbunny-app/cmd/cli/components"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/tickets/login"
)

type Model struct {
	login login.Model
}

const (
	hotPink     = lipgloss.Color("#FF06B7")
	pastelGreen = lipgloss.Color("#6A994E")
	darkGray    = lipgloss.Color("#767676")
)

var (
	pinkStyle    = lipgloss.NewStyle().Foreground(hotPink)
	successStyle = lipgloss.NewStyle().Foreground(pastelGreen)
	grayStyle    = lipgloss.NewStyle().Foreground(darkGray)
)

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	//switch msg := msg.(type) {
	//default:
	//	msg = msg
	//}
	m.login, cmd = utils.Propagate(m.login, msg)
	return m, cmd
}

func (m Model) View() string {
	if !m.login.LoggedIn() {
		return m.login.View()
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		successStyle.Render("Logged in"),
		pinkStyle.Render("Username: ")+grayStyle.Render(m.login.User().Username),
		pinkStyle.Render("Session ID: ")+grayStyle.Render(m.login.User().Sid),
	)
}

func New(config *apis.Config) Model {
	return Model{
		login: login.New(config),
	}
}
