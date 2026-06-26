package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type CacheEntry struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	UploadDate  string `json:"upload_date"`
	Description string `json:"description,omitempty"`
}

type Cache struct {
	Entries map[string]CacheEntry `json:"entries"` // keyed by video ID
}

func LoadCache(dir, id string) (map[string]CacheEntry, error) {
	path := filepath.Join(dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]CacheEntry), nil
		}
		return nil, fmt.Errorf("reading cache %s: %w", path, err)
	}

	var c Cache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing cache %s: %w", path, err)
	}
	if c.Entries == nil {
		c.Entries = make(map[string]CacheEntry)
	}
	return c.Entries, nil
}

func SaveCache(dir, id string, entries map[string]CacheEntry) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}

	path := filepath.Join(dir, id+".json")
	c := Cache{Entries: entries}

	// Write atomically via temp file
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(c); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("encoding cache: %w", err)
	}
	f.Close()

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("renaming cache: %w", err)
	}
	return nil
}
