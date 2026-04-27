package pricing

import (
	"os"
	"testing"
)

func TestGetKnownModel(t *testing.T) {
	p := Get("claude-sonnet-4-6")
	if p.Input != 3.0 || p.Output != 15.0 || p.Cache5m != 3.75 || p.Cache1h != 6.0 || p.CacheRead != 0.30 {
		t.Errorf("unexpected pricing for claude-sonnet-4-6: %+v", p)
	}
}

func TestGetOpus(t *testing.T) {
	p := Get("claude-opus-4-7")
	if p.Input != 5.0 || p.Output != 25.0 {
		t.Errorf("unexpected opus pricing: %+v", p)
	}
}

func TestGetHaiku(t *testing.T) {
	p := Get("claude-haiku-4-5")
	if p.Input != 1.0 || p.Output != 5.0 {
		t.Errorf("unexpected haiku pricing: %+v", p)
	}
}

func TestGetPrefixMatch(t *testing.T) {
	p := Get("claude-opus-4-7-20260101")
	if p.Input != 5.0 {
		t.Errorf("prefix match failed, got input price %f want 5.0", p.Input)
	}
}

func TestGetUnknownFallsBackToSonnet(t *testing.T) {
	p := Get("claude-unknown-model-99")
	if p.Input != 3.0 {
		t.Errorf("fallback should use sonnet pricing, got input %f", p.Input)
	}
}

func TestLoadFileAndApply(t *testing.T) {
	json := `{
		"updated": "2026-01-01",
		"source": "test",
		"models": {
			"claude-test-model": {"input": 9.0, "cache5m": 1.0, "cache1h": 2.0, "cacheRead": 0.5, "output": 45.0}
		}
	}`
	f, err := os.CreateTemp(t.TempDir(), "pricing-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(json)
	f.Close()

	pf, err := LoadFile(f.Name())
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if pf.Updated != "2026-01-01" {
		t.Errorf("Updated: got %q want 2026-01-01", pf.Updated)
	}

	// Apply and verify Get uses the new prices.
	ApplyFile(pf)
	p := Get("claude-test-model")
	if p.Input != 9.0 || p.Output != 45.0 {
		t.Errorf("ApplyFile: unexpected prices %+v", p)
	}

	// Restore defaults so other tests are unaffected.
	active = defaults
}
