package lsp

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// DefaultManager is the package-level manager instance.
var DefaultManager *Manager

// Start creates and sets the DefaultManager for the given working directory.
func Start(workingDir string) {
	log.Printf("lsp: initializing manager for %s", workingDir)
	DefaultManager = NewManager(workingDir)
}

// Stop shuts down the DefaultManager if it exists.
func Stop() {
	if DefaultManager != nil {
		DefaultManager.Shutdown()
	}
}

// Manager manages LSP servers by file extension, starting them lazily.
type Manager struct {
	mu         sync.Mutex
	servers    map[string]*Server       // extension -> running server
	configs    map[string]ServerConfig   // extension -> config (only for servers on PATH)
	workingDir string
}

func init() {
	// Append common tool directories to PATH so LookPath can find language servers.
	home, _ := os.UserHomeDir()
	if home == "" {
		return
	}
	extra := []string{
		filepath.Join(home, "go", "bin"),
		filepath.Join(home, ".cargo", "bin"),
		filepath.Join(home, ".local", "bin"),
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		extra = append(extra, filepath.Join(gopath, "bin"))
	}
	if runtime.GOOS == "darwin" {
		extra = append(extra, "/opt/homebrew/bin", "/usr/local/bin")
	}
	path := os.Getenv("PATH")
	for _, dir := range extra {
		if _, err := os.Stat(dir); err == nil {
			path += string(os.PathListSeparator) + dir
		}
	}
	os.Setenv("PATH", path)
}

// NewManager creates a Manager, registering only servers whose binaries can be found.
func NewManager(workingDir string) *Manager {
	configs := make(map[string]ServerConfig)
	for _, sc := range KnownServers {
		if _, err := exec.LookPath(sc.Command); err != nil {
			log.Printf("lsp: %s not found on PATH, skipping", sc.Command)
			continue
		}
		log.Printf("lsp: found %s on PATH", sc.Command)
		for _, ext := range sc.Extensions {
			configs[ext] = sc
		}
	}
	if len(configs) == 0 {
		log.Printf("lsp: no language servers found")
	}
	return &Manager{
		servers:    make(map[string]*Server),
		configs:    configs,
		workingDir: workingDir,
	}
}

// CheckFile looks up the appropriate server by file extension, starts it lazily,
// and returns diagnostics. Returns nil, nil if no server handles this extension.
func (m *Manager) CheckFile(filePath string, content string) ([]Diagnostic, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return nil, nil
	}

	m.mu.Lock()
	cfg, ok := m.configs[ext]
	if !ok {
		m.mu.Unlock()
		return nil, nil
	}

	srv, running := m.servers[ext]
	if !running {
		srv = NewServer(cfg, m.workingDir)
		if err := srv.Start(); err != nil {
			m.mu.Unlock()
			log.Printf("lsp: failed to start %s: %v", cfg.Name, err)
			return nil, nil
		}
		// Register this server for all its extensions
		for _, e := range cfg.Extensions {
			m.servers[e] = srv
		}
		log.Printf("lsp: started %s", cfg.Name)
	}
	m.mu.Unlock()

	log.Printf("lsp: checking %s with %s", filepath.Base(filePath), cfg.Name)
	diags, err := srv.CheckFile(filePath, content)
	if err != nil {
		log.Printf("lsp: %s CheckFile error: %v", cfg.Name, err)
		// Server may have crashed — remove it so next call restarts
		m.mu.Lock()
		for _, e := range cfg.Extensions {
			delete(m.servers, e)
		}
		m.mu.Unlock()
		return nil, nil
	}
	if len(diags) == 0 {
		log.Printf("lsp: %s returned 0 diagnostics for %s", cfg.Name, filepath.Base(filePath))
	} else {
		log.Printf("lsp: %s returned %d diagnostic(s) for %s", cfg.Name, len(diags), filepath.Base(filePath))
		for i, d := range diags {
			sev := "error"
			switch d.Severity {
			case SeverityWarning:
				sev = "warning"
			case SeverityInfo:
				sev = "info"
			case SeverityHint:
				sev = "hint"
			}
			log.Printf("lsp:   [%d] %s:%d:%d %s: %s", i+1,
				filepath.Base(filePath), d.Range.Start.Line+1, d.Range.Start.Character+1,
				sev, d.Message)
		}
	}
	return diags, nil
}

// Shutdown stops all running servers.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	stopped := make(map[*Server]bool)
	for ext, srv := range m.servers {
		if !stopped[srv] {
			srv.Stop()
			stopped[srv] = true
		}
		delete(m.servers, ext)
	}
}

// FormatDiagnostics formats diagnostics into a concise string for tool feedback.
// Returns empty string if there are no diagnostics.
func FormatDiagnostics(filePath string, diags []Diagnostic) string {
	if len(diags) == 0 {
		return ""
	}
	n := len(diags)
	if n > 3 {
		n = 3
	}
	var lines []string
	base := filepath.Base(filePath)
	for _, d := range diags[:n] {
		sev := "error"
		switch d.Severity {
		case SeverityWarning:
			sev = "warning"
		case SeverityInfo:
			sev = "info"
		case SeverityHint:
			sev = "hint"
		}
		lines = append(lines, fmt.Sprintf("%s:%d:%d: %s: %s",
			base, d.Range.Start.Line+1, d.Range.Start.Character+1,
			sev, d.Message))
	}
	if len(diags) > 3 {
		lines = append(lines, fmt.Sprintf("... and %d more", len(diags)-3))
	}
	return strings.Join(lines, "; ")
}
