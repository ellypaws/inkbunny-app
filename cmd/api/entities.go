package main

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

type SubmissionRequest struct {
	Config       llm.Config      `json:"config"`
	User         api.Credentials `json:"user"`
	SubmissionID string          `json:"submission_id"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
