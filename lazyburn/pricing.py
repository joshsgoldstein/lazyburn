# Source: https://platform.claude.com/docs/en/about-claude/pricing
# All prices per million tokens.
# cache_5m = 5-minute cache write (1.25x input)
# cache_1h = 1-hour cache write (2x input)
# cache_read = cache hit (0.1x input)

PRICING: dict[str, dict[str, float]] = {
    "claude-sonnet-4-6":         {"input": 3.0,  "cache_5m": 3.75, "cache_1h": 6.0,  "cache_read": 0.30, "output": 15.0},
    "claude-sonnet-4-5":         {"input": 3.0,  "cache_5m": 3.75, "cache_1h": 6.0,  "cache_read": 0.30, "output": 15.0},
    "claude-sonnet-4":           {"input": 3.0,  "cache_5m": 3.75, "cache_1h": 6.0,  "cache_read": 0.30, "output": 15.0},
    "claude-opus-4-7":           {"input": 5.0,  "cache_5m": 6.25, "cache_1h": 10.0, "cache_read": 0.50, "output": 25.0},
    "claude-opus-4-6":           {"input": 5.0,  "cache_5m": 6.25, "cache_1h": 10.0, "cache_read": 0.50, "output": 25.0},
    "claude-opus-4-5":           {"input": 5.0,  "cache_5m": 6.25, "cache_1h": 10.0, "cache_read": 0.50, "output": 25.0},
    "claude-haiku-4-5":          {"input": 1.0,  "cache_5m": 1.25, "cache_1h": 2.0,  "cache_read": 0.10, "output": 5.0},
    "claude-haiku-3-5-20241022": {"input": 0.80, "cache_5m": 1.0,  "cache_1h": 1.6,  "cache_read": 0.08, "output": 4.0},
    "claude-haiku-3-5":          {"input": 0.80, "cache_5m": 1.0,  "cache_1h": 1.6,  "cache_read": 0.08, "output": 4.0},
}

_FALLBACK = {"input": 3.0, "cache_5m": 3.75, "cache_1h": 6.0, "cache_read": 0.30, "output": 15.0}


def get_pricing(model: str) -> dict[str, float]:
    if model in PRICING:
        return PRICING[model]
    for key in PRICING:
        if model.startswith(key) or key.startswith(model):
            return PRICING[key]
    return _FALLBACK
