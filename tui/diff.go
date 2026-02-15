package tui

import (
	"fmt"
	"strings"

	"go-tui/config"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Styles are defined in theme.go

// renderDiff renders a unified-style diff with colored +/- lines and line numbers.
// When OldText is empty, all lines are shown as additions (new file).
func renderDiff(d DiffData) string {
	var sb strings.Builder

	if d.OldText == "" {
		icon := config.WriteIcon
		sb.WriteString(diffHeaderStyle.Render(icon + d.FilePath + " (new file)"))
		sb.WriteString("\n")

		for i, line := range strings.Split(d.NewText, "\n") {
			num := diffLineNumStyle.Render(fmt.Sprintf("   %4d ", i+1))
			sb.WriteString(num + diffAddedStyle.Render("+ "+line))
			sb.WriteString("\n")
		}
		return strings.TrimRight(sb.String(), "\n")
	}

	dmp := diffmatchpatch.New()
	a, b, lines := dmp.DiffLinesToChars(d.OldText, d.NewText)
	diffs := dmp.DiffMain(a, b, false)
	diffs = dmp.DiffCharsToLines(diffs, lines)
	diffs = dmp.DiffCleanupSemantic(diffs)

	icon := config.ToolIcon
	sb.WriteString(diffHeaderStyle.Render(icon + d.FilePath))
	sb.WriteString("\n")

	startLine := d.StartLine
	if startLine < 1 {
		startLine = 1
	}
	oldLine := startLine
	newLine := startLine

	for _, diff := range diffs {
		text := strings.TrimRight(diff.Text, "\n")
		for _, line := range strings.Split(text, "\n") {
			switch diff.Type {
			case diffmatchpatch.DiffInsert:
				num := diffLineNumStyle.Render(fmt.Sprintf("   %4d ", newLine))
				sb.WriteString(num + diffAddedStyle.Render("+ "+line))
				newLine++
			case diffmatchpatch.DiffDelete:
				num := diffLineNumStyle.Render(fmt.Sprintf("%4d     ", oldLine))
				sb.WriteString(num + diffRemovedStyle.Render("- "+line))
				oldLine++
			case diffmatchpatch.DiffEqual:
				num := diffLineNumStyle.Render(fmt.Sprintf("%4d %4d ", oldLine, newLine))
				sb.WriteString(num + diffHunkStyle.Render("  "+line))
				oldLine++
				newLine++
			}
			sb.WriteString("\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// findStartLine returns the 1-based line number where needle starts within fileContent.
// Returns 1 if not found.
func findStartLine(fileContent, needle string) int {
	idx := strings.Index(fileContent, needle)
	if idx < 0 {
		return 1
	}
	return strings.Count(fileContent[:idx], "\n") + 1
}

// getDiffForPermission renders a diff preview for the permission prompt.
func getDiffForPermission(toolName, argsJSON, workingDir string) string {
	d := parseDiffFromArgs(toolName, argsJSON, workingDir)
	if d == nil {
		return ""
	}
	return renderDiff(*d)
}
