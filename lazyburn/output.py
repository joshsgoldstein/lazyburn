import csv
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

from rich import box
from rich.console import Console
from rich.table import Table

from .models import Session, TokenUsage
from .parser import aggregate

console = Console()


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


def common_prefix(paths: list[str]) -> str:
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


def _date_range(sessions: list[Session]) -> str:
    dates = [s.start_time for s in sessions if s.start_time]
    if not dates:
        return ""
    lo, hi = min(dates), max(dates)
    if lo.date() == hi.date():
        return lo.strftime("%Y-%m-%d")
    return f"{lo.strftime('%Y-%m-%d')} – {hi.strftime('%Y-%m-%d')}"


def print_groups(sorted_groups: list[tuple[str, list[Session]]], all_sessions: list[Session]) -> None:
    total_cost = sum(aggregate(s).cost for _, s in sorted_groups)
    total_sessions = len(all_sessions)

    folders = [f for f, _ in sorted_groups]
    prefix = common_prefix(folders)

    date_str = _date_range(all_sessions)
    if date_str:
        console.print(f"[dim]{date_str}[/dim]")

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


def print_sessions(sessions: list[Session]) -> None:
    sessions = sorted(sessions, key=lambda s: s.usage.cost, reverse=True)

    date_str = _date_range(sessions)
    if date_str:
        console.print(f"[dim]{date_str}[/dim]")

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


def export_groups_csv(path: str, groups: list[tuple[str, list[Session]]]) -> None:
    with open(path, "w", newline="") as f:
        w = csv.writer(f)
        w.writerow(["folder", "sessions", "turns", "cache_write_tokens", "cache_read_tokens", "output_tokens", "estimated_cost_usd"])
        for folder, sess_list in groups:
            u = aggregate(sess_list)
            turns = sum(s.turn_count for s in sess_list)
            w.writerow([folder, len(sess_list), turns, u.cache_write, u.cache_read, u.output, f"{u.cost:.6f}"])


def export_sessions_csv(path: str, sessions: list[Session]) -> None:
    with open(path, "w", newline="") as f:
        w = csv.writer(f)
        w.writerow(["session_id", "slug", "project_path", "date", "turns", "models",
                    "cache_write_5m", "cache_write_1h", "cache_read", "output", "estimated_cost_usd", "last_prompt"])
        for s in sorted(sessions, key=lambda x: x.usage.cost, reverse=True):
            date_str = s.start_time.strftime("%Y-%m-%d") if s.start_time else ""
            w.writerow([
                s.session_id, s.slug, s.project_path, date_str, s.turn_count,
                "|".join(sorted(s.models)),
                s.usage.cache_write_5m, s.usage.cache_write_1h,
                s.usage.cache_read, s.usage.output,
                f"{s.usage.cost:.6f}", s.last_prompt,
            ])
