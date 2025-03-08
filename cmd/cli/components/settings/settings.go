package settings

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ellypaws/inkbunny-app/cmd/cli/apis"
)

type Model struct {
	config *apis.Config

	inputs  []textinput.Model
	focused int
	err     error
}

type (
	errMsg error
)

const inputs = 3

const (
	sdUrl = iota
	llmUrl
	apiUrl
)

const (
	hotPink     = lipgloss.Color("#FF06B7")
	pastelGreen = lipgloss.Color("#6A994E")
	darkGray    = lipgloss.Color("#767676")
)

var (
	inputStyle    = lipgloss.NewStyle().Foreground(hotPink)
	successStyle  = lipgloss.NewStyle().Foreground(pastelGreen)
	continueStyle = lipgloss.NewStyle().Foreground(darkGray)

	normalStyle = lipgloss.NewStyle()
)

func New(c *apis.Config) Model {
	var inputs []textinput.Model = make([]textinput.Model, inputs)

	for i := 0; i < len(inputs); i++ {
		switch i {
		case sdUrl:
			inputs[sdUrl] = textinput.New()
			inputs[sdUrl].Placeholder = "http://localhost:7860"
			inputs[sdUrl].CharLimit = 64
			inputs[sdUrl].Width = 64
			inputs[sdUrl].Prompt = ""
			inputs[sdUrl].Validate = urlValidator

			if c.SD != nil {
				inputs[sdUrl].SetValue(c.SD.String())
			}
		case llmUrl:
			inputs[llmUrl] = textinput.New()
			inputs[llmUrl].Placeholder = "http://localhost:7869"
			inputs[llmUrl].CharLimit = 64
			inputs[llmUrl].Width = 64
			inputs[llmUrl].Prompt = ""
			inputs[llmUrl].Validate = urlValidator

			if c.LLM != nil {
				inputs[llmUrl].SetValue(c.SD.String())
			}
		case apiUrl:
			inputs[apiUrl] = textinput.New()
			inputs[apiUrl].Placeholder = "http://localhost:1323"
			inputs[apiUrl].CharLimit = 64
			inputs[apiUrl].Width = 64
			inputs[apiUrl].Prompt = ""
			inputs[apiUrl].Validate = urlValidator

			if c.API != nil {
				inputs[apiUrl].SetValue(c.API.String())
			}
		}
	}

	return Model{
		config:  c,
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
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.focused == len(m.inputs)-1 {
				return m, m.save()
			}
			m.nextInput()
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
	return fmt.Sprintf(` Settings

 %s
 %s

 %s
 %s

 %s
 %s

 %s`,
		inputStyle.Width(64).Render("Stable Diffusion"),
		m.render(sdUrl),
		inputStyle.Width(64).Render("LLM"),
		m.render(llmUrl),
		inputStyle.Width(64).Render("API"),
		m.render(apiUrl),
		continueStyle.Render("Continue ->"),
	) + "\n"
}

func (m Model) render(i int) string {
	if success([]*url.URL{
		(*url.URL)(m.config.SD),
		m.config.LLM,
		m.config.API,
	}[i], m.inputs[i]) {
		m.inputs[i].TextStyle = successStyle
	} else {
		m.inputs[i].TextStyle = normalStyle
	}
	return m.inputs[i].View()
}

// Validator functions to ensure valid input
func urlValidator(s string) error {
	if len(s) < 7 {
		return nil
	}
	if strings.HasSuffix(s, ":") {
		return nil

	}
	if strings.HasSuffix(s, "/") {
		return nil
	}

	_, err := url.Parse(s)
	if err != nil {
		return err
	}
	return nil
}

func success(u *url.URL, t textinput.Model) bool {
	if u == nil {
		return false
	}
	return u.String() == t.Value()
}

func (m Model) save() tea.Cmd {
	return func() tea.Msg {
		var err error
		for i, input := range m.inputs {
			if input.Value() == "" {
				continue
			}
			switch i {
			case sdUrl:
				return m.config.SetSD(input.Value())
			case llmUrl:
				return m.config.SetLLM(input.Value())
			case apiUrl:
				return m.config.SetAPI(input.Value())
			}
		}
		return err
	}
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
