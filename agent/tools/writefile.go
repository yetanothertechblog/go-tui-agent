package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go-tui/llm"
)

type WriteFileArgs struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

var WriteFileTool = llm.Tool{
	Type: "function",
	Function: llm.ToolFunction{
		Name:        "write_file",
		Description: "Create or overwrite a file with the given content. Parent directories are created automatically.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"file_path": {
					"type": "string",
					"description": "Path to the file to write (relative to working directory or absolute)"
				},
				"content": {
					"type": "string",
					"description": "The full content to write to the file"
				}
			},
			"required": ["file_path", "content"]
		}`),
	},
}

func ExecuteWriteFile(argsJSON string, workingDir string) string {
	var args WriteFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("error: invalid arguments: %v", err)
	}

	if args.FilePath == "" {
		return "error: file_path is required"
	}

	path := args.FilePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(workingDir, path)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	if err := os.WriteFile(path, []byte(args.Content), 0o644); err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	return fmt.Sprintf("OK: wrote %s", args.FilePath)
}
