package description

import (
	"fmt"
	"strings"

	stick "github.com/76creates/stickers/flexbox"
	"github.com/TheZoraiz/ascii-image-converter/aic_package"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ellypaws/inkbunny/api"

	"github.com/ellypaws/inkbunny-app/cmd/cli/apis"
	utils "github.com/ellypaws/inkbunny-app/cmd/cli/components"
	"github.com/ellypaws/inkbunny-app/cmd/cli/entle"
	sd "github.com/ellypaws/inkbunny-sd/stable_diffusion"
)

const (
	hotPink     = lipgloss.Color("#FF06B7")
	pastelGreen = lipgloss.Color("#6A994E")
	darkGray    = lipgloss.Color("#767676")
)

var (
	pink  = lipgloss.NewStyle().Foreground(hotPink)
	green = lipgloss.NewStyle().Foreground(pastelGreen)
	gray  = lipgloss.NewStyle().Foreground(darkGray)

	normalStyle = lipgloss.NewStyle()

	titleStyle = pink.Bold(true)

	bodyStyle = normalStyle

	buttonStyle = green.Border(lipgloss.RoundedBorder()).Padding(0, 1)
)

type Model struct {
	config      *apis.Config
	submissions map[string]api.Submission
	images      *map[string]string

	Active  string
	loading bool
	spinner spinner.Model
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case []api.Submission:
		m.loading = false
		if m.submissions == nil {
			m.submissions = make(map[string]api.Submission)
		}
		for _, submission := range msg {
			m.submissions[submission.SubmissionID] = submission
		}
		for id := range *m.images {
			if _, ok := m.submissions[id]; !ok {
				delete(*m.images, id)
			}
		}
	case api.Submission:
		m.loading = false
		if m.submissions == nil {
			m.submissions = make(map[string]api.Submission)
		}
		m.submissions[msg.SubmissionID] = msg
	case SetActive:
		m.Active = string(msg)
		if _, ok := (*m.images)[string(msg)]; ok {
			return m, nil
		}
		m.loading = true
		if s, ok := m.submissions[string(msg)]; ok {
			go m.processImage(
				entle.Screen{Width: entle.Width(), Height: entle.Height()},
				s,
			)
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(m.spinner.Tick())
			return m, cmd
		}
		return m, nil
	case AsciiArt:
		m.loading = false
		return m.storeImage(string(msg)), nil
	case error:
		m.loading = false
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) storeImage(s string) Model {
	(*m.images)[m.Active] = s
	return m
}

func (m Model) SetActive(id string) func() tea.Msg {
	return func() tea.Msg { return SetActive(id) }
}

type SetActive string

func (m Model) View() string {
	if m.loading {
		return m.spinner.View()
	}
	return m.Render(entle.Screen{
		Width: entle.Width(), Height: entle.Height(),
	})
}

func (m Model) Render(s entle.Screen) string {
	// if m.loading {
	//	return lipgloss.Place(s.Width-100, s.Height, lipgloss.Center, lipgloss.Center, m.spinner.View())
	// }
	sub, ok := m.submissions[m.Active]
	if !ok {
		return lipgloss.Place(s.Width-100, s.Height, lipgloss.Center, lipgloss.Center, "select a submission")
	}
	title := fmt.Sprintf("%s by %s", titleStyle.Render(sub.Title), sub.Username)
	url := gray.Render(fmt.Sprintf("https://inkbunny.net/s/%s", sub.SubmissionID))

	body := bodyStyle.Render(sub.Description)

	top := stick.NewCell(1, 1).SetContent(lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		sub.UpdateDateUser,
		url))

	content := stick.New(s.Width-100, s.Height)
	content.SetRows(
		[]*stick.Row{
			content.NewRow().AddCells(
				top,
			),
			content.NewRow().AddCells(
				stick.NewCell(1, 6).SetContent(m.renderImage(sub.SubmissionID)),
			),
			content.NewRow().AddCells(
				stick.NewCell(1, 3).SetContent(body),
			),
			content.NewRow().AddCells(
				stick.NewCell(1, 1).SetContent(buttonStyle.Render("Reply...")),
				stick.NewCell(1, 1).SetContent(buttonStyle.Render("Mute this thread")),
			),
		})

	return content.Render()
}

func (m Model) renderImage(id string) string {
	if s, ok := (*m.images)[id]; ok {
		return s
	}
	if m.loading {
		return m.spinner.View()
	}
	return "image not found"
}

func (m Model) processImage(s entle.Screen, sub api.Submission) {
	if _, ok := (*m.images)[sub.SubmissionID]; ok {
		return
	}

	var file *api.File
	for _, f := range sub.Files {
		if strings.HasPrefix(f.MimeType, "image") {
			file = &f
			break
		}
	}
	if file == nil {
		(*m.images)[sub.SubmissionID] = "no image"
		return
	}
	flags := aic_package.DefaultFlags()

	size := [2]int{file.ThumbNonCustomX.Int(), file.ThumbNonCustomY.Int()}
	scaled := utils.Scale([2]int{max(15, s.Width-50), max(15, s.Height-50)}, size)
	flags.Dimensions = []int{scaled[0] + 5, scaled[1]}

	flags.Colored = true
	flags.Braille = true
	flags.Threshold = 128 / 2
	// flags.Dither = true
	// Conversion for an image
	url := file.ThumbnailURLMediumNonCustom
	if url == "" {
		url = file.ThumbnailURLMediumNonCustom
	}
	if url == "" {
		(*m.images)[sub.SubmissionID] = "no image"
		return
	}
	asciiArt, err := aic_package.Convert(fmt.Sprintf("%s/image?url=%s", (*sd.Host)(m.config.API).Base(), url), flags)
	if err != nil {
		(*m.images)[sub.SubmissionID] = "no image"
		return
	}
	(*m.images)[sub.SubmissionID] = asciiArt
}

type AsciiArt string

var basic = api.Submission{
	SubmissionBasic: api.SubmissionBasic{
		Title:          "Title",
		Username:       "Username",
		SubmissionID:   "123456",
		UpdateDateUser: "2021-01-01 12:00:00",
	},
	Description: "Description",
}

func New(config *apis.Config) Model {
	images := make(map[string]string)
	return Model{
		config:      config,
		submissions: nil,
		images:      &images,
		spinner:     spinner.New(spinner.WithSpinner(spinner.Moon)),
	}
}
