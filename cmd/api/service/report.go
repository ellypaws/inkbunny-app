package service

import (
	"encoding/json"
	"fmt"
	"github.com/ellypaws/inkbunny-app/cmd/api/cache"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	"math/rand/v2"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

type File struct {
	FileID       *string `json:"file_id,omitempty"`
	FileName     *string `json:"file_name,omitempty"`
	SubmissionID *string `json:"submission_id,omitempty"`
	Page         int     `json:"page,omitempty"`
	InitialMD5   *string `json:"initial_md5,omitempty"`
	FullFileMD5  *string `json:"full_file_md5,omitempty"`
	FileURLFull  *string `json:"file_url_full,omitempty"`
}

type SubInfo struct {
	Title *string `json:"title,omitempty"`
	URL   *string `json:"url,omitempty"`

	Generated *bool `json:"generated,omitempty"`
	Assisted  *bool `json:"assisted,omitempty"`

	Flags   []db.TicketLabel `json:"flags,omitempty"`
	Artists []db.Artist      `json:"artists,omitempty"`
	Files   []File           `json:"files,omitempty"`
}

type User struct {
	db.Auditor
	Role       string `json:"role"`
	AuditCount int    `json:"audit_count,omitempty"`
}

type Report struct {
	Auditor *User `json:"auditor,omitempty"`
	api.UsernameID
	Violations  int       `json:"violations"`
	Ratio       float64   `json:"violation_ratio"`
	Audited     int       `json:"total_audited"`
	ReportDate  time.Time `json:"report_date"`
	Submissions []SubInfo `json:"submissions"`
}

func CreateReport(processed []Detail, auditor *db.Auditor) Report {
	out := Report{
		ReportDate: time.Now().UTC(),
	}

	if auditor != nil {
		out.Auditor = &User{
			Auditor:    *auditor,
			Role:       auditor.Role.String(),
			AuditCount: auditor.AuditCount,
		}
	}

	for _, sub := range processed {
		out.UsernameID = sub.User

		if !sub.Submission.Metadata.AISubmission {
			continue
		}

		out.Audited++

		if len(sub.Ticket.Labels) == 0 {
			continue
		}

		out.Violations++

		info := SubInfo{
			Title:   &sub.Submission.Title,
			URL:     &sub.Submission.URL,
			Flags:   sub.Ticket.Labels,
			Artists: sub.Submission.Metadata.ArtistUsed,
			Files:   make([]File, len(sub.Submission.Files)),

			Generated: &sub.Submission.Metadata.Generated,
			Assisted:  &sub.Submission.Metadata.Assisted,
		}

		for i, f := range sub.Submission.Files {
			info.Files[i] = File{
				FileID:       &f.File.FileID,
				FileName:     &f.File.FileName,
				SubmissionID: &f.File.SubmissionID,
				Page:         int(f.File.SubmissionFileOrder) + 1,
				InitialMD5:   &f.File.InitialFileMD5,
				FullFileMD5:  &f.File.FullFileMD5,
				FileURLFull:  &f.File.FileURLFull,
			}
		}

		out.Submissions = append(out.Submissions, info)
	}
	if len(processed) > 0 {
		out.Ratio = float64(out.Violations) / float64(out.Audited)
	}

	return out
}

type TicketReport struct {
	Ticket     db.Ticket   `json:"ticket"`
	Report     Report      `json:"report"`
	Thumbnails []Thumbnail `json:"thumbnails"`
}

type Thumbnail struct {
	SubmissionID *int64  `json:"id,omitempty"`
	Title        *string `json:"title,omitempty"`
	PageCount    int     `json:"pagecount,omitempty"`
	URL          *string `json:"thumbnail_url,omitempty"`
	Width        *int    `json:"thumbnail_width,omitempty"`
	Height       *int    `json:"thumbnail_height,omitempty"`
	Generated    *bool   `json:"generated,omitempty"`
	Assisted     *bool   `json:"assisted,omitempty"`
}

func applyLabelColor(labels []db.TicketLabel, colors map[string]string) []string {
	if labels == nil || colors == nil {
		return nil
	}

	out := make([]string, len(labels))
	for i, label := range labels {
		out[i] = fmt.Sprintf("[color=%s]%s[/color]", getColor(label, colors), label)
	}

	return out
}

func getColor(label db.TicketLabel, colors map[string]string) string {
	if _, ok := colors[string(label)]; !ok {
		colors[string(label)] = randomColor()
	}
	return colors[string(label)]
}

func randomColor() string {
	var palette = [6]string{
		"#6169C0",
		"#2D4F7B",
		"#253C73",
		"#795577",
		"#4E4B76",
		"#1A2D65",
	}

	return palette[rand.IntN(6)]
}

func CreateTicketReport(auditor *db.Auditor, details []Detail, host *url.URL) TicketReport {
	report := CreateReport(details, auditor)
	auditorAsUser := AuditorAsUsernameID(auditor)

	var info struct {
		Labels  []db.TicketLabel
		Files   []*db.File
		MD5     []string
		Artists []api.UsernameID
		IDs     []int64
		Objects []map[string]entities.TextToImageRequest
		Thumbs  []Thumbnail

		Categories map[string][]int64
	}

	var colors = make(map[string]string)
	for _, sub := range details {
		if len(sub.Ticket.Labels) == 0 {
			continue
		}

		for _, label := range sub.Ticket.Labels {
			if !slices.Contains(info.Labels, label) {
				info.Labels = append(info.Labels, label)
			}
		}

		for _, file := range sub.Submission.Files {
			info.Files = append(info.Files, &file)
			info.MD5 = append(info.MD5, file.File.FullFileMD5)
		}

		for _, used := range sub.Submission.Metadata.ArtistUsed {
			if !slices.ContainsFunc(info.Artists, func(artist api.UsernameID) bool {
				return artist.Username == used.Username
			}) {
				user := api.UsernameID{
					Username: used.Username,
				}
				if used.UserID != nil {
					user.UserID = strconv.FormatInt(*used.UserID, 10)
				}
				info.Artists = append(info.Artists, user)
			}
		}

		info.IDs = append(info.IDs, int64(sub.ID))

		if sub.Submission.Metadata.Objects != nil {
			info.Objects = append(info.Objects, sub.Submission.Metadata.Objects)
		}

		info.Thumbs = append(info.Thumbs, Thumbnail{
			SubmissionID: &sub.Submission.ID,
			Title:        &sub.Submission.Title,
			PageCount:    len(sub.Submission.Files),
			URL:          &sub.Extra.ThumbnailURL,
			Width:        &sub.Extra.ThumbnailWidth,
			Height:       &sub.Extra.ThumbnailHeight,
		})

		if info.Categories == nil {
			info.Categories = make(map[string][]int64)
		}

		category := strings.Join(applyLabelColor(sub.Ticket.Labels, colors), ", ")
		if _, ok := info.Categories[category]; !ok {
			info.Categories[category] = []int64{sub.Submission.ID}
		} else {
			info.Categories[category] = append(info.Categories[category], sub.Submission.ID)
		}
	}

	var message strings.Builder

	message.WriteString(fmt.Sprintf("[u]AI Submissions by @%s ", report.UsernameID.Username))
	if len(info.Labels) > 0 {
		message.WriteString(fmt.Sprintf("do not follow the AI ACP[/u] (%d violations, %.2f%%):\n", report.Violations, report.Ratio*100))
	} else {
		message.WriteString(fmt.Sprintf("needs to be reviewed[/u]: (%d submissions)\n", len(info.IDs)))
	}

	for i, label := range info.Labels {
		if i == 0 {
			message.WriteString("\nThe following flags were detected:\n")
		} else {
			message.WriteString(", ")
		}
		message.WriteString(fmt.Sprintf("[b]%s[/b]", fmt.Sprintf("[color=%s]%s[/color]", getColor(label, colors), label)))
	}

	message.WriteString("\n\n[u]Submissions[/u]:")
	for category, submission := range info.Categories {
		message.WriteString(fmt.Sprintf("\n[b]%s[/b]:\n", category))
		for _, id := range submission {
			message.WriteString(fmt.Sprintf("#M%d", id))
		}
	}

	if len(info.Artists) > 0 {
		message.WriteString("\n\n")
		message.WriteString("The prompt may have used these artists: ")
	}

	var added int
	for _, detail := range details {
		if len(detail.Submission.Metadata.ArtistUsed) == 0 {
			continue
		}
		if added > 0 {
			message.WriteString("\n")
		}
		message.WriteString(writeArtistUsed(detail.Submission))
		added++
	}

	var lastSubmission string
	for i, image := range info.Files {
		if i == 0 {
			message.WriteString(fmt.Sprintf("\n\n[u]MD5 Checksums at the time of writing[/u] ([url=https://inkbunny.net/submissionsviewall.php?text=%s&md5=yes&mode=search]search all[/url]):", strings.Join(info.MD5, "%20")))
		}
		if lastSubmission != image.File.SubmissionID {
			if lastSubmission != "" {
				message.WriteString("\n")
			}
			message.WriteString(fmt.Sprintf("\nSubmission #[url=https://inkbunny.net/s/%s]%s[/url]:", image.File.SubmissionID, image.File.SubmissionID))
			lastSubmission = image.File.SubmissionID
		}
		message.WriteString(fmt.Sprintf("\n[url=%s]%s[/url] (%s)",
			image.File.FileURLFull, image.File.FileName, image.File.FullFileMD5))
	}

	message.WriteString(fmt.Sprintf("\n\nA copy of this report is available at: %s/report/%s/%s",
		host, report.Username, report.ReportDate.Format("2006-01-02")))

	out := TicketReport{db.Ticket{
		Subject: fmt.Sprintf("AI Submissions by %s - %d (%.2f%%) violations",
			report.UsernameID.Username, report.Violations, report.Ratio*100),
		DateOpened: time.Now().UTC(),
		Status:     "triage",
		Labels:     info.Labels,
		Priority:   "low",
		Closed:     false,
		Responses: []db.Response{
			{
				SupportTeam: false,
				User:        auditorAsUser,
				Date:        time.Now().UTC(),
				Message:     message.String(),
			},
		},
		SubmissionIDs: info.IDs,
		AssignedID:    &auditor.UserID,
		UsersInvolved: db.Involved{
			Reporter:    auditorAsUser,
			ReportedIDs: info.Artists,
		},
	}, report,
		info.Thumbs}

	return out
}

func StoreReport(c echo.Context, database *db.Sqlite, ticket TicketReport) {
	reportKey := fmt.Sprintf(
		"%s:report:%s:%s",
		echo.MIMEApplicationJSON,
		c.Param("id"),
		ticket.Report.ReportDate.Format(db.TicketDateLayout),
	)
	report := any(ticket.Report)
	bin, err := json.Marshal(report)
	if err != nil {
		c.Logger().Errorf("error marshaling report: %v", err)
		return
	}
	StoreReview(c, reportKey, &report, cache.Indefinite, bin...)

	err = database.UpsertTicketReport(db.TicketReport{
		Username:   ticket.Report.UsernameID.Username,
		ReportDate: ticket.Report.ReportDate,
		Report:     bin,
	})

	if err != nil {
		c.Logger().Error("error upserting ticket report:", err)
	}
}
