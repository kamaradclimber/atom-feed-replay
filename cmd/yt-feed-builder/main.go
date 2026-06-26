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
	srv.StartPolling(cfg)

	_, port, _ := net.SplitHostPort(cfg.Listen)
	for _, sc := range cfg.Sources {
		log.Printf("serving feed %q at http://localhost:%s%s", sc.ID, port, sc.Path)
	}

	log.Printf("listening on %s", cfg.Listen)
	if err := http.ListenAndServe(cfg.Listen, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
