package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Copper-brass steampunk color palette.
var (
	colorBrass      = lipgloss.Color("#B5A642")
	colorCopper     = lipgloss.Color("#B87333")
	colorBronze     = lipgloss.Color("#CD7F32")
	colorRust       = lipgloss.Color("#B7410E")
	colorSteam      = lipgloss.Color("#C8C8B4")
	colorDarkSteel  = lipgloss.Color("#3B3B2E")
	colorPatina     = lipgloss.Color("#4A7C6F")
	colorAmber      = lipgloss.Color("#FFBF00")
	colorOxidized   = lipgloss.Color("#6B4226")
	colorParchment  = lipgloss.Color("#D4C5A9")
	colorDimBrass   = lipgloss.Color("#8B7D3C")
	colorForgedIron = lipgloss.Color("#555548")
)

// Shared styles used across TUI components.
var (
	// Layout
	separatorStyle = lipgloss.NewStyle().
			Foreground(colorForgedIron)

	statusStyle = lipgloss.NewStyle().
			Foreground(colorDimBrass)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(colorBrass)

	// Messages
	toolCmdStyle = lipgloss.NewStyle().
			Foreground(colorCopper).
			Bold(true)

	userMessageStyle = lipgloss.NewStyle().
				Background(colorDarkSteel)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorRust)

	deniedStyle = lipgloss.NewStyle().
			Foreground(colorRust).
			Bold(true)

	// Diffs
	diffAddedStyle = lipgloss.NewStyle().
			Foreground(colorPatina)

	diffRemovedStyle = lipgloss.NewStyle().
				Foreground(colorRust)

	diffHeaderStyle = lipgloss.NewStyle().
			Foreground(colorBrass).
			Bold(true)

	diffHunkStyle = lipgloss.NewStyle().
			Foreground(colorDimBrass)

	diffLineNumStyle = lipgloss.NewStyle().
				Foreground(colorForgedIron)

	toolBulletStyle = lipgloss.NewStyle().
			Foreground(colorCopper).
			Bold(true)

	toolIndentStyle = lipgloss.NewStyle().
			Foreground(colorForgedIron)

	// Permission prompt
	permBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAmber).
			Padding(1, 2)

	permTitleStyle = lipgloss.NewStyle().
			Foreground(colorAmber).
			Bold(true)

	permOptionStyle = lipgloss.NewStyle().
			Foreground(colorParchment)

	permSelectedStyle = lipgloss.NewStyle().
				Foreground(colorAmber).
				Bold(true)
)

// robotFace is a visually square steampunk robot (14w × 7h to account
// for the ~2:1 terminal character aspect ratio).
var robotFace = "" +
	"╔══════⚙═══════╗  \n" +
	"║  ╭────────╮  ║  \n" +
	"║  │ ◉    ◉ │  ║  \n" +
	"║  ╰───▽────╯  ║  \n" +
	"╠══════╤═══════╣  \n" +
	"║  ▄▄▐██▌▄▄    ║  \n" +
	"╚══════╧═══════╝  \n"

var robotStyle = lipgloss.NewStyle().
	Foreground(colorCopper).
	MarginLeft(1)

// Piston animation frames (4-frame cycle).
var pistonFrames = []string{
	"▄█▀▀█▄╶╴▄█▀▀█▄",
	"██▄▄██╶╴██▄▄██",
	"▀█▄▄█▀╶╴▀█▄▄█▀",
	"██▄▄██╶╴██▄▄██",
}

var pistonStyle = lipgloss.NewStyle().
	Foreground(colorBronze)

// Robot dialogue lines — cycles through these for flavor.
var robotDialogues = []string{
	"Steam pressure nominal",
	"Gears turning...",
	"Boiler temperature stable",
	"Cogs well-oiled",
	"Pistons firing smoothly",
	"Valve pressure steady",
}

var dialogueStyle = lipgloss.NewStyle().
	Foreground(colorParchment)

// tokenBarColor returns the appropriate color for the token usage bar.
func tokenBarColor(ratio float64) lipgloss.Color {
	switch {
	case ratio > 0.8:
		return colorRust
	case ratio > 0.5:
		return colorBronze
	default:
		return colorPatina
	}
}
