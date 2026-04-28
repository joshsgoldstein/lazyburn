# lazyburn

<div align="center">

**See exactly where your Claude Code tokens are going.**

[![Go 1.22+](https://img.shields.io/badge/go-1.22%2B-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![CI](https://github.com/joshsgoldstein/lazyburn/actions/workflows/ci.yml/badge.svg)](https://github.com/joshsgoldstein/lazyburn/actions/workflows/ci.yml)

</div>

---

## Why this exists

I use Claude Code across a lot of projects and had no idea where my money or time was actually going. The Anthropic dashboard shows you a total — it doesn't tell you which project cost $40 last week or how many hours you've sunk into a specific codebase. I built lazyburn to answer those questions: what am I spending, on what, and is it worth it?

If you're a consultant, it's even more direct — your Claude Code spend maps to client work. lazyburn tells you exactly how much AI cost went to each project so you can bill accurately, protect your margins, and stop guessing.

If you're building something with friends or a small team, it's an easy way to keep everyone honest about what the project is actually costing — no surprises at the end of the month.

If you're running Claude Code across multiple projects and want that visibility, this is for you.

---

`lazyburn` reads directly from `~/.claude/projects/` — no config, no API key — and shows your spending broken down by folder, sub-project, and session.

```
$ lazyburn --all

2026-01-15 – 2026-04-26
  Folder                Sess   Turns     Time      Tokens    Cache W    Cache R    Output        Cost
 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Documents/acme          18     163    148.4h    248.3M     10.2M    237.3M    920.5k    $125.74
  Documents/globex         5      52     24.1h     61.4M      1.8M     59.5M     95.9k     $26.87
  Documents/initech        4      10      8.3h     11.5M      1.1M     10.3M     54.1k      $8.56
  Documents/lab            3       6      2.1h      2.4M    190.3k      2.2M      5.7k      $1.82
 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  TOTAL                   30                      323.6M                                  $162.99
```

Run it from inside any project and it automatically scopes to that directory — like `git`.

```
$ cd ~/Documents/acme/alpha-platform && lazyburn

2026-03-01 – 2026-04-26
Documents/acme/alpha-platform/
  Session               Date        Time   Turns     Tokens    Cache W    Cache R    Output        Cost   Last Prompt
 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  brave-ancient-reef    2026-04-22   6.3h      22    85.6M      3.1M     82.4M    312.1k    $43.21   implement the auth flow…
  sleepy-golden-tide    2026-04-19   4.1h      18    74.1M      2.8M     71.2M    268.5k    $37.44   fix the dashboard load…
  clever-rushing-wind   2026-04-15   2.9h      11    41.2M      1.4M     39.7M    122.8k    $20.18   add export to CSV…
```

---

## Install

```sh
curl -sSf https://raw.githubusercontent.com/joshsgoldstein/lazyburn/main/install.sh | sh
```

Or, if you have Go installed:

```sh
go install github.com/joshsgoldstein/lazyburn@latest
```

Binaries for macOS (arm64/amd64), Linux (arm64/amd64), and Windows are on the [releases page](https://github.com/joshsgoldstein/lazyburn/releases).

---

## Usage

```
lazyburn [flags]
lazyburn sessions [flags]
lazyburn update-pricing
```

### Flags

| Flag | Description |
|---|---|
| `--all` | Show all projects; ignore current directory |
| `--path <string>` | Filter by path substring (e.g. `acme`) |
| `--depth <int>` | Folder grouping depth (default: 2) |
| `--sessions` | Show per-session breakdown below the folder table |
| `--since <YYYY-MM-DD>` | Only include sessions after this date |
| `--until <YYYY-MM-DD>` | Only include sessions before this date |
| `--export <file>` | Export results — `.csv`, `.json`, or `.md` |

### Examples

```sh
# see everything, grouped by top-level folder
lazyburn --all

# drill into one client or project
lazyburn --path acme

# what did I spend this month?
lazyburn --all --since 2026-04-01

# folder summary + individual sessions together
lazyburn --path acme --sessions

# just the session list
lazyburn sessions --path alpha-platform

# export to CSV for a spreadsheet
lazyburn --all --export costs.csv

# export to JSON for automations or dashboards
lazyburn --all --export costs.json

# export to Markdown for sharing with clients or pasting into a doc
lazyburn --path acme --export acme-costs.md
```

### Understanding the columns

| Column | What it means |
|---|---|
| **Folder / Session** | The project directory or the name Claude gave to that conversation |
| **Sess** | Number of separate conversations started in that folder |
| **Turns** | How many times you sent a message and got a response — one back-and-forth = one turn |
| **Time** | How long the session was open (wall clock, not active time) |
| **Tokens** | Total volume processed — Claude charges by the token, roughly 750 words = 1,000 tokens |
| **Cache W** | Tokens written to Claude's memory so it remembers your context across turns |
| **Cache R** | Tokens read back from that memory — much cheaper than re-sending everything fresh |
| **Output** | Tokens Claude generated in its responses to you |
| **Cost** | Estimated cost based on Anthropic's API pricing for the models used |
| **Last Prompt** | The first message you sent in that session — useful for remembering what you were working on |

> Cache W and Cache R are shown separately because they're priced differently and most tools get this wrong. See [How it works](#how-it-works) for the full breakdown.

### Drilling into a folder

Filtering to a path automatically groups one level deeper so you can see sub-project breakdown:

```
$ lazyburn --path acme

2026-03-01 – 2026-04-26
Documents/acme/
  Folder              Sess   Turns     Time      Tokens    Cache W    Cache R    Output        Cost
 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  alpha-platform        13     147    112.3h    225.1M      8.2M    214.5M    774.4k    $108.62
  api-service            3      14     18.7h     22.5M      2.0M     20.4M    132.3k     $15.86
  data-pipeline          2       2      4.2h      1.8M     89.0k      1.7M     13.0k      $1.05
  (this folder)          2       4      8.1h      3.5M    320.0k      3.1M     20.0k      $2.10
 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  TOTAL                 20     167             252.9M                                   $127.63
```

The common path prefix is shown above the table so folder names stay compact. Sessions run directly from the filtered folder (not from a sub-project) appear as `(this folder)`.

### Keeping pricing up to date

Model pricing can change. Run this to pull the latest rates from the repo without updating the binary:

```sh
lazyburn update-pricing
```

Pricing is cached at `~/.claude/lazyburn/pricing.json` and used automatically on future runs. Falls back to compiled-in defaults if no cache exists.

---

## How it works

lazyburn reads `~/.claude/projects/` — the same directory Claude Code writes session logs to. Each subfolder is a project (the directory where you ran `claude`), and each `.jsonl` file inside is one session.

### Why the token numbers might differ from other tools

Claude's cache has two write tiers priced differently — a 5-minute tier and a 1-hour tier. Most tools collapse these into one and get the math wrong. lazyburn tracks all four buckets separately:

| Bucket | JSON field | Multiplier |
|---|---|---|
| Input | `input_tokens` | 1× |
| Cache write (5 min) | `cache_creation.ephemeral_5m_input_tokens` | 1.25× |
| Cache write (1 hr) | `cache_creation.ephemeral_1h_input_tokens` | 2× |
| Cache read | `cache_read_input_tokens` | 0.1× |
| Output | `output_tokens` | 5× |

Claude Code also replays each assistant message multiple times as tokens stream in. lazyburn deduplicates by request ID and keeps the final count so nothing is double-billed.

### A note on duration

Session duration is wall-clock time from the first message to the last. A session you left open overnight will show many hours even if you only sent a few messages — there's no way to distinguish idle time from active time in the log data.

> **Subscription users:** Token counts are accurate. Dollar amounts reflect API pricing — not your actual subscription cost.

---

## Pricing reference

| Model | Input | Cache Write 5m | Cache Write 1h | Cache Read | Output |
|---|---|---|---|---|---|
| claude-sonnet-4-6 | $3.00 | $3.75 | $6.00 | $0.30 | $15.00 |
| claude-opus-4-7 | $5.00 | $6.25 | $10.00 | $0.50 | $25.00 |
| claude-haiku-4-5 | $1.00 | $1.25 | $2.00 | $0.10 | $5.00 |

Per million tokens. Unknown models fall back to Sonnet 4.6 pricing. Source: [Anthropic pricing](https://platform.claude.com/docs/en/about-claude/pricing)

---

## Contributing

PRs are welcome. For significant changes, open an issue first.

All changes go through a pull request — CI must pass before merge.

See [ROADMAP.md](ROADMAP.md) for what's planned. Vote on features by reacting 👍 to the corresponding GitHub issue.

---

## License

MIT — see [LICENSE](LICENSE)

---

<div align="center">
<sub>Built for people who want to know where their tokens went.</sub>
</div>
