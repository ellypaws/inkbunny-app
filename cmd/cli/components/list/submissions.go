// Copyright (c) Liam Stanley <me@liamstanley.io>. All rights reserved. Use
// of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package list

import (
	stick "github.com/76creates/stickers/flexbox"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ellypaws/inkbunny-app/cmd/cli/entle"
	zone "github.com/lrstanley/bubblezone"
)

const (
	Submissions = "submissions"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type item struct {
	id    string
	title string
	desc  string
}

func (i item) Title() string       { return zone.Mark(i.id, i.title) }
func (i item) Description() string { return zone.Mark(i.id, i.desc) }
func (i item) FilterValue() string { return zone.Mark(i.id, i.title) }

type List struct {
	list.Model
	Active bool
}

func (m List) Init() tea.Cmd {
	return nil
}

func (m List) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.SetSize(msg.Width-h-3, msg.Height-v-3)
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonWheelUp {
			m.CursorUp()
			return m, nil
		}

		if msg.Button == tea.MouseButtonWheelDown {
			m.CursorDown()
			return m, nil
		}

		if msg.Action == tea.MouseActionPress {
			for i, listItem := range m.VisibleItems() {
				item := listItem.(item)
				// Check each item to see if it's in bounds.
				if zone.Get(item.id).InBounds(msg) {
					// If so, select it in the list.
					m.Select(i)
					break
				}
			}
		}

		return m, nil
	}

	var cmd tea.Cmd
	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}

func (m List) View() string {
	if !m.Active {
		return ""
	}
	return docStyle.Render(m.Model.View())
}

func (m List) Render(s entle.Screen) func() string {
	return func() string {
		submissionList := stick.New(s.Width, s.Height)
		submissionList.SetRows(
			[]*stick.Row{submissionList.NewRow().AddCells(
				stick.NewCell(1, 1).SetContent(zone.Mark(Submissions, "Press '1' to view submissions")),
				stick.NewCell(3, 1).SetContent(m.View()),
			)})
		return submissionList.Render()
	}
}

func New() List {
	items := []list.Item{
		item{id: "14576", title: "Inkbunny Logo (Mascot Only) by Inkbunny", desc: "rabbit"},
		item{id: "1258063", title: "Inktober 2016 roundup - with pictures! by Inkbunny", desc: "inktober"},
	}

	m := list.New(items, list.NewDefaultDelegate(), 0, 0)
	m.Title = "Select a submission"

	return List{Model: m}
}
