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

	Extra
}

type Extra struct {
	DescriptionSanitized string `json:"description_sanitized,omitempty"`
	ThumbnailURL         string `json:"thumbnail_url,omitempty"`
	ThumbnailWidth       int    `json:"thumbnail_width,omitempty"`
	ThumbnailHeight      int    `json:"thumbnail_height,omitempty"`
}

const (
	OutputSingleTicket    OutputType = "single_ticket"
	OutputReport          OutputType = "report"
	OutputReportIDs       OutputType = "report_ids"
	OutputMultipleTickets OutputType = "multiple_tickets"
	OutputSubmissions     OutputType = "submissions"
	OutputFull            OutputType = "full"
	OutputBadges          OutputType = "badges"
)

type OutputType = string

type Config struct {
	SubmissionDetails api.SubmissionDetailsResponse
	Artists           []db.Artist
	Cache             cache.Cache
	Host              *sd.Host
	Output            OutputType
	Parameters        bool
	Interrogate       bool
	Auditor           *db.Auditor
	ApiHost           *url.URL
	Query             url.Values
	Writer            http.Flusher

	wg    sync.WaitGroup
	mutex sync.Mutex
}

func ProcessResponse(c echo.Context, config *Config) []Detail {
	submissionCount := len(config.SubmissionDetails.Submissions)

	processed := make(chan Detail, submissionCount)
	defer close(processed)

	for i := range submissionCount {
		go spawnSubmissionWorker(c, config, &config.SubmissionDetails.Submissions[i], processed)
	}

	var details []Detail
	for range submissionCount {
		detail := <-processed
		if c.QueryParam("stream") == "true" {
			stream(c, config.Writer, detail)
		}

		go setCache(c, config, &detail)
		details = append(details, detail)
	}

	return details
}

func spawnSubmissionWorker(c echo.Context, config *Config, sub *api.Submission, processed chan<- Detail) {
	processed <- processSubmission(c, sub, config)
}

