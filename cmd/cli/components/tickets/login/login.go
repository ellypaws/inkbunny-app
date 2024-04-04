package login

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ellypaws/inkbunny/api"
	"strings"
)

type Model struct {
	user *api.Credentials

	inputs  []textinput.Model
	focused int
	err     error
}

func (m Model) LoggedIn() bool {
	return m.user != nil && m.user.Sid != ""
}

func (m Model) User() *api.Credentials {
	return m.user
}

type (
	errMsg error
)

const (
	username = iota
	password
)

const (
	hotPink  = lipgloss.Color("#FF06B7")
	darkGray = lipgloss.Color("#767676")
)

var (
	inputStyle    = lipgloss.NewStyle().Foreground(hotPink)
	continueStyle = lipgloss.NewStyle().Foreground(darkGray)
)

// Validator functions to ensure valid input
func textValidator(s string) error {
	if len(s) == 0 {
		return nil
	}
	if strings.ContainsAny(s, " \t\n") {
		return fmt.Errorf("no spaces allowed")
	}
	return nil
}

func New() Model {
	var inputs []textinput.Model = make([]textinput.Model, 2)
	inputs[username] = textinput.New()
	inputs[username].Placeholder = "guest"
	inputs[username].Focus()
	inputs[username].CharLimit = 22
	inputs[username].Width = 32
	inputs[username].Prompt = ""
	inputs[username].Validate = textValidator

	inputs[password] = textinput.New()
	inputs[password].Placeholder = "password"
	inputs[password].CharLimit = 32
	inputs[password].Width = 32
	inputs[password].Prompt = ""
	inputs[password].EchoMode = textinput.EchoPassword
	inputs[password].EchoCharacter = '•'

	return Model{
		inputs:  inputs,
		focused: 0,
		err:     nil,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd = make([]tea.Cmd, len(m.inputs))

	switch msg := msg.(type) {
	case *api.Credentials:
		m.user = msg
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.focused == len(m.inputs)-1 {
				return m, m.login()
			}
			m.nextInput()
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyShiftTab, tea.KeyCtrlP:
			m.prevInput()
		case tea.KeyTab, tea.KeyCtrlN:
			m.nextInput()
		default:

		}
		for i := range m.inputs {
			m.inputs[i].Blur()
		}
		m.inputs[m.focused].Focus()
	case errMsg:
		m.err = msg
		return m, nil
	}

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return fmt.Sprintf(
		` Login

 %s
 %s

 %s
 %s

 %s
`,
		inputStyle.Width(32).Render("Username"),
		m.inputs[username].View(),
		inputStyle.Width(32).Render("Password"),
		m.inputs[password].View(),
		continueStyle.Render("Continue ->"),
	) + "\n"
}

// nextInput focuses the next input field
func (m *Model) nextInput() {
	m.focused = (m.focused + 1) % len(m.inputs)
}

// prevInput focuses the previous input field
func (m *Model) prevInput() {
	m.focused--
	// Wrap around
	if m.focused < 0 {
		m.focused = len(m.inputs) - 1
	}
}

func (m Model) login() tea.Cmd {
	return func() tea.Msg {
		m.user = &api.Credentials{
			Username: m.inputs[username].Value(),
			Password: m.inputs[password].Value(),
		}

		for i := range m.inputs {
			m.inputs[i].SetValue("")
		}

		var err error
		m.user, err = m.user.Login()
		if err != nil {
			return errMsg(err)
		}

		if m.user.Sid == "" {
			return errMsg(fmt.Errorf("login failed, sid still empty"))
		}

		return m.user
	}
}
