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
	Updated string             `json:"updated"`
	Source  string             `json:"source"`
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
var defaults = map[string]Prices{
	"claude-sonnet-4-6": {3.0, 3.75, 6.0, 0.30, 15.0},
	"claude-opus-4-7":   {5.0, 6.25, 10.0, 0.50, 25.0},
	"claude-haiku-4-5":  {1.0, 1.25, 2.0, 0.10, 5.0},
}

// active holds the pricing table actually in use (defaults until overridden).
var active = defaults

func init() {
	if pf, err := LoadFile(CachePath()); err == nil {
		active = toMap(pf)
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
	return active["claude-sonnet-4-6"]
}

// ApplyFile replaces the active pricing table with the contents of a PricingFile.
func ApplyFile(pf *PricingFile) {
	active = toMap(pf)
}

func toMap(pf *PricingFile) map[string]Prices {
	m := make(map[string]Prices, len(pf.Models))
	for model, e := range pf.Models {
		m[model] = Prices{e.Input, e.Cache5m, e.Cache1h, e.CacheRead, e.Output}
	}
	return m
}
