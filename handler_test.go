package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/user/atom-feed-replay/feed"
)

func TestHandlerUnknownPath(t *testing.T) {
	cfg := &Config{
		Listen:          ":8080",
		RefreshInterval: time.Minute,
		Feeds: []FeedConfig{
			{
				ID:            "test",
				SourceURL:     "http://example.com/feed.xml",
				Path:          "/feeds/test",
				CatchupStart:  time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
				CatchupWindow: 720 * time.Hour,
				MinInterval:   time.Hour,
			},
		},
	}

	srv := NewServer(cfg)
	req := httptest.NewRequest(http.MethodGet, "/feeds/unknown", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandlerMethodNotAllowed(t *testing.T) {
	cfg := &Config{
		Listen:          ":8080",
		RefreshInterval: time.Minute,
		Feeds: []FeedConfig{
			{
				ID:            "test",
				SourceURL:     "http://example.com/feed.xml",
				Path:          "/feeds/test",
				CatchupStart:  time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
				CatchupWindow: 720 * time.Hour,
				MinInterval:   time.Hour,
			},
		},
	}

	srv := NewServer(cfg)
	req := httptest.NewRequest(http.MethodPost, "/feeds/test", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandlerServesAtom(t *testing.T) {
	cfg := &Config{
		Listen:          ":8080",
		RefreshInterval: time.Minute,
		Feeds: []FeedConfig{
			{
				ID:            "test",
				SourceURL:     "http://example.com/feed.xml",
				Path:          "/feeds/test",
				CatchupStart:  time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
				CatchupWindow: 720 * time.Hour,
				MinInterval:   time.Hour,
			},
		},
	}

	srv := NewServer(cfg)

	// Pre-populate with some entries
	now := time.Now()
	srv.states["/feeds/test"].entries = []feed.Entry{
		{
			ID:         "entry-1",
			Title:      "Test Entry",
			Link:       "https://example.com/1",
			ReplayDate: now,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/feeds/test", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/atom+xml; charset=utf-8" {
		t.Fatalf("expected atom content type, got %s", ct)
	}

	body := w.Body.String()
	if body == "" {
		t.Fatal("expected non-empty body")
	}
}

func TestHandlerMultipleFeeds(t *testing.T) {
	cfg := &Config{
		Listen:          ":8080",
		RefreshInterval: time.Minute,
		Feeds: []FeedConfig{
			{
				ID:            "feed-a",
				SourceURL:     "http://example.com/a.xml",
				Path:          "/feeds/a",
				CatchupStart:  time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
				CatchupWindow: 720 * time.Hour,
				MinInterval:   time.Hour,
			},
			{
				ID:            "feed-b",
				SourceURL:     "http://example.com/b.xml",
				Path:          "/feeds/b",
				CatchupStart:  time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
				CatchupWindow: 720 * time.Hour,
				MinInterval:   time.Hour,
			},
		},
	}

	srv := NewServer(cfg)

	// Populate feed a
	now := time.Now()
	srv.states["/feeds/a"].entries = []feed.Entry{
		{ID: "a1", Title: "Entry A1", ReplayDate: now},
	}

	req := httptest.NewRequest(http.MethodGet, "/feeds/a", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("feed a expected 200, got %d", w.Code)
	}

	// Feed b should be empty but still serve
	req = httptest.NewRequest(http.MethodGet, "/feeds/b", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("feed b expected 200, got %d", w.Code)
	}
}
