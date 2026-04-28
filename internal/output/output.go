package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/joshsgoldstein/lazyburn/internal/models"
	"github.com/joshsgoldstein/lazyburn/internal/parser"
)

// Group is a named bucket of sessions for table display.
type Group struct {
	Folder   string
	Sessions []models.Session
}

// ── Formatters ─────────────────────────────────────────────────────────────────

func FmtTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return strconv.Itoa(n)
	}
}

func FmtCost(cost float64) string {
	return fmt.Sprintf("$%.2f", cost)
}

func FmtDuration(minutes float64) string {
	switch {
	case minutes <= 0:
		return "-"
	case minutes >= 60:
		return fmt.Sprintf("%.1fh", minutes/60)
	default:
		return fmt.Sprintf("%.0fm", minutes)
	}
}

func DateRange(sessions []models.Session) string {
	var lo, hi time.Time
	for _, s := range sessions {
		if s.StartTime.IsZero() {
			continue
		}
		if lo.IsZero() || s.StartTime.Before(lo) {
			lo = s.StartTime
		}
		if hi.IsZero() || s.StartTime.After(hi) {
			hi = s.StartTime
		}
	}
	if lo.IsZero() {
		return ""
	}
	loStr := lo.Format("2006-01-02")
	hiStr := hi.Format("2006-01-02")
	if loStr == hiStr {
		return loStr
	}
	return loStr + " – " + hiStr
}

func GroupLabel(folder, prefix string) string {
	label := strings.TrimPrefix(folder, prefix)
	switch {
	case label == folder && prefix != "":
		return "(this folder)"
	case label == "":
		return folder
	}
	return label
}

func CommonPrefix(folders []string) string {
	if len(folders) <= 1 {
		return ""
	}
	split := make([][]string, len(folders))
	for i, f := range folders {
		split[i] = strings.Split(f, "/")
	}
	var common []string
	for i := 0; i < len(split[0]); i++ {
		seg := split[0][i]
		match := true
		for _, parts := range split[1:] {
			if i >= len(parts) || parts[i] != seg {
				match = false
				break
			}
		}
		if !match {
			break
		}
		common = append(common, seg)
	}
	if len(common) == 0 {
		return ""
	}
	return strings.Join(common, "/") + "/"
}

// ── Table style ────────────────────────────────────────────────────────────────

var tableStyle = table.Style{
	Name: "lazyburn",
	Box: table.BoxStyle{
		BottomLeft:       " ",
		BottomRight:      " ",
		BottomSeparator:  "━",
		EmptySeparator:   " ",
		Left:             " ",
		LeftSeparator:    " ",
		MiddleHorizontal: "━",
		MiddleSeparator:  " ",
		MiddleVertical:   " ",
		PaddingLeft:      " ",
		PaddingRight:     " ",
		PageSeparator:    "\n",
		Right:            " ",
		RightSeparator:   " ",
		TopLeft:          " ",
		TopRight:         " ",
		TopSeparator:     " ",
		UnfinishedRow:    " ",
	},
	Color: table.ColorOptionsDefault,
	Format: table.FormatOptions{
		Footer: text.FormatDefault,
		Header: text.FormatDefault,
		Row:    text.FormatDefault,
	},
	HTML:    table.DefaultHTMLOptions,
	Options: table.Options{
		DrawBorder:      false,
		SeparateColumns: false,
		SeparateFooter:  true,
		SeparateHeader:  true,
		SeparateRows:    false,
	},
	Title: table.TitleOptionsDefault,
}

// ── Group table ────────────────────────────────────────────────────────────────

