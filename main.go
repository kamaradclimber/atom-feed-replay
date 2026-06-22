package main

import (
	"log"
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

	log.Printf("listening on %s", cfg.Listen)
	if err := http.ListenAndServe(cfg.Listen, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
