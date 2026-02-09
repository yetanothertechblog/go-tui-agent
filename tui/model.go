package tui

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
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

const maxToolRounds = config.MaxToolRounds
const maxConsecutiveErrors = 3

type Model struct {
	viewport           viewport.Model
	textarea           textarea.Model
	spinner            spinner.Model
	messages           []ChatEntry
	agent              *agent.Agent
	waiting            bool
	width              int
	height             int
	ready              bool
	permission         *PermissionPrompt
	conv               *conversation.Data
	convDir            string
	markdownRenderer   *MarkdownRenderer
	history            []llm.Message
	workingDir         string
	alwaysAllow        map[string]bool
	toolRoundCount     int
	consecutiveErrors  int
	pendingToolCalls   []llm.ToolCall
	pendingToolIndex   int
	awaitingPermission  *llm.ToolCall
	totalTokens         int
	streamingTokens     int
	streamingThinking   bool
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

	// Initialize markdown renderer
	markdownRenderer, err := NewMarkdownRenderer()
	if err != nil {
		log.Printf("failed to initialize markdown renderer: %v", err)
		markdownRenderer = nil
	}

	var messages []ChatEntry
	if err := json.Unmarshal(conv.UIMessages, &messages); err != nil {
		log.Printf("failed to unmarshal UI messages: %v", err)
		messages = []ChatEntry{}
	}

	a := agent.New(workingDir)

	var history []llm.Message
	if err := json.Unmarshal(conv.AgentHistory, &history); err != nil {
		log.Printf("failed to unmarshal agent history: %v", err)
	}

	return Model{
		textarea:         ta,
		spinner:          s,
		messages:         messages,
		agent:            a,
		conv:             conv,
		convDir:          conversation.Dir(workingDir),
		markdownRenderer: markdownRenderer,
		history:          history,
		workingDir:       workingDir,
		alwaysAllow:      make(map[string]bool),
	}
}

func (m *Model) Shutdown() {
	m.agent.Shutdown()
}

