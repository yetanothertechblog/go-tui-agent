package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"go-tui/config"
)

var (
	toolCmdStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true)

	userMessageStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("240"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	deniedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

func renderMessages(messages []ChatEntry, perm *PermissionPrompt, width int, md *MarkdownRenderer) string {
	if len(messages) == 0 && perm == nil {
		return "Welcome! Type a message and press Enter to send."
	}

	boxWidth := width - config.BoxPadding
	if boxWidth < config.MinBoxWidth {
		boxWidth = config.MinBoxWidth
	}

	var sb strings.Builder
	i := 0
	for i < len(messages) {
		entry := messages[i]
		
		// Check if we can group this and next entries
		if entry.Type == EntryToolCall && canGroupToolCall(entry) && i+2 < len(messages) {
			groupEnd := findGroupEnd(messages, i)
			if groupEnd > i {
				sb.WriteString(renderGroupedToolCalls(messages[i:groupEnd+1], boxWidth))
				i = groupEnd + 1
				continue
			}
		}
		
		// Render individual entry
		switch entry.Type {
		case EntryToolCall:
			sb.WriteString(renderToolCallEntry(entry, boxWidth))

		case EntryMessage:
			switch entry.Role {
			case "user":
				sb.WriteString(userMessageStyle.Render(entry.Content))
			case "assistant":
				if md != nil && isMarkdown(entry.Content) {
					if rendered, err := md.Render(entry.Content); err == nil {
						sb.WriteString(rendered)
					} else {
						sb.WriteString(entry.Content)
					}
				} else {
					sb.WriteString(entry.Content)
				}
			default:
				sb.WriteString(fmt.Sprintf("%s: %s", entry.Role, entry.Content))
			}

		case EntryError:
			sb.WriteString(errorStyle.Render("Error: " + entry.Content))
		}

		i++
		if i < len(messages) {
			sb.WriteString("\n\n")
		}
	}

	// Show permission prompt inline
	if perm != nil {
		if len(messages) > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(perm.View(width))
	}

	return sb.String()
}

func renderToolCallEntry(entry ChatEntry, boxWidth int) string {
	if entry.Diff != nil {
		rendered := toolBoxStyle.Width(boxWidth).Render(renderDiff(*entry.Diff))
		if entry.Denied {
			return rendered + "\n" + deniedStyle.Render("denied")
		}
		return rendered
	}

	if entry.Denied {
		return toolCmdStyle.Render(formatCommand(entry.Command)) + " " + deniedStyle.Render("User declined")
	}

	header := formatCommand(entry.Command)
	name, _ := splitCommand(entry.Command)

	// For read_file, just show the header without content
	if name == "read_file" {
		return toolBoxStyle.Width(boxWidth).Render(toolCmdStyle.Render(header))
	}

	// Default: show formatted command + result
	result := entry.Result
	maxResultLines := config.MaxResultLines
	lines := strings.Split(result, "\n")
	if len(lines) > maxResultLines {
		result = strings.Join(lines[:maxResultLines], "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-maxResultLines)
	}
	content := toolCmdStyle.Render(header) + "\n" + result
	return toolBoxStyle.Width(boxWidth).Render(content)
}

// formatCommand turns "tool_name: {json}" into a human-readable string.
func formatCommand(command string) string {
	name, argsJSON := splitCommand(command)
	if argsJSON == "" {
		return command
	}

	var args map[string]json.RawMessage
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return command
	}

	str := func(key string) string {
		raw, ok := args[key]
		if !ok {
			return ""
		}
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return s
		}
		return ""
	}

	num := func(key string) (int, bool) {
		raw, ok := args[key]
		if !ok {
			return 0, false
		}
		var n int
		if json.Unmarshal(raw, &n) == nil {
			return n, true
		}
		return 0, false
	}

	var icon string
	switch name {
	case "read_file":
		icon = config.ReadIcon
	case "list_files":
		icon = config.ListIcon
	case "bash":
		icon = config.BashIcon
	case "search":
		icon = config.SearchIcon
	case "edit_file":
		icon = config.EditIcon
	case "write_file":
		icon = config.WriteIcon
	case "beads":
		icon = config.ToolIcon
	default:
		icon = config.ToolIcon
	}

	switch name {
	case "read_file":
		s := "Read: " + str("file_path")
		offset, hasOffset := num("offset")
		limit, hasLimit := num("limit")
		if hasOffset && hasLimit {
			s += fmt.Sprintf(" %d:%d", offset, offset+limit-1)
		} else if hasOffset {
			s += fmt.Sprintf(" from %d", offset)
		} else if hasLimit {
			s += fmt.Sprintf(" first %d lines", limit)
		}
		return icon + s
	case "list_files":
		path := str("path")
		if path == "" {
			path = "."
		}
		return icon + "List: " + path
	case "bash":
		return icon + "Bash: " + str("command")
	case "search":
		s := "Search: " + str("pattern")
		if p := str("path"); p != "" {
			s += " in " + p
		}
		return icon + s
	case "beads":
		s := "Beads: " + str("command")
		if a := str("args"); a != "" {
			s += " " + a
		}
		return icon + s
	default:
		return icon + command
	}
}

// splitCommand splits "tool_name: {json}" into name and argsJSON.
func splitCommand(command string) (string, string) {
	idx := strings.Index(command, ": ")
	if idx < 0 {
		return command, ""
	}
	return command[:idx], command[idx+2:]
}

