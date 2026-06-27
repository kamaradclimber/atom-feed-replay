package main

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/user/atom-feed-replay/feed"
)

type SourceState struct {
	mu      sync.RWMutex
	entries []feed.Entry
	cfg     SourceConfig
	ready   bool
	icon    string
}

type Server struct {
	mu       sync.Mutex
	states   map[string]*SourceState
	stateDir string
	apiKey   string
}

func NewServer(cfg *Config) *Server {
	s := &Server{
		states:   make(map[string]*SourceState),
		stateDir: cfg.StateDir,
		apiKey:   cfg.APIKey,
	}
	for _, sc := range cfg.Sources {
		s.states[sc.Path] = &SourceState{
			cfg: sc,
		}
	}
	return s
}

func (s *Server) StartPolling(cfg *Config) {
	for _, state := range s.states {
		state := state
		go func() {
			s.pollSource(state)
			ticker := time.NewTicker(cfg.PollInterval)
			defer ticker.Stop()
			for range ticker.C {
				s.pollSource(state)
			}
		}()
	}
}

func (s *Server) pollSource(state *SourceState) {
	cache, icon, err := Collect(state.cfg, s.apiKey)
	if err != nil {
		log.Printf("error collecting for %q: %v", state.cfg.ID, err)
		return
	}

	if err := SaveCache(s.stateDir, state.cfg.ID, cache); err != nil {
		log.Printf("error saving cache for %q: %v", state.cfg.ID, err)
	}

	entries := cacheToFeedEntries(cache)

	state.mu.Lock()
	state.entries = entries
	state.icon = icon
	state.ready = true
	state.mu.Unlock()
	log.Printf("source %q: %d entries", state.cfg.ID, len(entries))
}

func cacheToFeedEntries(cache map[string]CacheEntry) []feed.Entry {
	entries := make([]feed.Entry, 0, len(cache))
	for id, ce := range cache {
		e := feed.Entry{
			ID:    id,
			Title: ce.Title,
			Link:  ce.Link,
		}
		if ce.UploadDate != "" {
			t, err := parseUploadDate(ce.UploadDate)
			if err == nil {
				e.Published = t
				e.Updated = t
				e.ReplayDate = t
			}
		}
		if ce.Description != "" {
			e.Content = ce.Description
		}
		entries = append(entries, e)
	}

	sort.Slice(entries, func(i, j int) bool {
		ti, tj := entries[i].Published, entries[j].Published
		if ti.IsZero() && tj.IsZero() {
			return false
		}
		if ti.IsZero() {
			return true
		}
		if tj.IsZero() {
			return false
		}
		return ti.Before(tj)
	})

	return entries
}

func parseUploadDate(s string) (time.Time, error) {
	if len(s) != 8 {
		return time.Time{}, fmt.Errorf("invalid upload date %q: expected 8 digits (YYYYMMDD)", s)
	}
	t, err := time.Parse("20060102", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid upload date %q: %w", s, err)
	}
	return t, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state, ok := s.states[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}

	state.mu.RLock()
	ready := state.ready
	state.mu.RUnlock()
	if !ready {
		http.Error(w, "feed not ready: initial collection in progress", http.StatusServiceUnavailable)
		return
	}

	var scheme string
	if r.TLS != nil {
		scheme = "https"
	} else {
		scheme = "http"
	}
	selfURL := scheme + "://" + r.Host + r.URL.Path

	state.mu.RLock()
	entries := state.entries
	title := state.cfg.ID
	icon := state.icon
	state.mu.RUnlock()

	atom, err := feed.Render(entries, title, selfURL, icon)
	if err != nil {
		log.Printf("error rendering feed %q: %v", state.cfg.ID, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ct := "application/atom+xml; charset=utf-8"
	if isBrowserRequest(r) {
		ct = "application/xml; charset=utf-8"
	}
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(atom))
}

func isBrowserRequest(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html") || strings.Contains(accept, "application/xhtml+xml")
}
