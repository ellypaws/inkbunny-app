package llm

import (
	"net/url"
	"testing"
)

var defaultRequest = Request{
	Messages: []Message{{
		Role:    SystemRole,
		Content: "Just say hello!",
	}, UserMessage("Say hello!")},
	Temperature: 1.0,
	MaxTokens:   10,
	Stream:      false,
}

func TestLocalhost(t *testing.T) {
	infer, err := Localhost().Infer(&defaultRequest)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
		return
	}

	if infer.Choices[0].Message.Content == "" {
		t.Errorf("Expected content to be non-empty, got empty")
	}

	t.Logf("Infer: %+v", infer)
}

func TestStream(t *testing.T) {
	accumulatedTokens := make(chan *Response)
	go MonitorTokens(accumulatedTokens)
	infer, err := Localhost().Infer(&Request{
		Messages: []Message{{
			Role:    SystemRole,
			Content: "Write an essay",
		}, UserMessage("Write a generic API library for Inkbunny")},
		Temperature:   1.0,
		MaxTokens:     128,
		Stream:        true,
		StreamChannel: accumulatedTokens,
	})
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
		return
	}

	if infer.Choices[0].Message.Content == "" {
		t.Errorf("Expected content to be non-empty, got empty")
	}

	t.Logf("Infer: %+v", infer)
}

func TestOffline(t *testing.T) {
	_, err := Config{
		Host:   "localhost",
		APIKey: "",
		Endpoint: url.URL{
			Scheme: "http",
			Host:   "localhost",
			Path:   "/v1/chat/completions/FAIL",
		},
	}.Infer(&Request{
		Messages: []Message{
			DefaultSystem,
			UserMessage("Say hello!"),
		},
		Temperature: 1.0,
		MaxTokens:   1024,
		Stream:      false,
	})
	if err == nil {
		t.Errorf("Expected error, got nil")
		return
	}

	t.Logf("Got expected value: %s", err)
}
