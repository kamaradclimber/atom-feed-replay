package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/user/atom-feed-replay/feed"
)

func startTestSource(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
		w.Write([]byte(body))
	}))
}

func atomFeed(title string, entries ...feed.Entry) string {
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>%s</title>
  <id>http://example.com/feed</id>
  <updated>2026-06-01T00:00:00Z</updated>`, title)
	for _, e := range entries {
		published := e.Published.Format(time.RFC3339)
		xml += fmt.Sprintf(`
  <entry>
    <title>%s</title>
    <id>%s</id>
    <link href="%s" rel="alternate"/>
    <published>%s</published>
    <updated>%s</updated>
  </entry>`, e.Title, e.ID, e.Link, published, published)
	}
	xml += "\n</feed>"
	return xml
}

func TestHandlerUnknownPath(t *testing.T) {
	src := startTestSource(t, atomFeed("test", feed.Entry{ID: "e1", Title: "E1", Link: "https://e1"}))
	defer src.Close()

	cfg := &Config{
		Listen:          ":8080",
		RefreshInterval: time.Minute,
		Feeds: []FeedConfig{
			{
				ID:            "test",
				SourceURL:     src.URL,
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
	src := startTestSource(t, atomFeed("test", feed.Entry{ID: "e1", Title: "E1", Link: "https://e1"}))
	defer src.Close()

	cfg := &Config{
		Listen:          ":8080",
		RefreshInterval: time.Minute,
		Feeds: []FeedConfig{
			{
				ID:            "test",
				SourceURL:     src.URL,
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
	src := startTestSource(t, atomFeed("Test Feed",
		feed.Entry{ID: "entry-1", Title: "Test Entry", Link: "https://example.com/1",
			Published: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)},
	))
	defer src.Close()

	cfg := &Config{
		Listen:          ":8080",
		RefreshInterval: time.Minute,
		Feeds: []FeedConfig{
			{
				ID:            "test",
				SourceURL:     src.URL,
				Path:          "/feeds/test",
				CatchupStart:  time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
				CatchupWindow: 720 * time.Hour,
				MinInterval:   time.Hour,
			},
		},
	}

	srv := NewServer(cfg)
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
	srcA := startTestSource(t, atomFeed("Feed A",
		feed.Entry{ID: "a1", Title: "Entry A1", Link: "https://example.com/a1",
			Published: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)},
	))
	defer srcA.Close()

	srcB := startTestSource(t, atomFeed("Feed B",
		feed.Entry{ID: "b1", Title: "Entry B1", Link: "https://example.com/b1",
			Published: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
	))
	defer srcB.Close()

	cfg := &Config{
		Listen:          ":8080",
		RefreshInterval: time.Minute,
		Feeds: []FeedConfig{
			{
				ID:            "feed-a",
				SourceURL:     srcA.URL,
				Path:          "/feeds/a",
				CatchupStart:  time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
				CatchupWindow: 720 * time.Hour,
				MinInterval:   time.Hour,
			},
			{
				ID:            "feed-b",
				SourceURL:     srcB.URL,
				Path:          "/feeds/b",
				CatchupStart:  time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
				CatchupWindow: 720 * time.Hour,
				MinInterval:   time.Hour,
			},
		},
	}

	srv := NewServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/feeds/a", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("feed a expected 200, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/feeds/b", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("feed b expected 200, got %d", w.Code)
	}
}

func TestHandlerUpstreamUnavailable(t *testing.T) {
	cfg := &Config{
		Listen:          ":8080",
		RefreshInterval: time.Minute,
		Feeds: []FeedConfig{
			{
				ID:            "test",
				SourceURL:     "http://127.0.0.1:1/feed", // nothing listening there
				Path:          "/feeds/test",
				CatchupStart:  time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
				CatchupWindow: 720 * time.Hour,
				MinInterval:   time.Hour,
			},
		},
	}

	srv := NewServer(cfg)
	req := httptest.NewRequest(http.MethodGet, "/feeds/test", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 when upstream is down, got %d", w.Code)
	}
}
