package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-tui/llm"
)

type EditFileArgs struct {
	FilePath  string `json:"file_path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

var EditFileTool = llm.Tool{
	Type: "function",
	Function: llm.ToolFunction{
		Name:        "edit_file",
		Description: "Edit a file by replacing an exact string match. The old_string must be unique in the file. Read the file first to get the exact text.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"file_path": {
					"type": "string",
					"description": "Path to the file to edit (relative to working directory or absolute)"
				},
				"old_string": {
					"type": "string",
					"description": "The exact text to find and replace (must be unique in the file)"
				},
				"new_string": {
					"type": "string",
					"description": "The replacement text"
				}
			},
			"required": ["file_path", "old_string", "new_string"]
		}`),
	},
}

func ExecuteEditFile(argsJSON string, workingDir string) string {
	var args EditFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("error: invalid arguments: %v", err)
	}

	if args.FilePath == "" {
		return "error: file_path is required"
	}
	if args.OldString == "" {
		return "error: old_string is required"
	}
	if args.OldString == args.NewString {
		return "error: old_string and new_string are identical"
	}

	path := args.FilePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(workingDir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	content := string(data)
	count := strings.Count(content, args.OldString)

	if count == 0 {
		return "error: old_string not found in file"
	}
	if count > 1 {
		return fmt.Sprintf("error: old_string found %d times, must be unique. Include more surrounding context.", count)
	}

	newContent := strings.Replace(content, args.OldString, args.NewString, 1)

	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	return fmt.Sprintf("OK: edited %s", args.FilePath)
}
