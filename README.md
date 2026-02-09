# go-tui

A terminal-based AI coding assistant with rich TUI interface, built with Go and Bubble Tea. Features integrated LSP support for real-time code analysis, comprehensive tool ecosystem, and conversation persistence.

## Features

- **AI Coding Assistant**: Expert coding assistant that helps write, debug, and improve code with integrated LSP support
- **Rich Terminal UI**: Clean, responsive TUI using Bubble Tea framework with keyboard navigation and markdown rendering
- **Language Server Protocol (LSP)**: Real-time code analysis and diagnostics for multiple programming languages
- **Built-in Tools**: Comprehensive tool ecosystem for file operations, bash commands, code search, and task management
- **Conversation Persistence**: Save and resume conversations by UUID with full history preservation
- **Real-time Streaming**: Streaming responses with live content updates and tool execution
- **Permission System**: Interactive permission requests for potentially dangerous operations
- **Code Diff Visualization**: Visual diffs for file edits with side-by-side comparison
- **Task Management**: Integrated beads CLI for task tracking and project management
- **Logging Support**: Debug logging for monitoring and troubleshooting

## Architecture Overview

```
go-tui/
├── main.go              # Application entry point and initialization
├── config/              # Configuration constants and settings
├── conversation/        # Conversation management and persistence (UUID-based)
├── llm/                 # LLM integration with Z.AI API support (streaming + non-streaming)
├── agent/               # AI agent with system prompts and tool execution
│   └── tools/           # Built-in tool implementations (read, edit, write, bash, search, beads)
├── tui/                 # Terminal UI components (Bubble Tea models, markdown, diff rendering)
├── lsp/                 # Language Server Protocol integration for code analysis
├── conversations/       # Saved conversation files (JSON format)
└── log/                 # Application logs and debugging
```

### Core Components

**Main Application (`main.go`)**
- Initializes LLM API key and logging system
- Manages conversation creation/resumption
- Starts TUI with Bubble Tea framework

**LLM Integration (`llm/`)**
- Z.AI API client with streaming and non-streaming support
- Message handling and tool call management
- API configuration and error handling

**AI Agent (`agent/`)**
- System prompt with coding best practices
- Tool execution coordination
- LSP integration for code analysis

**Tool System (`agent/tools/`)**
- File operations: read, edit, write with diff visualization
- System commands: bash execution with safety checks
- Code analysis: file search and pattern matching
- Task management: beads CLI integration

**TUI Interface (`tui/`)**
- Bubble Tea model with viewport, textarea, and spinner
- Markdown rendering for rich content display
- Permission prompts for dangerous operations
- Real-time message updates and tool result display

**LSP Integration (`lsp/`)**
- Multi-language server management (Go, Rust, Python, etc.)
- Lazy server startup and automatic shutdown
- Diagnostic collection and formatting
- Code analysis and error detection

**Conversation Management (`conversation/`)**
- UUID-based conversation persistence
- Separate UI messages and agent history storage
- Conversation resumption and management

## Getting Started

1. **Install dependencies**:
   ```bash
   go mod tidy
   ```

2. **Set up your LLM API key**:
   ```bash
   # Create .env file in project root
   echo "ZAI_API_KEY=your-api-key-here" > .env
   ```

3. **Install language servers** (optional, for LSP support):
   ```bash
   # Install Go language server
   go install golang.org/x/tools/gopls@latest
   
   # Install Rust language server
   cargo install rust-analyzer
   
   # Install Python language server
   pip install python-lsp-server
   ```

4. **Run the application**:
   ```bash
   go run .
   ```

## Usage

### Basic Commands
- Start a new conversation: `go run .`
- Resume a specific conversation: `go run . -resume <uuid>`
- Resume the latest conversation: `go run . -resume`

### Available Tools
The AI assistant has access to these tools:
- **📖 Read File**: Read file contents with line number support and LSP diagnostics
- **📁 List Files**: List files and directories recursively with filtering
- **✏️ Edit File**: Edit files with visual diff preview and LSP validation
- **📝 Write File**: Create new files or overwrite existing ones
- **💻 Bash**: Execute shell commands with permission checks
- **🔍 Search**: Search for text patterns in files with regex support
- **🎯 Beads**: Integrate with task tracking system for project management

### Keyboard Shortcuts
- `Ctrl+C` - Exit application
- `Enter` - Send message
- `Esc` - Cancel current operation
- `Tab` - Navigate between UI elements

### LSP Features
- Real-time code analysis and diagnostics
- Multi-language support (Go, Rust, Python, JavaScript, etc.)
- Automatic error detection and suggestions
- Integration with tool execution results

## Requirements

- Go 1.25+
- [Z.AI API key](https://z.ai/) (currently supported)
- Terminal with UTF-8 support
- Optional: Language servers for LSP support

## Configuration

The application uses environment variables from `.env`:
- `ZAI_API_KEY`: Your Z.AI API key for LLM access

## Tool Execution Flow

1. User sends message to AI assistant
2. AI analyzes request and may call tools
3. System checks permissions for dangerous operations
4. Tools execute with proper error handling
5. Results are displayed with visual diffs when applicable
6. LSP diagnostics are integrated into tool feedback
7. Conversation state is preserved throughout

## License

This project is open source and available under the MIT License.