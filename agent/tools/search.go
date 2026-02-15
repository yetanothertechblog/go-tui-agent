package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type SearchArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
}

func init() {
	Register(Typed[SearchArgs]{
		ToolName:        "search",
		ToolDescription: "Search for a string pattern in files using grep. Returns matching lines with file paths and line numbers.",
		ToolSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"pattern": {
					"type": "string",
					"description": "The string pattern to search for"
				},
				"path": {
					"type": "string",
					"description": "Directory or file path to search in (default: current directory)"
				}
			},
			"required": ["pattern"]
		}`),
		Run: executeSearch,
	})
}

func executeSearch(args SearchArgs, workingDir string) (ToolResult, error) {
	if args.Pattern == "" {
		return ToolResult{}, NewToolError(ErrMissingField, "pattern is required")
	}

	searchPath := args.Path
	if searchPath == "" {
		searchPath = workingDir
	} else if !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(workingDir, searchPath)
	}

	if _, err := os.Stat(searchPath); os.IsNotExist(err) {
		return ToolResult{}, NewToolErrorWithDetails(ErrFileNotFound, "path does not exist", args.Path)
	}

	cmd := exec.Command("grep",
		"-rn",
		"--include=*.go",
		"--include=*.js",
		"--include=*.ts",
		"--include=*.tsx",
		"--include=*.jsx",
		"--include=*.py",
		"--include=*.rs",
		"--include=*.java",
		"--include=*.c",
		"--include=*.h",
		"--include=*.cpp",
		"--include=*.md",
		"--include=*.txt",
		"--include=*.yaml",
		"--include=*.yml",
		"--include=*.toml",
		"--include=*.json",
		"--include=*.html",
		"--include=*.css",
		"--include=*.sh",
		"--include=*.mod",
		"--include=*.sum",
		"--exclude-dir=.git",
		"--exclude-dir=node_modules",
		"--exclude-dir=vendor",
		"--exclude-dir=log",
		"--exclude-dir=conversations",
		args.Pattern,
		searchPath,
	)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return ToolResult{Output: "No matches found."}, nil
			}
		}
		return ToolResult{}, fmt.Errorf("grep failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(output), "\n"), "\n")

	for i, line := range lines {
		if rel, err := filepath.Rel(workingDir, line); err == nil {
			lines[i] = rel
		} else if strings.HasPrefix(line, workingDir) {
			lines[i] = strings.TrimPrefix(line, workingDir+"/")
		}
		if len(lines[i]) > 200 {
			lines[i] = lines[i][:200] + "..."
		}
	}

	total := len(lines)
	if total > 30 {
		lines = lines[:30]
		lines = append(lines, fmt.Sprintf("... (%d total matches, showing first 30)", total))
	}

	return ToolResult{Output: strings.Join(lines, "\n")}, nil
}
