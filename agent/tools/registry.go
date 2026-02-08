package tools

import (
	"fmt"

	"go-tui/llm"
)

var All = []llm.Tool{
	ReadFileTool,
	ListFilesTool,
	EditFileTool,
	WriteFileTool,
	BashTool,
	SearchTool,
}

func Execute(name string, argsJSON string, workingDir string) (string, error) {
	switch name {
	case "read_file":
		return ExecuteReadFile(argsJSON, workingDir)
	case "list_files":
		return ExecuteListFiles(argsJSON, workingDir)
	case "edit_file":
		return ExecuteEditFile(argsJSON, workingDir)
	case "write_file":
		return ExecuteWriteFile(argsJSON, workingDir)
	case "bash":
		return ExecuteBash(argsJSON, workingDir)
	case "search":
		return ExecuteSearch(argsJSON, workingDir)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}
