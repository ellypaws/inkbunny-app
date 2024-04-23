package llm

import (
	"bytes"
	"testing"
)

func TestUnmarshalRequest(t *testing.T) {
	data := []byte(`{
  "messages": [
    {"role": "system", "content": "Just say hello!"},
    {"role": "user"  , "content": "Say hello!"     }
  ],
  "temperature": 1,
  "max_tokens": 10,
  "stream": true
}`)
	req, err := UnmarshalRequest(data)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	if req.Messages[0].Role != SystemRole {
		t.Errorf("Expected role to be system, got %s", req.Messages[0].Role)
	}

	if req.Messages[1].Content != "Say hello!" {
		t.Errorf("Expected content to be 'Say hello!', got %s", req.Messages[1].Content)
	}

	if req.Temperature != 1 {
		t.Errorf("Expected temperature to be 1, got %f", req.Temperature)
	}

	if req.MaxTokens != 10 {
		t.Errorf("Expected max tokens to be 10, got %d", req.MaxTokens)
	}

	if req.Stream != true {
		t.Errorf("Expected stream to be true, got %t", req.Stream)
	}

	if req.StreamChannel != nil {
		t.Errorf("Expected stream channel to be nil, got %v", req.StreamChannel)
	}
}

func TestRequest_Marshal(t *testing.T) {
	req := Request{
		Messages: []Message{{
			Role:    SystemRole,
			Content: "Just say hello!",
		}, UserMessage("Say hello!")},
		Temperature: 1,
		MaxTokens:   10,
		Stream:      true,
	}
	data, err := req.Marshal()
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	expected := `{"messages":[{"role":"system","content":"Just say hello!"},{"role":"user","content":"Say hello!"}],"temperature":1,"max_tokens":10,"stream":true}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

var responseTest = []byte(`{
  "id": "chatcmpl-uefukdhy3krmm74ri9qk2f",
  "object": "chat.completion.chunk",
  "created": 1088474663,
  "model": "deepseek-coder-6.7b-instruct.Q4_K_S.gguf",
  "choices": [
    {
      "index": 0,
      "delta": {"role": "assistant", "content": " I"},
      "message": {"role": "assistant", "content": "Hello there, user 2317"},
      "finish_reason": "stop"
    }
  ],
  "usage": {"prompt_tokens": 15, "completion_tokens": 22, "total_tokens": 37}
}`)

func TestUnmarshalResponse(t *testing.T) {
	resp, err := UnmarshalResponse(responseTest)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	if resp.ID != "chatcmpl-uefukdhy3krmm74ri9qk2f" {
		t.Errorf("Expected ID to be chatcmpl-uefukdhy3krmm74ri9qk2f, got %s", resp.ID)
	}

	if resp.Object != "chat.completion.chunk" {
		t.Errorf("Expected object to be chat.completion.chunk, got %s", resp.Object)
	}

	if resp.Created != 1088474663 {
		t.Errorf("Expected created to be 1088474663, got %d", resp.Created)
	}

	if resp.Model != "deepseek-coder-6.7b-instruct.Q4_K_S.gguf" {
		t.Errorf("Expected model to be deepseek-coder-6.7b-instruct.Q4_K_S.gguf, got %s", resp.Model)
	}

	if resp.Choices[0].Index != 0 {
		t.Errorf("Expected index to be 0, got %d", resp.Choices[0].Index)
	}

	if resp.Choices[0].Delta.Role != AssistantRole {
		t.Errorf("Expected role to be assistant, got %s", resp.Choices[0].Delta.Role)
	}

	if resp.Choices[0].Delta.Content != " I" {
		t.Errorf("Expected content to be ' I', got %s", resp.Choices[0].Delta.Content)
	}

	if resp.Choices[0].Message.Role != AssistantRole {
		t.Errorf("Expected role to be assistant, got %s", resp.Choices[0].Message.Role)
	}

	if resp.Choices[0].Message.Content != "Hello there, user 2317" {
		t.Errorf("Expected content to be 'Hello there, user 2317', got %s", resp.Choices[0].Message.Content)
	}

	if resp.Choices[0].FinishReason != "stop" {
		t.Errorf("Expected finish reason to be stop, got %s", resp.Choices[0].FinishReason)
	}

	if resp.Usage.PromptTokens != 15 {
		t.Errorf("Expected prompt tokens to be 15, got %d", resp.Usage.PromptTokens)
	}

	if resp.Usage.CompletionTokens != 22 {
		t.Errorf("Expected completion tokens to be 22, got %d", resp.Usage.CompletionTokens)
	}

	if resp.Usage.TotalTokens != 37 {
		t.Errorf("Expected total tokens to be 37, got %d", resp.Usage.TotalTokens)
	}
}

func TestResponse_Marshal(t *testing.T) {
	resp := Response{
		ID:      "chatcmpl-uefukdhy3krmm74ri9qk2f",
		Object:  "chat.completion.chunk",
		Created: 1088474663,
		Model:   "deepseek-coder-6.7b-instruct.Q4_K_S.gguf",
		Choices: []Choice{{
			Index: 0,
			Delta: Message{
				Role:    AssistantRole,
				Content: " I",
			},
			Message: Message{
				Role:    AssistantRole,
				Content: "Hello there, user 2317",
			},
			FinishReason: "stop",
		}},
		Usage: Usage{
			PromptTokens:     15,
			CompletionTokens: 22,
			TotalTokens:      37,
		},
	}
	data, err := resp.Marshal()
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	if !bytes.Equal(data, []byte(`{"id":"chatcmpl-uefukdhy3krmm74ri9qk2f","object":"chat.completion.chunk","created":1088474663,"model":"deepseek-coder-6.7b-instruct.Q4_K_S.gguf","choices":[{"index":0,"delta":{"role":"assistant","content":" I"},"message":{"role":"assistant","content":"Hello there, user 2317"},"finish_reason":"stop"}],"usage":{"prompt_tokens":15,"completion_tokens":22,"total_tokens":37}}`)) {
		t.Errorf("Expected %s, got %s", responseTest, data)
	}
}
