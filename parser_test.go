package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── helpers ────────────────────────────────────────────────────────────────────

func writeJSONL(t *testing.T, lines []map[string]any) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range lines {
		b, _ := json.Marshal(line)
		f.Write(b)
		f.WriteString("\n")
	}
	f.Close()
	return f.Name()
}

func assistantMsg(msgID, reqID string, input, output int) map[string]any {
	return map[string]any{
		"type":      "assistant",
		"timestamp": "2026-04-01T10:00:00Z",
		"cwd":       "/home/user/project",
		"requestId": reqID,
		"sessionId": "sess-1",
		"message": map[string]any{
			"id":    msgID,
			"model": "claude-sonnet-4-6",
			"usage": map[string]any{
				"input_tokens":            input,
				"output_tokens":           output,
				"cache_read_input_tokens": 0,
				"cache_creation":          map[string]any{},
			},
		},
	}
}

func makeSession(projectPath string, cost float64) Session {
	s := Session{
		ID:          "test",
		ProjectPath: projectPath,
		Models:      make(map[string]bool),
	}
	s.Usage.Cost = cost
	return s
}

// ── deduplication ──────────────────────────────────────────────────────────────

func TestParseKeepsLastDuplicate(t *testing.T) {
	// streaming replays the same requestId — we must keep the last (highest) token counts
	path := writeJSONL(t, []map[string]any{
		assistantMsg("msg-1", "req-1", 100, 50),
		assistantMsg("msg-1", "req-1", 200, 150), // last = authoritative
	})
	s, err := parseJSONL(path, time.Time{}, time.Time{})
	if err != nil || s == nil {
		t.Fatalf("expected a session, got err=%v s=%v", err, s)
	}
	if s.Usage.Input != 200 {
		t.Errorf("Input: got %d want 200", s.Usage.Input)
	}
	if s.Usage.Output != 150 {
		t.Errorf("Output: got %d want 150", s.Usage.Output)
	}
}

func TestParseCountsDistinctRequests(t *testing.T) {
	path := writeJSONL(t, []map[string]any{
		assistantMsg("msg-1", "req-1", 100, 50),
		assistantMsg("msg-2", "req-2", 200, 100),
	})
	s, err := parseJSONL(path, time.Time{}, time.Time{})
	if err != nil || s == nil {
		t.Fatalf("expected a session, got err=%v s=%v", err, s)
	}
	if s.Usage.Input != 300 {
		t.Errorf("Input: got %d want 300", s.Usage.Input)
	}
	if s.Usage.Output != 150 {
		t.Errorf("Output: got %d want 150", s.Usage.Output)
	}
}

func TestParseSkipsSyntheticModel(t *testing.T) {
	msg := assistantMsg("msg-1", "req-1", 100, 50)
	msg["message"].(map[string]any)["model"] = "<synthetic>"
	path := writeJSONL(t, []map[string]any{msg})
	s, _ := parseJSONL(path, time.Time{}, time.Time{})
	if s != nil {
		t.Error("expected nil session for synthetic-only file")
	}
}

func TestParseEmptyFile(t *testing.T) {
	path := writeJSONL(t, nil)
	s, _ := parseJSONL(path, time.Time{}, time.Time{})
	if s != nil {
		t.Error("expected nil for empty file")
	}
}

func TestParseReadsCwd(t *testing.T) {
	path := writeJSONL(t, []map[string]any{assistantMsg("msg-1", "req-1", 10, 5)})
	s, _ := parseJSONL(path, time.Time{}, time.Time{})
	if s == nil {
		t.Fatal("expected a session")
	}
	if s.ProjectPath != "/home/user/project" {
		t.Errorf("ProjectPath: got %q want /home/user/project", s.ProjectPath)
	}
}

func TestParseSinceFilter(t *testing.T) {
	// message is before the since cutoff — should be excluded
	path := writeJSONL(t, []map[string]any{assistantMsg("msg-1", "req-1", 100, 50)})
	since := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	s, _ := parseJSONL(path, since, time.Time{})
	if s != nil {
		t.Error("expected nil when all messages are before --since")
	}
}

func TestParseUntilFilter(t *testing.T) {
	// message is after the until cutoff — should be excluded
	path := writeJSONL(t, []map[string]any{assistantMsg("msg-1", "req-1", 100, 50)})
	until := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s, _ := parseJSONL(path, time.Time{}, until)
	if s != nil {
		t.Error("expected nil when all messages are after --until")
	}
}

func TestParseTurnCount(t *testing.T) {
	path := writeJSONL(t, []map[string]any{
		assistantMsg("msg-1", "req-1", 100, 50),
		{"type": "system", "subtype": "turn_duration", "slug": "brave-ancient-reef"},
		{"type": "system", "subtype": "turn_duration", "slug": "sleepy-golden-tide"},
	})
	s, _ := parseJSONL(path, time.Time{}, time.Time{})
	if s == nil {
		t.Fatal("expected a session")
	}
	if s.TurnCount != 2 {
		t.Errorf("TurnCount: got %d want 2", s.TurnCount)
	}
	if s.Slug != "sleepy-golden-tide" {
		t.Errorf("Slug: got %q want sleepy-golden-tide", s.Slug)
	}
}

