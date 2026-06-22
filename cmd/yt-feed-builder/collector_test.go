package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseUploadDate(t *testing.T) {
	tests := []struct {
		input string
		want  time.Time
		err   bool
	}{
		{"20260115", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), false},
		{"20260601", time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), false},
		{"2026-01-15", time.Time{}, true},   // wrong format
		{"", time.Time{}, true},              // empty
		{"short", time.Time{}, true},         // too short
		{"202601151", time.Time{}, true},     // too long
		{"abcdefgh", time.Time{}, true},      // not a date
	}

	for _, tt := range tests {
		got, err := parseUploadDate(tt.input)
		if (err != nil) != tt.err {
			t.Fatalf("parseUploadDate(%q): err=%v, want err=%v", tt.input, err, tt.err)
		}
		if err == nil && !got.Equal(tt.want) {
			t.Fatalf("parseUploadDate(%q): got %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseYTDLPJSONLine(t *testing.T) {
	line := `{"id":"abc123","title":"My Video","webpage_url":"https://www.youtube.com/watch?v=abc123","upload_date":"20260615","channel":"Tech Channel","description":"A great video"}`

	var v ytVideo
	if err := json.Unmarshal([]byte(line), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v.ID != "abc123" {
		t.Fatalf("expected abc123, got %s", v.ID)
	}
	if v.Title != "My Video" {
		t.Fatalf("expected 'My Video', got %s", v.Title)
	}
	if v.WebpageURL != "https://www.youtube.com/watch?v=abc123" {
		t.Fatalf("unexpected URL: %s", v.WebpageURL)
	}
	if v.UploadDate != "20260615" {
		t.Fatalf("expected 20260615, got %s", v.UploadDate)
	}
	if v.Channel != "Tech Channel" {
		t.Fatalf("unexpected channel: %s", v.Channel)
	}
	if v.Description != "A great video" {
		t.Fatalf("unexpected description: %s", v.Description)
	}
}

func TestParseYTDLPMinimalLine(t *testing.T) {
	// yt-dlp flat-playlist minimal output
	line := `{"id":"xyz789","title":"Minimal","webpage_url":"https://www.youtube.com/watch?v=xyz789"}`

	var v ytVideo
	if err := json.Unmarshal([]byte(line), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v.ID != "xyz789" {
		t.Fatalf("expected xyz789, got %s", v.ID)
	}
	if v.UploadDate != "" {
		t.Fatalf("expected empty upload_date, got %s", v.UploadDate)
	}
}

func TestParseYTDLPInvalidLine(t *testing.T) {
	line := `not json at all`
	var v ytVideo
	if err := json.Unmarshal([]byte(line), &v); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
