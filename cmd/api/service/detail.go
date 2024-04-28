package service

import (
	"encoding/json"
	"fmt"
	"github.com/ellypaws/inkbunny-app/api/cache"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	sd "github.com/ellypaws/inkbunny-sd/stable_diffusion"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Detail struct {
	URL        string          `json:"url"`
	ID         api.IntString   `json:"id"`
	User       api.UsernameID  `json:"user"`
	Submission *db.Submission  `json:"submission,omitempty"`
	Inkbunny   *api.Submission `json:"inkbunny,omitempty"`
	Ticket     *db.Ticket      `json:"ticket,omitempty"`
	Images     []*db.File      `json:"images,omitempty"`

	DescriptionSanitized string `json:"description_sanitized,omitempty"`
}

const (
	OutputSingleTicket    OutputType = "single_ticket"
	OutputMultipleTickets OutputType = "multiple_tickets"
	OutputSubmissions     OutputType = "submissions"
	OutputFull            OutputType = "full"
	OutputBadges          OutputType = "badges"
)

type OutputType = string

type Config struct {
	SubmissionDetails api.SubmissionDetailsResponse
	Database          *db.Sqlite
	Cache             cache.Cache
	Host              *sd.Host
	Output            OutputType
	Auditor           *db.Auditor
	ApiHost           *url.URL

	wg      sync.WaitGroup
	mutex   sync.Mutex
	artists []db.Artist
}

func ProcessResponse(c echo.Context, config *Config) []Detail {
	var details = make([]Detail, len(config.SubmissionDetails.Submissions))

	config.artists = config.Database.AllArtists()

	for i, submission := range config.SubmissionDetails.Submissions {
		config.wg.Add(1)
		go processSubmission(c, &submission, config, &details[i])
	}
	config.wg.Wait()

	return details
}

func processSubmission(c echo.Context, submission *api.Submission, config *Config, detail *Detail) {
	defer config.wg.Done()

	sub := db.InkbunnySubmissionToDBSubmission(*submission)

	if sub.Metadata.AISubmission {
		c.Logger().Infof("processing files for %s %s", sub.URL, sub.Title)
		parseFiles(c, &sub, config.Cache, config.Host, config.artists)
	}

	//config.mutex.Lock()
	//defer config.mutex.Unlock()

	//for _, obj := range sub.Metadata.Objects {
	//	for hash, model := range obj.LoraHashes {
	//		config.Database.Wait()
	//		err := config.Database.UpsertModel(db.ModelHashes{
	//			hash: []string{model},
	//		})
	//		if err != nil {
	//			c.Logger().Errorf("error inserting model %s: %s", hash, err)
	//		}
	//	}
	//}

	if c.QueryParam("stream") == "true" {
		enc := json.NewEncoder(c.Response())
		if err := enc.Encode(sub); err != nil {
			c.Logger().Errorf("error encoding submission %v: %v", sub.ID, err)
		}
		c.Logger().Debugf("flushing %v", sub.ID)
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)
		c.Get("writer").(http.Flusher).Flush()
		c.Logger().Infof("finished processing %v", sub.ID)
	}

	//err := config.Database.InsertSubmission(sub)
	//if err != nil {
	//	c.Logger().Errorf("error inserting submission %v: %v", sub.ID, err)
	//}

	user := api.UsernameID{UserID: strconv.FormatInt(sub.UserID, 10), Username: sub.Username}

	*detail = Detail{
		URL:        sub.URL,
		ID:         api.IntString(sub.ID),
		User:       user,
		Submission: &sub,
	}

	auditorAsUser := AuditorAsUsernameID(config.Auditor)

	switch config.Output {
	case OutputBadges:
		detail.Ticket = new(db.Ticket)
	case OutputFull:
		detail.Inkbunny = submission
		for f, file := range sub.Files {
			if !strings.Contains(file.File.MimeType, "image") {
				continue
			}
			detail.Images = append(detail.Images, &sub.Files[f])
		}
		detail.DescriptionSanitized = sanitizeDescription(submission.DescriptionBBCodeParsed, config.ApiHost)
		fallthrough
	case OutputSubmissions:
		fallthrough
	case OutputSingleTicket:
		detail.Ticket = &db.Ticket{
			DateOpened: time.Now().UTC(),
			Responses: []db.Response{
				{
					SupportTeam: false,
					User:        auditorAsUser,
					Date:        time.Now().UTC(),
					Message:     "",
				},
			},
		}
	case OutputMultipleTickets:
		detail.Ticket = &db.Ticket{
			ID:         sub.ID,
			Subject:    fmt.Sprintf("Review for %v", sub.URL),
			DateOpened: time.Now().UTC(),
			Status:     "triage",
			Labels:     nil,
			Priority:   "low",
			Closed:     false,
			Responses: []db.Response{
				{
					SupportTeam: false,
					User:        auditorAsUser,
					Date:        time.Now().UTC(),
					Message:     "",
				},
			},
			SubmissionIDs: []int64{sub.ID},
			AssignedID:    &config.Auditor.UserID,
			UsersInvolved: db.Involved{
				Reporter: auditorAsUser,
				ReportedIDs: []api.UsernameID{
					user,
				},
			},
		}
	}
}

var apiImage = regexp.MustCompile(`(?i)(https://(?:\w+\.ib\.metapix|inkbunny)\.net(?:/[\w\-.]+)+\.(?:jpe?g|png|gif))`)

func sanitizeDescription(description string, apiHost *url.URL) string {
	description = strings.ReplaceAll(description, "href='/", "href='https://inkbunny.net/")
	description = strings.ReplaceAll(description, "thumbnails/large", "thumbnails/medium")
	description = apiImage.ReplaceAllString(description, fmt.Sprintf("%s/image?url=${1}", apiHost))
	return description
}

func parseFiles(c echo.Context, sub *db.Submission, cache cache.Cache, host *sd.Host, artists []db.Artist) {
	var wg sync.WaitGroup
	if c.QueryParam("parameters") == "true" {
		wg.Add(1)
		go RetrieveParams(c, &wg, sub, cache, artists)
	}
	if c.QueryParam("interrogate") == "true" {
		for i := range sub.Files {
			wg.Add(1)
			go RetrieveCaptions(c, &wg, sub, i, host)
		}
	}
	wg.Wait()
}

func AuditorAsUsernameID(auditor *db.Auditor) api.UsernameID {
	return api.UsernameID{UserID: strconv.FormatInt(auditor.UserID, 10), Username: auditor.Username}
}
