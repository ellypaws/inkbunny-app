package service

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ellypaws/inkbunny/api"

	"github.com/ellypaws/inkbunny-app/pkg/db"
	"github.com/ellypaws/inkbunny-sd/utils"
)

// InkbunnyTimeLayout e.g. 2010-03-03 13:26:46.357649+00
const InkbunnyTimeLayout = "2006-01-02 15:04:05.999999-07"

func InkbunnySubmissionToDBSubmission(submission api.Submission, override bool) db.Submission {
	id, _ := strconv.ParseInt(submission.SubmissionID, 10, 64)
	userID, _ := strconv.ParseInt(submission.UserID, 10, 64)

	parsedTime, err := time.Parse(InkbunnyTimeLayout, submission.UpdateDateSystem)
	if err != nil {
		log.Printf("error: parsing date: %v", err)
		parsedTime = time.Now().UTC()
	}

	dbSubmission := db.Submission{
		ID:          id,
		UserID:      userID,
		Username:    submission.Username,
		URL:         fmt.Sprintf("https://inkbunny.net/s/%v", id),
		Title:       submission.Title,
		Description: submission.Description,
		Updated:     parsedTime,
		Ratings:     submission.Ratings,
		Keywords:    submission.Keywords,
	}

	for _, f := range submission.Files {
		dbSubmission.Files = append(dbSubmission.Files, db.File{
			File:    f,
			Caption: nil,
		})
	}

	SetSubmissionMeta(&dbSubmission, override)

	return dbSubmission
}

var PrivateTools = regexp.MustCompile(`(?i)\b(midjourney|novelai|bing|dall[- ]?e|nijijourney|craiyon|image[- ]*fx|perchance)\b`)

// SetSubmissionMeta modifies a submission's Metadata based on its Keywords and other fields.
func SetSubmissionMeta(submission *db.Submission, override bool) {
	if submission == nil {
		return
	}
	if override {
		submission.Metadata.AISubmission = true
	}
	for _, keyword := range submission.Keywords {
		switch keyword.KeywordName {
		case "ai generated", "ai art":
			submission.Metadata.Generated = true
			submission.Metadata.AIKeywords = append(submission.Metadata.AIKeywords, keyword.KeywordName)
			submission.Metadata.AISubmission = true
		case "ai assisted":
			submission.Metadata.Assisted = true
			submission.Metadata.AIKeywords = append(submission.Metadata.AIKeywords, keyword.KeywordName)
			submission.Metadata.AISubmission = true
		case "img2img":
			submission.Metadata.Img2Img = true
			submission.Metadata.AIKeywords = append(submission.Metadata.AIKeywords, keyword.KeywordName)
			submission.Metadata.AISubmission = true
		case "stable diffusion":
			submission.Metadata.StableDiffusion = true
			submission.Metadata.AIKeywords = append(submission.Metadata.AIKeywords, keyword.KeywordName)
			submission.Metadata.AISubmission = true
		case "comfyui", "comfy ui":
			submission.Metadata.ComfyUI = true
			submission.Metadata.AIKeywords = append(submission.Metadata.AIKeywords, keyword.KeywordName)
			submission.Metadata.AISubmission = true
		case "human":
			submission.Metadata.TaggedHuman = true
		}
		switch keyword.KeywordID {
		case db.AIGeneratedID, db.AIArt:
			submission.Metadata.Generated = true
		case db.AIAssistedID:
			submission.Metadata.Assisted = true
		case db.Img2ImgID:
			submission.Metadata.Img2Img = true
		case db.StableDiffusionID:
			submission.Metadata.StableDiffusion = true
		case db.ComfyUIID, db.ComfyUI:
			submission.Metadata.ComfyUI = true
		}
	}

	if tool := PrivateTools.FindString(submission.Description); tool != "" {
		submission.Metadata.AISubmission = true
		submission.Metadata.PrivateTool = true
		submission.Metadata.Generator = tool
	}

	var images int
	for _, file := range submission.Files {
		if strings.HasPrefix(file.File.MimeType, "image") {
			images++
			continue
		}
		if file.File.MimeType == "text/plain" {
			submission.Metadata.HasTxt = true
			continue
		}
		if file.File.MimeType == "application/json" {
			submission.Metadata.HasJSON = true
			continue
		}
	}
	if images > 1 {
		submission.Metadata.MultipleImages = true
	}
	submission.Metadata.MissingPrompt = true
	submission.Metadata.MissingModel = true
	if submission.Metadata.Objects != nil {
		submission.Metadata.AISubmission = true
		for _, obj := range submission.Metadata.Objects {
			if obj.Prompt != "" {
				submission.Metadata.MissingPrompt = false
			}
			if obj.OverrideSettings.SDModelCheckpoint != nil || obj.OverrideSettings.SDCheckpointHash != "" {
				submission.Metadata.MissingModel = false
			}
		}
	}
	if submission.Metadata.Params != nil && len(submission.Metadata.Params) > 0 {
		submission.Metadata.AISubmission = true
	}
	if aiRegex.MatchString(submission.Title) {
		submission.Metadata.AITitle = true
		submission.Metadata.AISubmission = true
	}
	if strings.Contains(submission.Description, "stable diffusion") {
		submission.Metadata.StableDiffusion = true
		submission.Metadata.AIDescription = true
		submission.Metadata.AISubmission = true
	}
	if strings.Contains(submission.Description, "comfyui") {
		submission.Metadata.ComfyUI = true
		submission.Metadata.AIDescription = true
		submission.Metadata.AISubmission = true
	}

	switch {
	case submission.Metadata.AISubmission:
		break
	case utils.StepsStart.MatchString(submission.Description):
		submission.Metadata.AISubmission = true
	case utils.ParametersStart.MatchString(submission.Description):
		submission.Metadata.AISubmission = true
	}

	if submission.Metadata.AISubmission && len(submission.Metadata.AIKeywords) == 0 {
		submission.Metadata.MissingTags = true
	}
}

