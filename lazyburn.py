#!/usr/bin/env python3
"""lazyburn - Claude Code session cost tracker by folder"""

import csv
import json
import os
from collections import defaultdict
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

import click
from rich import box
from rich.console import Console
from rich.table import Table

console = Console()

# ── Pricing ────────────────────────────────────────────────────────────────────
# Source: https://platform.claude.com/docs/en/about-claude/pricing
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

FALLBACK_PRICING = {"input": 3.0, "cache_5m": 3.75, "cache_1h": 6.0, "cache_read": 0.30, "output": 15.0}


def get_pricing(model: str) -> dict:
    if model in PRICING:
        return PRICING[model]
    for key in PRICING:
        if model.startswith(key) or key.startswith(model):
            return PRICING[key]
    return FALLBACK_PRICING


# ── Data structures ────────────────────────────────────────────────────────────

@dataclass
class TokenUsage:
    input: int = 0
    cache_write_5m: int = 0
    cache_write_1h: int = 0
    cache_read: int = 0
    output: int = 0
    cost: float = 0.0

    def __iadd__(self, other: "TokenUsage") -> "TokenUsage":
        self.input += other.input
        self.cache_write_5m += other.cache_write_5m
        self.cache_write_1h += other.cache_write_1h
        self.cache_read += other.cache_read
        self.output += other.output
        self.cost += other.cost
        return self

    @property
    def cache_write(self) -> int:
        return self.cache_write_5m + self.cache_write_1h

    @property
    def total(self) -> int:
        return self.input + self.cache_write + self.cache_read + self.output


@dataclass
class Session:
    session_id: str
    project_path: str
    slug: str = ""
    start_time: Optional[datetime] = None
    end_time: Optional[datetime] = None
    usage: TokenUsage = field(default_factory=TokenUsage)
    models: set = field(default_factory=set)
    turn_count: int = 0
    last_prompt: str = ""

    @property
    def duration_minutes(self) -> float:
        if self.start_time and self.end_time:
            return (self.end_time - self.start_time).total_seconds() / 60
        return 0.0


# ── Parser ─────────────────────────────────────────────────────────────────────

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
            if path_filter and path_filter not in session.project_path:
                continue
            sessions.append(session)
    return sessions


def _parse_jsonl(
    path: Path,
    since: Optional[datetime],
    until: Optional[datetime],
) -> Optional[Session]:
    seen: set[str] = set()
    session = Session(session_id=path.stem, project_path="")
    has_cost_data = False

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

            # skip synthetic model entries
            model = message.get("model", "unknown")
            if model == "<synthetic>":
                continue

            # dedup: prefer message.id + requestId, fall back to message.id + sessionId
            msg_id = message.get("id")
            req_id = msg.get("requestId")
            sess_id = msg.get("sessionId")

            if msg_id and req_id:
                dedup_key = f"req:{msg_id}:{req_id}"
            elif msg_id and sess_id:
                dedup_key = f"session:{msg_id}:{sess_id}"
            else:
                dedup_key = None

            if dedup_key:
                if dedup_key in seen:
                    continue
                seen.add(dedup_key)

            # date filter on individual messages
            if ts:
                if since and ts < since:
                    continue
                if until and ts > until:
                    continue

            # split cache creation into 5m vs 1h using the sub-object when available
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

            session.usage += usage
            session.models.add(model)
            has_cost_data = True

        elif msg_type == "system" and msg.get("subtype") == "turn_duration":
            session.turn_count += 1
            if slug := msg.get("slug"):
                session.slug = slug

        elif msg_type == "last-prompt":
            session.last_prompt = msg.get("lastPrompt", "")

    return session if has_cost_data else None


def _parse_ts(ts_str: Optional[str]) -> Optional[datetime]:
    if not ts_str:
        return None
    try:
        return datetime.fromisoformat(ts_str.replace("Z", "+00:00"))
    except (ValueError, AttributeError):
        return None


# ── Grouping ───────────────────────────────────────────────────────────────────

def group_by_depth(sessions: list[Session], depth: int, home: Path) -> dict[str, list[Session]]:
    groups: dict[str, list[Session]] = defaultdict(list)
    for session in sessions:
        key = _path_key(session.project_path, depth, home)
        groups[key].append(session)
    return dict(groups)


def _path_key(project_path: str, depth: int, home: Path) -> str:
    if not project_path:
        return "(unknown)"
    try:
        rel = Path(project_path).relative_to(home)
        parts = rel.parts
        return str(Path(*parts[:depth])) if len(parts) >= depth else str(rel)
    except ValueError:
        return project_path


def aggregate(sessions: list[Session]) -> TokenUsage:
    total = TokenUsage()
    for s in sessions:
        total += s.usage
    return total


# ── Formatting ─────────────────────────────────────────────────────────────────

