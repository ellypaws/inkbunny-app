package entities

import (
	"github.com/ellypaws/inkbunny-sd/llm"
	"github.com/ellypaws/inkbunny/api"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type InferenceRequest struct {
	Config  llm.Config  `json:"config"`
	Request llm.Request `json:"request"`
}

type InferenceSubmissionRequest struct {
	Config       llm.Config      `json:"config"`
	User         api.Credentials `json:"user"`
	SubmissionID string          `json:"submission_id,omitempty"`
	Request      *llm.Request    `json:"request,omitempty"`
}

type DescriptionRequest struct {
	SID           string `json:"sid"`
	SubmissionIDs string `json:"submission_ids"`
}

type DescriptionResponse struct {
	SubmissionID string `json:"submission_id"`
	Title        string `json:"title"`
	Username     string `json:"username"`
	Description  string `json:"description"`
}

type PrefillRequest struct {
	Description string `json:"description"`
}
