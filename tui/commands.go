package tui

import (
	"log"
	"strings"
	"sync/atomic"
	"time"

	"go-tui/agent"
	"go-tui/agent/tools"
	"go-tui/llm"

	tea "github.com/charmbracelet/bubbletea"
)

// Messages

type LLMResponseMsg struct {
	Content   string
	ToolCalls []llm.ToolCall
	Usage     *llm.Usage
	Err       error
}

type ToolResultMsg struct {
	ToolCallID string
	ToolName   string
	Args       string
	Result     string
	Err        error
}

// StreamTokenCountMsg is sent periodically during streaming to update the token counter.
type StreamTokenCountMsg struct {
	Count    int
	Thinking bool
	ch       <-chan tea.Msg
}

type PermissionDecision int

const (
	PermissionAllow PermissionDecision = iota
	PermissionAlwaysAllow
	PermissionDeny
)

// Cmd factories

func callLLM(a *agent.Agent, history []llm.Message) tea.Cmd {
	messages := make([]llm.Message, 0, len(history)+1)
	messages = append(messages, llm.Message{
		Role:    "system",
		Content: a.SystemPrompt(),
	})
	messages = append(messages, history...)

	ch := make(chan tea.Msg, 1000)

	go func() {
		defer close(ch)

		var wordCount int64
		var thinking int32

		onContent := func(content string, isThinking bool) {
			if isThinking {
				atomic.StoreInt32(&thinking, 1)
			} else {
				atomic.StoreInt32(&thinking, 0)
			}
			words := int64(len(strings.Fields(content)))
			total := atomic.AddInt64(&wordCount, words)
			estimated := int(float64(total) * 0.75)
			select {
			case ch <- StreamTokenCountMsg{
				Count:    estimated,
				Thinking: atomic.LoadInt32(&thinking) == 1,
				ch:       ch,
			}:
			default:
			}
		}

		result, err := llm.CallLLMStream(messages, tools.All(), onContent)
		if err != nil {
			log.Printf("llm error: %v", err)
			ch <- LLMResponseMsg{Err: err}
			return
		}
		ch <- LLMResponseMsg{
			Content:   result.Delta.Content,
			ToolCalls: result.Delta.ToolCalls,
			Usage:     result.Usage,
		}
	}()

	return waitForStream(ch)
}

func waitForStream(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(100 * time.Millisecond)
		var latest tea.Msg
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return latest
				}
				latest = msg
				if _, isToken := msg.(StreamTokenCountMsg); !isToken {
					return msg
				}
			default:
				if latest != nil {
					return latest
				}
				msg, ok := <-ch
				if !ok {
					return nil
				}
				return msg
			}
		}
	}
}

func executeTool(a *agent.Agent, tc llm.ToolCall) tea.Cmd {
	name := tc.Function.Name
	args := tc.Function.Arguments
	id := tc.ID

	return func() tea.Msg {
		result, err := a.ExecuteTool(name, args)
		if err != nil {
			log.Printf("tool error: %v", err)
			return ToolResultMsg{
				ToolCallID: id,
				ToolName:   name,
				Args:       args,
				Result:     err.Error(),
				Err:        err,
			}
		}
		log.Printf("tool result: %.200s", result.Output)
		return ToolResultMsg{
			ToolCallID: id,
			ToolName:   name,
			Args:       args,
			Result:     result.Output,
		}
	}
}
