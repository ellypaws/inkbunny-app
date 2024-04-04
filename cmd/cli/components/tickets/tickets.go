package tickets

import (
	tea "github.com/charmbracelet/bubbletea"
	utils "github.com/ellypaws/inkbunny-app/cmd/cli/components"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/tickets/login"
)

type Model struct {
	login login.Model
}

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
	return "Logged in"
}

func New() Model {
	return Model{
		login: login.New(),
	}
}
