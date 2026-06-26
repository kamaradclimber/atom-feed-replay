package main

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfigBasic(t *testing.T) {
	t.Setenv("YOUTUBE_API_KEY", "test-key-123")
	content := `
listen: ":9090"
poll_interval: 5m
sources:
  - id: tech-talks
    source: https://www.youtube.com/@tech
    type: channel
    path: /feeds/tech-talks
`
	f := writeTempConfig(t, content)
	defer os.Remove(f.Name())

	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Listen != ":9090" {
		t.Fatalf("expected :9090, got %s", cfg.Listen)
	}
	if cfg.PollInterval != 5*time.Minute {
		t.Fatalf("expected 5m, got %v", cfg.PollInterval)
	}
	if len(cfg.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(cfg.Sources))
	}
	if cfg.Sources[0].ID != "tech-talks" {
		t.Fatalf("expected tech-talks, got %s", cfg.Sources[0].ID)
	}
	if cfg.Sources[0].Type != "channel" {
		t.Fatalf("expected channel, got %s", cfg.Sources[0].Type)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("YOUTUBE_API_KEY", "test-key-456")
	content := `
sources:
  - id: test
    source: https://www.youtube.com/playlist?list=PLabc
    type: playlist
    path: /feeds/test
`
	f := writeTempConfig(t, content)
	defer os.Remove(f.Name())

	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Listen != ":8081" {
		t.Fatalf("expected default :8081, got %s", cfg.Listen)
	}
	if cfg.PollInterval != time.Hour {
		t.Fatalf("expected default 1h, got %v", cfg.PollInterval)
	}
}

func TestLoadConfigMissingFields(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{"missing id", `
sources:
  - source: https://youtube.com/@x
    type: channel
    path: /feeds/x
`},
		{"missing source", `
sources:
  - id: x
    type: channel
    path: /feeds/x
`},
		{"invalid type", `
sources:
  - id: x
    source: https://youtube.com/@x
    type: invalid
    path: /feeds/x
`},
		{"missing path", `
sources:
  - id: x
    source: https://youtube.com/@x
    type: channel
`},
		{"duplicate path", `
sources:
  - id: a
    source: https://youtube.com/@a
    type: channel
    path: /feeds/x
  - id: b
    source: https://youtube.com/@b
    type: channel
    path: /feeds/x
`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := writeTempConfig(t, tt.config)
			defer os.Remove(f.Name())
			_, err := LoadConfig(f.Name())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestLoadConfigNoSources(t *testing.T) {
	content := `listen: ":8081"`
	f := writeTempConfig(t, content)
	defer os.Remove(f.Name())

	_, err := LoadConfig(f.Name())
	if err == nil {
		t.Fatal("expected error for no sources")
	}
}

func TestLoadConfigAPIKeyFromConfigFile(t *testing.T) {
	keyFile := writeTempConfig(t, "file-based-key\n")
	defer os.Remove(keyFile.Name())

	cfgYAML := `
api_key_file: ` + keyFile.Name() + `
sources:
  - id: test
    source: https://www.youtube.com/@test
    type: channel
    path: /feeds/test
`
	f := writeTempConfig(t, cfgYAML)
	defer os.Remove(f.Name())

	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.APIKey != "file-based-key" {
		t.Fatalf("expected file-based-key, got %q", cfg.APIKey)
	}
}

func TestLoadConfigAPIKeyFromEnv(t *testing.T) {
	t.Setenv("YOUTUBE_API_KEY", "env-key")
	content := `
sources:
  - id: test
    source: https://www.youtube.com/@test
    type: channel
    path: /feeds/test
`
	f := writeTempConfig(t, content)
	defer os.Remove(f.Name())

	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.APIKey != "env-key" {
		t.Fatalf("expected env-key, got %q", cfg.APIKey)
	}
}

func TestLoadConfigMissingAPIKey(t *testing.T) {
	content := `
sources:
  - id: test
    source: https://www.youtube.com/@test
    type: channel
    path: /feeds/test
`
	f := writeTempConfig(t, content)
	defer os.Remove(f.Name())

	_, err := LoadConfig(f.Name())
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func writeTempConfig(t *testing.T, content string) *os.File {
	t.Helper()
	f, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f
}
