package tools

import (
	"encoding/json"
	"os"
	"path/filepath"

	"go-tui/config"
	"go-tui/llm"
	"go-tui/lsp"
)

type WriteFileArgs struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

var WriteFileTool = llm.Tool{
	Type: "function",
	Function: llm.ToolFunction{
		Name:        "write_file",
		Description: "Create or overwrite a file with the given content. Parent directories are created automatically. NOTE: The system runs LSP diagnostics on the new content and provides feedback in the result. If LSP feedback indicates errors, you should fix them in subsequent tool calls.",
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

type WriteFileResult struct {
	FilePath    string `json:"file_path"`
	OldContent  string `json:"old_content"`
	NewContent  string `json:"new_content"`
	IsNewFile   bool   `json:"is_new_file"`
	LSPFeedback string `json:"lsp_feedback,omitempty"`
}

func ExecuteWriteFile(argsJSON string, workingDir string) (string, error) {
	var args WriteFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", NewToolErrorWithDetails(ErrInvalidArguments, "invalid arguments", err.Error())
	}

	if args.FilePath == "" {
		return "", NewToolError(ErrMissingField, "file_path is required")
	}

	path := args.FilePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(workingDir, path)
	}

	// Read existing content before overwriting
	oldContent := ""
	isNewFile := true
	if existing, err := os.ReadFile(path); err == nil {
		oldContent = string(existing)
		isNewFile = false
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, config.DirPermissions); err != nil {
		return "", NewToolErrorWithDetails(ErrFileWrite, "failed to create directory", err.Error())
	}

	if err := os.WriteFile(path, []byte(args.Content), config.FilePermissions); err != nil {
		return "", NewToolErrorWithDetails(ErrFileWrite, "failed to write file", err.Error())
	}

	result := WriteFileResult{
		FilePath:   args.FilePath,
		OldContent: oldContent,
		NewContent: args.Content,
		IsNewFile:  isNewFile,
	}

	if lsp.DefaultManager != nil {
		if diags, diagErr := lsp.DefaultManager.CheckFile(path, args.Content); diagErr == nil {
			if feedback := lsp.FormatDiagnostics(path, diags); feedback != "" {
				result.LSPFeedback = "LSP Feedback: " + feedback
			}
		}
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", NewToolErrorWithDetails(ErrJSONMarshal, "failed to marshal result", err.Error())
	}
	
	return string(resultJSON), nil
}
