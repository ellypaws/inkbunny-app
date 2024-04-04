package utils

import tea "github.com/charmbracelet/bubbletea"

func IF[T any](condition bool, a, b T) T {
	if condition {
		return a
	}
	return b
}

func If[T any](condition bool, a, b func() T) T {
	return IF(condition, a, b)()
}

func Propagate[T tea.Model](m T, msg tea.Msg) (T, tea.Cmd) {
	model, cmd := m.Update(msg)
	return model.(T), cmd
}

type RerenderMsg struct{}

var AlwaysRender bool = true

func ForceRender() tea.Cmd {
	if AlwaysRender {
		return func() tea.Msg {
			return RerenderMsg{}
		}
	}
	return nil
}

func AsCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}
