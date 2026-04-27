package pricing

import "strings"

type Prices struct {
	Input     float64
	Cache5m   float64
	Cache1h   float64
	CacheRead float64
	Output    float64
}

var table = map[string]Prices{
	"claude-sonnet-4-6": {3.0, 3.75, 6.0, 0.30, 15.0},
	"claude-opus-4-7":   {5.0, 6.25, 10.0, 0.50, 25.0},
	"claude-haiku-4-5":  {1.0, 1.25, 2.0, 0.10, 5.0},
}

var fallback = table["claude-sonnet-4-6"]

func Get(model string) Prices {
	if p, ok := table[model]; ok {
		return p
	}
	for key, p := range table {
		if strings.HasPrefix(model, key) {
			return p
		}
	}
	return fallback
}
