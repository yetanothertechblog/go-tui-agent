package slashcmd

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Command represents a single slash command available to the user.
type Command struct {
	Name        string
	Description string
}

var registry []Command

// Register adds a slash command to the global registry.
func Register(cmd Command) {
	registry = append(registry, cmd)
}

// All returns all registered slash commands.
func All() []Command {
	return registry
}

// Filter returns commands whose Name starts with the given prefix.
func Filter(query string) []Command {
	if query == "" {
		return registry
	}
	var filtered []Command
	for _, cmd := range registry {
		if strings.HasPrefix(cmd.Name, query) {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}

// Overlay holds the state of the slash command picker overlay.
type Overlay struct {
	Commands []Command
	Cursor   int
}

// Steampunk color palette (mirrored from tui/theme.go for overlay styling).
var (
	colorBrass     = lipgloss.Color("#B5A642")
	colorParchment = lipgloss.Color("#D4C5A9")
	colorDimBrass  = lipgloss.Color("#8B7D3C")
	colorAmber     = lipgloss.Color("#FFBF00")
)

var (
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBrass).
			Padding(0, 1)

	itemStyle = lipgloss.NewStyle().
			Foreground(colorParchment)

	selectedStyle = lipgloss.NewStyle().
			Foreground(colorBrass).
			Bold(true)

	descStyle = lipgloss.NewStyle().
			Foreground(colorDimBrass)

	overlayBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAmber).
			Padding(1, 2)

	overlayTitleStyle = lipgloss.NewStyle().
				Foreground(colorAmber).
				Bold(true)

	overlayOptionStyle = lipgloss.NewStyle().
				Foreground(colorParchment)

	overlaySelectedStyle = lipgloss.NewStyle().
				Foreground(colorAmber).
				Bold(true)
)

// View renders the slash command overlay box.
func (s *Overlay) View(width int) string {
	if len(s.Commands) == 0 {
		return ""
	}

	var lines []string
	for i, cmd := range s.Commands {
		name := cmd.Name
		desc := descStyle.Render(" " + cmd.Description)
		if i == s.Cursor {
			name = selectedStyle.Render(name)
		} else {
			name = itemStyle.Render(name)
		}
		lines = append(lines, name+desc)
	}

	content := strings.Join(lines, "\n")

	boxWidth := width - 4
	if boxWidth < 20 {
		boxWidth = 20
	}

	return boxStyle.Width(boxWidth).Render(content)
}
