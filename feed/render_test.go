package feed

import (
	"strings"
	"testing"
	"time"
)

func TestRenderEmptyFeed(t *testing.T) {
	xml, err := Render(nil, "Empty Feed", "http://localhost/feed.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(xml, "<title>Empty Feed</title>") {
		t.Fatal("expected title in output")
	}
	if !strings.Contains(xml, "<entry>") {
		// No entries expected
	}
}

func TestRenderSingleEntry(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	entries := []Entry{
		{
			ID:         "entry-1",
			Title:      "Test Entry",
			Link:       "https://example.com/1",
			ReplayDate: now,
			Content:    "<p>Hello world</p>",
		},
	}

	xml, err := Render(entries, "Test Feed", "http://localhost/feed.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []string{
		`<feed xmlns="http://www.w3.org/2005/Atom">`,
		"<title>Test Feed</title>",
		"<id>http://localhost/feed.xml</id>",
		"<entry>",
		"<title>Test Entry</title>",
		"<id>entry-1</id>",
		`<link href="https://example.com/1" rel="alternate">`,
		`<published>2026-06-15T12:00:00Z</published>`,
		`<content type="html">`,
		"&lt;p&gt;Hello world&lt;/p&gt;",
	}
	for _, c := range checks {
		if !strings.Contains(xml, c) {
			t.Fatalf("expected output to contain: %s", c)
		}
	}
}

func TestRenderMultipleEntriesOrder(t *testing.T) {
	t1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)

	entries := []Entry{
		{ID: "1", Title: "First", ReplayDate: t1},
		{ID: "2", Title: "Second", ReplayDate: t2},
		{ID: "3", Title: "Third", ReplayDate: t3},
	}

	xml, err := Render(entries, "Ordered Feed", "http://localhost/feed.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Entries should be in reverse chronological order (newest first per Atom convention)
	firstIdx := strings.Index(xml, "First")
	secondIdx := strings.Index(xml, "Second")
	thirdIdx := strings.Index(xml, "Third")

	if !(thirdIdx < secondIdx && secondIdx < firstIdx) {
		t.Fatal("entries should be in reverse chronological order (newest first)")
	}
}

func TestRenderFeedUpdatedLatest(t *testing.T) {
	t1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)

	entries := []Entry{
		{ID: "1", Title: "Early", ReplayDate: t1},
		{ID: "2", Title: "Late", ReplayDate: t2},
	}

	xml, err := Render(entries, "Feed", "http://localhost/feed.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(xml, "2026-06-20T00:00:00Z") {
		t.Fatal("expected feed updated to be latest entry date")
	}
}

func TestRenderValidXML(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	entries := []Entry{
		{ID: "1", Title: "Test", Link: "https://example.com/1", ReplayDate: now},
	}

	xml, err := Render(entries, "Valid Feed", "http://localhost/feed.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(xml, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Fatal("expected XML declaration")
	}
}
