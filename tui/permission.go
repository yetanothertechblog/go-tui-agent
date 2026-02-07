package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	permBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2)

	permTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	permToolStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	permArgsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	permOptionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	permSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)
)

type PermissionPrompt struct {
	ToolName string
	Args     string
	Cursor   int
}

func (p PermissionPrompt) View(width int) string {
	title := permTitleStyle.Render("Tool Permission Required")
	tool := permToolStyle.Render(p.ToolName)
	args := permArgsStyle.Render(p.Args)

	options := []string{
		"Allow",
		fmt.Sprintf("Always Allow %s", p.ToolName),
		"Deny",
	}

	var optionsView string
	for i, opt := range options {
		cursor := "  "
		style := permOptionStyle
		if i == p.Cursor {
			cursor = "> "
			style = permSelectedStyle
		}
		optionsView += cursor + style.Render(opt) + "\n"
	}

	content := fmt.Sprintf("%s\n\n%s %s\n\n%s", title, tool, args, optionsView)

	boxWidth := width - 6
	if boxWidth < 30 {
		boxWidth = 30
	}

	return permBoxStyle.Width(boxWidth).Render(content)
}
