package agent

import (
	"fmt"
	"log"

	"go-tui/agent/tools"
	"go-tui/config"
	"go-tui/llm"

	tea "github.com/charmbracelet/bubbletea"
)

type ToolCallMsg struct {
	Command string
	Result  string
	Denied  bool
}

type ResponseMsg struct {
	Content string
	Err     error
	Denied  bool
}

type PermissionRequestMsg struct {
	ToolName string
	Args     string
}

type PermissionDecision int

const (
	PermissionAllow PermissionDecision = iota
	PermissionAlwaysAllow
	PermissionDeny
)

const systemPromptTemplate = `You are an expert coding assistant. You help users write, debug, and improve code.

Working directory: %s

Rules:
- Give concise, direct answers. Avoid unnecessary preamble.
- When showing code, include only the relevant parts unless the user asks for the full file.
- If a question is ambiguous, ask a brief clarifying question before answering.
- When fixing bugs, explain the root cause in one sentence, then show the fix.
- Prefer simple, readable solutions over clever ones.
- If you don't know something, say so. Don't guess.
- Use the tools available to you when needed.
- When reading files, use paths relative to the working directory unless an absolute path is given.`

const maxToolRounds = config.MaxToolRounds

type Agent struct {
	history      []llm.Message
	workingDir   string
	alwaysAllow  map[string]bool
	permissionCh chan PermissionDecision
	program      *tea.Program
}

func New(workingDir string) *Agent {
	return &Agent{
		workingDir:   workingDir,
		alwaysAllow:  make(map[string]bool),
		permissionCh: make(chan PermissionDecision),
	}
}

func (a *Agent) SetProgram(p *tea.Program) {
	a.program = p
}

func (a *Agent) RespondPermission(decision PermissionDecision) {
	a.permissionCh <- decision
}

func (a *Agent) requestPermission(toolName string, args string) PermissionDecision {
	if a.alwaysAllow[toolName] {
		return PermissionAllow
	}

	a.program.Send(PermissionRequestMsg{
		ToolName: toolName,
		Args:     args,
	})

	decision := <-a.permissionCh

	if decision == PermissionAlwaysAllow {
		a.alwaysAllow[toolName] = true
	}

	return decision
}

func (a *Agent) WorkingDir() string {
	return a.workingDir
}

func (a *Agent) History() []llm.Message {
	return a.history
}

func (a *Agent) SetHistory(history []llm.Message) {
	a.history = history
}

func (a *Agent) Send(userText string) tea.Cmd {
	a.history = append(a.history, llm.Message{
		Role:    "user",
		Content: userText,
	})

	return func() tea.Msg {
		consecutiveErrors := 0
		const maxConsecutiveErrors = 3

		for range maxToolRounds {
			messages := make([]llm.Message, 0, len(a.history)+1)
			messages = append(messages, llm.Message{
				Role:    "system",
				Content: fmt.Sprintf(systemPromptTemplate, a.workingDir),
			})
			messages = append(messages, a.history...)

			delta, err := llm.CallLLM(messages, tools.All)
			if err != nil {
				log.Printf("llm error: %v", err)
				return ResponseMsg{Err: err}
			}

			if len(delta.ToolCalls) == 0 {
				log.Printf("assistant response: %s", delta.Content)
				a.history = append(a.history, llm.Message{
					Role:    "assistant",
					Content: delta.Content,
				})
				return ResponseMsg{Content: delta.Content}
			}

			// Append assistant message with tool calls
			a.history = append(a.history, llm.Message{
				Role:      "assistant",
				ToolCalls: delta.ToolCalls,
			})

			// Execute each tool call with permission check
			for _, tc := range delta.ToolCalls {
				command := tc.Function.Name + ": " + tc.Function.Arguments

				decision := a.requestPermission(tc.Function.Name, tc.Function.Arguments)

				if decision == PermissionDeny {
					log.Printf("tool call denied: %s", tc.Function.Name)
					result := "Tool call denied by user."
					a.program.Send(ToolCallMsg{
						Command: command,
						Denied:  true,
					})
					a.history = append(a.history, llm.Message{
						Role:       "tool",
						Content:    result,
						ToolCallID: tc.ID,
					})

					// Break the loop and return to allow user to add explanation
					return ResponseMsg{Denied: true}
				}

				result, err := tools.Execute(tc.Function.Name, tc.Function.Arguments, a.workingDir)
				if err != nil {
					log.Printf("tool error: %v", err)
					consecutiveErrors++
					errMsg := err.Error()
					if consecutiveErrors >= maxConsecutiveErrors {
						errMsg += " (Too many consecutive errors. Stop retrying and tell the user what went wrong.)"
					}
					a.program.Send(ToolCallMsg{
						Command: command,
						Result:  errMsg,
					})
					a.history = append(a.history, llm.Message{
						Role:       "tool",
						Content:    errMsg,
						ToolCallID: tc.ID,
					})
					continue
				}

				consecutiveErrors = 0
				log.Printf("tool result: %.200s", result)
				a.program.Send(ToolCallMsg{
					Command: command,
					Result:  result,
				})
				a.history = append(a.history, llm.Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: tc.ID,
				})
			}
		}

		return ResponseMsg{Err: fmt.Errorf("tool call limit reached")}
	}
}
