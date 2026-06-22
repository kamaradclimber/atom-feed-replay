package feed

import (
	"testing"
	"time"
)

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestEmptyEntries(t *testing.T) {
	start := mustTime("2026-06-01T00:00:00Z")
	now := mustTime("2026-06-15T00:00:00Z")
	result := ReplaySchedule(nil, start, 720*time.Hour, time.Hour, now)
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestSingleBacklogEntry(t *testing.T) {
	start := mustTime("2026-06-01T00:00:00Z")
	now := mustTime("2026-06-15T00:00:00Z")
	entries := []Entry{
		{ID: "1", Published: mustTime("2025-01-01T00:00:00Z"), Title: "old video"},
	}
	result := ReplaySchedule(entries, start, 720*time.Hour, time.Hour, now)

	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	if !result[0].ReplayDate.Equal(start) {
		t.Fatalf("expected replay date %v, got %v", start, result[0].ReplayDate)
	}
}

func TestSingleBacklogEntryBeforeNow(t *testing.T) {
	start := mustTime("2026-07-01T00:00:00Z") // future
	now := mustTime("2026-06-15T00:00:00Z")
	entries := []Entry{
		{ID: "1", Published: mustTime("2025-01-01T00:00:00Z"), Title: "old video"},
	}
	result := ReplaySchedule(entries, start, 720*time.Hour, time.Hour, now)

	if len(result) != 0 {
		t.Fatalf("expected 0 entries (start in future), got %d", len(result))
	}
}

func TestMultipleEntriesEvenlySpaced(t *testing.T) {
	start := mustTime("2026-06-01T00:00:00Z")
	now := mustTime("2026-06-30T00:00:00Z")
	window := 10 * 24 * time.Hour // 10 days
	minInterval := time.Hour

	entries := []Entry{
		{ID: "1", Published: mustTime("2025-01-01T00:00:00Z")},
		{ID: "2", Published: mustTime("2025-02-01T00:00:00Z")},
		{ID: "3", Published: mustTime("2025-03-01T00:00:00Z")},
		{ID: "4", Published: mustTime("2025-04-01T00:00:00Z")},
		{ID: "5", Published: mustTime("2025-05-01T00:00:00Z")},
	}

	result := ReplaySchedule(entries, start, window, minInterval, now)

	if len(result) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(result))
	}

	// All should be released since now is after the window end
	// Window end = start + max(10d, 5*1h) = June 1 + 10 days = June 11
	// now = June 30, so all should be visible
	if len(result) != 5 {
		t.Fatalf("expected all 5 entries visible, got %d", len(result))
	}

	// Check even spacing: each pair should be ~48h apart (10 days / 4 intervals)
	spacing := result[1].ReplayDate.Sub(result[0].ReplayDate)
	expected := 10 * 24 * time.Hour / 4 // 60h
	tolerance := time.Second
	if diff := spacing - expected; diff < -tolerance || diff > tolerance {
		t.Fatalf("expected spacing ~%v, got %v", expected, spacing)
	}
}

func TestMinIntervalDominates(t *testing.T) {
	start := mustTime("2026-06-01T00:00:00Z")
	now := mustTime("2026-07-15T00:00:00Z")
	window := 24 * time.Hour       // 1 day ideal window
	minInterval := 12 * time.Hour  // 12 hour min gap

	// 5 entries: 5 * 12h = 60h needed, but window is only 24h
	// So catchup_end = start + max(24h, 60h) = start + 60h
	entries := []Entry{
		{ID: "1", Published: mustTime("2025-01-01T00:00:00Z")},
		{ID: "2", Published: mustTime("2025-02-01T00:00:00Z")},
		{ID: "3", Published: mustTime("2025-03-01T00:00:00Z")},
		{ID: "4", Published: mustTime("2025-04-01T00:00:00Z")},
		{ID: "5", Published: mustTime("2025-05-01T00:00:00Z")},
	}

	result := ReplaySchedule(entries, start, window, minInterval, now)

	if len(result) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(result))
	}

	// With 5 entries over 60h, spacing should be 60h/4 = 15h each
	spacing := result[1].ReplayDate.Sub(result[0].ReplayDate)
	expected := 60 * time.Hour / 4
	tolerance := time.Second
	if diff := spacing - expected; diff < -tolerance || diff > tolerance {
		t.Fatalf("expected spacing ~%v, got %v", expected, spacing)
	}

	// Last entry should be at start + 60h
	expectedEnd := start.Add(60 * time.Hour)
	if !result[4].ReplayDate.Equal(expectedEnd) {
		t.Fatalf("expected last entry at %v, got %v", expectedEnd, result[4].ReplayDate)
	}
}

