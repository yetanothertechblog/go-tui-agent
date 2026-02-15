package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ListFilesArgs struct {
	Path      string `json:"path,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`
}

func init() {
	Register(Typed[ListFilesArgs]{
		ToolName:        "list_files",
		ToolDescription: "List files and directories at the given path. Directories have a trailing '/'. Skips .git.",
		ToolSchema: json.RawMessage(`{
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
		Run: executeListFiles,
	})
}

func executeListFiles(args ListFilesArgs, workingDir string) (ToolResult, error) {
	path := args.Path
	if path == "" {
		path = workingDir
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(workingDir, path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return ToolResult{}, NewToolErrorWithDetails(ErrFileNotFound, "path not found", err.Error())
	}
	if !info.IsDir() {
		return ToolResult{}, NewToolError(ErrInvalidArguments, fmt.Sprintf("%s is not a directory", args.Path))
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
			return ToolResult{}, NewToolErrorWithDetails(ErrFileNotFound, "failed to read directory", err.Error())
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

	return ToolResult{Output: sb.String()}, nil
}
