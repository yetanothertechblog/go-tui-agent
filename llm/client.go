package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"go-tui/config"
)

const (
	apiURL    = config.APIURL
	modelName = config.ModelName
)

var apiKey string

func InitAPIKey() error {
	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("error loading .env file: %w", err)
	}
	apiKey = os.Getenv("ZAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ZAI_API_KEY not set in .env file")
	}
	return nil
}

func CallLLM(messages []Message, tools []Tool) (*LLMResult, error) {
	req := ChatRequest{
		Model:    modelName,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody bytes.Buffer
		errBody.ReadFrom(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, errBody.String())
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &LLMResult{
		Delta: chatResp.Choices[0].Message,
		Usage: chatResp.Usage,
	}, nil
}

func CallLLMStream(messages []Message, tools []Tool, onContent func(string, bool)) (*LLMResult, error) {
	req := ChatRequest{
		Model:    modelName,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody bytes.Buffer
		errBody.ReadFrom(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, errBody.String())
	}

	full := &Delta{}
	var usage *Usage
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk ChatResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if chunk.Usage != nil {
			usage = chunk.Usage
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta
		if delta == nil {
			continue
		}

		if delta.ReasoningContent != "" {
			full.ReasoningContent += delta.ReasoningContent
			onContent(delta.ReasoningContent, true)
		}
		if delta.Content != "" {
			full.Content += delta.Content
			onContent(delta.Content, false)
		}
		if len(delta.ToolCalls) > 0 {
			for _, tc := range delta.ToolCalls {
				if tc.ID != "" {
					full.ToolCalls = append(full.ToolCalls, tc)
				} else if len(full.ToolCalls) > 0 {
					last := &full.ToolCalls[len(full.ToolCalls)-1]
					last.Function.Arguments += tc.Function.Arguments
				}
			}
		}
	}

	return &LLMResult{Delta: full, Usage: usage}, scanner.Err()
}
