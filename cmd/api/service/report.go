package service

import (
	"fmt"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny/api"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

type File struct {
	FileID       string `json:"file_id,omitempty"`
	FileName     string `json:"file_name,omitempty"`
	SubmissionID string `json:"submission_id,omitempty"`
	Page         int    `json:"page,omitempty"`
	FullFileMD5  string `json:"full_file_md5,omitempty"`
	FileURLFull  string `json:"file_url_full,omitempty"`
}

type SubInfo struct {
	Title   string           `json:"title,omitempty"`
	URL     string           `json:"url,omitempty"`
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
			Title:   sub.Submission.Title,
			URL:     sub.Submission.URL,
			Flags:   sub.Ticket.Labels,
			Artists: sub.Submission.Metadata.ArtistUsed,
		}

		for _, f := range sub.Submission.Files {
			info.Files = append(info.Files, File{
				FileID:       f.File.FileID,
				FileName:     f.File.FileName,
				SubmissionID: f.File.SubmissionID,
				Page:         int(f.File.SubmissionFileOrder) + 1,
				FullFileMD5:  f.File.FullFileMD5,
				FileURLFull:  f.File.FileURLFull,
			})
		}

		out.Submissions = append(out.Submissions, info)
	}
	if len(processed) > 0 {
		out.Ratio = float64(out.Violations) / float64(len(processed))
	}

	return out
}

type TicketReport struct {
	Ticket db.Ticket `json:"ticket"`
	Report Report    `json:"report"`
}

func CreateTicketReport(auditor *db.Auditor, details []Detail, host *url.URL, store func(TicketReport)) TicketReport {
	report := CreateReport(details, auditor)
	auditorAsUser := AuditorAsUsernameID(auditor)

	var info struct {
		Labels  []db.TicketLabel
		Files   []*db.File
		MD5     []string
		Artists []api.UsernameID
		IDs     []int64
		Objects []map[string]entities.TextToImageRequest
	}

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
		message.WriteString(fmt.Sprintf("[b]%s[/b]", label))
	}

	for i, id := range info.IDs {
		if i == 0 {
			message.WriteString("\n\n[u]Submissions[/u]:\n")
		} else {
			message.WriteString("  ")
		}
		message.WriteString(fmt.Sprintf("#M%d", id))
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

	for i, image := range info.Files {
		if i == 0 {
			message.WriteString(fmt.Sprintf("\n[u]MD5 Checksums at the time of writing[/u] ([url=https://inkbunny.net/submissionsviewall.php?text=%s&md5=yes&mode=search]search all[/url]):", strings.Join(info.MD5, "%20")))
		}
		message.WriteString("\n")
		message.WriteString(fmt.Sprintf("[url=https://inkbunny.net/s/%s]%s[/url]: [url=%s]%s[/url] ([url=https://inkbunny.net/submissionsviewall.php?text=%s&md5=yes&mode=search]%s[/url])",
			image.File.SubmissionID, image.File.SubmissionID,
			image.File.FileURLFull, image.File.FileName, image.File.FullFileMD5, image.File.FullFileMD5))
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
	}, report}

	store(out)
	return out
}
