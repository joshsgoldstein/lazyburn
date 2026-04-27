package main

import "strings"

// Prices holds per-million-token rates for one model.
type Prices struct {
	Input     float64
	Cache5m   float64
	Cache1h   float64
	CacheRead float64
	Output    float64
}

// pricing maps model IDs to their rates.
// Source: https://platform.claude.com/docs/en/about-claude/pricing
var pricing = map[string]Prices{
	"claude-sonnet-4-6":         {3.0, 3.75, 6.0, 0.30, 15.0},
	"claude-sonnet-4-5":         {3.0, 3.75, 6.0, 0.30, 15.0},
	"claude-sonnet-4":           {3.0, 3.75, 6.0, 0.30, 15.0},
	"claude-opus-4-7":           {5.0, 6.25, 10.0, 0.50, 25.0},
	"claude-opus-4-6":           {5.0, 6.25, 10.0, 0.50, 25.0},
	"claude-opus-4-5":           {5.0, 6.25, 10.0, 0.50, 25.0},
	"claude-haiku-4-5":          {1.0, 1.25, 2.0, 0.10, 5.0},
	"claude-haiku-3-5-20241022": {0.80, 1.0, 1.6, 0.08, 4.0},
	"claude-haiku-3-5":          {0.80, 1.0, 1.6, 0.08, 4.0},
}

var fallbackPricing = Prices{3.0, 3.75, 6.0, 0.30, 15.0}

// getPricing returns the pricing for a model, falling back to Sonnet 4.6 rates.
func getPricing(model string) Prices {
	if p, ok := pricing[model]; ok {
		return p
	}
	for key, p := range pricing {
		if strings.HasPrefix(model, key) || strings.HasPrefix(key, model) {
			return p
		}
	}
	return fallbackPricing
}