func TestParseLastPrompt(t *testing.T) {
	path := writeJSONL(t, []map[string]any{
		assistantMsg("msg-1", "req-1", 10, 5),
		{"type": "last-prompt", "lastPrompt": "implement the auth flow"},
	})
	s, _ := parseJSONL(path, time.Time{}, time.Time{})
	if s == nil {
		t.Fatal("expected a session")
	}
	if s.LastPrompt != "implement the auth flow" {
		t.Errorf("LastPrompt: got %q", s.LastPrompt)
	}
}

// ── path matching ──────────────────────────────────────────────────────────────

func TestPathMatchesAbsolutePrefix(t *testing.T) {
	if !pathMatches("/home/user/acme/project", "/home/user/acme") {
		t.Error("expected match for path under filter")
	}
}

func TestPathMatchesAbsoluteExact(t *testing.T) {
	if !pathMatches("/home/user/acme", "/home/user/acme") {
		t.Error("expected match for exact path")
	}
}

func TestPathMatchesNoFalsePositiveOnSimilarPrefix(t *testing.T) {
	// /acme must NOT match /acme-backend
	if pathMatches("/home/user/acme-backend/proj", "/home/user/acme") {
		t.Error("acme-backend should not match acme filter")
	}
}

func TestPathMatchesSubstring(t *testing.T) {
	if !pathMatches("/home/user/acme/project", "acme") {
		t.Error("expected substring match")
	}
}

func TestPathMatchesEmptyProjectPath(t *testing.T) {
	if pathMatches("", "acme") {
		t.Error("empty project path should not match")
	}
}

// ── grouping ───────────────────────────────────────────────────────────────────

func TestGroupByDepth(t *testing.T) {
	home := "/home/user"
	sessions := []Session{
		makeSession("/home/user/acme/alpha", 1),
		makeSession("/home/user/acme/beta", 1),
		makeSession("/home/user/globex/main", 1),
	}
	groups := GroupByDepth(sessions, 1, home)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d: %v", len(groups), groups)
	}
	if len(groups["acme"]) != 2 {
		t.Errorf("acme group: got %d sessions want 2", len(groups["acme"]))
	}
	if len(groups["globex"]) != 1 {
		t.Errorf("globex group: got %d sessions want 1", len(groups["globex"]))
	}
}

func TestAggregate(t *testing.T) {
	sessions := []Session{makeSession("/p", 1.5), makeSession("/p", 2.5)}
	total := Aggregate(sessions)
	if total.Cost != 4.0 {
		t.Errorf("Aggregate cost: got %f want 4.0", total.Cost)
	}
}

// ── filter depth ───────────────────────────────────────────────────────────────

func TestFilterDepthAbsolutePath(t *testing.T) {
	home := "/home/user"
	sessions := []Session{makeSession("/home/user/acme/project", 1)}
	depth := FilterDepth(sessions, "/home/user/acme", home)
	if depth != 1 {
		t.Errorf("FilterDepth: got %d want 1", depth)
	}
}

func TestFilterDepthSubstring(t *testing.T) {
	home := "/home/user"
	sessions := []Session{makeSession("/home/user/acme/project", 1)}
	depth := FilterDepth(sessions, "acme", home)
	if depth != 1 {
		t.Errorf("FilterDepth substring: got %d want 1", depth)
	}
}

// ── ParseAllSessions integration ───────────────────────────────────────────────

func TestParseAllSessionsPathFilter(t *testing.T) {
	// build a fake ~/.claude/projects/ structure
	dir := t.TempDir()
	proj1 := filepath.Join(dir, "projects", "-home-user-acme-alpha")
	proj2 := filepath.Join(dir, "projects", "-home-user-globex-main")
	os.MkdirAll(proj1, 0755)
	os.MkdirAll(proj2, 0755)

	writeSessionFile := func(projDir, cwd string) {
		msg := assistantMsg("msg-1", "req-1", 100, 50)
		msg["cwd"] = cwd
		b, _ := json.Marshal(msg)
		os.WriteFile(filepath.Join(projDir, "session.jsonl"), append(b, '\n'), 0644)
	}
	writeSessionFile(proj1, "/home/user/acme/alpha")
	writeSessionFile(proj2, "/home/user/globex/main")

	sessions, err := ParseAllSessions(dir, time.Time{}, time.Time{}, "acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session after acme filter, got %d", len(sessions))
	}
	if sessions[0].ProjectPath != "/home/user/acme/alpha" {
		t.Errorf("wrong project path: %s", sessions[0].ProjectPath)
	}
}
