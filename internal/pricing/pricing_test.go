package pricing

import "testing"

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
