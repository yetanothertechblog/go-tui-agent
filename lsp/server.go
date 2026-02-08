package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ServerConfig defines how to start a language server.
type ServerConfig struct {
	Name       string
	Command    string
	Args       []string
	Extensions []string
}

// KnownServers lists language servers we know how to start.
var KnownServers = []ServerConfig{
	{Name: "gopls", Command: "gopls", Args: []string{"serve"}, Extensions: []string{".go"}},
	{Name: "typescript-language-server", Command: "typescript-language-server", Args: []string{"--stdio"}, Extensions: []string{".ts", ".tsx", ".js", ".jsx"}},
	{Name: "pyright", Command: "pyright-langserver", Args: []string{"--stdio"}, Extensions: []string{".py"}},
	{Name: "rust-analyzer", Command: "rust-analyzer", Args: nil, Extensions: []string{".rs"}},
}

// Server manages a single LSP server subprocess.
type Server struct {
	config     ServerConfig
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     *bufio.Reader
	mu         sync.Mutex
	nextID     int
	pending    map[int]chan *Response
	diagCh     chan []Diagnostic
	workingDir string
	stopped    bool
}

// NewServer creates a new server instance (does not start the process).
func NewServer(config ServerConfig, workingDir string) *Server {
	return &Server{
		config:     config,
		workingDir: workingDir,
		nextID:     1,
		pending:    make(map[int]chan *Response),
		diagCh:     make(chan []Diagnostic, 4),
	}
}

// Start spawns the language server process and performs the initialize handshake.
func (s *Server) Start() error {
	// Hold mutex only while setting up process fields.
	s.mu.Lock()
	cmd := exec.Command(s.config.Command, s.config.Args...)
	cmd.Dir = s.workingDir
	cmd.Stderr = io.Discard

	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		s.mu.Unlock()
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		s.mu.Unlock()
		return fmt.Errorf("start %s: %w", s.config.Command, err)
	}

	s.cmd = cmd
	s.stdin = stdin
	s.stdout = bufio.NewReaderSize(stdout, 64*1024)
	s.mu.Unlock()

	// Start reader goroutine before sending initialize
	go s.readLoop()

	// Send initialize request (must be outside mutex — sendRequest acquires it)
	rootURI := filePathToURI(s.workingDir)
	initParams := InitializeParams{
		ProcessID: os.Getpid(),
		RootURI:   rootURI,
		Capabilities: ClientCapabilities{
			TextDocument: TextDocumentClientCapabilities{
				PublishDiagnostics: PublishDiagnosticsCapability{
					RelatedInformation: true,
				},
			},
		},
	}

	log.Printf("lsp: sending initialize to %s", s.config.Name)
	resp, err := s.sendRequest("initialize", initParams)
	if err != nil {
		s.kill()
		return fmt.Errorf("initialize: %w", err)
	}
	if resp.Error != nil {
		s.kill()
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	// Send initialized notification
	if err := s.sendNotification("initialized", struct{}{}); err != nil {
		s.kill()
		return fmt.Errorf("initialized notification: %w", err)
	}

	log.Printf("lsp: %s handshake complete", s.config.Name)
	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()

	// Try graceful shutdown with a timeout
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Send shutdown request (best effort)
		if resp, err := s.sendRequest("shutdown", nil); err == nil && resp != nil {
			// Send exit notification
			_ = s.sendNotification("exit", nil)
		}
	}()

	select {
	case <-done:
		// Wait briefly for process exit
		waitDone := make(chan struct{})
		go func() {
			_ = s.cmd.Wait()
			close(waitDone)
		}()
		select {
		case <-waitDone:
		case <-time.After(2 * time.Second):
			s.kill()
		}
	case <-time.After(3 * time.Second):
		s.kill()
	}
}

// CheckFile sends didOpen, waits for diagnostics, then sends didClose.
func (s *Server) CheckFile(filePath string, content string) ([]Diagnostic, error) {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil, fmt.Errorf("server stopped")
	}
	s.mu.Unlock()

	uri := filePathToURI(filePath)
	langID := extensionToLanguageID(filepath.Ext(filePath))

	// Drain any stale diagnostics
	for range len(s.diagCh) {
		<-s.diagCh
	}

	// Send didOpen
	err := s.sendNotification("textDocument/didOpen", DidOpenParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: langID,
			Version:    1,
			Text:       content,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("didOpen: %w", err)
	}

	// Wait for diagnostics with timeout
	var diags []Diagnostic
	select {
	case diags = <-s.diagCh:
	case <-time.After(5 * time.Second):
		// Timeout — return empty
	}

	// Send didClose
	_ = s.sendNotification("textDocument/didClose", DidCloseParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	})

	return diags, nil
}

// sendRequest sends a JSON-RPC request and waits for the response.
func (s *Server) sendRequest(method string, params interface{}) (*Response, error) {
	s.mu.Lock()
	id := s.nextID
	s.nextID++
	ch := make(chan *Response, 1)
	s.pending[id] = ch
	s.mu.Unlock()

	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	if err := WriteMessage(s.stdin, req); err != nil {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(10 * time.Second):
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
		return nil, fmt.Errorf("timeout waiting for response to %s (id=%d)", method, id)
	}
}

// sendNotification sends a JSON-RPC notification (no response expected).
func (s *Server) sendNotification(method string, params interface{}) error {
	notif := Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return WriteMessage(s.stdin, notif)
}

// readLoop continuously reads messages from the server and dispatches them.
func (s *Server) readLoop() {
	for {
		raw, err := ReadMessage(s.stdout)
		if err != nil {
			s.mu.Lock()
			stopped := s.stopped
			// Close all pending channels
			for id, ch := range s.pending {
				close(ch)
				delete(s.pending, id)
			}
			s.mu.Unlock()
			if !stopped {
				log.Printf("lsp %s: read error: %v", s.config.Name, err)
			}
			return
		}

		var msg Response
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("lsp %s: unmarshal error: %v", s.config.Name, err)
			continue
		}

		// Response to a request (has ID)
		if msg.ID != nil {
			s.mu.Lock()
			ch, ok := s.pending[*msg.ID]
			if ok {
				delete(s.pending, *msg.ID)
			}
			s.mu.Unlock()
			if ok {
				ch <- &msg
			}
			continue
		}

		// Notification from server
		if msg.Method == "textDocument/publishDiagnostics" {
			var params PublishDiagnosticsParams
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				log.Printf("lsp %s: unmarshal diagnostics: %v", s.config.Name, err)
				continue
			}
			// Non-blocking send
			select {
			case s.diagCh <- params.Diagnostics:
			default:
			}
		}
	}
}

func (s *Server) kill() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}
}

func filePathToURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	// Use url.URL to correctly encode the path while preserving slashes.
	u := &url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	return u.String()
}

func extensionToLanguageID(ext string) string {
	switch strings.ToLower(ext) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescriptreact"
	case ".js":
		return "javascript"
	case ".jsx":
		return "javascriptreact"
	case ".rs":
		return "rust"
	default:
		return "plaintext"
	}
}
