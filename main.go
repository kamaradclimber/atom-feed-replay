package main

import (
	"log"
	"net"
	"net/http"
	"os"
)

func main() {
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	srv := NewServer(cfg)
	srv.StartPolling(cfg.RefreshInterval)

	_, port, _ := net.SplitHostPort(cfg.Listen)
	for _, fc := range cfg.Feeds {
		log.Printf("serving feed %q at http://localhost:%s%s", fc.ID, port, fc.Path)
	}

	log.Printf("listening on %s", cfg.Listen)
	if err := http.ListenAndServe(cfg.Listen, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