func processSubmission(c echo.Context, submission *api.Submission, config *Config) Detail {
	sub := InkbunnySubmissionToDBSubmission(*submission, config.Output == OutputReportIDs)

	if sub.Metadata.AISubmission {
		c.Logger().Infof("processing files for %s %s", sub.URL, sub.Title)
		parseFiles(c, &sub, config)
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

	var detail = Detail{
		URL:        sub.URL,
		ID:         api.IntString(sub.ID),
		User:       user,
		Submission: &sub,
	}

	switch config.Output {
	case OutputReport, OutputReportIDs:
		detail.Extra = Extra{
			ThumbnailURL:    submission.ThumbnailURLMediumNonCustom,
			ThumbnailWidth:  int(submission.ThumbMediumNonCustomX),
			ThumbnailHeight: int(submission.ThumbMediumNonCustomY),
		}
		fallthrough
	case OutputBadges:
		detail.Ticket = &db.Ticket{
			Labels: TicketLabels(sub),
		}
	case OutputFull:
		detail.Inkbunny = submission
		for f, file := range sub.Files {
			if !strings.Contains(file.File.MimeType, "image") {
				continue
			}
			detail.Images = append(detail.Images, &sub.Files[f])
		}
		detail.Extra.DescriptionSanitized = sanitizeDescription(submission.DescriptionBBCodeParsed, config.ApiHost)
		fallthrough
	case OutputSubmissions:
		fallthrough
	case OutputSingleTicket:
		fallthrough
	case OutputMultipleTickets:
		auditorAsUser := AuditorAsUsernameID(config.Auditor)
		flags := TicketLabels(sub)
		detail.Ticket = &db.Ticket{
			ID:         sub.ID,
			Subject:    fmt.Sprintf("AI Submission #%d by @%s %s", sub.ID, sub.Username, ticketSubject(flags)),
			DateOpened: time.Now().UTC(),
			Status:     "triage",
			Labels:     flags,
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

	return detail
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
		config.Query.Encode(),
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

func stream(c echo.Context, writer http.Flusher, detail Detail) {
	if writer == nil {
		return
	}
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	enc := json.NewEncoder(c.Response())
	if err := enc.Encode(detail); err != nil {
		c.Logger().Errorf("error encoding submission %s: %s", detail.ID, err)
		c.Response().WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Logger().Debugf("flushing %v", detail.ID)

	writer.Flush()
}

var apiImage = regexp.MustCompile(`(?i)(https://(?:\w+\.ib\.metapix|inkbunny)\.net(?:/[\w\-.]+)+\.(?:jpe?g|png|gif))`)

func sanitizeDescription(description string, apiHost *url.URL) string {
	description = strings.ReplaceAll(description, "href='/", "href='https://inkbunny.net/")
	description = strings.ReplaceAll(description, "thumbnails/large", "thumbnails/medium")
	description = apiImage.ReplaceAllString(description, fmt.Sprintf("%s/image?url=${1}", apiHost))
	return description
}

func parseFiles(c echo.Context, sub *db.Submission, config *Config) {
	var wg sync.WaitGroup
	if config.Parameters {
		wg.Add(1)
		go RetrieveParams(c, &wg, sub, config.Cache, config.Artists)
	}
	if config.Interrogate {
		for i := range sub.Files {
			wg.Add(1)
			go RetrieveCaptions(c, &wg, sub, i, config.Host)
		}
	}
	wg.Wait()
}

// ticketSubject returns the subject of the ticket based on the flags detected in the submission.
func ticketSubject(flags []db.TicketLabel) string {
	if len(flags) == 0 {
		return "needs to be reviewed"
	} else {
		switch {
		case slices.Contains(flags, db.LabelArtistUsed):
			return "has used an artist in the prompt"
		case slices.Contains(flags, db.LabelMissingParams):
			return "does not have any parameters"
		case slices.Contains(flags, db.LabelMissingPrompt):
			return "is missing the prompt"
		case slices.Contains(flags, db.LabelMissingModel):
			return "does not include the model information"
		case slices.Contains(flags, db.LabelMissingSeed):
			return "is missing the generation seed"
		case slices.Contains(flags, db.LabelSoldArt):
			return "is a selling content"
		case slices.Contains(flags, db.LabelPrivateTool):
			return "was generated using a private tool"
		case slices.Contains(flags, db.LabelPrivateLora):
			return "was generated using a private Lora model"
		case slices.Contains(flags, db.LabelPrivateModel):
			return "was generated using a private checkpoint model"
		case slices.Contains(flags, db.LabelMissingTags):
			return "is missing the AI tags"
		default:
			return "is not following AI ACP"
		}
	}
}

func ticketFlagSummary(flags []db.TicketLabel, colors map[string]string) string {
	var sb strings.Builder
	for i, label := range flags {
		if i == 0 {
			sb.WriteString("\nThe following flags were detected:\n")
		} else {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("[b][color=%s]%s[/color][/b]", getColor(label, colors), label))
	}
	return sb.String()
}

func submissionMessage(sub *db.Submission) string {
	sb := NewChunkedWriter(10000, "\n--------✂️--------")
	sb.WriteString(fmt.Sprintf("[u]AI Submission %d by @%s ", sub.ID, sub.Username))

	flags := TicketLabels(*sub)
	sb.WriteString(ticketSubject(flags))
	sb.WriteString("[/u]\n")

	slices.Sort(flags)
	colors := make(map[string]string)
	sb.WriteString(ticketFlagSummary(flags, colors))

	sb.WriteString(fmt.Sprintf("\n%s by @%s\n#M%d", sub.URL, sub.Username, sub.ID))

	if len(sub.Metadata.ArtistUsed) > 0 {
		sb.WriteString("\n\n")
		sb.WriteString("The prompt may have used these artists: ")
		sb.WriteString(writeArtistUsed(sub))
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

	var md5 []string
	for _, file := range sub.Files {
		md5 = append(md5, file.File.FullFileMD5)
	}

	sb.Split()

	for i, file := range sub.Files {
		switch file.File.MimeType {
		//case echo.MIMEApplicationJSON, echo.MIMETextPlain:
		default:
			if i == 0 {
				sb.WriteString(fmt.Sprintf("\n\n[u]MD5 Checksums at the time of writing[/u] ([url=https://inkbunny.net/submissionsviewall.php?text=%s&md5=yes&mode=search]search all[/url]):", strings.Join(md5, "%20")))
			}
			sb.WriteString("\n")
			sb.WriteString(fmt.Sprintf("Page %d: [url=%s]%s[/url] ([url=https://inkbunny.net/submissionsviewall.php?text=%s&md5=yes&mode=search]%s[/url])",
				file.File.SubmissionFileOrder+1,
				file.File.FileURLFull, file.File.FileName, file.File.FullFileMD5, file.File.FullFileMD5))
			sb.Split()
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

func writeArtistUsed(sub *db.Submission) string {
	sb := NewChunkedWriter(10000, "\n--------✂️--------")
	if len(sub.Metadata.ArtistUsed) == 0 {
		return ""
	}

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
		var replaced bool
		for _, artist := range sub.Metadata.ArtistUsed {
			re, err := regexp.Compile(fmt.Sprintf(`(?i)\b(%s)\b`, artist.Username))
			if err != nil {
				continue
			}

			if !replaced {
				if re.MatchString(obj.Prompt) {
					replaced = true
					highlight[name] = obj.Prompt
				}
				if re.MatchString(obj.NegativePrompt) {
					replaced = true
					highlight[name] = fmt.Sprintf("%s\n(Found in negative prompt)\n%s", highlight[name], obj.NegativePrompt)
				}
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
			if strings.HasPrefix(title, f.File.FileName) {
				file = &f
				return true
			}
			return false
		}) {
			sb.WriteString(fmt.Sprintf("\nFile %d: [url=%s]%s[/url] ([url=https://inkbunny.net/submissionsviewall.php?text=%s&md5=yes&mode=search]%s[/url]) https://inkbunny.net/s/%s",
				file.File.SubmissionFileOrder+1, file.File.FileURLFull, file.File.FileName, file.File.FullFileMD5, file.File.FullFileMD5, file.File.SubmissionID))
		} else {
			sb.WriteString(fmt.Sprintf("\nFrom description: [url=%s]%s[/url] %s", sub.URL, title, sub.URL))
		}
		sb.WriteString(fmt.Sprintf("\n[q=%s]%s[/q]", title, prompt))
		sb.Split()
	}

	return sb.String()
}

func AuditorAsUsernameID(auditor *db.Auditor) api.UsernameID {
	if auditor == nil {
		return api.UsernameID{UserID: "0", Username: "guest"}
	}
	return api.UsernameID{UserID: strconv.FormatInt(auditor.UserID, 10), Username: auditor.Username}
}
