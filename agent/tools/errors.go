package tools

import "fmt"

// ToolError represents a structured error from tool execution
type ToolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e *ToolError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewToolError creates a new ToolError
func NewToolError(code, message string) *ToolError {
	return &ToolError{
		Code:    code,
		Message: message,
	}
}

// NewToolErrorWithDetails creates a new ToolError with additional details
func NewToolErrorWithDetails(code, message, details string) *ToolError {
	return &ToolError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Error codes
const (
	ErrInvalidArguments = "INVALID_ARGUMENTS"
	ErrMissingField     = "MISSING_FIELD"
	ErrIdenticalContent = "IDENTICAL_CONTENT"
	ErrFileNotFound     = "FILE_NOT_FOUND"
	ErrStringNotFound   = "STRING_NOT_FOUND"
	ErrStringNotUnique  = "STRING_NOT_UNIQUE"
	ErrFileWrite        = "FILE_WRITE_ERROR"
	ErrJSONMarshal      = "JSON_MARSHAL_ERROR"
)