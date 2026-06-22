package main

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfigBasic(t *testing.T) {
	content := `
listen: ":9090"
refresh_interval: 1m
feeds:
  - id: test-feed
    source_url: https://example.com/feed.xml
    path: /feeds/test
    catchup_start: 2026-06-01T00:00:00Z
    catchup_window: 720h
    min_interval: 1h
`
	f, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString(content)
	f.Close()

	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Listen != ":9090" {
		t.Fatalf("expected :9090, got %s", cfg.Listen)
	}
	if cfg.RefreshInterval != time.Minute {
		t.Fatalf("expected 1m, got %v", cfg.RefreshInterval)
	}
	if len(cfg.Feeds) != 1 {
		t.Fatalf("expected 1 feed, got %d", len(cfg.Feeds))
	}
	if cfg.Feeds[0].ID != "test-feed" {
		t.Fatalf("expected test-feed, got %s", cfg.Feeds[0].ID)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	content := `
feeds:
  - id: test
    source_url: https://example.com/feed.xml
    path: /feeds/test
    catchup_start: 2026-06-01T00:00:00Z
    catchup_window: 720h
    min_interval: 1h
`
	f, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString(content)
	f.Close()

	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Listen != ":8080" {
		t.Fatalf("expected default :8080, got %s", cfg.Listen)
	}
	if cfg.RefreshInterval != 5*time.Minute {
		t.Fatalf("expected default 5m, got %v", cfg.RefreshInterval)
	}
}

func TestLoadConfigMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name: "missing source_url",
			config: `
feeds:
  - id: test
    path: /feeds/test
    catchup_start: 2026-06-01T00:00:00Z
    catchup_window: 720h
    min_interval: 1h
`,
			wantErr: "missing source_url",
		},
		{
			name: "missing path",
			config: `
feeds:
  - id: test
    source_url: https://example.com/feed.xml
    catchup_start: 2026-06-01T00:00:00Z
    catchup_window: 720h
    min_interval: 1h
`,
			wantErr: "missing path",
		},
		{
			name: "missing id",
			config: `
feeds:
  - source_url: https://example.com/feed.xml
    path: /feeds/test
    catchup_start: 2026-06-01T00:00:00Z
    catchup_window: 720h
    min_interval: 1h
`,
			wantErr: "feed missing id",
		},
		{
			name: "missing catchup_start",
			config: `
feeds:
  - id: test
    source_url: https://example.com/feed.xml
    path: /feeds/test
    catchup_window: 720h
    min_interval: 1h
`,
			wantErr: "missing catchup_start",
		},
		{
			name: "zero catchup_window",
			config: `
feeds:
  - id: test
    source_url: https://example.com/feed.xml
    path: /feeds/test
    catchup_start: 2026-06-01T00:00:00Z
    catchup_window: 0
    min_interval: 1h
`,
			wantErr: "catchup_window must be positive",
		},
		{
			name: "duplicate path",
			config: `
feeds:
  - id: a
    source_url: https://a.example.com/feed.xml
    path: /feeds/test
    catchup_start: 2026-06-01T00:00:00Z
    catchup_window: 720h
    min_interval: 1h
  - id: b
    source_url: https://b.example.com/feed.xml
    path: /feeds/test
    catchup_start: 2026-06-01T00:00:00Z
    catchup_window: 720h
    min_interval: 1h
`,
			wantErr: "duplicate path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.CreateTemp("", "config-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(f.Name())
			f.WriteString(tt.config)
			f.Close()

			_, err = LoadConfig(f.Name())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestLoadConfigNoFeeds(t *testing.T) {
	content := `
listen: ":8080"
`
	f, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString(content)
	f.Close()

	_, err = LoadConfig(f.Name())
	if err == nil {
		t.Fatal("expected error for no feeds")
	}
}
