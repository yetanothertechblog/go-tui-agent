package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-tui/config"
	"go-tui/lsp"
)

type EditFileArgs struct {
	FilePath  string `json:"file_path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

type EditFileResult struct {
	FilePath    string `json:"file_path"`
	OldString   string `json:"old_string"`
	NewString   string `json:"new_string"`
	LSPFeedback string `json:"lsp_feedback,omitempty"`
}

func init() {
	Register(Typed[EditFileArgs]{
		ToolName:        "edit_file",
		ToolDescription: "Edit a file by replacing an exact string match. The old_string must be unique in the file. Read the file first to get the exact text. NOTE: After editing, the system runs LSP diagnostics and provides feedback in the result. If LSP feedback indicates errors, you should fix them in subsequent tool calls.",
		ToolSchema: json.RawMessage(`{
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
		Run: executeEditFile,
	})
}

func executeEditFile(args EditFileArgs, workingDir string) (ToolResult, error) {
	if args.FilePath == "" {
		return ToolResult{}, NewToolError(ErrMissingField, "file_path is required")
	}
	if args.OldString == "" {
		return ToolResult{}, NewToolError(ErrMissingField, "old_string is required")
	}
	if args.OldString == args.NewString {
		return ToolResult{}, NewToolError(ErrIdenticalContent, "old_string and new_string are identical. No changes needed. Do not retry this edit.")
	}

	path := args.FilePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(workingDir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{}, NewToolErrorWithDetails(ErrFileNotFound, "file not found", err.Error())
	}

	content := string(data)
	count := strings.Count(content, args.OldString)

	if count == 0 {
		return ToolResult{}, NewToolError(ErrStringNotFound, "old_string not found in file. The file may have already been edited. Use read_file to check the current content before retrying.")
	}
	if count > 1 {
		return ToolResult{}, NewToolErrorWithDetails(ErrStringNotUnique, "old_string found multiple times",
			fmt.Sprintf("found %d times, must be unique. Include more surrounding context.", count))
	}

	newContent := strings.Replace(content, args.OldString, args.NewString, 1)

	if err := os.WriteFile(path, []byte(newContent), config.FilePermissions); err != nil {
		return ToolResult{}, NewToolErrorWithDetails(ErrFileWrite, "failed to write file", err.Error())
	}

	editResult := EditFileResult{
		FilePath:  args.FilePath,
		OldString: args.OldString,
		NewString: args.NewString,
	}

	if lsp.DefaultManager != nil {
		if diags, err := lsp.DefaultManager.CheckFile(path, newContent); err == nil {
			if feedback := lsp.FormatDiagnostics(path, diags); feedback != "" {
				editResult.LSPFeedback = "LSP Feedback: " + feedback
			}
		}
	}

	resultJSON, err := json.Marshal(editResult)
	if err != nil {
		return ToolResult{}, NewToolErrorWithDetails(ErrJSONMarshal, "failed to marshal result", err.Error())
	}

	return ToolResult{Output: string(resultJSON)}, nil
}
