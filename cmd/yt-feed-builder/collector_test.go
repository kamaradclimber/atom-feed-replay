package main

import (
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
		{"2026-01-15", time.Time{}, true},  // wrong format
		{"", time.Time{}, true},             // empty
		{"short", time.Time{}, true},        // too short
		{"202601151", time.Time{}, true},    // too long
		{"abcdefgh", time.Time{}, true},     // not a date
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

func TestExtractHandle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"@TechIngredients", "TechIngredients"},
		{"https://www.youtube.com/@TechIngredients", "TechIngredients"},
		{"https://www.youtube.com/@TechIngredients/videos", "TechIngredients"},
		{"https://www.youtube.com/channel/UCabc123", "UCabc123"},
		{"@", ""},
		{"https://youtube.com/@handle", "handle"},
		{"just-a-handle", "just-a-handle"},
	}
	for _, tt := range tests {
		got := extractHandle(tt.input)
		if got != tt.want {
			t.Fatalf("extractHandle(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractPlaylistID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"PLabc123", "PLabc123"},
		{"https://www.youtube.com/playlist?list=PLabc123", "PLabc123"},
		{"https://youtube.com/playlist?list=PLabc123", "PLabc123"},
		{"https://www.youtube.com/playlist?list=PLabc123&si=xyz", "PLabc123"},
		{"https://youtu.be/abc123?list=PLabc123", "PLabc123"},
		{"", ""},
	}
	for _, tt := range tests {
		got := extractPlaylistID(tt.input)
		if got != tt.want {
			t.Fatalf("extractPlaylistID(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractPlaylistIDNoQuery(t *testing.T) {
	got := extractPlaylistID("https://www.youtube.com/watch?v=abc")
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
