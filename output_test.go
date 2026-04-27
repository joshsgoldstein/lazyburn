package main

import (
	"testing"
	"time"
)

func TestFmtTokens(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0k"},
		{1500, "1.5k"},
		{999999, "1000.0k"},
		{1_000_000, "1.0M"},
		{2_500_000, "2.5M"},
	}
	for _, c := range cases {
		got := fmtTokens(c.in)
		if got != c.want {
			t.Errorf("fmtTokens(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFmtCost(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0.0, "$0.00"},
		{1.5, "$1.50"},
		{125.749, "$125.75"},
		{0.004, "$0.00"},
		{0.005, "$0.01"},
	}
	for _, c := range cases {
		got := fmtCost(c.in)
		if got != c.want {
			t.Errorf("fmtCost(%f) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFmtDuration(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "-"},
		{-1, "-"},
		{30, "30m"},
		{59, "59m"},
		{60, "1.0h"},
		{90, "1.5h"},
		{148.4, "2.5h"},
	}
	for _, c := range cases {
		got := fmtDuration(c.in)
		if got != c.want {
			t.Errorf("fmtDuration(%f) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDateRangeEmpty(t *testing.T) {
	if dateRange(nil) != "" {
		t.Error("expected empty string for no sessions")
	}
	if dateRange([]Session{{}}) != "" {
		t.Error("expected empty string for session with zero time")
	}
}

func TestDateRangeSingleDay(t *testing.T) {
	s := Session{StartTime: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)}
	got := dateRange([]Session{s})
	if got != "2026-04-01" {
		t.Errorf("single day: got %q want 2026-04-01", got)
	}
}

func TestDateRangeSpan(t *testing.T) {
	sessions := []Session{
		{StartTime: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		{StartTime: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)},
		{StartTime: time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)},
	}
	got := dateRange(sessions)
	if got != "2026-04-01 – 2026-04-30" {
		t.Errorf("span: got %q want 2026-04-01 – 2026-04-30", got)
	}
}

func TestCommonPrefixNone(t *testing.T) {
	if commonPrefix(nil) != "" {
		t.Error("nil input should return empty")
	}
	if commonPrefix([]string{"a"}) != "" {
		t.Error("single item should return empty")
	}
}

func TestCommonPrefixShared(t *testing.T) {
	got := commonPrefix([]string{"Documents/acme/alpha", "Documents/acme/beta"})
	if got != "Documents/acme/" {
		t.Errorf("got %q want Documents/acme/", got)
	}
}

func TestCommonPrefixNoShared(t *testing.T) {
	got := commonPrefix([]string{"acme/alpha", "globex/main"})
	if got != "" {
		t.Errorf("no common prefix: got %q want empty", got)
	}
}

func TestGroupLabel(t *testing.T) {
	cases := []struct {
		folder, prefix, want string
	}{
		{"acme/alpha", "acme/", "alpha"},
		{"acme/beta", "acme/", "beta"},
		{"acme", "acme/", "(this folder)"},  // sessions directly in the filtered folder
		{"acme", "", "acme"},                 // no prefix — use full name
		{"acme/alpha", "", "acme/alpha"},     // no prefix — use full name
	}
	for _, c := range cases {
		got := groupLabel(c.folder, c.prefix)
		if got != c.want {
			t.Errorf("groupLabel(%q, %q) = %q, want %q", c.folder, c.prefix, got, c.want)
		}
	}
}
