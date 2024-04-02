package utils

import tea "github.com/charmbracelet/bubbletea"

func IF[T any](condition bool, a, b T) T {
	if condition {
		return a
	}
	return b
}

func Propagate[T tea.Model](m T, msg tea.Msg) (T, tea.Cmd) {
	model, cmd := m.Update(msg)
	return model.(T), cmd
}
