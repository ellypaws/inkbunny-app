package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	tasks    []task
	cursor   int
	selected map[int]struct{}
}

type task struct {
	id       string
	category string
	title    string
	status   string
	priority string
}

func initialModel() model {
	return model{
		// Initialize with some example tasks
		tasks: []task{
			{"TASK-8782", "Documentation", "You can't compress the program without quantifying the open-source SSD...", "In Progress", "Medium"},
			{"TASK-7878", "Documentation", "Try to calculate the EXE feed, maybe it will index the multi-byte pixel!", "Backlog", "Medium"},
			// Add more tasks...
		},
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// Define keybindings here
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}
		case "enter", " ":
			// Toggle selection of the current task
			if _, ok := m.selected[m.cursor]; ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
			// Add more keybindings as needed
		}
	}
	return m, nil
}

func (m model) View() string {
	// Define styles
	var (
		bodyStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				PaddingRight(2).
				PaddingTop(1).
				PaddingBottom(1).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#875f9a"))

		taskStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#674172")).
				PaddingLeft(2)

		selectedTaskStyle = taskStyle.Copy().
					Foreground(lipgloss.Color("#d1a3ff")).
					Background(lipgloss.Color("#ead3ff")).
					Bold(true)

		cursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#d1a3ff")).
				Bold(true)
	)

	var b strings.Builder

	// Render tasks
	for i, t := range m.tasks {
		taskStr := fmt.Sprintf("%s - %s", t.id, t.title)
		if _, ok := m.selected[i]; ok {
			taskStr = selectedTaskStyle.Render(taskStr)
		} else {
			taskStr = taskStyle.Render(taskStr)
		}

		if i == m.cursor {
			fmt.Fprintf(&b, "%s\n", cursorStyle.Render(taskStr))
		} else {
			fmt.Fprintf(&b, "%s\n", taskStr)
		}
	}

	return bodyStyle.Render(b.String())
}

func main() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
