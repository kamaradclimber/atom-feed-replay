package feed

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
)

type FetchResult struct {
	Title   string
	Entries []Entry
}

// Fetch retrieves and parses an Atom/RSS feed from the given URL.
func Fetch(client *http.Client, url string) (*FetchResult, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: status %d", url, resp.StatusCode)
	}

	fp := gofeed.NewParser()
	parsed, err := fp.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", url, err)
	}

	result := &FetchResult{
		Title: parsed.Title,
	}

	entries := make([]Entry, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		e := Entry{
			ID:    item.GUID,
			Title: item.Title,
		}

		if item.Link != "" {
			e.Link = item.Link
		} else if len(item.Links) > 0 {
			e.Link = item.Links[0]
		}

		if item.PublishedParsed != nil {
			e.Published = *item.PublishedParsed
		}
		if item.UpdatedParsed != nil {
			e.Updated = *item.UpdatedParsed
		} else if item.PublishedParsed != nil {
			e.Updated = *item.PublishedParsed
		}

		if item.Content != "" {
			e.Content = item.Content
		} else if item.Description != "" {
			e.Content = item.Description
		}

		if e.ID == "" && e.Link != "" {
			e.ID = e.Link
		}
		if e.ID == "" {
			e.ID = fmt.Sprintf("%s-%d", url, time.Now().UnixNano())
		}

		entries = append(entries, e)
	}

	result.Entries = entries
	return result, nil
}
