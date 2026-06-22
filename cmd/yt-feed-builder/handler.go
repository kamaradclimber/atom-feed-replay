package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/user/atom-feed-replay/feed"
)

type SourceState struct {
	mu      sync.RWMutex
	entries []feed.Entry
	cfg     SourceConfig
}

type Server struct {
	states map[string]*SourceState
}

func NewServer(cfg *Config) *Server {
	s := &Server{
		states: make(map[string]*SourceState),
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
	entries, err := Collect(state.cfg.Source)
	if err != nil {
		log.Printf("error collecting %q: %v", state.cfg.ID, err)
		return
	}

	state.mu.Lock()
	state.entries = entries
	state.mu.Unlock()

	log.Printf("source %q: %d entries", state.cfg.ID, len(entries))
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
	state.mu.RUnlock()

	atom, err := feed.Render(entries, title, selfURL)
	if err != nil {
		log.Printf("error rendering feed %q: %v", state.cfg.ID, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(atom))
}
