package tui

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go-tui/agent"
	"go-tui/config"
	"go-tui/conversation"
	"go-tui/llm"
)

type EntryType int

const (
	EntryMessage EntryType = iota
	EntryToolCall
	EntryError
)

type DiffData struct {
	FilePath  string `json:"file_path"`
	OldText   string `json:"old_text"`
	NewText   string `json:"new_text"`
	StartLine int    `json:"start_line,omitempty"`
}

type ChatEntry struct {
	Type    EntryType `json:"type"`
	Role    string    `json:"role,omitempty"`
	Content string    `json:"content,omitempty"`
	Command string    `json:"command,omitempty"`
	Result  string    `json:"result,omitempty"`
	Denied  bool      `json:"denied,omitempty"`
	Diff    *DiffData `json:"diff,omitempty"`
}

type Model struct {
	viewport   viewport.Model
	textarea   textarea.Model
	spinner    spinner.Model
	messages   []ChatEntry
	agent      *agent.Agent
	waiting    bool
	width      int
	height     int
	ready      bool
	permission *PermissionPrompt
	conv       *conversation.Data
	convDir    string
}

var separatorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("240"))

var statusStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("243"))

func New(workingDir string, conv *conversation.Data) Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Focus()
	ta.ShowLineNumbers = false
	ta.SetHeight(config.TextareaHeight)
	ta.CharLimit = 0

	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	var messages []ChatEntry
	if err := json.Unmarshal(conv.UIMessages, &messages); err != nil {
		log.Printf("failed to unmarshal UI messages: %v", err)
		messages = []ChatEntry{}
	}

	a := agent.New(workingDir)

	var history []llm.Message
	if err := json.Unmarshal(conv.AgentHistory, &history); err != nil {
		log.Printf("failed to unmarshal agent history: %v", err)
	} else if len(history) > 0 {
		a.SetHistory(history)
	}

	return Model{
		textarea: ta,
		spinner:  s,
		messages: messages,
		agent:    a,
		conv:     conv,
		convDir:  conversation.Dir(workingDir),
	}
}

func (m *Model) SetProgram(p *tea.Program) {
	m.agent.SetProgram(p)
}

func (m *Model) saveConversation() {
	uiJSON, err := json.Marshal(m.messages)
	if err != nil {
		log.Printf("failed to marshal UI messages: %v", err)
		return
	}
	histJSON, err := json.Marshal(m.agent.History())
	if err != nil {
		log.Printf("failed to marshal agent history: %v", err)
		return
	}
	m.conv.UIMessages = uiJSON
	m.conv.AgentHistory = histJSON
	if err := m.conv.Save(m.convDir); err != nil {
		log.Printf("failed to save conversation: %v", err)
	}
}

func parseDiffFromToolCall(msg agent.ToolCallMsg, workingDir string) *DiffData {
	name, argsJSON := splitCommand(msg.Command)

	if msg.Denied {
		return parseDiffFromArgs(name, argsJSON, workingDir)
	}

	if msg.Result == "" {
		return nil
	}

	switch name {
	case "edit_file", "write_file":
		var r struct {
			FilePath   string `json:"file_path"`
			OldString  string `json:"old_string"`
			NewString  string `json:"new_string"`
			OldContent string `json:"old_content"`
			NewContent string `json:"new_content"`
			IsNewFile  bool   `json:"is_new_file"`
		}
		if json.Unmarshal([]byte(msg.Result), &r) != nil || r.FilePath == "" {
			return parseDiffFromArgs(name, argsJSON, workingDir)
		}
		old := r.OldString + r.OldContent
		new_ := r.NewString + r.NewContent
		startLine := 1
		if name == "edit_file" {
			path := r.FilePath
			if !filepath.IsAbs(path) {
				path = filepath.Join(workingDir, path)
			}
			if data, err := os.ReadFile(path); err == nil {
				startLine = findStartLine(string(data), r.OldString)
			}
		}
		return &DiffData{
			FilePath:  r.FilePath,
			OldText:   old,
			NewText:   new_,
			StartLine: startLine,
		}
	}
	return nil
}

func parseDiffFromArgs(name, argsJSON, workingDir string) *DiffData {
	switch name {
	case "edit_file":
		var args struct {
			FilePath  string `json:"file_path"`
			OldString string `json:"old_string"`
			NewString string `json:"new_string"`
		}
		if json.Unmarshal([]byte(argsJSON), &args) != nil || args.FilePath == "" {
			return nil
		}
		startLine := 1
		path := args.FilePath
		if !filepath.IsAbs(path) {
			path = filepath.Join(workingDir, path)
		}
		if data, err := os.ReadFile(path); err == nil {
			startLine = findStartLine(string(data), args.OldString)
		}
		return &DiffData{
			FilePath:  args.FilePath,
			OldText:   args.OldString,
			NewText:   args.NewString,
			StartLine: startLine,
		}

	case "write_file":
		var args struct {
			FilePath string `json:"file_path"`
			Content  string `json:"content"`
		}
		if json.Unmarshal([]byte(argsJSON), &args) != nil || args.FilePath == "" {
			return nil
		}
		path := args.FilePath
		if !filepath.IsAbs(path) {
			path = filepath.Join(workingDir, path)
		}
		d := &DiffData{
			FilePath:  args.FilePath,
			NewText:   args.Content,
			StartLine: 1,
		}
		if data, err := os.ReadFile(path); err == nil {
			d.OldText = string(data)
		}
		return d
	}
	return nil
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, spinner.Tick)
}

func (m *Model) refreshViewport() {
	m.viewport.SetContent(renderMessages(m.messages, m.permission, m.width))
	m.viewport.GotoBottom()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		textareaHeight := config.TextareaHeight
		separatorHeight := config.SeparatorHeight
		statusHeight := config.StatusHeight
		vpHeight := m.height - textareaHeight - separatorHeight - statusHeight

		if !m.ready {
			m.viewport = viewport.New(m.width, vpHeight)
			m.refreshViewport()
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = vpHeight
		}

		m.textarea.SetWidth(m.width)
		return m, nil

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		var cmd tea.Cmd
		m, cmd = handleKeyMsg(m, msg)
		return m, cmd

	case agent.PermissionRequestMsg:
		m.permission = &PermissionPrompt{
			ToolName:   msg.ToolName,
			Args:       msg.Args,
			Cursor:     0,
			WorkingDir: m.agent.WorkingDir(),
		}
		m.refreshViewport()
		return m, nil

	case agent.ToolCallMsg:
		entry := ChatEntry{
			Type:    EntryToolCall,
			Command: msg.Command,
			Result:  msg.Result,
			Denied:  msg.Denied,
			Diff:    parseDiffFromToolCall(msg, m.agent.WorkingDir()),
		}
		m.messages = append(m.messages, entry)
		m.saveConversation()
		m.refreshViewport()
		return m, nil

	case agent.ResponseMsg:
		m.waiting = false
		m.textarea.Focus()
		if msg.Denied {
			// Permission was denied — no message to show
		} else if msg.Err != nil {
			m.messages = append(m.messages, ChatEntry{
				Type:    EntryError,
				Content: msg.Err.Error(),
			})
		} else {
			m.messages = append(m.messages, ChatEntry{
				Type:    EntryMessage,
				Role:    "assistant",
				Content: msg.Content,
			})
		}
		m.saveConversation()
		m.refreshViewport()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	separator := separatorStyle.Render(strings.Repeat("─", m.width))

	var status string
	if m.waiting {
		status = m.spinner.View() + " Processing..."
	} else {
		status = statusStyle.Render("Waiting for your input")
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		status,
		separator,
		m.textarea.View(),
	)
}
