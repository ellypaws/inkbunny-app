// Copyright (c) Liam Stanley <me@liamstanley.io>. All rights reserved. Use
// of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package submissions

import (
	"errors"

	stick "github.com/76creates/stickers/flexbox"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ellypaws/inkbunny/api"
	zone "github.com/lrstanley/bubblezone"

	"github.com/ellypaws/inkbunny-app/cmd/cli/apis"
	utils "github.com/ellypaws/inkbunny-app/cmd/cli/components"
	"github.com/ellypaws/inkbunny-app/cmd/cli/components/submissions/description"
	"github.com/ellypaws/inkbunny-app/cmd/cli/entle"
	lib "github.com/ellypaws/inkbunny-app/pkg/api/library"
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

	description description.Model

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
			return m, tea.Batch(utils.ForceRender(), m.description.SetActive(m.SelectedItem().(item).id))
		}

		if msg.Button == tea.MouseButtonWheelDown {
			m.CursorDown()
			return m, tea.Batch(utils.ForceRender(), m.description.SetActive(m.SelectedItem().(item).id))
		}

		if msg.Action == tea.MouseActionPress {
			for i, listItem := range m.VisibleItems() {
				item := listItem.(item)
				// Check each item to see if it's in bounds.
				if zone.Get(item.id).InBounds(msg) {
					// If so, select it in the submissions.
					m.Select(i)
					cmd = tea.Batch(utils.ForceRender(), m.description.SetActive(item.id))
					break
				}
			}

			if zone.Get(ButtonViewSubmissions).InBounds(msg) {
				m.Active = !m.Active
				if m.Active {
					return m, utils.ForceRender()
				}
				cmd = utils.ForceRender()
			}

			if zone.Get("search").InBounds(msg) {
				m.searching = true
				m.err = nil
				return m, tea.Batch(m.GetList(), m.Model.StartSpinner())
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
		m.searching = false
		m.err = nil
		m.Active = true
		m.StopSpinner()
		m.Model.SetItems(msg)
		return m, nil
	case api.SubmissionSearchResponse:
		m.searching = false
		return m, m.responseToListItems(msg)
	case finishSearch:
		m.searching = false
		return m, nil
	case error:
		m.searching = false
		m.err = msg
		m.StopSpinner()
		return m, nil
	}

	return m.propagate(msg, cmd)
}

func (m List) propagate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
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
	m.description, cmd = utils.Propagate(m.description, msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if cmds != nil {
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m List) responseToListItems(msg api.SubmissionSearchResponse) tea.Cmd {
	var items []list.Item
	var ids []string
	for _, submission := range msg.Submissions {
		items = append(items, item{
			id:    submission.SubmissionID,
			title: submission.Title,
			desc:  submission.Username,
		})
		ids = append(ids, submission.SubmissionID)
	}
	return tea.Batch(
		func() tea.Msg { return items },
		m.getDescriptions(ids),
	)
}

func (m List) getDescriptions(s []string) tea.Cmd {
	return func() tea.Msg {
		response, err := (*lib.Host)(m.config.API).
			GetSubmission(m.config.User(), api.SubmissionDetailsRequest{
				SID:               m.config.User().Sid,
				SubmissionIDSlice: s,
				ShowDescription:   api.Yes,
			})
		if err != nil {
			return err
		}
		if len(response.Submissions) > 0 {
			return response.Submissions
		}
		return errors.New("no submissions found")
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
		submissionList := stick.New(s.Width-25, s.Height)
		submissionContent := submissionList.NewRow().AddCells(
			stick.NewCell(1, 1).SetContent(panel.Render()),
		)
		if m.Active {
			submissionContent.AddCells(
				stick.NewCell(3, 1).SetContent(m.View()),
				stick.NewCell(3, 1).SetContent(m.description.Render(s)),
			)
		}
		submissionList.SetRows([]*stick.Row{submissionContent})
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
			Type:               api.SubmissionTypes{api.SubmissionTypePicturePinup},
		}
		if m.input.Value() != "" {
			request.Text = m.input.Value()
		}

		if m.config.API != nil {
			response, err := (*lib.Host)(m.config.API).GetSearch(m.config.User(), request)
			if err != nil {
				return nil
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
	m.SetSpinner(spinner.Moon)

	input := textinput.New()

	input.Placeholder = "ai_generated"
	input.Focus()
	input.CharLimit = 22
	input.Width = 22
	input.Prompt = ""

	return List{Model: m, input: input, config: config, description: description.New(config)}
}