func PrintGroups(groups []Group, allSessions []models.Session) {
	if dr := DateRange(allSessions); dr != "" {
		fmt.Println("📅 " + dr)
	}

	folders := make([]string, len(groups))
	for i, g := range groups {
		folders[i] = g.Folder
	}
	prefix := CommonPrefix(folders)
	if prefix != "" {
		fmt.Printf("%s\n", prefix)
	}

	totalCost := 0.0
	totalTokens := 0
	for _, g := range groups {
		u := parser.Aggregate(g.Sessions)
		totalCost += u.Cost
		totalTokens += u.Input + u.CacheWrite() + u.CacheRead + u.Output
	}

	// Find the top cost for 🔥 threshold (top 25% or at least the #1 row).
	maxCost := 0.0
	for _, g := range groups {
		if c := parser.Aggregate(g.Sessions).Cost; c > maxCost {
			maxCost = c
		}
	}
	fireThreshold := maxCost * 0.75

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(tableStyle)
	t.AppendHeader(table.Row{"Folder", "Sess", "Turns", "Time", "Tokens", "Cache W", "Cache R", "Output", "💰 Cost"})
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, Colors: text.Colors{text.FgCyan}},
		{Number: 2, Align: text.AlignRight},
		{Number: 3, Align: text.AlignRight},
		{Number: 4, Align: text.AlignRight},
		{Number: 5, Align: text.AlignRight},
		{Number: 6, Align: text.AlignRight},
		{Number: 7, Align: text.AlignRight},
		{Number: 8, Align: text.AlignRight},
		{Number: 9, Align: text.AlignRight, Colors: text.Colors{text.FgGreen}, WidthMin: 10},
	})

	for _, g := range groups {
		u := parser.Aggregate(g.Sessions)
		turns := 0
		totalMins := 0.0
		for _, s := range g.Sessions {
			turns += s.TurnCount
			totalMins += s.DurationMinutes()
		}
		label := GroupLabel(g.Folder, prefix)
		if u.Cost >= fireThreshold {
			label = "🔥 " + label
		}
		tokens := u.Input + u.CacheWrite() + u.CacheRead + u.Output
		t.AppendRow(table.Row{
			label,
			len(g.Sessions),
			turns,
			FmtDuration(totalMins),
			FmtTokens(tokens),
			FmtTokens(u.CacheWrite()),
			FmtTokens(u.CacheRead),
			FmtTokens(u.Output),
			FmtCost(u.Cost),
		})
	}

	t.AppendFooter(table.Row{"TOTAL", len(allSessions), "", "", FmtTokens(totalTokens), "", "", "", FmtCost(totalCost)})
	t.Render()
}

// ── Session table ──────────────────────────────────────────────────────────────

func PrintSessions(sessions []models.Session) {
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Usage.Cost > sessions[j].Usage.Cost
	})

	if dr := DateRange(sessions); dr != "" {
		fmt.Println("📅 " + dr)
	}

	maxCost := 0.0
	for _, s := range sessions {
		if s.Usage.Cost > maxCost {
			maxCost = s.Usage.Cost
		}
	}
	fireThreshold := maxCost * 0.75

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(tableStyle)
	t.AppendHeader(table.Row{"Session", "Project", "Date", "Time", "Turns", "Tokens", "Cache W", "Cache R", "Output", "💰 Cost", "Last Prompt"})
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, Colors: text.Colors{text.FgCyan}},
		{Number: 10, Align: text.AlignRight, Colors: text.Colors{text.FgGreen}, WidthMin: 10},
	})

	for _, s := range sessions {
		slug := s.Slug
		if slug == "" && len(s.ID) >= 8 {
			slug = s.ID[:8]
		}
		if s.Usage.Cost >= fireThreshold {
			slug = "🔥 " + slug
		}
		proj := filepath.Base(s.ProjectPath)
		dateStr := "?"
		if !s.StartTime.IsZero() {
			dateStr = s.StartTime.Format("2006-01-02")
		}
		prompt := s.LastPrompt
		if len([]rune(prompt)) > 40 {
			prompt = string([]rune(prompt)[:40]) + "…"
		}
		tokens := s.Usage.Input + s.Usage.CacheWrite() + s.Usage.CacheRead + s.Usage.Output
		t.AppendRow(table.Row{
			slug, proj, dateStr,
			FmtDuration(s.DurationMinutes()),
			s.TurnCount,
			FmtTokens(tokens),
			FmtTokens(s.Usage.CacheWrite()),
			FmtTokens(s.Usage.CacheRead),
			FmtTokens(s.Usage.Output),
			FmtCost(s.Usage.Cost),
			prompt,
		})
	}
	t.Render()
}

