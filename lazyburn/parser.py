import json
import os
from collections import defaultdict
from datetime import datetime
from pathlib import Path
from typing import Optional

from .models import Session, TokenUsage
from .pricing import get_pricing


def parse_all_sessions(
    claude_dir: Path,
    since: Optional[datetime] = None,
    until: Optional[datetime] = None,
    path_filter: Optional[str] = None,
) -> list[Session]:
    projects_dir = claude_dir / "projects"
    sessions = []
    for proj_dir in projects_dir.iterdir():
        if not proj_dir.is_dir():
            continue
        for jsonl_file in proj_dir.glob("*.jsonl"):
            session = _parse_jsonl(jsonl_file, since, until)
            if session is None:
                continue
            if path_filter and not _path_matches(session.project_path, path_filter):
                continue
            sessions.append(session)
    return sessions


def group_by_depth(sessions: list[Session], depth: int, home: Path) -> dict[str, list[Session]]:
    groups: dict[str, list[Session]] = defaultdict(list)
    for session in sessions:
        key = _path_key(session.project_path, depth, home)
        groups[key].append(session)
    return dict(groups)


def aggregate(sessions: list[Session]) -> TokenUsage:
    total = TokenUsage()
    for s in sessions:
        total += s.usage
    return total


def filter_depth(sessions: list[Session], active_filter: str, home: Path) -> int:
    """Return how deep the filter path sits relative to home."""
    filter_path = Path(active_filter)
    if filter_path.is_absolute():
        try:
            return len(filter_path.relative_to(home).parts)
        except ValueError:
            pass

    filter_clean = active_filter.strip("/")
    for s in sessions:
        if not s.project_path:
            continue
        try:
            rel = Path(s.project_path).relative_to(home)
            parts = rel.parts
            for i in range(1, len(parts) + 1):
                segment = str(Path(*parts[:i]))
                if segment == filter_clean or segment.endswith("/" + filter_clean):
                    return i
        except ValueError:
            pass
    return 2


def _path_matches(project_path: str, path_filter: str) -> bool:
    if not project_path:
        return False
    if os.path.isabs(path_filter):
        return project_path == path_filter or project_path.startswith(path_filter + "/")
    return path_filter in project_path


def _path_key(project_path: str, depth: int, home: Path) -> str:
    if not project_path:
        return "(unknown)"
    try:
        rel = Path(project_path).relative_to(home)
        parts = rel.parts
        return str(Path(*parts[:depth])) if len(parts) >= depth else str(rel)
    except ValueError:
        return project_path


def _parse_jsonl(
    path: Path,
    since: Optional[datetime],
    until: Optional[datetime],
) -> Optional[Session]:
    # keyed by dedup_key -> (TokenUsage, model); overwritten each occurrence so we keep the last
    deduped: dict[str, tuple[TokenUsage, str]] = {}
    session = Session(session_id=path.stem, project_path="")

    try:
        lines = path.read_text().splitlines()
    except OSError:
        return None

    for line in lines:
        line = line.strip()
        if not line:
            continue
        try:
            msg = json.loads(line)
        except json.JSONDecodeError:
            continue

        msg_type = msg.get("type")
        ts = _parse_ts(msg.get("timestamp"))

        if ts:
            if session.start_time is None or ts < session.start_time:
                session.start_time = ts
            if session.end_time is None or ts > session.end_time:
                session.end_time = ts

        if cwd := msg.get("cwd"):
            if not session.project_path:
                session.project_path = cwd

        if msg_type == "assistant":
            message = msg.get("message", {})
            usage_raw = message.get("usage")
            if not usage_raw:
                continue

            model = message.get("model", "unknown")
            if model == "<synthetic>":
                continue

            if ts:
                if since and ts < since:
                    continue
                if until and ts > until:
                    continue

            cache_creation_obj = usage_raw.get("cache_creation", {})
            if cache_creation_obj:
                c5m = cache_creation_obj.get("ephemeral_5m_input_tokens", 0)
                c1h = cache_creation_obj.get("ephemeral_1h_input_tokens", 0)
            else:
                c5m = usage_raw.get("cache_creation_input_tokens", 0)
                c1h = 0

            p = get_pricing(model)
            usage = TokenUsage(
                input=usage_raw.get("input_tokens", 0),
                cache_write_5m=c5m,
                cache_write_1h=c1h,
                cache_read=usage_raw.get("cache_read_input_tokens", 0),
                output=usage_raw.get("output_tokens", 0),
            )
            usage.cost = (
                usage.input * p["input"]
                + usage.cache_write_5m * p["cache_5m"]
                + usage.cache_write_1h * p["cache_1h"]
                + usage.cache_read * p["cache_read"]
                + usage.output * p["output"]
            ) / 1_000_000

            # streaming replays the same request multiple times — keep the last (most complete) entry
            msg_id = message.get("id")
            req_id = msg.get("requestId")
            sess_id = msg.get("sessionId")

            if msg_id and req_id:
                dedup_key = f"req:{msg_id}:{req_id}"
            elif msg_id and sess_id:
                dedup_key = f"session:{msg_id}:{sess_id}"
            else:
                dedup_key = f"anon:{len(deduped)}"

            deduped[dedup_key] = (usage, model)

        elif msg_type == "system" and msg.get("subtype") == "turn_duration":
            session.turn_count += 1
            if slug := msg.get("slug"):
                session.slug = slug

        elif msg_type == "last-prompt":
            session.last_prompt = msg.get("lastPrompt", "")

    for usage, model in deduped.values():
        session.usage += usage
        session.models.add(model)

    return session if deduped else None


def _parse_ts(ts_str: Optional[str]) -> Optional[datetime]:
    if not ts_str:
        return None
    try:
        return datetime.fromisoformat(ts_str.replace("Z", "+00:00"))
    except (ValueError, AttributeError):
        return None
