package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

// MarkdownRenderer handles markdown rendering for the TUI.
type MarkdownRenderer struct {
	renderer *glamour.TermRenderer
}

// NewMarkdownRenderer creates a new markdown renderer.
func NewMarkdownRenderer() (*MarkdownRenderer, error) {
	renderer, err := glamour.NewTermRenderer(glamour.WithStandardStyle("auto"))
	if err != nil {
		return nil, err
	}
	return &MarkdownRenderer{renderer: renderer}, nil
}

// Render converts markdown content to styled terminal text.
func (r *MarkdownRenderer) Render(markdown string) (string, error) {
	if r == nil || r.renderer == nil {
		return markdown, nil
	}
	rendered, err := r.renderer.Render(markdown)
	if err != nil {
		return markdown, err
	}
	return strings.TrimRight(rendered, "\n"), nil
}

// isMarkdown returns true if the content likely contains markdown formatting.
func isMarkdown(content string) bool {
	markers := []string{
		"```", "# ", "## ", "### ",
		"**", "* ", "- ", "| ",
		"1. ", "> ",
	}
	for _, m := range markers {
		if strings.Contains(content, m) {
			return true
		}
	}
	return false
}
