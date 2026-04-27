package main

import (
	"testing"
	"time"
)

func TestTokenUsageAdd(t *testing.T) {
	a := TokenUsage{Input: 100, Output: 50, Cost: 1.0}
	b := TokenUsage{Input: 200, Output: 100, Cost: 2.0}
	a.Add(b)
	if a.Input != 300 {
		t.Errorf("Input: got %d want 300", a.Input)
	}
	if a.Output != 150 {
		t.Errorf("Output: got %d want 150", a.Output)
	}
	if a.Cost != 3.0 {
		t.Errorf("Cost: got %f want 3.0", a.Cost)
	}
}

func TestTokenUsageCacheWrite(t *testing.T) {
	u := TokenUsage{CacheWrite5m: 100, CacheWrite1h: 200}
	if u.CacheWrite() != 300 {
		t.Errorf("CacheWrite: got %d want 300", u.CacheWrite())
	}
}

func TestSessionDurationMinutes(t *testing.T) {
	s := Session{
		StartTime: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 4, 1, 11, 30, 0, 0, time.UTC),
	}
	if s.DurationMinutes() != 90.0 {
		t.Errorf("DurationMinutes: got %f want 90.0", s.DurationMinutes())
	}
}

func TestSessionDurationMinutesZeroTimes(t *testing.T) {
	s := Session{}
	if s.DurationMinutes() != 0.0 {
		t.Errorf("expected 0 for empty times, got %f", s.DurationMinutes())
	}
}
