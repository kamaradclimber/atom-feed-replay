package main

import (
	"fmt"
	"os"
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
	Sources      []SourceConfig `yaml:"sources"`
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

	return &cfg, nil
}
