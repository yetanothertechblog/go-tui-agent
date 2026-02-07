package tools

import "go-tui/llm"

var All = []llm.Tool{
	ReadFileTool,
	ListFilesTool,
	EditFileTool,
	WriteFileTool,
	BashTool,
}

func Execute(name string, argsJSON string, workingDir string) string {
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
	default:
		return "error: unknown tool: " + name
	}
}
