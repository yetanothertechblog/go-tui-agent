package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-tui/llm"
)

type ListFilesArgs struct {
	Path      string `json:"path,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`
}

var ListFilesTool = llm.Tool{
	Type: "function",
	Function: llm.ToolFunction{
		Name:        "list_files",
		Description: "List files and directories at the given path. Directories have a trailing '/'. Skips .git.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Directory to list (relative to working directory or absolute, default: working directory)"
				},
				"recursive": {
					"type": "boolean",
					"description": "List files recursively (default: false)"
				}
			}
		}`),
	},
}

func ExecuteListFiles(argsJSON string, workingDir string) string {
	var args ListFilesArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("error: invalid arguments: %v", err)
	}

	path := args.Path
	if path == "" {
		path = workingDir
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(workingDir, path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	if !info.IsDir() {
		return fmt.Sprintf("error: %s is not a directory", args.Path)
	}

	var sb strings.Builder

	if args.Recursive {
		filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.Name() == ".git" && info.IsDir() {
				return filepath.SkipDir
			}
			rel, _ := filepath.Rel(workingDir, p)
			name := rel
			if info.IsDir() {
				name += "/"
			}
			sb.WriteString(name + "\n")
			return nil
		})
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		for _, entry := range entries {
			if entry.Name() == ".git" {
				continue
			}
			name := entry.Name()
			if entry.IsDir() {
				name += "/"
			}
			sb.WriteString(name + "\n")
		}
	}

	return sb.String()
}
