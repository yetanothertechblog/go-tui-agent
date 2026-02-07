package tui

import (
	"strings"

	"go-tui/agent"

	tea "github.com/charmbracelet/bubbletea"
)

func handleKeyMsg(m *Model, msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	}

	// Permission prompt mode
	if m.permission != nil {
		return handlePermissionKey(m, msg)
	}

	switch msg.Type {
	case tea.KeyEnter:
		if m.waiting {
			return m, nil
		}

		text := strings.TrimSpace(m.textarea.Value())
		if text == "" {
			return m, nil
		}

		m.messages = append(m.messages, ChatEntry{
			Type:    EntryMessage,
			Role:    "user",
			Content: text,
		})

		m.textarea.Reset()
		m.textarea.Blur()
		m.waiting = true

		m.refreshViewport()

		return m, m.agent.Send(text)
	}

	if m.waiting {
		return m, nil
	}

	// Handle scrolling keys before textarea consumes them
	switch msg.Type {
	case tea.KeyPgUp:
		m.viewport.ViewUp()
		return m, nil
	case tea.KeyPgDown:
		m.viewport.ViewDown()
		return m, nil
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func handlePermissionKey(m *Model, msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.permission.Cursor > 0 {
			m.permission.Cursor--
			m.refreshViewport()
		}
		return m, nil

	case tea.KeyDown:
		if m.permission.Cursor < 2 {
			m.permission.Cursor++
			m.refreshViewport()
		}
		return m, nil

	case tea.KeyEnter:
		var decision agent.PermissionDecision
		switch m.permission.Cursor {
		case 0:
			decision = agent.PermissionAllow
		case 1:
			decision = agent.PermissionAlwaysAllow
		case 2:
			decision = agent.PermissionDeny
		}
		m.permission = nil
		m.refreshViewport()
		go m.agent.RespondPermission(decision)
		return m, nil
	}

	return m, nil
}
