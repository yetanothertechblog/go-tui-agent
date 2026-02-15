package tools

import (
	"encoding/json"
	"fmt"

	"go-tui/llm"
)

// ToolResult replaces bare string returns from tool execution.
type ToolResult struct {
	Output string
}

// ToolImpl is the non-generic interface so the registry can hold []ToolImpl.
type ToolImpl interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Execute(argsJSON string, workingDir string) (ToolResult, error)
}

// Typed is a generic adapter that handles json.Unmarshal once.
type Typed[A any] struct {
	ToolName        string
	ToolDescription string
	ToolSchema      json.RawMessage
	Run             func(args A, workingDir string) (ToolResult, error)
}

func (t Typed[A]) Name() string              { return t.ToolName }
func (t Typed[A]) Description() string        { return t.ToolDescription }
func (t Typed[A]) Schema() json.RawMessage    { return t.ToolSchema }

func (t Typed[A]) Execute(argsJSON string, workingDir string) (ToolResult, error) {
	var args A
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return ToolResult{}, fmt.Errorf("invalid arguments: %w", err)
	}
	return t.Run(args, workingDir)
}

// ToLLMTool converts a ToolImpl to the wire format used by the LLM client.
func ToLLMTool(t ToolImpl) llm.Tool {
	return llm.Tool{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Schema(),
		},
	}
}