// ── CSV export ─────────────────────────────────────────────────────────────────

func ExportGroupsCSV(path string, groups []Group) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write([]string{"folder", "sessions", "turns", "cache_write_tokens", "cache_read_tokens", "output_tokens", "estimated_cost_usd"})
	for _, g := range groups {
		u := parser.Aggregate(g.Sessions)
		turns := 0
		for _, s := range g.Sessions {
			turns += s.TurnCount
		}
		w.Write([]string{
			g.Folder, strconv.Itoa(len(g.Sessions)), strconv.Itoa(turns),
			strconv.Itoa(u.CacheWrite()), strconv.Itoa(u.CacheRead), strconv.Itoa(u.Output),
			fmt.Sprintf("%.6f", u.Cost),
		})
	}
	w.Flush()
	return w.Error()
}

func ExportGroupsJSON(path string, groups []Group) error {
	type row struct {
		Folder  string  `json:"folder"`
		Sessions int    `json:"sessions"`
		Turns   int     `json:"turns"`
		CacheWriteTokens int `json:"cache_write_tokens"`
		CacheReadTokens  int `json:"cache_read_tokens"`
		OutputTokens     int `json:"output_tokens"`
		EstimatedCostUSD float64 `json:"estimated_cost_usd"`
	}
	rows := make([]row, 0, len(groups))
	for _, g := range groups {
		u := parser.Aggregate(g.Sessions)
		turns := 0
		for _, s := range g.Sessions {
			turns += s.TurnCount
		}
		rows = append(rows, row{
			Folder:           g.Folder,
			Sessions:         len(g.Sessions),
			Turns:            turns,
			CacheWriteTokens: u.CacheWrite(),
			CacheReadTokens:  u.CacheRead,
			OutputTokens:     u.Output,
			EstimatedCostUSD: u.Cost,
		})
	}
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}

func ExportSessionsJSON(path string, sessions []models.Session) error {
	type row struct {
		SessionID        string   `json:"session_id"`
		Slug             string   `json:"slug"`
		ProjectPath      string   `json:"project_path"`
		Date             string   `json:"date"`
		Turns            int      `json:"turns"`
		Models           []string `json:"models"`
		CacheWrite5m     int      `json:"cache_write_5m"`
		CacheWrite1h     int      `json:"cache_write_1h"`
		CacheReadTokens  int      `json:"cache_read_tokens"`
		OutputTokens     int      `json:"output_tokens"`
		EstimatedCostUSD float64  `json:"estimated_cost_usd"`
		LastPrompt       string   `json:"last_prompt"`
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Usage.Cost > sessions[j].Usage.Cost
	})
	rows := make([]row, 0, len(sessions))
	for _, s := range sessions {
		modelList := make([]string, 0, len(s.Models))
		for m := range s.Models {
			modelList = append(modelList, m)
		}
		sort.Strings(modelList)
		dateStr := ""
		if !s.StartTime.IsZero() {
			dateStr = s.StartTime.Format("2006-01-02")
		}
		rows = append(rows, row{
			SessionID:        s.ID,
			Slug:             s.Slug,
			ProjectPath:      s.ProjectPath,
			Date:             dateStr,
			Turns:            s.TurnCount,
			Models:           modelList,
			CacheWrite5m:     s.Usage.CacheWrite5m,
			CacheWrite1h:     s.Usage.CacheWrite1h,
			CacheReadTokens:  s.Usage.CacheRead,
			OutputTokens:     s.Usage.Output,
			EstimatedCostUSD: s.Usage.Cost,
			LastPrompt:       s.LastPrompt,
		})
	}
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}

