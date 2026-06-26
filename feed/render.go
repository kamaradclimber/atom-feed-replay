package feed

import (
	"encoding/xml"
	"fmt"
	"time"
)

const atomNamespace = "http://www.w3.org/2005/Atom"
const timeFormat = time.RFC3339

type AtomFeed struct {
	XMLName xml.Name   `xml:"feed"`
	Xmlns   string     `xml:"xmlns,attr"`
	Title   string     `xml:"title"`
	ID      string     `xml:"id"`
	Updated string     `xml:"updated"`
	Link    AtomLink   `xml:"link"`
	Icon    string     `xml:"icon,omitempty"`
	Entries []AtomEntry `xml:"entry"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
}

type AtomEntry struct {
	Title     string    `xml:"title"`
	ID        string    `xml:"id"`
	Link      AtomLink  `xml:"link"`
	Published string    `xml:"published"`
	Updated   string    `xml:"updated"`
	Content   AtomContent `xml:"content,omitempty"`
}

type AtomContent struct {
	Type  string `xml:"type,attr"`
	Body  string `xml:",chardata"`
}

// Render produces an Atom XML string for the given entries, feed title, and
// feed URL. Each entry uses its ReplayDate as both published and updated.
func Render(entries []Entry, feedTitle string, feedURL string, feedIcon ...string) (string, error) {
	var latest time.Time
	atomEntries := make([]AtomEntry, 0, len(entries))

	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		replay := e.ReplayDate
		if replay.After(latest) {
			latest = replay
		}

		atomEntry := AtomEntry{
			Title: e.Title,
			ID:    e.ID,
			Link: AtomLink{
				Href: e.Link,
				Rel:  "alternate",
			},
			Published: replay.Format(timeFormat),
			Updated:   replay.Format(timeFormat),
		}
		if e.Content != "" {
			atomEntry.Content = AtomContent{
				Type: "html",
				Body: e.Content,
			}
		}
		atomEntries = append(atomEntries, atomEntry)
	}

	if latest.IsZero() {
		latest = time.Now()
	}

	f := AtomFeed{
		Xmlns:   atomNamespace,
		Title:   feedTitle,
		ID:      feedURL,
		Updated: latest.Format(timeFormat),
		Link: AtomLink{
			Href: feedURL,
			Rel:  "self",
		},
		Entries: atomEntries,
	}
	if len(feedIcon) > 0 {
		f.Icon = feedIcon[0]
	}

	data, err := xml.MarshalIndent(f, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling atom feed: %w", err)
	}

	return xml.Header + string(data), nil
}
