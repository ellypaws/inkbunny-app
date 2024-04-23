package main

import (
	"github.com/ellypaws/inkbunny-sd/llm"
	"log"
)

func main() {
	accumulatedTokens := make(chan *llm.Response)
	go llm.MonitorTokens(accumulatedTokens)
	infer, err := llm.Localhost().Infer(&llm.Request{
		Messages: []llm.Message{{
			Role:    llm.SystemRole,
			Content: "Write an essay",
		}, llm.UserMessage("Write a generic API library for Inkbunny")},
		Temperature:   1.0,
		MaxTokens:     1024,
		Stream:        true,
		StreamChannel: accumulatedTokens,
	})
	if err != nil {
		log.Fatalf("Expected no error, got %s", err)
		return
	}

	if infer.Choices[0].Message.Content == "" {
		log.Fatalf("Expected content to be non-empty, got empty")
		return
	}

	log.Printf("Infer: %+v", infer)
}
