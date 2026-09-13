package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
)

// JSONConfig is the file-backed Config repository.
type JSONConfig struct {
	path string
}

// NewJSONConfig returns a Config repository stored as JSON at path.
func NewJSONConfig(path string) *JSONConfig { return &JSONConfig{path: path} }

// Path returns the file this repository reads and writes.
func (r *JSONConfig) Path() string { return r.path }

// Load implements Config.
func (r *JSONConfig) Load() (domain.Config, error) {
	data, err := os.ReadFile(r.path)
	if os.IsNotExist(err) {
		return domain.DefaultConfig(), nil
	}
	if err != nil {
		return domain.Config{}, fmt.Errorf("load config from %s: %w", r.path, err)
	}

	cfg := domain.DefaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return domain.Config{}, fmt.Errorf("parse config at %s: %w", r.path, err)
	}
	return cfg, nil
}

// Save implements Config.
func (r *JSONConfig) Save(cfg domain.Config) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return fmt.Errorf("save config: create %s: %w", filepath.Dir(r.path), err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("save config: encode: %w", err)
	}
	if err := os.WriteFile(r.path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("save config to %s: %w", r.path, err)
	}
	return nil
}