def _filter_depth(sessions: list[Session], active_filter: str, home: Path) -> int:
    """Find how deep the filter path sits relative to home, so we can group one level below it."""
    # If the filter is an absolute path (e.g. cwd), compute depth directly
    filter_path = Path(active_filter)
    if filter_path.is_absolute():
        try:
            return len(filter_path.relative_to(home).parts)
        except ValueError:
            pass

    # Otherwise it's a substring — search session paths for a matching segment
    for s in sessions:
        if not s.project_path:
            continue
        try:
            rel = Path(s.project_path).relative_to(home)
            parts = rel.parts
            for i in range(1, len(parts) + 1):
                segment = str(Path(*parts[:i]))
                if active_filter.strip("/") in segment:
                    return i
        except ValueError:
            pass
    return 2  # fallback


def _common_prefix(paths: list[str]) -> str:
    if not paths or len(paths) == 1:
        return ""
    parts = [p.split("/") for p in paths]
    common = []
    for segment in zip(*parts):
        if len(set(segment)) == 1:
            common.append(segment[0])
        else:
            break
    return "/".join(common) + "/" if common else ""


def fmt_tokens(n: int) -> str:
    if n >= 1_000_000:
        return f"{n / 1_000_000:.1f}M"
    if n >= 1_000:
        return f"{n / 1_000:.1f}k"
    return str(n)


def fmt_cost(cost: float) -> str:
    return f"${cost:,.4f}"


def fmt_duration(minutes: float) -> str:
    if minutes <= 0:
        return "-"
    if minutes >= 60:
        return f"{minutes / 60:.1f}h"
    return f"{minutes:.0f}m"


def parse_date(s: Optional[str]) -> Optional[datetime]:
    if not s:
        return None
    return datetime.fromisoformat(s).replace(tzinfo=timezone.utc)


# ── CLI ────────────────────────────────────────────────────────────────────────

@click.group(invoke_without_command=True)
@click.option("--path", "path_filter", default=None, metavar="PATH", help="Filter by path substring (e.g. acme)")
@click.option("--depth", default=2, show_default=True, help="Folder depth to group by")
@click.option("--all", "show_all", is_flag=True, help="Show all projects, ignore current directory")
@click.option("--sessions", "show_sessions", is_flag=True, help="Also show per-session breakdown below folder summary")
@click.option("--since", default=None, metavar="YYYY-MM-DD", help="Only include sessions after this date")
@click.option("--until", default=None, metavar="YYYY-MM-DD", help="Only include sessions before this date")
@click.option("--export", default=None, metavar="FILE", help="Export results to CSV")
@click.pass_context
def cli(ctx, path_filter, depth, show_all, show_sessions, since, until, export):
    """Claude Code cost tracker. Run from any project directory to see its burn.

    Filter to a folder: lazyburn --path acme
    """
    if ctx.invoked_subcommand:
        return

    since_dt = parse_date(since)
    until_dt = parse_date(until)
    home = Path.home()
    cwd = Path.cwd()

    # resolve what to filter on — explicit arg wins, then cwd, then everything
    if path_filter:
        active_filter = path_filter
    elif not show_all and cwd != home:
        active_filter = str(cwd)
    else:
        active_filter = None

    with console.status("[dim]Reading sessions...[/dim]"):
        sessions = parse_all_sessions(home / ".claude", since_dt, until_dt, active_filter)

    if not sessions:
        label = active_filter or "anywhere"
        console.print(f"[yellow]No sessions found matching[/yellow] {label}")
        return

    if active_filter:
        # group one level deeper than the filter path itself
        filter_depth = _filter_depth(sessions, active_filter, home)
        groups = group_by_depth(sessions, filter_depth + 1, home)
        sorted_groups = sorted(groups.items(), key=lambda x: aggregate(x[1]).cost, reverse=True)
        if len(groups) > 1:
            _print_groups(sorted_groups, sessions)
            if show_sessions:
                console.print()
                _print_sessions(sessions)
        else:
            _print_sessions(sessions)
    else:
        groups = group_by_depth(sessions, depth, home)
        sorted_groups = sorted(groups.items(), key=lambda x: aggregate(x[1]).cost, reverse=True)
        _print_groups(sorted_groups, sessions)
        if show_sessions:
            console.print()
            _print_sessions(sessions)

    if export:
        _export_csv(export, sorted_groups)
        console.print(f"[green]Exported to {export}[/green]")


@cli.command()
@click.option("--path", "path_filter", default=None, metavar="PATH", help="Filter by path substring")
@click.option("--since", default=None, metavar="YYYY-MM-DD")
@click.option("--until", default=None, metavar="YYYY-MM-DD")
@click.option("--export", default=None, metavar="FILE")
def sessions(path_filter, since, until, export):
    """Show individual session breakdown.

    \b
    lazyburn sessions                    # current directory
    lazyburn sessions --path acme        # filter by path
    """
    since_dt = parse_date(since)
    until_dt = parse_date(until)
    home = Path.home()

    cwd = Path.cwd()
    if not path_filter and cwd != home:
        path_filter = str(cwd)

    with console.status("[dim]Reading sessions...[/dim]"):
        all_sessions = parse_all_sessions(home / ".claude", since_dt, until_dt, path_filter)

    if not all_sessions:
        console.print("[yellow]No sessions found.[/yellow]")
        return

    _print_sessions(all_sessions)

    if export:
        _export_sessions_csv(export, all_sessions)
        console.print(f"[green]Exported to {export}[/green]")


