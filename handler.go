package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/user/atom-feed-replay/feed"
)

type FeedState struct {
	mu     sync.RWMutex
	cfg    FeedConfig
	title  string
	url    string
	entries []feed.Entry
}

type Server struct {
	states map[string]*FeedState
	client *http.Client
}

func NewServer(cfg *Config) *Server {
	s := &Server{
		states: make(map[string]*FeedState),
		client: &http.Client{Timeout: 30 * time.Second},
	}

	for _, fc := range cfg.Feeds {
		s.states[fc.Path] = &FeedState{
			cfg: fc,
		}
	}

	return s
}

func (s *Server) StartPolling(interval time.Duration) {
	for _, state := range s.states {
		state := state
		go func() {
			s.pollFeed(state)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for range ticker.C {
				s.pollFeed(state)
			}
		}()
	}
}

func (s *Server) pollFeed(state *FeedState) {
	result, err := feed.Fetch(s.client, state.cfg.SourceURL)
	if err != nil {
		log.Printf("error fetching feed %q: %v", state.cfg.ID, err)
		return
	}

	now := time.Now()
	scheduled := feed.ReplaySchedule(result.Entries, state.cfg.CatchupStart, state.cfg.CatchupWindow, state.cfg.MinInterval, now)

	state.mu.Lock()
	state.entries = scheduled
	state.title = result.Title
	state.mu.Unlock()

	log.Printf("feed %q: %d entries fetched, %d scheduled", state.cfg.ID, len(result.Entries), len(scheduled))
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
	title := state.title
	if title == "" {
		title = state.cfg.ID
	}
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
