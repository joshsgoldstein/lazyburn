package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// These structs mirror the shape of each JSON line in a .jsonl file.
// The `json:"fieldName"` tags tell Go which JSON key maps to which field.

type jsonMessage struct {
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	Cwd       string          `json:"cwd"`
	RequestID string          `json:"requestId"`
	SessionID string          `json:"sessionId"`
	Subtype   string          `json:"subtype"`
	Slug      string          `json:"slug"`
	Message   jsonInnerMsg    `json:"message"`
	LastPrompt string         `json:"lastPrompt"`
}

type jsonInnerMsg struct {
	ID    string   `json:"id"`
	Model string   `json:"model"`
	Usage jsonUsage `json:"usage"`
}

type jsonUsage struct {
	InputTokens          int              `json:"input_tokens"`
	OutputTokens         int              `json:"output_tokens"`
	CacheReadInputTokens int              `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int          `json:"cache_creation_input_tokens"`
	CacheCreation        jsonCacheCreation `json:"cache_creation"`
}

type jsonCacheCreation struct {
	Ephemeral5m int `json:"ephemeral_5m_input_tokens"`
	Ephemeral1h int `json:"ephemeral_1h_input_tokens"`
}

// dedupEntry holds the last-seen usage for a given dedup key.
type dedupEntry struct {
	usage TokenUsage
	model string
}

// ParseAllSessions reads every .jsonl file under claudeDir/projects/.
func ParseAllSessions(claudeDir string, since, until time.Time, pathFilter string) ([]Session, error) {
	projectsDir := filepath.Join(claudeDir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}

	var sessions []Session
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projDir := filepath.Join(projectsDir, entry.Name())
		jsonlFiles, err := filepath.Glob(filepath.Join(projDir, "*.jsonl"))
		if err != nil {
			continue
		}
		for _, f := range jsonlFiles {
			s, err := parseJSONL(f, since, until)
			if err != nil || s == nil {
				continue
			}
			if pathFilter != "" && !pathMatches(s.ProjectPath, pathFilter) {
				continue
			}
			sessions = append(sessions, *s)
		}
	}
	return sessions, nil
}

// parseJSONL reads one session file and returns a Session, or nil if it has no cost data.
func parseJSONL(path string, since, until time.Time) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	id := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	session := &Session{
		ID:     id,
		Models: make(map[string]bool),
	}
	deduped := make(map[string]dedupEntry)

	scanner := bufio.NewScanner(f)
	// Claude sessions can have very long lines; bump the buffer.
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var msg jsonMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		ts := parseTimestamp(msg.Timestamp)
		if !ts.IsZero() {
			if session.StartTime.IsZero() || ts.Before(session.StartTime) {
				session.StartTime = ts
			}
			if session.EndTime.IsZero() || ts.After(session.EndTime) {
				session.EndTime = ts
			}
		}

		if msg.Cwd != "" && session.ProjectPath == "" {
			session.ProjectPath = msg.Cwd
		}

		switch msg.Type {
		case "assistant":
			if msg.Message.Model == "<synthetic>" || msg.Message.Model == "" {
				continue
			}
			if msg.Message.Usage.InputTokens == 0 && msg.Message.Usage.OutputTokens == 0 {
				continue
			}

			if !ts.IsZero() {
				if !since.IsZero() && ts.Before(since) {
					continue
				}
				if !until.IsZero() && ts.After(until) {
					continue
				}
			}

			cc := msg.Message.Usage.CacheCreation
			c5m, c1h := cc.Ephemeral5m, cc.Ephemeral1h
			if c5m == 0 && c1h == 0 {
				c5m = msg.Message.Usage.CacheCreationInputTokens
			}

			p := getPricing(msg.Message.Model)
			u := msg.Message.Usage
			usage := TokenUsage{
				Input:        u.InputTokens,
				CacheWrite5m: c5m,
				CacheWrite1h: c1h,
				CacheRead:    u.CacheReadInputTokens,
				Output:       u.OutputTokens,
			}
			usage.Cost = (float64(usage.Input)*p.Input +
				float64(usage.CacheWrite5m)*p.Cache5m +
				float64(usage.CacheWrite1h)*p.Cache1h +
				float64(usage.CacheRead)*p.CacheRead +
				float64(usage.Output)*p.Output) / 1_000_000

			// Build dedup key — keep last occurrence (streaming replays each request).
			key := dedupKey(msg.Message.ID, msg.RequestID, msg.SessionID, len(deduped))
			deduped[key] = dedupEntry{usage: usage, model: msg.Message.Model}

		case "system":
			if msg.Subtype == "turn_duration" {
				session.TurnCount++
				if msg.Slug != "" {
					session.Slug = msg.Slug
				}
			}

		case "last-prompt":
			session.LastPrompt = msg.LastPrompt
		}
	}

	if len(deduped) == 0 {
		return nil, nil
	}
	for _, entry := range deduped {
		session.Usage.Add(entry.usage)
		session.Models[entry.model] = true
	}
	return session, nil
}

func dedupKey(msgID, reqID, sessID string, count int) string {
	if msgID != "" && reqID != "" {
		return "req:" + msgID + ":" + reqID
	}
	if msgID != "" && sessID != "" {
		return "session:" + msgID + ":" + sessID
	}
	return fmt.Sprintf("anon:%d", count)
}

func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	s = strings.Replace(s, "Z", "+00:00", 1)
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// pathMatches checks whether projectPath is under pathFilter (absolute) or contains it (substring).
func pathMatches(projectPath, pathFilter string) bool {
	if projectPath == "" {
		return false
	}
	if filepath.IsAbs(pathFilter) {
		return projectPath == pathFilter || strings.HasPrefix(projectPath, pathFilter+"/")
	}
	return strings.Contains(projectPath, pathFilter)
}

// GroupByDepth buckets sessions by the first `depth` components of their path relative to home.
func GroupByDepth(sessions []Session, depth int, home string) map[string][]Session {
	groups := make(map[string][]Session)
	for _, s := range sessions {
		key := pathKey(s.ProjectPath, depth, home)
		groups[key] = append(groups[key], s)
	}
	return groups
}

func pathKey(projectPath string, depth int, home string) string {
	if projectPath == "" {
		return "(unknown)"
	}
	rel, err := filepath.Rel(home, projectPath)
	if err != nil {
		return projectPath
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) >= depth {
		return strings.Join(parts[:depth], "/")
	}
	return rel
}

// Aggregate sums token usage across a slice of sessions.
func Aggregate(sessions []Session) TokenUsage {
	var total TokenUsage
	for _, s := range sessions {
		total.Add(s.Usage)
	}
	return total
}

// FilterDepth returns the depth of the filter path relative to home.
func FilterDepth(sessions []Session, activeFilter, home string) int {
	filterPath := filepath.Clean(activeFilter)
	if filepath.IsAbs(filterPath) {
		rel, err := filepath.Rel(home, filterPath)
		if err == nil {
			parts := strings.Split(filepath.ToSlash(rel), "/")
			return len(parts)
		}
	}
	filterClean := strings.Trim(activeFilter, "/")
	for _, s := range sessions {
		if s.ProjectPath == "" {
			continue
		}
		rel, err := filepath.Rel(home, s.ProjectPath)
		if err != nil {
			continue
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		for i := 1; i <= len(parts); i++ {
			segment := strings.Join(parts[:i], "/")
			if segment == filterClean || strings.HasSuffix(segment, "/"+filterClean) {
				return i
			}
		}
	}
	return 2
}
