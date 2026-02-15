package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-tui/config"
)

const maxDefaultLines = config.MaxDefaultLines

type ReadFileArgs struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

func init() {
	Register(Typed[ReadFileArgs]{
		ToolName:        "read_file",
		ToolDescription: "Read the contents of a file. Returns lines with line numbers. Use offset and limit to paginate large files.",
		ToolSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"file_path": {
					"type": "string",
					"description": "Path to the file to read (relative to working directory or absolute)"
				},
				"offset": {
					"type": "integer",
					"description": "Line number to start reading from (1-based, default 1)"
				},
				"limit": {
					"type": "integer",
					"description": "Maximum number of lines to read (default 2000)"
				}
			},
			"required": ["file_path"]
		}`),
		Run: executeReadFile,
	})
}

func executeReadFile(args ReadFileArgs, workingDir string) (ToolResult, error) {
	if args.FilePath == "" {
		return ToolResult{}, NewToolError(ErrMissingField, "file_path is required")
	}

	path := args.FilePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(workingDir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{}, NewToolErrorWithDetails(ErrFileNotFound, "file not found", err.Error())
	}

	lines := strings.Split(string(data), "\n")
	totalLines := len(lines)

	offset := args.Offset
	if offset < 1 {
		offset = 1
	}
	if offset > totalLines {
		return ToolResult{}, NewToolErrorWithDetails(ErrInvalidArguments, "offset exceeds file length",
			fmt.Sprintf("offset %d exceeds %d lines", offset, totalLines))
	}

	limit := args.Limit
	if limit <= 0 {
		limit = maxDefaultLines
	}

	startIdx := offset - 1
	endIdx := startIdx + limit
	if endIdx > totalLines {
		endIdx = totalLines
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("File: %s (%d total lines, showing %d-%d)\n\n", args.FilePath, totalLines, offset, endIdx))
	for i := startIdx; i < endIdx; i++ {
		sb.WriteString(fmt.Sprintf("%4d: %s\n", i+1, lines[i]))
	}

	return ToolResult{Output: sb.String()}, nil
}
