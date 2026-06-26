package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/user/atom-feed-replay/feed"
)

func TestHandlerUnknownPath(t *testing.T) {
	srv := NewServer(&Config{})
	req := httptest.NewRequest(http.MethodGet, "/feeds/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandlerMethodNotAllowed(t *testing.T) {
	cfg := &Config{
		Listen: ":8081",
		Sources: []SourceConfig{
			{ID: "test", Source: "https://youtube.com/@test", Type: "channel", Path: "/feeds/test"},
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
		Listen: ":8081",
		Sources: []SourceConfig{
			{ID: "test", Source: "https://youtube.com/@test", Type: "channel", Path: "/feeds/test"},
		},
	}
	srv := NewServer(cfg)

	now := time.Now()
	srv.states["/feeds/test"].entries = []feed.Entry{
		{ID: "vid1", Title: "Test Video", Link: "https://youtube.com/watch?v=vid1", ReplayDate: now},
	}
	srv.states["/feeds/test"].ready = true

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

func TestHandlerMultipleSources(t *testing.T) {
	cfg := &Config{
		Listen: ":8081",
		Sources: []SourceConfig{
			{ID: "a", Source: "https://youtube.com/@a", Type: "channel", Path: "/feeds/a"},
			{ID: "b", Source: "https://youtube.com/playlist?list=PLb", Type: "playlist", Path: "/feeds/b"},
		},
	}
	srv := NewServer(cfg)

	now := time.Now()
	srv.states["/feeds/a"].entries = []feed.Entry{
		{ID: "a1", Title: "A1", ReplayDate: now},
	}
	srv.states["/feeds/a"].ready = true
	// Feed b stays not ready → should return 503

	req := httptest.NewRequest(http.MethodGet, "/feeds/a", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("feed a expected 200, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/feeds/b", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("feed b expected 503, got %d", w.Code)
	}
}
