package service

import (
	"encoding/json"
	"fmt"
	"github.com/ellypaws/inkbunny-app/cmd/api/cache"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	sd "github.com/ellypaws/inkbunny-sd/stable_diffusion"
	"github.com/ellypaws/inkbunny/api"
	"github.com/labstack/echo/v4"
	units "github.com/labstack/gommon/bytes"
	"net/http"
	"net/url"
	"regexp"
	"slices"
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
	Query             string
	Writer            http.Flusher

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

	switch config.Output {
	case OutputBadges:
		detail.Ticket = &db.Ticket{
			Labels: db.TicketLabels(sub),
		}
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
		fallthrough
	case OutputMultipleTickets:
		auditorAsUser := AuditorAsUsernameID(config.Auditor)
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
					Message:     submissionMessage(&sub),
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

	if c.QueryParam("stream") == "true" {
		config.mutex.Lock()
		defer config.mutex.Unlock()
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

		enc := json.NewEncoder(c.Response())
		if err := enc.Encode(detail); err != nil {
			c.Logger().Errorf("error encoding submission %v: %v", sub.ID, err)
			c.Response().WriteHeader(http.StatusInternalServerError)
			return
		}
		c.Logger().Debugf("flushing %v", sub.ID)

		config.Writer.Flush()
		c.Logger().Infof("finished processing %v", sub.ID)
	}

	go setCache(c, config, detail)
}

func setCache(c echo.Context, config *Config, detail *Detail) {
	bin, err := json.Marshal(detail)
	if err != nil {
		c.Logger().Errorf("error marshaling submission %v: %v", detail.ID, err)
		return
	}

	key := fmt.Sprintf(
		"%s:review:%s:%d?%s",
		echo.MIMEApplicationJSON,
		config.Output,
		detail.ID,
		config.Query,
	)
	err = config.Cache.Set(key, &cache.Item{
		Blob:     bin,
		MimeType: echo.MIMEApplicationJSON,
	}, cache.Week)
	if err != nil {
		c.Logger().Errorf("error caching caption: %v", err)
	} else {
		c.Logger().Infof("Cached %s %dKiB", key, len(bin)/units.KiB)
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

func submissionMessage(sub *db.Submission) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[u]AI Submission %d by @%s ", sub.ID, sub.Username))

	flags := db.TicketLabels(*sub)
	if len(flags) == 0 {
		sb.WriteString("needs to be reviewed[/u]\n")
	}
	for i, flag := range flags {
		switch i {
		case 0:
			switch flag {
			case db.LabelArtistUsed:
				sb.WriteString("has used an artist in the prompt[/u]\n")
			case db.LabelMissingParams:
				sb.WriteString("does not have any parameters[/u]\n")
			case db.LabelMissingPrompt:
				sb.WriteString("is missing the prompt[/u]\n")
			case db.LabelMissingModel:
				sb.WriteString("does not include the model information[/u]\n")
			case db.LabelMissingSeed:
				sb.WriteString("is missing the generation seed[/u]\n")
			case db.LabelSoldArt:
				sb.WriteString("is a selling content[/u]\n")
			case db.LabelPrivateTool:
				sb.WriteString(fmt.Sprintf("was generated using a private tool %s[/u]\n", sub.Metadata.Generator))
			case db.LabelPrivateLora:
				sb.WriteString("was generated using a private Lora model[/u]\n")
			case db.LabelPrivateModel:
				sb.WriteString("was generated using a private checkpoint model[/u]\n")
			default:
				sb.WriteString("is not following AI ACP[/u]\n")
			}
		case 1:
			sb.WriteString("\n\nIn addition, the following flags were detected:")
			fallthrough
		default:
			sb.WriteString("\n")
			sb.WriteString(string(flag))
		}
	}

	sb.WriteString(fmt.Sprintf("\n%s by @%s\n#M%d", sub.URL, sub.Username, sub.ID))

	if len(sub.Metadata.ArtistUsed) > 0 {
		sb.WriteString("\n\n")
		sb.WriteString("The prompt may have used these artists: ")

		for i, artist := range sub.Metadata.ArtistUsed {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("[b]")
			if artist.UserID != nil {
				sb.WriteString(fmt.Sprintf("ib!%s[/b]", artist.Username))
			} else {
				sb.WriteString(artist.Username)
				sb.WriteString("[/b]")
			}
			if i == len(sub.Metadata.ArtistUsed)-1 {
				sb.WriteString("\n")
			}
		}

		highlight := make(map[string]string)
		for name, obj := range sub.Metadata.Objects {
			meta := strings.ToLower(obj.Prompt + obj.NegativePrompt)

			var replaced bool
			for _, artist := range sub.Metadata.ArtistUsed {
				re, err := regexp.Compile(fmt.Sprintf(`(?i)\b(%s)\b`, artist.Username))
				if err != nil {
					continue
				}

				if !replaced && re.MatchString(meta) {
					replaced = true
					highlight[name] = meta
				}
				if replaced {
					highlight[name] = re.ReplaceAllStringFunc(highlight[name], func(s string) string {
						if artist.UserID != nil {
							return fmt.Sprintf("[b]>>> [u][name]%s[/name][/u] <<<[/b]", s)
						}
						return fmt.Sprintf("[b] >>> [color=#F78C6C][u]%s[/u][/color] <<< [/b]", s)
					})
				}
			}
		}

		for title, prompt := range highlight {
			var file *db.File
			if slices.ContainsFunc(sub.Files, func(f db.File) bool {
				if f.File.FileName == title {
					file = &f
					return true
				}
				return false
			}) {
				sb.WriteString(fmt.Sprintf("\nFile: [url=%s]%s[/url]", file.File.FileURLFull, file.File.FileName))
			}
			sb.WriteString(fmt.Sprintf("\n[q=%s]%s[/q]", title, prompt))
		}
	}

	if sub.Metadata.MissingPrompt {
		sb.WriteString("\n")
		sb.WriteString("The submission is missing the prompt")
	}

	if len(sub.Metadata.AIKeywords) == 0 {
		if sub.Metadata.AISubmission {
			sb.WriteString("\n")
			sb.WriteString("The submission was detected to have AI content, but was not tagged as such")
		}
	}

	var added uint
	for i, file := range sub.Files {
		switch file.File.MimeType {
		//case echo.MIMEApplicationJSON, echo.MIMETextPlain:
		default:
			if added == 0 {
				sb.WriteString("\n[u]MD5 Checksums at the time of writing[/u]:")
			}
			sb.WriteString("\n")
			sb.WriteString(fmt.Sprintf("Page %d: [url=%s]%s[/url] (%s)", i+1,
				file.File.FileURLFull, file.File.FileName, file.File.FullFileMD5))
			added++
		}
	}

	if sub.Metadata.DetectedHuman {
		sb.WriteString("\n")
		if !sub.Metadata.TaggedHuman {
			sb.WriteString("A human was detected in the submission but was not tagged\n")
		} else {
			sb.WriteString("A human was detected in the submission and was tagged\n")
		}
		sb.WriteString("The detection rate is: ")
		sb.WriteString(fmt.Sprintf("%.2f", sub.Metadata.HumanConfidence))
	}

	return sb.String()
}

func AuditorAsUsernameID(auditor *db.Auditor) api.UsernameID {
	return api.UsernameID{UserID: strconv.FormatInt(auditor.UserID, 10), Username: auditor.Username}
}
