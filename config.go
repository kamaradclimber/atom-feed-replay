package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type FeedConfig struct {
	ID            string        `yaml:"id"`
	SourceURL     string        `yaml:"source_url"`
	Path          string        `yaml:"path"`
	CatchupStart  time.Time     `yaml:"catchup_start"`
	CatchupWindow time.Duration `yaml:"catchup_window"`
	MinInterval   time.Duration `yaml:"min_interval"`
}

type Config struct {
	Listen          string        `yaml:"listen"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	Feeds           []FeedConfig  `yaml:"feeds"`
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
		cfg.Listen = ":8080"
	}
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = 5 * time.Minute
	}

	if len(cfg.Feeds) == 0 {
		return nil, fmt.Errorf("no feeds configured")
	}

	seen := map[string]bool{}
	for _, f := range cfg.Feeds {
		if f.ID == "" {
			return nil, fmt.Errorf("feed missing id")
		}
		if f.SourceURL == "" {
			return nil, fmt.Errorf("feed %q missing source_url", f.ID)
		}
		if f.Path == "" {
			return nil, fmt.Errorf("feed %q missing path", f.ID)
		}
		if f.CatchupStart.IsZero() {
			return nil, fmt.Errorf("feed %q missing catchup_start", f.ID)
		}
		if f.CatchupWindow <= 0 {
			return nil, fmt.Errorf("feed %q: catchup_window must be positive", f.ID)
		}
		if f.MinInterval <= 0 {
			return nil, fmt.Errorf("feed %q: min_interval must be positive", f.ID)
		}
		if seen[f.Path] {
			return nil, fmt.Errorf("duplicate path %q", f.Path)
		}
		seen[f.Path] = true
	}

	return &cfg, nil
}