# ── Output helpers ─────────────────────────────────────────────────────────────

def _print_groups(sorted_groups, all_sessions):
    total_cost = sum(aggregate(s).cost for _, s in sorted_groups)
    total_sessions = len(all_sessions)

    # strip common prefix to keep folder names short
    folders = [f for f, _ in sorted_groups]
    prefix = _common_prefix(folders)

    table = Table(box=box.SIMPLE_HEAVY, show_footer=True, padding=(0, 1))
    table.add_column("Folder", footer="TOTAL", style="cyan", no_wrap=True)
    table.add_column("Sess", justify="right", footer=str(total_sessions))
    table.add_column("Turns", justify="right", footer="")
    table.add_column("Time", justify="right", footer="")
    table.add_column("Cache W", justify="right", footer="")
    table.add_column("Cache R", justify="right", footer="")
    table.add_column("Output", justify="right", footer="")
    table.add_column("Cost", justify="right", footer=fmt_cost(total_cost), style="green", no_wrap=True, min_width=10)

    for folder, sess_list in sorted_groups:
        u = aggregate(sess_list)
        turns = sum(s.turn_count for s in sess_list)
        total_mins = sum(s.duration_minutes for s in sess_list)
        label = folder[len(prefix):] or folder
        table.add_row(
            label,
            str(len(sess_list)),
            str(turns),
            fmt_duration(total_mins),
            fmt_tokens(u.cache_write),
            fmt_tokens(u.cache_read),
            fmt_tokens(u.output),
            fmt_cost(u.cost),
        )

    if prefix:
        console.print(f"[dim]{prefix}[/dim]")
    console.print(table)


def _print_sessions(sessions: list[Session]):
    sessions = sorted(sessions, key=lambda s: s.usage.cost, reverse=True)

    table = Table(box=box.SIMPLE_HEAVY, padding=(0, 1))
    table.add_column("Session", style="cyan", no_wrap=True)
    table.add_column("Project", no_wrap=True)
    table.add_column("Date")
    table.add_column("Time", justify="right")
    table.add_column("Turns", justify="right")
    table.add_column("Cache W", justify="right")
    table.add_column("Cache R", justify="right")
    table.add_column("Output", justify="right")
    table.add_column("Cost", justify="right", style="green", no_wrap=True, min_width=10)
    table.add_column("Last Prompt")

    for s in sessions:
        date_str = s.start_time.strftime("%Y-%m-%d") if s.start_time else "?"
        slug = s.slug or s.session_id[:8]
        proj = Path(s.project_path).name if s.project_path else "?"
        prompt = s.last_prompt[:40] + "…" if len(s.last_prompt) > 40 else s.last_prompt
        table.add_row(
            slug,
            proj,
            date_str,
            fmt_duration(s.duration_minutes),
            str(s.turn_count),
            fmt_tokens(s.usage.cache_write),
            fmt_tokens(s.usage.cache_read),
            fmt_tokens(s.usage.output),
            fmt_cost(s.usage.cost),
            prompt,
        )

    console.print(table)


# ── CSV export ─────────────────────────────────────────────────────────────────

def _export_csv(path: str, groups):
    with open(path, "w", newline="") as f:
        w = csv.writer(f)
        w.writerow(["folder", "sessions", "turns", "cache_write_tokens", "cache_read_tokens", "output_tokens", "estimated_cost_usd"])
        for folder, sess_list in groups:
            u = aggregate(sess_list)
            turns = sum(s.turn_count for s in sess_list)
            w.writerow([folder, len(sess_list), turns, u.cache_write, u.cache_read, u.output, f"{u.cost:.6f}"])


def _export_sessions_csv(path: str, sessions: list[Session]):
    with open(path, "w", newline="") as f:
        w = csv.writer(f)
        w.writerow(["session_id", "slug", "project_path", "date", "turns", "models", "cache_write_5m", "cache_write_1h", "cache_read", "output", "estimated_cost_usd", "last_prompt"])
        for s in sorted(sessions, key=lambda x: x.usage.cost, reverse=True):
            date_str = s.start_time.strftime("%Y-%m-%d") if s.start_time else ""
            w.writerow([
                s.session_id, s.slug, s.project_path, date_str, s.turn_count,
                "|".join(sorted(s.models)),
                s.usage.cache_write_5m, s.usage.cache_write_1h,
                s.usage.cache_read, s.usage.output,
                f"{s.usage.cost:.6f}", s.last_prompt,
            ])


if __name__ == "__main__":
    cli()
