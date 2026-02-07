package conversation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gofrs/uuid/v5"
)

type Data struct {
	ID           string          `json:"id"`
	UIMessages   json.RawMessage `json:"ui_messages"`
	AgentHistory json.RawMessage `json:"agent_history"`
}

func New() *Data {
	id, _ := uuid.NewV7()
	return &Data{
		ID:           id.String(),
		UIMessages:   []byte("[]"),
		AgentHistory: []byte("[]"),
	}
}

func Load(path string) (*Data, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading conversation file: %w", err)
	}
	var d Data
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("parsing conversation file: %w", err)
	}
	return &d, nil
}

func (d *Data) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating conversation dir: %w", err)
	}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling conversation: %w", err)
	}
	path := filepath.Join(dir, d.ID+".json")
	return os.WriteFile(path, b, 0o644)
}

func LatestInDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var jsonFiles []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			jsonFiles = append(jsonFiles, e.Name())
		}
	}
	if len(jsonFiles) == 0 {
		return "", fmt.Errorf("no conversation files found")
	}

	sort.Strings(jsonFiles)
	return jsonFiles[len(jsonFiles)-1], nil
}

func Dir(workingDir string) string {
	return filepath.Join(workingDir, "conversations")
}
