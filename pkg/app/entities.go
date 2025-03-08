// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    mails, err := UnmarshalMails(bytes)
//    bytes, err = mails.Marshal()

package app

import (
	"encoding/json"
	"strings"
)

type Mails []Mail

func UnmarshalMails(data []byte) (Mails, error) {
	var r Mails
	err := json.Unmarshal(data, &r)
	return r, err
}

type Mail struct {
	SubmissionID   string  `json:"id"`
	Username       string  `json:"name"`
	ProfilePicture string  `json:"photo,omitempty"`
	Files          []File  `json:"files"`
	Link           string  `json:"email"`
	Title          string  `json:"subject"`
	Description    string  `json:"text"`
	Html           string  `json:"html,omitempty"`
	Date           string  `json:"date"`
	Read           bool    `json:"read"`
	Labels         []Label `json:"labels"`
}

type File struct {
	FileID             string `json:"file_id"`
	FileName           string `json:"file_name"`
	FilePreview        string `json:"thumbnail_url,omitempty"`
	NonCustomThumb     string `json:"thumbnail_url_noncustom,omitempty"`
	FileURL            string `json:"file_url,omitempty"`
	UserID             string `json:"user_id"`
	CreateDateTime     string `json:"create_datetime"`
	CreateDateTimeUser string `json:"create_datetime_usertime"`
}

type Label string

func (l Label) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.SetSpace())
}

func (l *Label) UnmarshalJSON(data []byte) error {
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*l = Label(l.SetUnderscore())
	return nil
}

const (
	Unread = false
	Read   = true

	Generated       Label = "ai_generated"
	Assisted        Label = "ai_assisted"
	StableDiffusion Label = "stable_diffusion"

	YiffyMix  Label = "yiffymix"
	YiffyMix3 Label = "yiffymix3"
)

func (l Label) SetUnderscore() string {
	return strings.ReplaceAll(string(l), " ", "_")
}

func (l Label) SetSpace() string {
	return strings.ReplaceAll(string(l), "_", " ")
}

func (l Label) String() string {
	return l.SetSpace()
}
