// Copyright (c) Liam Stanley <me@liamstanley.io>. All rights reserved. Use
// of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package list

import (
	"errors"
	stick "github.com/76creates/stickers/flexbox"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lib "github.com/ellypaws/inkbunny-app/api/library"
	"github.com/ellypaws/inkbunny-app/cmd/cli/apis"
	utils "github.com/ellypaws/inkbunny-app/cmd/cli/components"
	"github.com/ellypaws/inkbunny-app/cmd/cli/entle"
	"github.com/ellypaws/inkbunny/api"
	zone "github.com/lrstanley/bubblezone"
	"net/http"
)

const (
	ButtonViewSubmissions = "submissions"
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
	config *apis.Config
	Active bool

	searching bool
	input     textinput.Model

	err error
}

func (m List) Init() tea.Cmd {
	return textinput.Blink
}

func (m List) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.SetSize(msg.Width-h-3, msg.Height-v-3)
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonWheelUp {
			m.CursorUp()
			return m, utils.ForceRender()
		}

		if msg.Button == tea.MouseButtonWheelDown {
			m.CursorDown()
			return m, utils.ForceRender()
		}

		if msg.Action == tea.MouseActionPress {
			for i, listItem := range m.VisibleItems() {
				item := listItem.(item)
				// Check each item to see if it's in bounds.
				if zone.Get(item.id).InBounds(msg) {
					// If so, select it in the list.
					m.Select(i)
					cmd = utils.ForceRender()
					break
				}
			}

			if zone.Get(ButtonViewSubmissions).InBounds(msg) {
				m.Active = !m.Active
				if m.Active {
					return m, tea.Batch(m.GetList(), utils.ForceRender())
				}
				cmd = utils.ForceRender()
			}

			if zone.Get("search").InBounds(msg) {
				m.searching = true
				m.err = nil
				return m, m.GetList()
			}
		}

		return m, cmd
	case tea.KeyMsg:
		switch msg.String() {
		case "1":
			m.Active = !m.Active
			cmd = tea.Batch(cmd, utils.ForceRender())
		case "enter":
			if !m.Active {
				return m, m.GetList()
			}
		}
	case []list.Item:
		m.Model.SetItems(msg)
		return m, nil
	case api.SubmissionSearchResponse:
		return m, responseToListItems(msg)
	case finishSearch:
		m.searching = false
		return m, nil
	case error:
		m.searching = false
		m.err = msg
		return m, nil
	}

	var cmds []tea.Cmd
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	m.Model, cmd = m.Model.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	m.input, cmd = m.input.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if cmds != nil {
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func responseToListItems(msg api.SubmissionSearchResponse) tea.Cmd {
	return func() tea.Msg {
		var items []list.Item
		for _, submission := range msg.Submissions {
			items = append(items, item{
				id:    submission.SubmissionID,
				title: submission.Title,
				desc:  submission.Username,
			})
		}
		return items
	}
}

func (m List) View() string {
	if !m.Active {
		return ""
	}
	return docStyle.Render(m.Model.View())
}

const (
	hotPink  = lipgloss.Color("#FF06B7")
	darkGray = lipgloss.Color("#767676")
)

var (
	pinkStyle = lipgloss.NewStyle().Foreground(hotPink)
	grayStyle = lipgloss.NewStyle().Foreground(darkGray)
)

func (m List) Render(s entle.Screen) func() string {
	return func() string {
		panel := stick.New(s.Width, s.Height)
		inputRender := lipgloss.JoinVertical(
			lipgloss.Top,
			pinkStyle.Render("Search"),
			m.input.View(),
			"",
			utils.If(
				m.searching,
				func() string { return "Searching..." },
				func() string { return zone.Mark("search", grayStyle.Render("Submit")) },
			),
			errString(m.err),
		)
		panel.SetRows(
			[]*stick.Row{
				panel.NewRow().AddCells(
					stick.NewCell(1, 1).SetContent(zone.Mark(ButtonViewSubmissions, "Press '1' to view submissions")),
				),
				panel.NewRow().AddCells(
					stick.NewCell(1, 4).SetContent(inputRender),
				),
			})
		submissionList := stick.New(s.Width, s.Height)
		submissionList.SetRows(
			[]*stick.Row{submissionList.NewRow().AddCells(
				stick.NewCell(1, 1).SetContent(panel.Render()),
				stick.NewCell(3, 1).SetContent(m.View()),
			)})
		return submissionList.Render()
	}
}

func errString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func empty() string {
	return ""
}

func (m List) GetList() tea.Cmd {
	return func() tea.Msg {
		if m.config.User() == nil {
			return errors.New("logged out")
		}
		var request = api.SubmissionSearchRequest{
			SID:                m.config.User().Sid,
			Text:               "ai_generated",
			SubmissionsPerPage: 10,
			Random:             api.Yes,
			Type:               api.SubmissionTypePicturePinup,
		}
		if m.input.Value() != "" {
			request.Text = m.input.Value()
		}

		if m.config.API != nil {
			var response api.SubmissionSearchResponse
			_, err := (&lib.Request{
				Host:      (*lib.Host)(m.config.API).WithPath("/inkbunny/search"),
				Method:    http.MethodGet,
				Data:      request,
				MarshalTo: &response,
			}).Do()
			if err != nil {
				return err
			}
			return response
		}

		return []list.Item{
			item{id: "14576", title: "Test Logo (Mascot Only) by Inkbunny", desc: "rabbit"},
			item{id: "1258063", title: "Inktober 2016 roundup - with pictures! by Inkbunny", desc: "inktober"},
		}
	}
}

type finishSearch struct{}

func New(config *apis.Config) List {
	items := []list.Item{
		item{id: "14576", title: "Inkbunny Logo (Mascot Only) by Inkbunny", desc: "rabbit"},
		item{id: "1258063", title: "Inktober 2016 roundup - with pictures! by Inkbunny", desc: "inktober"},
	}

	m := list.New(items, list.NewDefaultDelegate(), 0, 0)
	m.Title = "Select a submission"

	input := textinput.New()

	input.Placeholder = "ai_generated"
	input.Focus()
	input.CharLimit = 22
	input.Width = 22
	input.Prompt = ""

	return List{Model: m, input: input, config: config}
}
