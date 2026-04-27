# Roadmap

Features are prioritized by community votes — 👍 the GitHub issue for anything you want to see.

## In progress

Nothing currently in active development.

## Planned

### Git-aware cost tracking
Correlate sessions with git branches so you can see what a feature or PR actually cost to build. Uses the git reflog to map session time windows to the branch that was active.

[Vote on this feature →](https://github.com/joshsgoldstein/lazyburn/issues/1)

---

### Budget system
Set a spend limit per project. lazyburn warns you when you're approaching it and shows remaining budget in the table footer.

```sh
lazyburn budget set acme 200    # $200 limit
lazyburn budget list            # see all budgets and current spend
```

[Vote on this feature →](https://github.com/joshsgoldstein/lazyburn/issues/2)

---

### Cross-directory session tracking
Sessions are currently attributed to the directory where `claude` was launched (`cwd`). In practice, a session often touches files across multiple directories. This would parse tool use results in the session log to show which directories were actually accessed.

[Vote on this feature →](https://github.com/joshsgoldstein/lazyburn/issues/3)

---

### Active time estimation
Current duration is wall-clock — first message to last. A session left open overnight shows many hours even if only a few messages were sent. This would use message timestamps to separate actual working time from idle gaps, showing both wall-clock and estimated active time.

Particularly useful if you're reporting time to clients and don't want to explain why a session shows 40 hours.

[Vote on this feature →](https://github.com/joshsgoldstein/lazyburn/issues/4)

---

### AI-generated session summaries
Generate a plain-English summary of what happened in each session — what was built, what problems were solved, what was left open. Useful for reporting to clients, reviewing your own work, or just remembering what that 6-hour session was actually about.

[Vote on this feature →](https://github.com/joshsgoldstein/lazyburn/issues/5)

---

## Known limitations

- **Duration is wall-clock time** — time between first and last message in a session, not active API time. A session left open overnight will show many hours.
- **Sessions attributed to launch directory only** — if claude is started in `/acme/tracker` but reads files in `/acme/shared`, the spend is attributed to `tracker`. See cross-directory tracking above.
- **Cost estimates reflect API pricing** — if you're on a Pro or Max subscription, dollar amounts represent the API equivalent, not your actual subscription cost.

---

## Suggest a feature

[Open an issue](https://github.com/joshsgoldstein/lazyburn/issues/new) with the `enhancement` label.
