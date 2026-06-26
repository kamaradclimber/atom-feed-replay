package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type SourceConfig struct {
	ID     string `yaml:"id"`
	Source string `yaml:"source"`
	Type   string `yaml:"type"`
	Path   string `yaml:"path"`
}

type Config struct {
	Listen       string         `yaml:"listen"`
	PollInterval time.Duration  `yaml:"poll_interval"`
	StateDir     string         `yaml:"state_dir"`
	APIKeyFile   string         `yaml:"api_key_file"`
	APIKey       string         `yaml:"-"` // resolved from file or env
	Sources      []SourceConfig `yaml:"sources"`
}

func resolveAPIKey(keyFile string) (string, error) {
	if keyFile != "" {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return "", fmt.Errorf("reading api_key_file: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	if key := os.Getenv("YOUTUBE_API_KEY"); key != "" {
		return key, nil
	}
	return "", fmt.Errorf("neither api_key_file (config) nor YOUTUBE_API_KEY (env) is set")
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Listen == "" {
		cfg.Listen = ":8081"
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Hour
	}
	if cfg.StateDir == "" {
		cfg.StateDir = "./data"
	}
	if len(cfg.Sources) == 0 {
		return nil, fmt.Errorf("no sources configured")
	}

	seen := map[string]bool{}
	for _, s := range cfg.Sources {
		if s.ID == "" {
			return nil, fmt.Errorf("source missing id")
		}
		if s.Source == "" {
			return nil, fmt.Errorf("source %q missing source URL", s.ID)
		}
		if s.Type != "channel" && s.Type != "playlist" {
			return nil, fmt.Errorf("source %q: type must be 'channel' or 'playlist'", s.ID)
		}
		if s.Path == "" {
			return nil, fmt.Errorf("source %q missing path", s.ID)
		}
		if seen[s.Path] {
			return nil, fmt.Errorf("duplicate path %q", s.Path)
		}
		seen[s.Path] = true
	}

	cfg.APIKey, err = resolveAPIKey(cfg.APIKeyFile)
	if err != nil {
		return nil, fmt.Errorf("resolving API key: %w", err)
	}

	return &cfg, nil
}
