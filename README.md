# go-tui

A terminal-based AI coding assistant with rich TUI interface built with Go and Bubble Tea.

## Features

- **AI Coding Assistant**: Expert coding assistant that helps write, debug, and improve code
- **Rich Terminal UI**: Clean, responsive TUI using Bubble Tea framework with keyboard navigation
- **Built-in Tools**: Integrated file operations (read, edit, write), bash commands, and code search
- **Conversation Persistence**: Save and resume conversations by UUID
- **Real-time Streaming**: Streaming responses with live content updates
- **Permission System**: Interactive permission requests for potentially dangerous operations
- **Code Diff Visualization**: Visual diffs for file edits with side-by-side comparison
- **Logging Support**: Debug logging for monitoring and troubleshooting

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

3. **Run the application**:
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
- **📖 Read File**: Read file contents with line number support
- **📁 List Files**: List files and directories recursively
- **✏️ Edit File**: Edit files with visual diff preview
- **📝 Write File**: Create new files or overwrite existing ones
- **💻 Bash**: Execute shell commands
- **🔍 Search**: Search for text patterns in files

### Keyboard Shortcuts
- `Ctrl+C` - Exit application
- `Enter` - Send message
- `Esc` - Cancel current operation
- `Tab` - Navigate between UI elements

## Requirements

- Go 1.25+
- [Z.AI API key](https://z.ai/) (currently supported)
- Terminal with UTF-8 support

## Project Structure

```
go-tui/
├── main.go              # Application entry point
├── config/              # Configuration constants
├── conversation/        # Conversation management and persistence
├── llm/                 # LLM integration and API handling
├── agent/               # AI agent and tool execution
│   └── tools/           # Built-in tool implementations
├── tui/                 # Terminal user interface components
├── conversations/       # Saved conversation files
└── log/                 # Application logs
```

## Configuration

The application uses environment variables from `.env`:
- `ZAI_API_KEY`: Your Z.AI API key for LLM access

## License

This project is open source and available under the MIT License.