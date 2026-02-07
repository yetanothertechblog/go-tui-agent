package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	toolBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1).
			Foreground(lipgloss.Color("243"))

	toolCmdStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	deniedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

func renderMessages(messages []ChatEntry, perm *PermissionPrompt, width int) string {
	if len(messages) == 0 && perm == nil {
		return "Welcome! Type a message and press Enter to send."
	}

	boxWidth := width - 4
	if boxWidth < 20 {
		boxWidth = 20
	}

	var sb strings.Builder
	for i, entry := range messages {
		switch entry.Type {
		case EntryToolCall:
			if entry.Denied {
				line := toolCmdStyle.Render(entry.Command) + " " + deniedStyle.Render("User declined")
				sb.WriteString(line)
			} else {
				result := entry.Result
				maxResultLines := 10
				lines := strings.Split(result, "\n")
				if len(lines) > maxResultLines {
					result = strings.Join(lines[:maxResultLines], "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-maxResultLines)
				}
				content := toolCmdStyle.Render(entry.Command) + "\n" + result
				box := toolBoxStyle.Width(boxWidth).Render(content)
				sb.WriteString(box)
			}

		case EntryMessage:
			var prefix string
			switch entry.Role {
			case "user":
				prefix = "You"
			case "assistant":
				prefix = "Assistant"
			default:
				prefix = entry.Role
			}
			sb.WriteString(fmt.Sprintf("%s: %s", prefix, entry.Content))

		case EntryError:
			sb.WriteString(errorStyle.Render("Error: " + entry.Content))
		}

		if i < len(messages)-1 {
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
