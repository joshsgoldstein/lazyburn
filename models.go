package main

import "time"

// TokenUsage holds token counts and cost for one or more messages.
// Think of it like the Python TokenUsage dataclass.
type TokenUsage struct {
	Input        int
	CacheWrite5m int
	CacheWrite1h int
	CacheRead    int
	Output       int
	Cost         float64
}

// CacheWrite is the total of both cache write tiers.
func (u TokenUsage) CacheWrite() int {
	return u.CacheWrite5m + u.CacheWrite1h
}

// Add accumulates another TokenUsage into this one (like Python's __iadd__).
func (u *TokenUsage) Add(other TokenUsage) {
	u.Input += other.Input
	u.CacheWrite5m += other.CacheWrite5m
	u.CacheWrite1h += other.CacheWrite1h
	u.CacheRead += other.CacheRead
	u.Output += other.Output
	u.Cost += other.Cost
}

// Session represents one Claude Code session (.jsonl file).
type Session struct {
	ID          string
	ProjectPath string
	Slug        string
	StartTime   time.Time
	EndTime     time.Time
	Usage       TokenUsage
	Models      map[string]bool
	TurnCount   int
	LastPrompt  string
}

// DurationMinutes returns wall-clock duration of the session in minutes.
func (s Session) DurationMinutes() float64 {
	if s.StartTime.IsZero() || s.EndTime.IsZero() {
		return 0
	}
	return s.EndTime.Sub(s.StartTime).Minutes()
}