var aiRegex = regexp.MustCompile(`(?i)\b(ai|ia|ai generated|ai assisted|img2img|stable diffusion|comfyui)\b`)

var payment = regexp.MustCompile(`(?i)\b(ko-?fi|paypal|patreon|subscribestar|donate|bitcoin|ethereum|monero)\b`)

var sortedTicketLabels = []db.TicketLabel{
	db.LabelArtistUsed,
	db.LabelPrivateTool,
	db.LabelMissingTags,
	db.LabelMissingParams,
	db.LabelMissingPrompt,
	db.LabelMissingModel,
	db.LabelMissingSeed,
}

func TicketLabels(submission db.Submission) []db.TicketLabel {
	labels := make(map[db.TicketLabel]bool)
	metadata := submission.Metadata

	if metadata.TaggedHuman {
		labels[db.LabelTaggedHuman] = true
	}
	if metadata.DetectedHuman {
		labels[db.LabelDetectedHuman] = true
	}

	if metadata.AISubmission {
		if len(metadata.Objects) == 0 {
			if metadata.HasTxt || metadata.HasJSON {
				labels[db.LabelCannotParse] = true
			} else {
				labels[db.LabelMissingParams] = true
			}
		}
		if metadata.MissingTags {
			labels[db.LabelMissingTags] = true
		}
		if len(metadata.ArtistUsed) > 0 {
			labels[db.LabelArtistUsed] = true
		}

		if p := payment.FindString(submission.Description); p != "" {
			labels[db.TicketLabel(fmt.Sprintf("%s:%s", db.LabelPayMention, p))] = true
		}

		if submission.Updated.Before(db.Nov21) {
			labels[db.LabelBeforeRuleRevision] = true
		}

		if metadata.PrivateTool {
			labels[db.TicketLabel(fmt.Sprintf("%s:%s", db.LabelPrivateTool, metadata.Generator))] = true
		}

		const (
			prompt  = "prompt"
			model   = "model"
			seed    = "seed"
			steps   = "steps"
			cfg     = "cfg"
			sampler = "sampler"
		)
		hints := [...]hint{
			0: {label: prompt},
			1: {label: model},
			2: {label: seed},
			3: {label: steps},
			4: {label: cfg},
			5: {label: sampler},
		}
		for _, obj := range metadata.Objects {
			hints[0].assert(obj.Prompt != "")
			hints[1].assert(obj.OverrideSettings.SDModelCheckpoint != nil || obj.OverrideSettings.SDCheckpointHash != "")
			hints[2].assert(obj.Seed != 0 && obj.Seed != -1)
			hints[3].assert(obj.Steps > 0)
			hints[4].assert(obj.CFGScale != 0.0)
			hints[5].assert(obj.SamplerName != "")
		}
		for _, v := range hints {
			if v.missing {
				if v.partial {
					labels[db.TicketLabel("partial_"+v.label)] = true
				} else {
					labels[db.TicketLabel("missing_"+v.label)] = true
				}
			}
		}
	}

	out := make([]db.TicketLabel, 0, len(labels))
	for _, label := range sortedTicketLabels {
		if labels[label] {
			out = append(out, label)
			labels[label] = false
		}
	}

	for label, ok := range labels {
		if !ok {
			continue
		}
		out = append(out, label)
	}

	return out
}

type hint struct {
	label   string
	missing bool
	partial bool
}

func (h *hint) assert(condition bool) {
	if condition {
		h.partial = true
	} else {
		h.missing = true
	}
}
