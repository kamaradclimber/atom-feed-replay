package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type youTubeClient struct {
	apiKey string
	http   *http.Client
}

func newYouTubeClient(apiKey string) *youTubeClient {
	return &youTubeClient{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

type channelResponse struct {
	Items []struct {
		ContentDetails struct {
			RelatedPlaylists struct {
				Uploads string `json:"uploads"`
			} `json:"relatedPlaylists"`
		} `json:"contentDetails"`
	} `json:"items"`
}

type playlistItemsResponse struct {
	Items          []playlistItemJSON `json:"items"`
	NextPageToken  string             `json:"nextPageToken"`
	PageInfo       struct {
		TotalResults int `json:"totalResults"`
	} `json:"pageInfo"`
}

type playlistItemJSON struct {
	Snippet struct {
		PublishedAt string `json:"publishedAt"`
		Title       string `json:"title"`
		Description string `json:"description"`
		ResourceID  struct {
			VideoID string `json:"videoId"`
		} `json:"resourceId"`
	} `json:"snippet"`
}

type apiPlaylistItem struct {
	ID          string
	Title       string
	Link        string
	PublishedAt time.Time
	Description string
}

func (c *youTubeClient) resolveChannel(handle string) (string, error) {
	u := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/channels?part=contentDetails&forHandle=%s&key=%s",
		url.QueryEscape(handle), c.apiKey,
	)

	resp, err := c.http.Get(u)
	if err != nil {
		return "", fmt.Errorf("calling channels API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", c.apiError("channels", resp)
	}

	var ch channelResponse
	if err := json.NewDecoder(resp.Body).Decode(&ch); err != nil {
		return "", fmt.Errorf("decoding channels response: %w", err)
	}

	if len(ch.Items) == 0 {
		return "", fmt.Errorf("channel %q not found", handle)
	}

	playlistID := ch.Items[0].ContentDetails.RelatedPlaylists.Uploads
	if playlistID == "" {
		return "", fmt.Errorf("channel %q has no uploads playlist", handle)
	}

	return playlistID, nil
}

func (c *youTubeClient) listPlaylistItems(playlistID string) ([]apiPlaylistItem, error) {
	var all []apiPlaylistItem
	pageToken := ""

	for {
		u := fmt.Sprintf(
			"https://www.googleapis.com/youtube/v3/playlistItems?part=snippet&playlistId=%s&maxResults=50&key=%s",
			url.QueryEscape(playlistID), c.apiKey,
		)
		if pageToken != "" {
			u += "&pageToken=" + url.QueryEscape(pageToken)
		}

		resp, err := c.http.Get(u)
		if err != nil {
			return nil, fmt.Errorf("calling playlistItems API: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, c.apiError("playlistItems", resp)
		}

		var pr playlistItemsResponse
		if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decoding playlistItems response: %w", err)
		}
		resp.Body.Close()

		for _, item := range pr.Items {
			published, err := time.Parse(time.RFC3339Nano, item.Snippet.PublishedAt)
			if err != nil {
				log.Printf("warning: failed to parse publishedAt %q: %v", item.Snippet.PublishedAt, err)
			}
			videoID := item.Snippet.ResourceID.VideoID
			all = append(all, apiPlaylistItem{
				ID:          videoID,
				Title:       item.Snippet.Title,
				Link:        "https://www.youtube.com/watch?v=" + videoID,
				PublishedAt: published,
				Description: item.Snippet.Description,
			})
		}

		if pr.NextPageToken == "" {
			break
		}
		pageToken = pr.NextPageToken
	}

	return all, nil
}

func (c *youTubeClient) apiError(api string, resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return fmt.Errorf("%s API: HTTP %d: %s", api, resp.StatusCode, strings.TrimSpace(string(body)))
}

// resolveSource extracts the playlist ID from a source config.
func resolveSource(cfg SourceConfig, client *youTubeClient) (string, error) {
	switch cfg.Type {
	case "channel":
		handle := extractHandle(cfg.Source)
		if handle == "" {
			return "", fmt.Errorf("could not extract handle from %q", cfg.Source)
		}
		return client.resolveChannel(handle)
	case "playlist":
		pid := extractPlaylistID(cfg.Source)
		if pid == "" {
			return "", fmt.Errorf("could not extract playlist ID from %q", cfg.Source)
		}
		return pid, nil
	default:
		return "", fmt.Errorf("unknown source type: %s", cfg.Type)
	}
}

func extractHandle(s string) string {
	if u, err := url.Parse(s); err == nil && (strings.Contains(s, "youtube.com") || strings.Contains(s, "youtu.be")) {
		parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
		for _, p := range parts {
			if strings.HasPrefix(p, "@") {
				return strings.TrimPrefix(p, "@")
			}
		}
		for i, p := range parts {
			if p == "channel" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
		return ""
	}
	return strings.TrimPrefix(s, "@")
}

func extractPlaylistID(s string) string {
	if u, err := url.Parse(s); err == nil && (strings.Contains(s, "youtube.com") || strings.Contains(s, "youtu.be")) {
		if id := u.Query().Get("list"); id != "" {
			return id
		}
		return ""
	}
	if !strings.Contains(s, "://") && !strings.Contains(s, "/") {
		return s
	}
	return ""
}

// Collect fetches all videos from a YouTube source via the Data API.
func Collect(cfg SourceConfig, apiKey string) (map[string]CacheEntry, error) {
	client := newYouTubeClient(apiKey)
	playlistID, err := resolveSource(cfg, client)
	if err != nil {
		return nil, fmt.Errorf("resolving source: %w", err)
	}

	items, err := client.listPlaylistItems(playlistID)
	if err != nil {
		return nil, fmt.Errorf("listing playlist items: %w", err)
	}

	log.Printf("fetched %d items from playlist %s", len(items), playlistID)

	cache := make(map[string]CacheEntry, len(items))
	for _, item := range items {
		cache[item.ID] = CacheEntry{
			Title:       item.Title,
			Link:        item.Link,
			UploadDate:  item.PublishedAt.Format("20060102"),
			Description: item.Description,
		}
	}
	return cache, nil
}
