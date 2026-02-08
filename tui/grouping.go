package tui

import (
	"fmt"
	"strings"
)

func canGroupToolCall(entry ChatEntry) bool {
	if entry.Type != EntryToolCall || entry.Denied {
		return false
	}
	
	name, _ := splitCommand(entry.Command)
	return name == "read_file" || name == "list_files" || name == "search"
}

func findGroupEnd(messages []ChatEntry, start int) int {
	end := start
	for i := start + 1; i < len(messages); i++ {
		entry := messages[i]
		if entry.Type != EntryToolCall || entry.Denied || !canGroupToolCall(entry) {
			break
		}
		end = i
	}
	return end
}

func countOperations(group []ChatEntry) (reads, searches, lists int) {
	for _, entry := range group {
		name, _ := splitCommand(entry.Command)
		switch name {
		case "read_file":
			reads++
		case "search":
			searches++
		case "list_files":
			lists++
		}
	}
	return reads, searches, lists
}

func renderGroupedToolCalls(group []ChatEntry, boxWidth int) string {
	reads, searches, lists := countOperations(group)
	
	var parts []string
	if reads > 0 {
		parts = append(parts, fmt.Sprintf("📖 Read %d files", reads))
	}
	if searches > 0 {
		parts = append(parts, fmt.Sprintf("🔍 Searched for %d patterns", searches))
	}
	if lists > 0 {
		parts = append(parts, fmt.Sprintf("📁 Listed %d directories", lists))
	}
	
	content := strings.Join(parts, ", ")
	return toolBoxStyle.Width(boxWidth).Render(content)
}