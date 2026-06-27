package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/user/atom-feed-replay/feed"
)

type Server struct {
	feeds  map[string]FeedConfig
	client *http.Client
}

func NewServer(cfg *Config) *Server {
	s := &Server{
		feeds:  make(map[string]FeedConfig),
		client: &http.Client{Timeout: 30 * time.Second},
	}

	for _, fc := range cfg.Feeds {
		s.feeds[fc.Path] = fc
	}

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fc, ok := s.feeds[r.URL.Path]
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

	result, err := feed.Fetch(s.client, fc.SourceURL)
	if err != nil {
		log.Printf("error fetching feed %q: %v", fc.ID, err)
		http.Error(w, "upstream feed unavailable", http.StatusBadGateway)
		return
	}

	now := time.Now()
	entries := feed.ReplaySchedule(result.Entries, fc.CatchupStart, fc.CatchupWindow, fc.MinInterval, now)

	title := result.Title
	if title == "" {
		title = fc.ID
	}

	atom, err := feed.Render(entries, title, selfURL, result.Icon)
	if err != nil {
		log.Printf("error rendering feed %q: %v", fc.ID, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	log.Printf("feed %q: %d entries fetched, %d scheduled", fc.ID, len(result.Entries), len(entries))

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
