package tui

import (
	"strings"

	"go-tui/llm"

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

		// Append user message to history (now on Model, not Agent)
		m.history = append(m.history, llm.Message{
			Role:    "user",
			Content: text,
		})

		m.textarea.Reset()
		m.textarea.Blur()
		m.waiting = true
		m.toolRoundCount = 0
		m.consecutiveErrors = 0

		m.refreshViewport()

		return m, callLLM(m.agent, m.history)
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
		// Read cursor and tool call before clearing permission state
		cursor := m.permission.Cursor
		tc := m.awaitingPermission

		// Clear permission state
		m.permission = nil
		m.awaitingPermission = nil
		m.refreshViewport()

		switch cursor {
		case 0: // Allow
			return m, executeTool(m.agent, *tc)

		case 1: // Always Allow
			m.alwaysAllow[tc.Function.Name] = true
			return m, executeTool(m.agent, *tc)

		case 2: // Deny
			command := tc.Function.Name + ": " + tc.Function.Arguments
			result := "Tool call denied by user."

			// Append denial to history
			m.history = append(m.history, llm.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})

			// Append denied tool call to UI messages
			m.messages = append(m.messages, ChatEntry{
				Type:    EntryToolCall,
				Command: command,
				Denied:  true,
				Diff:    parseDiffFromToolCall(tc.Function.Name, tc.Function.Arguments, "", m.workingDir, true),
			})

			// Stop the loop — return to user input
			m.pendingToolCalls = nil
			m.pendingToolIndex = 0
			m.waiting = false
			m.textarea.Focus()
			m.saveConversation()
			m.refreshViewport()
			return m, nil
		}
	}

	return m, nil
}
