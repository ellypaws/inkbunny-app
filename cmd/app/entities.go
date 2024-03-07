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
	SubmissionID string  `json:"id"`
	Username     string  `json:"name"`
	Link         string  `json:"email"`
	Title        string  `json:"subject"`
	Description  string  `json:"text"`
	Date         string  `json:"date"`
	Read         bool    `json:"read"`
	Labels       []Label `json:"labels"`
}

type Label string

func (l Label) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ReplaceAll(string(l), "_", " "))
}

func (l *Label) UnmarshalJSON(data []byte) error {
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*l = Label(strings.ReplaceAll(v, " ", "_"))
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
