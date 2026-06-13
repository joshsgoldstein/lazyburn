package pricing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Prices struct {
	Input     float64
	Cache5m   float64
	Cache1h   float64
	CacheRead float64
	Output    float64
}

// PricingFile is the shape of pricing.json.
type PricingFile struct {
	Updated string                `json:"updated"`
	Source  string                `json:"source"`
	Models  map[string]priceEntry `json:"models"`
}

type priceEntry struct {
	Input     float64 `json:"input"`
	Cache5m   float64 `json:"cache5m"`
	Cache1h   float64 `json:"cache1h"`
	CacheRead float64 `json:"cacheRead"`
	Output    float64 `json:"output"`
}

// CachePath is where lazyburn stores downloaded pricing.
func CachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "lazyburn", "pricing.json")
}

// defaults are compiled in so the binary works with no network and no cache.
// Verified against https://platform.claude.com/docs/en/about-claude/pricing (2026-06-13).
// Fields: Input, Cache5m, Cache1h, CacheRead, Output (USD per 1M tokens).
var defaults = map[string]Prices{
	"claude-fable-5":    {10.0, 12.50, 20.0, 1.0, 50.0},
	"claude-mythos-5":   {10.0, 12.50, 20.0, 1.0, 50.0},
	"claude-opus-4-8":   {5.0, 6.25, 10.0, 0.50, 25.0},
	"claude-opus-4-7":   {5.0, 6.25, 10.0, 0.50, 25.0},
	"claude-opus-4-6":   {5.0, 6.25, 10.0, 0.50, 25.0},
	"claude-opus-4-5":   {5.0, 6.25, 10.0, 0.50, 25.0},
	"claude-sonnet-4-6": {3.0, 3.75, 6.0, 0.30, 15.0},
	"claude-sonnet-4-5": {3.0, 3.75, 6.0, 0.30, 15.0},
	"claude-haiku-4-5":  {1.0, 1.25, 2.0, 0.10, 5.0},
}

// active holds the pricing table actually in use (defaults until overridden).
var active = defaults

func init() {
	if pf, err := LoadFile(CachePath()); err == nil {
		active = mergeDefaults(pf)
	}
}

// LoadFile reads and parses a pricing JSON file.
func LoadFile(path string) (*PricingFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pf PricingFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, err
	}
	return &pf, nil
}

// Get returns the pricing for a model, using the active table (cached file or defaults).
func Get(model string) Prices {
	if p, ok := active[model]; ok {
		return p
	}
	for key, p := range active {
		if strings.HasPrefix(model, key) {
			return p
		}
	}
	// Unknown model: fall back by family so a new Opus/Haiku is never billed
	// as Sonnet (which silently undercounts the most expensive tier).
	switch {
	case strings.Contains(model, "opus"):
		if p, ok := active["claude-opus-4-8"]; ok {
			return p
		}
	case strings.Contains(model, "haiku"):
		if p, ok := active["claude-haiku-4-5"]; ok {
			return p
		}
	}
	return active["claude-sonnet-4-6"]
}

// ApplyFile overlays a PricingFile on top of the compiled defaults and makes it
// the active table. A model absent from the file keeps its built-in price.
func ApplyFile(pf *PricingFile) {
	active = mergeDefaults(pf)
}

// mergeDefaults returns a pricing table seeded from the compiled defaults with
// the file's entries overlaid on top. This prevents a model that's missing from
// the cached/downloaded pricing.json from silently falling back to Sonnet.
func mergeDefaults(pf *PricingFile) map[string]Prices {
	m := make(map[string]Prices, len(defaults)+len(pf.Models))
	for model, p := range defaults {
		m[model] = p
	}
	for model, e := range pf.Models {
		m[model] = Prices{e.Input, e.Cache5m, e.Cache1h, e.CacheRead, e.Output}
	}
	return m
}
