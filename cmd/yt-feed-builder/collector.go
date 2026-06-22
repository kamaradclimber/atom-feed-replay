package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/user/atom-feed-replay/feed"
)

type ytVideo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	WebpageURL  string `json:"webpage_url"`
	UploadDate  string `json:"upload_date"` // YYYYMMDD
	Channel     string `json:"channel"`
	Description string `json:"description"`
}

func parseUploadDate(s string) (time.Time, error) {
	if len(s) != 8 {
		return time.Time{}, fmt.Errorf("invalid upload_date format: %q", s)
	}
	return time.Parse("20060102", s)
}

// Collect runs yt-dlp for the given source URL and returns parsed entries.
func Collect(sourceURL string) ([]feed.Entry, error) {
	cmd := exec.Command("yt-dlp",
		"--flat-playlist",
		"--dump-json",
		"--playlist-end", "-1",
		sourceURL,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting yt-dlp: %w", err)
	}

	var entries []feed.Entry
	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var v ytVideo
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			continue
		}

		e := feed.Entry{
			ID:    v.ID,
			Title: v.Title,
			Link:  v.WebpageURL,
		}

		if v.UploadDate != "" {
			t, err := parseUploadDate(v.UploadDate)
			if err == nil {
				e.Published = t
				e.Updated = t
			}
		}

		if v.Description != "" {
			e.Content = v.Description
		}

		entries = append(entries, e)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading yt-dlp output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("yt-dlp failed: %w", err)
	}

	return entries, nil
}