func ExportSessionsCSV(path string, sessions []models.Session) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write([]string{"session_id", "slug", "project_path", "date", "turns", "models", "cache_write_5m", "cache_write_1h", "cache_read", "output", "estimated_cost_usd", "last_prompt"})
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Usage.Cost > sessions[j].Usage.Cost
	})
	for _, s := range sessions {
		modelList := make([]string, 0, len(s.Models))
		for m := range s.Models {
			modelList = append(modelList, m)
		}
		sort.Strings(modelList)
		dateStr := ""
		if !s.StartTime.IsZero() {
			dateStr = s.StartTime.Format("2006-01-02")
		}
		w.Write([]string{
			s.ID, s.Slug, s.ProjectPath, dateStr, strconv.Itoa(s.TurnCount),
			strings.Join(modelList, "|"),
			strconv.Itoa(s.Usage.CacheWrite5m), strconv.Itoa(s.Usage.CacheWrite1h),
			strconv.Itoa(s.Usage.CacheRead), strconv.Itoa(s.Usage.Output),
			fmt.Sprintf("%.6f", s.Usage.Cost), s.LastPrompt,
		})
	}
	w.Flush()
	return w.Error()
}

// ── Markdown export ────────────────────────────────────────────────────────────

func ExportGroupsMD(path string, groups []Group) error {
	var sb strings.Builder
	sb.WriteString("# lazyburn — Cost by Folder\n\n")
	sb.WriteString("| Folder | Sessions | Turns | Cache Write | Cache Read | Output | Cost |\n")
	sb.WriteString("|--------|----------|-------|------------|------------|--------|------|\n")
	for _, g := range groups {
		u := parser.Aggregate(g.Sessions)
		turns := 0
		for _, s := range g.Sessions {
			turns += s.TurnCount
		}
		sb.WriteString(fmt.Sprintf("| %s | %d | %d | %s | %s | %s | %s |\n",
			g.Folder,
			len(g.Sessions),
			turns,
			FmtTokens(u.CacheWrite()),
			FmtTokens(u.CacheRead),
			FmtTokens(u.Output),
			FmtCost(u.Cost),
		))
	}
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func ExportSessionsMD(path string, sessions []models.Session) error {
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Usage.Cost > sessions[j].Usage.Cost
	})
	var sb strings.Builder
	sb.WriteString("# lazyburn — Session Breakdown\n\n")
	sb.WriteString("| Session | Project | Date | Turns | Tokens | Cache Write | Cache Read | Output | Cost | Last Prompt |\n")
	sb.WriteString("|---------|---------|------|-------|--------|------------|------------|--------|------|-------------|\n")
	for _, s := range sessions {
		slug := s.Slug
		if slug == "" && len(s.ID) >= 8 {
			slug = s.ID[:8]
		}
		proj := filepath.Base(s.ProjectPath)
		dateStr := ""
		if !s.StartTime.IsZero() {
			dateStr = s.StartTime.Format("2006-01-02")
		}
		prompt := s.LastPrompt
		if len([]rune(prompt)) > 60 {
			prompt = string([]rune(prompt)[:60]) + "…"
		}
		tokens := s.Usage.Input + s.Usage.CacheWrite() + s.Usage.CacheRead + s.Usage.Output
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s | %s | %s | %s | %s | %s |\n",
			slug, proj, dateStr, s.TurnCount,
			FmtTokens(tokens),
			FmtTokens(s.Usage.CacheWrite()),
			FmtTokens(s.Usage.CacheRead),
			FmtTokens(s.Usage.Output),
			FmtCost(s.Usage.Cost),
			strings.ReplaceAll(prompt, "|", "\\|"),
		))
	}
	return os.WriteFile(path, []byte(sb.String()), 0644)
}
