from pathlib import Path

import click

from .output import (
    console,
    export_groups_csv,
    export_sessions_csv,
    parse_date,
    print_groups,
    print_sessions,
)
from .parser import aggregate, filter_depth, group_by_depth, parse_all_sessions


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
        fd = filter_depth(sessions, active_filter, home)
        groups = group_by_depth(sessions, fd + 1, home)
        sorted_groups = sorted(groups.items(), key=lambda x: aggregate(x[1]).cost, reverse=True)
        if len(groups) > 1:
            print_groups(sorted_groups, sessions)
            if show_sessions:
                console.print()
                print_sessions(sessions)
            if export:
                export_groups_csv(export, sorted_groups)
                console.print(f"[green]Exported to {export}[/green]")
        else:
            print_sessions(sessions)
            if export:
                export_sessions_csv(export, sessions)
                console.print(f"[green]Exported to {export}[/green]")
    else:
        groups = group_by_depth(sessions, depth, home)
        sorted_groups = sorted(groups.items(), key=lambda x: aggregate(x[1]).cost, reverse=True)
        print_groups(sorted_groups, sessions)
        if show_sessions:
            console.print()
            print_sessions(sessions)
        if export:
            export_groups_csv(export, sorted_groups)
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

    print_sessions(all_sessions)

    if export:
        export_sessions_csv(export, all_sessions)
        console.print(f"[green]Exported to {export}[/green]")
