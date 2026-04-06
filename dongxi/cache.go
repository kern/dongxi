package dongxi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Cache holds the locally cached Things Cloud history.
type Cache struct {
	HistoryKey string           `json:"history_key"`
	ItemCount  int              `json:"item_count"`
	Items      []map[string]any `json:"items"`
}

// CachePath returns the path to ~/.config/dongxi/history.json.
func CachePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.json"), nil
}

// LoadCache reads the cached history from disk.
// Returns an empty cache (not an error) if the file does not exist.
func LoadCache() (*Cache, error) {
	path, err := CachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Cache{}, nil
		}
		return nil, fmt.Errorf("read cache: %w", err)
	}
	var c Cache
	if err := json.Unmarshal(data, &c); err != nil {
		// Corrupt cache — start fresh.
		return &Cache{}, nil
	}
	return &c, nil
}

// SaveCache writes the cache to disk.
func SaveCache(c *Cache) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}
	path := filepath.Join(dir, "history.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write cache: %w", err)
	}
	return nil
}
