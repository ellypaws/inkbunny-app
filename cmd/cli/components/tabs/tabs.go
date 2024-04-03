// Package tabs is a component for rendering a list of tabs using [bubblezone].
// Copyright (c) Liam Stanley <me@liamstanley.io>. All rights reserved. Use
// of this [source code] is governed by the [MIT license] that can be found in
// the [LICENSE] file.
//
// [bubblezone]: https://github.com/lrstanley/bubblezone
// [source code]: https://github.com/lrstanley/bubblezone/blob/master/examples/full-lipgloss/tabs.go
// [MIT license]: https://opensource.org/license/mit
// [LICENSE]: https://github.com/lrstanley/bubblezone/blob/master/LICENSE
package tabs

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"strings"
)

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	divider = lipgloss.NewStyle().
		SetString("•").
		Padding(0, 1).
		Foreground(subtle).
		String()
)

var (
	activeTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}

	tabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┴",
		BottomRight: "┴",
	}

	tab = lipgloss.NewStyle().
		Border(tabBorder, true).
		BorderForeground(highlight).
		Padding(0, 1)

	activeTab = tab.Copy().Border(activeTabBorder, true)

	tabGap = tab.Copy().
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false)
)

type Tabs struct {
	prefix string
	height int
	width  int

	Active      string
	activeIndex uint8
	Items       []string
}

func (m Tabs) Init() tea.Cmd {
	return nil
}

func (m Tabs) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}

		for i, item := range m.Items {
			// Check each item to see if it's in bounds.
			if zone.Get(m.prefix + item).InBounds(msg) {
				m.Active = item
				m.activeIndex = uint8(i)
				break
			}
		}

		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			return m.Next(), nil
		case tea.KeyShiftTab:
			return m.Previous(), nil
		default:
			return m, nil
		}
	}
	return m, nil
}

func (m Tabs) Next() Tabs {
	m.activeIndex = (m.activeIndex + 1) % uint8(len(m.Items))
	m.Active = m.Items[m.activeIndex]
	return m
}

func (m Tabs) Previous() Tabs {
	m.activeIndex = (m.activeIndex - 1) % uint8(len(m.Items))
	m.Active = m.Items[m.activeIndex]
	return m
}

func (m Tabs) View() string {
	var out []string

	for _, item := range m.Items {
		// Make sure to mark each tab when rendering.
		if item == m.Active {
			out = append(out, zone.Mark(m.prefix+item, activeTab.Render(item)))
		} else {
			out = append(out, zone.Mark(m.prefix+item, tab.Render(item)))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, out...)
	gap := tabGap.Render(strings.Repeat(" ", max(0, m.width-lipgloss.Width(row)-2)))
	row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
	return row
}

func New(items []string) Tabs {
	return Tabs{
		prefix: "tab",
		Active: items[0],
		Items:  items,
	}
}
