package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type BeadsArgs struct {
	Command string `json:"command"`
	Args    string `json:"args"`
}

func init() {
	Register(Typed[BeadsArgs]{
		ToolName:        "beads",
		ToolDescription: "Manage task tracking using beads CLI. When dealing with complex tasks, break them down into small subtasks and use beads to track progress. Use 'bd ready' to find work ready to claim, 'bd create' to create new tasks (break complex tasks into smaller ones), 'bd list' to view all tasks, 'bd show' to view task details and dependencies, 'bd update' to modify task status, 'bd dep add' to add dependencies between tasks, and 'bd close' to complete tasks. Always break down large tasks into smaller, manageable subtasks.",
		ToolSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"command": {
					"type": "string",
					"description": "The beads command to execute (create, list, show, update, close, ready, dep add, etc.)"
				},
				"args": {
					"type": "string",
					"description": "Command arguments (e.g., 'Fix login bug' for create, 'go-tui-123' for show, '--status in_progress' for update, 'go-tui-123 go-tui-456' for dep add)",
					"default": ""
				}
			},
			"required": ["command"]
		}`),
		Run: executeBeads,
	})
}

func executeBeads(args BeadsArgs, workingDir string) (ToolResult, error) {
	if args.Command == "" {
		return ToolResult{}, NewToolError(ErrMissingField, "command is required")
	}

	cmdArgs := strings.Fields(args.Command)
	if args.Args != "" {
		cmdArgs = append(cmdArgs, strings.Fields(args.Args)...)
	}

	cmd := exec.Command("bd", cmdArgs...)
	cmd.Dir = workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		output := stdout.String()
		if stderr.Len() > 0 {
			if output != "" {
				output += "\n"
			}
			output += stderr.String()
		}
		if err != nil {
			if output != "" {
				output += "\n"
			}
			output += fmt.Sprintf("exit status: %v", err)
		}
		if output == "" {
			output = "(no output)"
		}
		return ToolResult{Output: output}, nil
	case <-time.After(10 * time.Second):
		cmd.Process.Kill()
		return ToolResult{}, fmt.Errorf("bd command timed out after 10s")
	}
}
