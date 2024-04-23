// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    request, err := UnmarshalRequest(bytes)
//    bytes, err = request.Marshal()

package llm

import "encoding/json"

func UnmarshalRequest(data []byte) (Request, error) {
	var r Request
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Request) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Role string

const (
	SystemRole    Role = "system"
	UserRole      Role = "user"
	AssistantRole Role = "assistant"
)

type Request struct {
	Messages      []Message      `json:"messages"`
	Model         string         `json:"model,omitempty"`
	Temperature   float64        `json:"temperature"`
	MaxTokens     int64          `json:"max_tokens"`
	Stream        bool           `json:"stream"`
	StreamChannel chan *Response `json:"-"`
}

type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

func UnmarshalResponse(data []byte) (Response, error) {
	var r Response
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Response) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Response struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int64   `json:"index"`
	Delta        Message `json:"delta"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}