func (m *Model) saveConversation() {
	uiJSON, err := json.Marshal(m.messages)
	if err != nil {
		log.Printf("failed to marshal UI messages: %v", err)
		return
	}
	histJSON, err := json.Marshal(m.history)
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

func parseDiffFromToolCall(toolName, args, result, workingDir string, denied bool) *DiffData {
	if denied {
		return parseDiffFromArgs(toolName, args, workingDir)
	}

	if result == "" {
		return nil
	}

	switch toolName {
	case "edit_file", "write_file":
		var r struct {
			FilePath   string `json:"file_path"`
			OldString  string `json:"old_string"`
			NewString  string `json:"new_string"`
			OldContent string `json:"old_content"`
			NewContent string `json:"new_content"`
			IsNewFile  bool   `json:"is_new_file"`
		}
		if json.Unmarshal([]byte(result), &r) != nil || r.FilePath == "" {
			return parseDiffFromArgs(toolName, args, workingDir)
		}
		old := r.OldString + r.OldContent
		new_ := r.NewString + r.NewContent
		startLine := 1
		if toolName == "edit_file" {
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
	m.viewport.SetContent(renderMessages(m.messages, m.permission, m.width, m.markdownRenderer))
	m.viewport.GotoBottom()
}

// dispatchNextTool returns a Cmd to execute the next pending tool call,
// or starts the next LLM round if all tools are done.
func (m *Model) dispatchNextTool() tea.Cmd {
	if m.pendingToolIndex >= len(m.pendingToolCalls) {
		// All tools done for this round
		m.pendingToolCalls = nil
		m.pendingToolIndex = 0
		if m.toolRoundCount >= maxToolRounds {
			m.waiting = false
			m.textarea.Focus()
			m.messages = append(m.messages, ChatEntry{
				Type:    EntryError,
				Content: "Tool call limit reached",
			})
			m.saveConversation()
			m.refreshViewport()
			return nil
		}
		// Start next LLM round
		return callLLM(m.agent, m.history)
	}

	tc := m.pendingToolCalls[m.pendingToolIndex]

	if m.alwaysAllow[tc.Function.Name] {
		return executeTool(m.agent, tc)
	}

	// Need permission
	m.awaitingPermission = &tc
	m.permission = &PermissionPrompt{
		ToolName:   tc.Function.Name,
		Args:       tc.Function.Arguments,
		Cursor:     0,
		WorkingDir: m.workingDir,
	}
	m.refreshViewport()
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		vpHeight := m.height - config.TextareaHeight - 2*config.SeparatorHeight - config.StatusHeight - config.TokenBarHeight

		if !m.ready {
			m.viewport = viewport.New(m.width, vpHeight)
			m.refreshViewport()
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = vpHeight
			m.textarea.SetWidth(m.width)
			m.refreshViewport()
		}

		return m, nil

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		var cmd tea.Cmd
		m, cmd = handleKeyMsg(m, msg)
		return m, cmd

	case StreamTokenCountMsg:
		m.streamingTokens = msg.Count
		m.streamingThinking = msg.Thinking
		return m, waitForStream(msg.ch)

	case LLMResponseMsg:
		m.streamingTokens = 0
		m.streamingThinking = false
		if msg.Usage != nil {
			m.totalTokens = msg.Usage.TotalTokens
		}
		if msg.Err != nil {
			m.waiting = false
			m.textarea.Focus()
			m.messages = append(m.messages, ChatEntry{
				Type:    EntryError,
				Content: msg.Err.Error(),
			})
			m.saveConversation()
			m.refreshViewport()
			return m, nil
		}

		if len(msg.ToolCalls) == 0 {
			// No tools — plain assistant response
			m.waiting = false
			m.textarea.Focus()
			m.history = append(m.history, llm.Message{
				Role:    "assistant",
				Content: msg.Content,
			})
			m.messages = append(m.messages, ChatEntry{
				Type:    EntryMessage,
				Role:    "assistant",
				Content: msg.Content,
			})
			m.saveConversation()
			m.refreshViewport()
			return m, nil
		}

		// Has tool calls — append assistant message with both content and tool calls
		m.history = append(m.history, llm.Message{
			Role:      "assistant",
			Content:   msg.Content,
			ToolCalls: msg.ToolCalls,
		})

		// If there's content alongside tool calls, show it (fixes dropped-content bug)
		if msg.Content != "" {
			m.messages = append(m.messages, ChatEntry{
				Type:    EntryMessage,
				Role:    "assistant",
				Content: msg.Content,
			})
		}

		m.pendingToolCalls = msg.ToolCalls
		m.pendingToolIndex = 0
		m.toolRoundCount++
		m.refreshViewport()

		cmd := m.dispatchNextTool()
		return m, cmd

	case ToolResultMsg:
		command := msg.ToolName + ": " + msg.Args
		resultStr := msg.Result

		if msg.Err != nil {
			m.consecutiveErrors++
			if m.consecutiveErrors >= maxConsecutiveErrors {
				resultStr += " (Too many consecutive errors. Stop retrying and tell the user what went wrong.)"
			}
		} else {
			m.consecutiveErrors = 0
		}

		// Append tool result to history
		m.history = append(m.history, llm.Message{
			Role:       "tool",
			Content:    resultStr,
			ToolCallID: msg.ToolCallID,
		})

		// Append tool call entry to UI messages
		entry := ChatEntry{
			Type:    EntryToolCall,
			Command: command,
			Result:  msg.Result,
			Diff:    parseDiffFromToolCall(msg.ToolName, msg.Args, msg.Result, m.workingDir, false),
		}
		m.messages = append(m.messages, entry)
		m.saveConversation()
		m.refreshViewport()

		// Advance to next tool
		m.pendingToolIndex++
		cmd := m.dispatchNextTool()
		return m, cmd

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
		if m.streamingTokens > 0 {
			thinkingStr := ""
			if m.streamingThinking {
				thinkingStr = " · ( thinking )"
			}
			status = m.spinner.View() + fmt.Sprintf(" Processing · ⬇ %d%s tokens", m.streamingTokens, thinkingStr)
		} else {
			status = m.spinner.View() + " Processing"
		}
	} else {
		status = statusStyle.Render("Waiting for your input")
	}

	tokenLabel := fmt.Sprintf("tokens: %d/%d ", m.totalTokens, config.MaxContextTokens)
	barWidth := m.width - len(tokenLabel) - 2
	if barWidth < 10 {
		barWidth = 10
	}
	tokenBar := statusStyle.Render(tokenLabel) + renderBar(m.totalTokens, config.MaxContextTokens, barWidth)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		status,
		separator,
		m.textarea.View(),
		separator,
		tokenBar,
	)
}

func renderBar(value, max, width int) string {
	ratio := float64(value) / float64(max)
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	color := "42"
	if ratio > 0.8 {
		color = "196"
	} else if ratio > 0.5 {
		color = "214"
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(bar)
}
