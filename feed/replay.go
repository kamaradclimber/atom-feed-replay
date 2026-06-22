package feed

import (
	"sort"
	"time"
)

type Entry struct {
	ID          string
	Title       string
	Link        string
	Published   time.Time
	Updated     time.Time
	Content     string
	ReplayDate  time.Time
}

// ReplaySchedule computes smoothed replay dates for a list of entries.
//
// Entries whose original Published date is ≤ catchupStart are treated as
// "backlog" and get linearly interpolated between catchupStart and
// catchupEnd, where catchupEnd = catchupStart + max(window, N * minInterval).
// Entries published after catchupStart are "live" and keep their original
// date. Only entries with replay_date ≤ now are returned, sorted by
// replay_date ascending.
func ReplaySchedule(entries []Entry, catchupStart time.Time, window time.Duration, minInterval time.Duration, now time.Time) []Entry {
	if len(entries) == 0 {
		return nil
	}

	var backlog, live []Entry
	for _, e := range entries {
		if !e.Published.After(catchupStart) {
			backlog = append(backlog, e)
		} else {
			live = append(live, e)
		}
	}

	sort.Slice(backlog, func(i, j int) bool {
		return backlog[i].Published.Before(backlog[j].Published)
	})
	sort.Slice(live, func(i, j int) bool {
		return live[i].Published.Before(live[j].Published)
	})

	n := len(backlog)
	if n > 0 {
		needed := time.Duration(n) * minInterval
		duration := window
		if needed > duration {
			duration = needed
		}
		catchupEnd := catchupStart.Add(duration)
		total := catchupEnd.Sub(catchupStart)

		for i := range backlog {
			var offset time.Duration
			if n == 1 {
				offset = 0
			} else {
				offset = total * time.Duration(i) / time.Duration(n-1)
			}
			backlog[i].ReplayDate = catchupStart.Add(offset)
		}
	}

	for i := range live {
		live[i].ReplayDate = live[i].Published
	}

	result := append(backlog, live...)
	sort.Slice(result, func(i, j int) bool {
		return result[i].ReplayDate.Before(result[j].ReplayDate)
	})

	var filtered []Entry
	for _, e := range result {
		if !e.ReplayDate.After(now) {
			filtered = append(filtered, e)
		}
	}

	return filtered
}
