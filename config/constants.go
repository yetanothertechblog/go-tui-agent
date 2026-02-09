package config

// Configuration constants for the application
const (
	// LLM Configuration
	MaxToolRounds = 100 // Maximum number of tool execution rounds per LLM response

	// File Reading Configuration
	MaxDefaultLines = 2000 // Default maximum lines to read when no limit is specified

	// UI Configuration
	TextareaHeight  = 3  // Height of the input textarea
	SeparatorHeight = 1  // Height of the separator line
	StatusHeight    = 1  // Height of the status line
	TokenBarHeight  = 2  // Height of the token bar
	MaxResultLines  = 10 // Maximum lines to display for tool results before truncating
	MinBoxWidth     = 30 // Minimum width for UI boxes
	BoxPadding      = 4  // Padding for UI boxes (2 sides)
	DiffBoxPadding  = 2  // Padding for diff boxes (1 side)

	// Tool Icons
	ToolIcon   = "🔧 "
	EditIcon   = "✏️ "
	WriteIcon  = "📝 "
	ReadIcon   = "📖 "
	ListIcon   = "📁 "
	BashIcon   = "💻 "
	SearchIcon = "🔍 "

	// API Configuration
	APIURL             = "https://api.z.ai/api/paas/v4/chat/completions"
	ModelName          = "glm-4.5-air"
	MaxContextTokens = 128000

	// File Permissions
	DirPermissions  = 0o755 // Directory permissions
	FilePermissions = 0o644 // File permissions
	LogPermissions  = 0o644 // Log file permissions
)
