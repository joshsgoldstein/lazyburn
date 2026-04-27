import json
import tempfile
from datetime import datetime, timezone
from pathlib import Path

import pytest

from lazyburn.parser import (
    _parse_jsonl,
    _path_matches,
    aggregate,
    filter_depth,
    group_by_depth,
)
from lazyburn.models import Session, TokenUsage
from lazyburn.pricing import get_pricing


# ── Pricing ────────────────────────────────────────────────────────────────────

def test_get_pricing_known_model():
    p = get_pricing("claude-sonnet-4-6")
    assert p["input"] == 3.0
    assert p["output"] == 15.0
    assert p["cache_5m"] == 3.75
    assert p["cache_1h"] == 6.0
    assert p["cache_read"] == 0.30


def test_get_pricing_fallback():
    p = get_pricing("claude-unknown-model-99")
    assert p["input"] == 3.0  # falls back to sonnet pricing


def test_get_pricing_prefix_match():
    p = get_pricing("claude-opus-4-7-20260101")
    assert p["input"] == 5.0


# ── Path matching ──────────────────────────────────────────────────────────────

def test_path_matches_absolute_prefix():
    assert _path_matches("/home/user/acme/project", "/home/user/acme") is True


def test_path_matches_absolute_exact():
    assert _path_matches("/home/user/acme", "/home/user/acme") is True


def test_path_matches_absolute_no_false_positive():
    # /acme should NOT match /acme-backend
    assert _path_matches("/home/user/acme-backend/proj", "/home/user/acme") is False


def test_path_matches_substring():
    assert _path_matches("/home/user/acme/project", "acme") is True


def test_path_matches_empty_project_path():
    assert _path_matches("", "acme") is False


# ── JSONL parsing ──────────────────────────────────────────────────────────────

def _make_assistant_msg(msg_id, req_id, input_tokens, output_tokens, model="claude-sonnet-4-6"):
    return {
        "type": "assistant",
        "timestamp": "2026-04-01T10:00:00Z",
        "cwd": "/home/user/project",
        "requestId": req_id,
        "sessionId": "sess-1",
        "message": {
            "id": msg_id,
            "model": model,
            "usage": {
                "input_tokens": input_tokens,
                "output_tokens": output_tokens,
                "cache_read_input_tokens": 0,
                "cache_creation": {},
            },
        },
    }


def _write_jsonl(lines: list[dict]) -> Path:
    f = tempfile.NamedTemporaryFile(mode="w", suffix=".jsonl", delete=False)
    for line in lines:
        f.write(json.dumps(line) + "\n")
    f.flush()
    return Path(f.name)


def test_parse_keeps_last_duplicate():
    """Streaming replays the same requestId — we must keep the last (highest) token counts."""
    path = _write_jsonl([
        _make_assistant_msg("msg-1", "req-1", input_tokens=100, output_tokens=50),
        _make_assistant_msg("msg-1", "req-1", input_tokens=200, output_tokens=150),  # last = authoritative
    ])
    session = _parse_jsonl(path, None, None)
    assert session is not None
    assert session.usage.input == 200
    assert session.usage.output == 150


def test_parse_deduplicates_distinct_requests():
    """Different requestIds are separate turns and should both be counted."""
    path = _write_jsonl([
        _make_assistant_msg("msg-1", "req-1", input_tokens=100, output_tokens=50),
        _make_assistant_msg("msg-2", "req-2", input_tokens=200, output_tokens=100),
    ])
    session = _parse_jsonl(path, None, None)
    assert session is not None
    assert session.usage.input == 300
    assert session.usage.output == 150


def test_parse_skips_synthetic_model():
    path = _write_jsonl([
        _make_assistant_msg("msg-1", "req-1", input_tokens=100, output_tokens=50, model="<synthetic>"),
    ])
    session = _parse_jsonl(path, None, None)
    assert session is None


def test_parse_empty_file():
    path = _write_jsonl([])
    assert _parse_jsonl(path, None, None) is None


def test_parse_reads_cwd():
    path = _write_jsonl([
        _make_assistant_msg("msg-1", "req-1", input_tokens=10, output_tokens=5),
    ])
    session = _parse_jsonl(path, None, None)
    assert session is not None
    assert session.project_path == "/home/user/project"


# ── Grouping ───────────────────────────────────────────────────────────────────

def _make_session(project_path: str, cost: float = 1.0) -> Session:
    s = Session(session_id="test", project_path=project_path)
    s.usage.cost = cost
    return s


def test_group_by_depth():
    home = Path("/home/user")
    sessions = [
        _make_session("/home/user/acme/alpha"),
        _make_session("/home/user/acme/beta"),
        _make_session("/home/user/globex/main"),
    ]
    groups = group_by_depth(sessions, 1, home)
    assert set(groups.keys()) == {"acme", "globex"}
    assert len(groups["acme"]) == 2
    assert len(groups["globex"]) == 1


def test_aggregate():
    sessions = [_make_session("/p", cost=1.5), _make_session("/p", cost=2.5)]
    total = aggregate(sessions)
    assert total.cost == pytest.approx(4.0)


# ── filter_depth ───────────────────────────────────────────────────────────────

def test_filter_depth_absolute_path():
    home = Path("/home/user")
    sessions = [_make_session("/home/user/acme/project")]
    assert filter_depth(sessions, "/home/user/acme", home) == 1


def test_filter_depth_substring():
    home = Path("/home/user")
    sessions = [_make_session("/home/user/acme/project")]
    assert filter_depth(sessions, "acme", home) == 1


# ── TokenUsage ─────────────────────────────────────────────────────────────────

def test_token_usage_iadd():
    a = TokenUsage(input=100, output=50, cost=1.0)
    b = TokenUsage(input=200, output=100, cost=2.0)
    a += b
    assert a.input == 300
    assert a.output == 150
    assert a.cost == pytest.approx(3.0)


def test_token_usage_cache_write_property():
    u = TokenUsage(cache_write_5m=100, cache_write_1h=200)
    assert u.cache_write == 300