func TestLiveEntriesPassThrough(t *testing.T) {
	start := mustTime("2026-06-01T00:00:00Z")
	now := mustTime("2026-06-15T00:00:00Z")
	window := 10 * 24 * time.Hour
	minInterval := time.Hour

	entries := []Entry{
		{ID: "1", Published: mustTime("2025-01-01T00:00:00Z")},     // backlog
		{ID: "2", Published: mustTime("2026-06-10T00:00:00Z")},     // live (after catchup_start)
	}

	result := ReplaySchedule(entries, start, window, minInterval, now)

	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}

	// Entry 1 is backlog, should have replay date = start
	if !result[0].ReplayDate.Equal(start) {
		t.Fatalf("backlog entry expected at %v, got %v", start, result[0].ReplayDate)
	}

	// Entry 2 is live, should keep original date
	expectedLive := mustTime("2026-06-10T00:00:00Z")
	if !result[1].ReplayDate.Equal(expectedLive) {
		t.Fatalf("live entry expected at %v, got %v", expectedLive, result[1].ReplayDate)
	}
}

func TestLiveEntryHiddenBeforeDate(t *testing.T) {
	start := mustTime("2026-06-01T00:00:00Z")
	now := mustTime("2026-06-05T00:00:00Z") // before the live entry
	window := 10 * 24 * time.Hour
	minInterval := time.Hour

	entries := []Entry{
		{ID: "1", Published: mustTime("2025-01-01T00:00:00Z")},
		{ID: "2", Published: mustTime("2026-06-10T00:00:00Z")}, // in the future relative to now
	}

	result := ReplaySchedule(entries, start, window, minInterval, now)

	if len(result) != 1 {
		t.Fatalf("expected 1 entry (live is future), got %d", len(result))
	}
	if result[0].ID != "1" {
		t.Fatalf("expected only backlog entry, got %s", result[0].ID)
	}
}

func TestAllEntriesLive(t *testing.T) {
	start := mustTime("2026-01-01T00:00:00Z")
	now := mustTime("2026-06-15T00:00:00Z")
	window := 10 * 24 * time.Hour
	minInterval := time.Hour

	entries := []Entry{
		{ID: "1", Published: mustTime("2026-06-10T00:00:00Z")},
		{ID: "2", Published: mustTime("2026-06-12T00:00:00Z")},
	}

	result := ReplaySchedule(entries, start, window, minInterval, now)

	if len(result) != 2 {
		t.Fatalf("expected 2 live entries, got %d", len(result))
	}
	if !result[0].ReplayDate.Equal(mustTime("2026-06-10T00:00:00Z")) {
		t.Fatalf("expected June 10, got %v", result[0].ReplayDate)
	}
	if !result[1].ReplayDate.Equal(mustTime("2026-06-12T00:00:00Z")) {
		t.Fatalf("expected June 12, got %v", result[1].ReplayDate)
	}
}

func TestEntriesSortedByReplayDate(t *testing.T) {
	start := mustTime("2026-06-01T00:00:00Z")
	now := mustTime("2026-06-20T00:00:00Z")
	window := 10 * 24 * time.Hour
	minInterval := time.Hour

	// Add entry in reverse chronological order of original date to test sorting
	entries := []Entry{
		{ID: "3", Published: mustTime("2025-03-01T00:00:00Z")},
		{ID: "1", Published: mustTime("2025-01-01T00:00:00Z")},
		{ID: "2", Published: mustTime("2025-02-01T00:00:00Z")},
	}

	result := ReplaySchedule(entries, start, window, minInterval, now)

	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}

	// Should be sorted by replay_date ascending
	for i := 1; i < len(result); i++ {
		if result[i].ReplayDate.Before(result[i-1].ReplayDate) {
			t.Fatalf("entries not sorted by replay_date: %v before %v", result[i].ReplayDate, result[i-1].ReplayDate)
		}
	}
}

