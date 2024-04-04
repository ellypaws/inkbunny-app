package description

import (
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
	inputStyle    = lipgloss.NewStyle().Foreground(hotPink)
	successStyle  = lipgloss.NewStyle().Foreground(pastelGreen)
	continueStyle = lipgloss.NewStyle().Foreground(darkGray)

	normalStyle = lipgloss.NewStyle()

	titleStyle = inputStyle.Bold(true)

	bodyStyle = normalStyle

	buttonStyle = successStyle.Border(lipgloss.RoundedBorder()).Padding(0, 1)
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
	//m.Title
	//m.Description
	//m.Username
	//m.UpdateDateUser
	//"https://inkbunny.net/s/" + m.SubmissionID
	title := titleStyle.Render("Meeting Tomorrow")
	username := normalStyle.Render("John Doe")
	body := bodyStyle.Render(`Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec blandit nunc at hendrerit ullamcorper. Vestibulum et tristique justo, sed iaculis dolor. Nullam ante diam, ultrices vitae faucibus vel, volutpat at sem. Sed congue nibh in lectus rutrum, egestas venenatis risus egestas. Nulla placerat ornare leo, nec vestibulum erat imperdiet vitae. Quisque dapibus felis eu ligula tempus mollis. In hac habitasse platea dictumst. Vestibulum venenatis quam urna, vitae vestibulum augue pulvinar ac. Sed sed nisl feugiat, efficitur enim vel, blandit elit.

Vestibulum et lacus mi. Duis sollicitudin, diam eget condimentum bibendum, lorem eros ullamcorper dui, in viverra nulla sapien ut arcu. Curabitur maximus sollicitudin ipsum, ac commodo dolor sollicitudin quis. Duis eu dolor quis turpis tempor dictum. Ut tempus tellus vitae iaculis hendrerit. Donec in elementum sem. Curabitur eleifend fringilla libero a feugiat. Mauris ac lacinia lacus. Nullam feugiat volutpat ipsum vehicula imperdiet. Proin eu fringilla nibh. Quisque vel dui lacus. Vivamus ipsum orci, fringilla vel semper a, scelerisque at quam. Ut in massa vitae dui molestie hendrerit. Sed a velit lobortis, gravida urna bibendum, porttitor erat. Donec sollicitudin nisl eu libero blandit faucibus. Proin quam diam, aliquet tristique sagittis sit amet, convallis at lectus.`)

	buttons := lipgloss.JoinHorizontal(lipgloss.Top,
		buttonStyle.Render("Reply..."),
		lipgloss.NewStyle().Width(4).Render(""), // Spacer
		buttonStyle.Render("Mute this thread"),
	)

	// Assemble the pieces
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		username,
		"",
		body,
		"",
		buttons,
	)
}

func New() Model {
	return Model{}
}
