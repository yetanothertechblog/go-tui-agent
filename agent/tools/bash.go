package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

const bashTimeout = 30 * time.Second

type BashArgs struct {
	Command string `json:"command"`
}

func init() {
	Register(Typed[BashArgs]{
		ToolName:        "bash",
		ToolDescription: "Execute a bash command and return the output. Use this to run shell commands, read files, list directories, search code, etc.",
		ToolSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"command": {
					"type": "string",
					"description": "The bash command to execute"
				}
			},
			"required": ["command"]
		}`),
		Run: executeBash,
	})
}

func executeBash(args BashArgs, workingDir string) (ToolResult, error) {
	if args.Command == "" {
		return ToolResult{}, NewToolError(ErrMissingField, "command is required")
	}

	cmd := exec.Command("bash", "-c", args.Command)
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
	case <-time.After(bashTimeout):
		cmd.Process.Kill()
		return ToolResult{}, fmt.Errorf("command timed out after 30s")
	}
}
