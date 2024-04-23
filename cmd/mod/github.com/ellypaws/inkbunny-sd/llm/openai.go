package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Config stores the API host and key.
type Config struct {
	Host     string
	APIKey   string
	Endpoint url.URL
}

// inference makes a POST request to the OpenAI API with the given request data.
func (c Config) inference(r *Request) (*http.Response, error) {
	requestBytes, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", c.Endpoint.String(), bytes.NewBuffer(requestBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.APIKey)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	return resp, nil
}

func (c Config) Infer(request *Request) (Response, error) {
	resp, err := c.inference(request)
	if err != nil {
		return Response{}, fmt.Errorf("failed to make inference request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("inference request failed with status code %d", resp.StatusCode)
	}

	var response Response
	if request.Stream {
		if request.StreamChannel == nil {
			return Response{}, fmt.Errorf("streaming request requires a channel")
		}
		messages, err := handleStreamedResponse(resp, request.StreamChannel)
		if err != nil {
			return Response{}, fmt.Errorf("failed to handle streamed response: %w", err)
		}
		response = messages
	} else {
		response, err = handleResponse(resp)
		if err != nil {
			return Response{}, fmt.Errorf("failed to handle response: %w", err)
		}
	}

	return response, nil
}

// TokenCount returns the number of tokens in the message.
// Implement this based on your message structure.
func (m Response) TokenCount() int {
	return len(m.Choices)
}

// handleResponse parses the HTTP response from the inference API call using io.ReadAll.
func handleResponse(response *http.Response) (Response, error) {
	defer response.Body.Close()

	// Use io.ReadAll to read the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Response{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var messages Response
	if err := json.Unmarshal(body, &messages); err != nil {
		return Response{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return messages, nil
}

// handleStreamedResponse processes streamed responses, sending each message through the response channel.
// It follows an SSE (a Server-Sent Events) format which prepends each message with "data: ".
//
//	data: {"id":"chatcmpl-uefukdhy3krmm74ri9qk2f","object":"chat.completion.chunk","created":1709571416,"model":"deepseek-coder-6.7b-instruct.Q4_K_S.gguf","choices":[{"index":0,"delta":{"role":"assistant","content":"?"},"finish_reason":null}]}
//	data: {"id":"chatcmpl-uefukdhy3krmm74ri9qk2f","object":"chat.completion.chunk","created":1709571416,"model":"deepseek-coder-6.7b-instruct.Q4_K_S.gguf","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}
//	data: [DONE]
func handleStreamedResponse(response *http.Response, responseChan chan<- *Response) (Response, error) {
	defer response.Body.Close()
	reader := bufio.NewReader(response.Body)

	var responses []Response
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				close(responseChan) // Close channel on EOF
				break
			}
			return Response{}, fmt.Errorf("failed to read line: %w", err)
		}

		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}

		line = bytes.TrimPrefix(line, []byte("data: "))

		if bytes.Equal(line, []byte("[DONE]")) {
			continue
		}

		var r Response
		if err := json.Unmarshal(line, &r); err != nil {
			return Response{}, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		// Send the token count for the message through the channel
		responseChan <- &r

		responses = append(responses, r)
	}

	if len(responses) == 0 {
		return Response{}, fmt.Errorf("no responses received")
	}

	var accumulatedTokens strings.Builder
	for _, r := range responses {
		accumulatedTokens.WriteString(r.Choices[0].Delta.Content)
	}

	out := responses[len(responses)-1]

	out.Choices[0].Message.Content = out.Choices[0].Delta.Content
	out.Choices[0].Message.Content = accumulatedTokens.String()

	return out, nil
}

func MonitorTokens(responseChan <-chan *Response) {
	var accumulatedTokens int
	for r := range responseChan {
		accumulatedTokens += r.TokenCount()
		fmt.Printf("\rAccumulated tokens: %d", accumulatedTokens)
		fmt.Print("\033[K") // Clear the line
	}
	fmt.Printf("\nStream ended.\n")
}

type AvailableModels struct {
	Data   []Datum `json:"data"`
	Object string  `json:"object"`
}

type Datum struct {
	ID         string       `json:"id"`
	Object     string       `json:"object"`
	OwnedBy    string       `json:"owned_by"`
	Permission []Permission `json:"permission"`
}

type Permission struct{}

func (c Config) AvailableModels() ([]string, error) {
	resp, err := http.Get(c.Endpoint.String())
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}

	var models AvailableModels
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	var modelIDs []string
	for _, model := range models.Data {
		modelIDs = append(modelIDs, model.ID)
	}

	return modelIDs, nil
}