func TestPartialCatchup(t *testing.T) {
	start := mustTime("2026-06-01T00:00:00Z")
	window := 10 * 24 * time.Hour // 10 days, catchup_end = June 11
	minInterval := time.Hour

	entries := []Entry{
		{ID: "1", Published: mustTime("2025-01-01T00:00:00Z")},
		{ID: "2", Published: mustTime("2025-02-01T00:00:00Z")},
		{ID: "3", Published: mustTime("2025-03-01T00:00:00Z")},
		{ID: "4", Published: mustTime("2025-04-01T00:00:00Z")},
		{ID: "5", Published: mustTime("2025-05-01T00:00:00Z")},
	}

	// now = June 5, which is 4 days into the 10-day catchup
	// With 5 entries over 10 days, spacing = 10d/4 = 2.5d per entry
	// i=0: June 1
	// i=1: June 3.5 (June 1 + 2.5d)
	// i=2: June 6 (June 1 + 5d) → not yet visible (now = June 5)
	// So only 2 entries should be visible
	now := mustTime("2026-06-05T00:00:00Z")

	result := ReplaySchedule(entries, start, window, minInterval, now)

	if len(result) != 2 {
		t.Fatalf("expected 2 entries visible by June 5, got %d", len(result))
	}
	if result[0].ID != "1" || result[1].ID != "2" {
		t.Fatalf("expected entries 1 and 2, got %s, %s", result[0].ID, result[1].ID)
	}
}

func TestAllBacklogReleasedAfterEnd(t *testing.T) {
	start := mustTime("2026-06-01T00:00:00Z")
	window := 10 * 24 * time.Hour // catchup_end = June 11
	minInterval := time.Hour
	now := mustTime("2026-06-30T00:00:00Z") // well after end

	entries := []Entry{
		{ID: "1", Published: mustTime("2025-01-01T00:00:00Z")},
		{ID: "2", Published: mustTime("2025-02-01T00:00:00Z")},
	}

	result := ReplaySchedule(entries, start, window, minInterval, now)

	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
}

func TestNewEntriesShiftSchedule(t *testing.T) {
	start := mustTime("2026-06-01T00:00:00Z")
	window := 10 * 24 * time.Hour
	minInterval := time.Hour
	now := mustTime("2026-06-20T00:00:00Z")

	// First, 3 entries
	entries1 := []Entry{
		{ID: "1", Published: mustTime("2025-01-01T00:00:00Z")},
		{ID: "2", Published: mustTime("2025-02-01T00:00:00Z")},
		{ID: "3", Published: mustTime("2025-03-01T00:00:00Z")},
	}
	result1 := ReplaySchedule(entries1, start, window, minInterval, now)

	// With 3 entries over 10 days: spacing = 10d/2 = 5d
	// Entry 1 at June 1, Entry 2 at June 6, Entry 3 at June 11

	// Now add a 4th entry
	entries2 := []Entry{
		{ID: "1", Published: mustTime("2025-01-01T00:00:00Z")},
		{ID: "2", Published: mustTime("2025-02-01T00:00:00Z")},
		{ID: "3", Published: mustTime("2025-03-01T00:00:00Z")},
		{ID: "4", Published: mustTime("2025-04-01T00:00:00Z")},
	}
	result2 := ReplaySchedule(entries2, start, window, minInterval, now)

	// With 4 entries over 10 days: spacing = 10d/3 ≈ 3.33d
	// Entry 1 at June 1, Entry 2 at ~June 4.33, Entry 3 at ~June 7.67, Entry 4 at June 11
	// Entry 2 shifted from June 6 to June 4.33. This is the "shift everything" behavior.

	if len(result2) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(result2))
	}

	// Verify the dates changed (shifted) between the two runs
	if result1[1].ReplayDate.Equal(result2[1].ReplayDate) {
		t.Fatal("expected entry 2 date to shift when new entry added, but it stayed the same")
	}
}

