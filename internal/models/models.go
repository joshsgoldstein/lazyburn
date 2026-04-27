package models

import "time"

type TokenUsage struct {
	Input        int
	CacheWrite5m int
	CacheWrite1h int
	CacheRead    int
	Output       int
	Cost         float64
}

func (u *TokenUsage) Add(other TokenUsage) {
	u.Input += other.Input
	u.CacheWrite5m += other.CacheWrite5m
	u.CacheWrite1h += other.CacheWrite1h
	u.CacheRead += other.CacheRead
	u.Output += other.Output
	u.Cost += other.Cost
}

func (u TokenUsage) CacheWrite() int {
	return u.CacheWrite5m + u.CacheWrite1h
}

type Session struct {
	ID          string
	ProjectPath string
	Slug        string
	LastPrompt  string
	StartTime   time.Time
	EndTime     time.Time
	Usage       TokenUsage
	Models      map[string]bool
	TurnCount   int
}

func (s Session) DurationMinutes() float64 {
	if s.StartTime.IsZero() || s.EndTime.IsZero() {
		return 0
	}
	return s.EndTime.Sub(s.StartTime).Minutes()
}
