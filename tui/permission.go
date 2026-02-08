package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"go-tui/config"
)

var (
	permBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2)

	permTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	permOptionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	permSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)
)

type PermissionPrompt struct {
	ToolName   string
	Args       string
	Cursor     int
	WorkingDir string
}

func (p PermissionPrompt) View(width int) string {
	boxWidth := width - config.BoxPadding
	if boxWidth < config.MinBoxWidth {
		boxWidth = config.MinBoxWidth
	}

	// Render the tool call preview box (diff for edit/write, formatted command for others)
	var toolSection string
	if diff := getDiffForPermission(p.ToolName, p.Args, p.WorkingDir); diff != "" {
		toolSection = toolBoxStyle.Width(boxWidth).Render(diff)
	} else {
		command := p.ToolName + ": " + p.Args
		header := formatCommand(command)
		toolSection = toolBoxStyle.Width(boxWidth).Render(toolCmdStyle.Render(header))
	}

	title := permTitleStyle.Render("Tool Permission Required")

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

	content := fmt.Sprintf("%s\n\n%s", title, optionsView)
	permBox := permBoxStyle.Width(boxWidth).Render(content)

	return toolSection + "\n" + permBox
}