func TestBacklogAndLiveMixed(t *testing.T) {
	start := mustTime("2026-06-01T00:00:00Z")
	window := 10 * 24 * time.Hour
	minInterval := time.Hour
	now := mustTime("2026-06-20T00:00:00Z")

	entries := []Entry{
		{ID: "old1", Published: mustTime("2025-01-01T00:00:00Z")},
		{ID: "live1", Published: mustTime("2026-06-15T00:00:00Z")},
		{ID: "old2", Published: mustTime("2025-06-01T00:00:00Z")},
		{ID: "live2", Published: mustTime("2026-06-18T00:00:00Z")},
	}

	result := ReplaySchedule(entries, start, window, minInterval, now)

	if len(result) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(result))
	}

	// Results sorted by replay_date ascending
	// old1: June 1 (first backlog)
	// old2: June 11 (last backlog, start + 10 days)
	// live1: June 15
	// live2: June 18
	if result[0].ID != "old1" {
		t.Fatalf("expected old1 first, got %s", result[0].ID)
	}
	if result[2].ID != "live1" {
		t.Fatalf("expected live1 third, got %s", result[2].ID)
	}
	if result[3].ID != "live2" {
		t.Fatalf("expected live2 last, got %s", result[3].ID)
	}
}

func TestEntryAtCatchupStartBoundary(t *testing.T) {
	start := mustTime("2026-06-01T00:00:00Z")
	window := 10 * 24 * time.Hour
	minInterval := time.Hour
	now := mustTime("2026-06-20T00:00:00Z")

	// Entry published exactly at catchup_start should be treated as backlog
	entries := []Entry{
		{ID: "at-boundary", Published: mustTime("2026-06-01T00:00:00Z")},
		{ID: "after", Published: mustTime("2026-06-02T00:00:00Z")},
	}

	result := ReplaySchedule(entries, start, window, minInterval, now)

	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}

	// The boundary entry should be treated as backlog and placed at start
	if !result[0].ReplayDate.Equal(start) {
		t.Fatalf("expected entry at boundary to have replay_date = catchup_start, got %v", result[0].ReplayDate)
	}
}

func TestLargeMinIntervalStretchesWindow(t *testing.T) {
	start := mustTime("2026-06-01T00:00:00Z")
	window := 24 * time.Hour
	minInterval := 10 * 24 * time.Hour // 10 days per entry
	now := mustTime("2026-08-01T00:00:00Z")

	// 3 entries, 3 * 10d = 30d needed > 1d window
	// catchup_end = start + 30d = July 1
	entries := []Entry{
		{ID: "1", Published: mustTime("2025-01-01T00:00:00Z")},
		{ID: "2", Published: mustTime("2025-02-01T00:00:00Z")},
		{ID: "3", Published: mustTime("2025-03-01T00:00:00Z")},
	}

	result := ReplaySchedule(entries, start, window, minInterval, now)

	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}

	// Spacing: 30d / 2 = 15d between entries
	spacing := result[1].ReplayDate.Sub(result[0].ReplayDate)
	expected := 30 * 24 * time.Hour / 2
	tolerance := time.Second
	if diff := spacing - expected; diff < -tolerance || diff > tolerance {
		t.Fatalf("expected spacing ~%v, got %v", expected, spacing)
	}

	if !result[2].ReplayDate.Equal(start.Add(30 * 24 * time.Hour)) {
		t.Fatalf("expected last entry at %v, got %v", start.Add(30*24*time.Hour), result[2].ReplayDate)
	}
}

func TestEntriesWithoutPublishedDates(t *testing.T) {
	start := mustTime("2026-06-01T00:00:00Z")
	window := 10 * 24 * time.Hour
	minInterval := time.Hour
	now := mustTime("2026-06-20T00:00:00Z")

	zeroTime := time.Time{}

	entries := []Entry{
		{ID: "1", Published: zeroTime}, // zero time - treated as not after catchup_start, so backlog
		{ID: "2", Published: mustTime("2026-06-15T00:00:00Z")}, // live
	}

	result := ReplaySchedule(entries, start, window, minInterval, now)

	// Entry 1: zero time is not after catchup_start, so it's backlog
	// Entry 2: live, June 15
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if !result[0].ReplayDate.Equal(start) {
		t.Fatalf("expected zero-time entry at start, got %v", result[0].ReplayDate)
	}
}
