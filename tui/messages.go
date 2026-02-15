package tui

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"go-tui/config"
)

// Styles are defined in theme.go

func renderMessages(messages []ChatEntry, perm *PermissionPrompt, width int, md *MarkdownRenderer) string {
	if len(messages) == 0 && perm == nil {
		return "Welcome! Type a message and press Enter to send."
	}

	var sb strings.Builder
	// Debug: log entry types to verify interleaving
	for idx, e := range messages {
		log.Printf("messages[%d]: Type=%d Role=%q Command=%q", idx, e.Type, e.Role, e.Command)
	}
	i := 0
	for i < len(messages) {
		entry := messages[i]

		var rendered string

		// Check if we can group this and next entries
		if entry.Type == EntryToolCall && canGroupToolCall(entry) && i+1 < len(messages) {
			groupEnd := findGroupEnd(messages, i)
			if groupEnd > i {
				rendered = renderGroupedToolCalls(messages[i : groupEnd+1])
				rendered = strings.TrimSpace(rendered)
				sb.WriteString(rendered)
				i = groupEnd + 1
				continue
			}
		}

		// Render individual entry
		switch entry.Type {
		case EntryToolCall:
			rendered = renderToolCallEntry(entry)

		case EntryMessage:
			switch entry.Role {
			case "user":
				rendered = userMessageStyle.Render(entry.Content)
			case "assistant":
				if md != nil && isMarkdown(entry.Content) {
					if r, err := md.Render(entry.Content); err == nil {
						rendered = r
					} else {
						rendered = entry.Content
					}
				} else {
					rendered = entry.Content
				}
			default:
				rendered = fmt.Sprintf("%s: %s", entry.Role, entry.Content)
			}

		case EntryError:
			rendered = errorStyle.Render("Error: " + entry.Content)
		}

		rendered = strings.Trim(rendered, "\n")

		// Extra blank line before user/assistant messages to separate conversation turns
		if entry.Type == EntryMessage && (entry.Role == "user" || entry.Role == "assistant") && sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(rendered)
		if entry.Type != EntryToolCall {
			sb.WriteString("\n")
		}

		// prevType = entry.Type

		i++
	}

	// Show permission prompt inline
	if perm != nil {
		if len(messages) > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(perm.View(width))
	}

	return sb.String()
}

func renderToolCallEntry(entry ChatEntry) string {
	header := formatCommand(entry.Command)
	bullet := toolBulletStyle.Render("⏺ ") + toolCmdStyle.Render(header)

	if entry.Denied {
		return bullet + " " + deniedStyle.Render("User declined")
	}

	if entry.Diff != nil {
		return bullet + "\n" + indentBlock(renderDiff(*entry.Diff))
	}

	name, _ := splitCommand(entry.Command)

	// For read_file, just show the bullet header
	if name == "read_file" {
		return bullet
	}

	// Default: show bullet + indented result
	result := entry.Result
	maxResultLines := config.MaxResultLines
	lines := strings.Split(result, "\n")
	if len(lines) > maxResultLines {
		result = strings.Join(lines[:maxResultLines], "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-maxResultLines)
	}
	return bullet + "\n" + indentBlock(result)
}

// indentBlock renders content with ⎿ on the first line, then spaces for the rest.
func indentBlock(content string) string {
	content = strings.TrimRight(content, "\n ")
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return ""
	}
	bar := toolIndentStyle.Render("⎿ ")
	pad := "  "
	out := bar + lines[0]
	for _, line := range lines[1:] {
		out += "\n" + pad + line
	}
	return out
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
