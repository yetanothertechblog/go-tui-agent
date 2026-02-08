package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-tui/config"
	"go-tui/llm"
)

type EditFileArgs struct {
	FilePath  string `json:"file_path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

type EditFileResult struct {
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

func ExecuteEditFile(argsJSON string, workingDir string) (string, error) {
	var args EditFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", NewToolErrorWithDetails(ErrInvalidArguments, "invalid arguments", err.Error())
	}

	if args.FilePath == "" {
		return "", NewToolError(ErrMissingField, "file_path is required")
	}
	if args.OldString == "" {
		return "", NewToolError(ErrMissingField, "old_string is required")
	}
	if args.OldString == args.NewString {
		return "", NewToolError(ErrIdenticalContent, "old_string and new_string are identical. No changes needed. Do not retry this edit.")
	}

	path := args.FilePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(workingDir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", NewToolErrorWithDetails(ErrFileNotFound, "file not found", err.Error())
	}

	content := string(data)
	count := strings.Count(content, args.OldString)

	if count == 0 {
		return "", NewToolError(ErrStringNotFound, "old_string not found in file. The file may have already been edited. Use read_file to check the current content before retrying.")
	}
	if count > 1 {
		return "", NewToolErrorWithDetails(ErrStringNotUnique, "old_string found multiple times",
			fmt.Sprintf("found %d times, must be unique. Include more surrounding context.", count))
	}

	newContent := strings.Replace(content, args.OldString, args.NewString, 1)

	if err := os.WriteFile(path, []byte(newContent), config.FilePermissions); err != nil {
		return "", NewToolErrorWithDetails(ErrFileWrite, "failed to write file", err.Error())
	}

	editResult := EditFileResult{
		FilePath:  args.FilePath,
		OldString: args.OldString,
		NewString: args.NewString,
	}
	resultJSON, err := json.Marshal(editResult)
	if err != nil {
		return "", NewToolErrorWithDetails(ErrJSONMarshal, "failed to marshal result", err.Error())
	}
	return string(resultJSON), nil
}
