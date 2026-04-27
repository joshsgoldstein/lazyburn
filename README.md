# lazyburn

<div align="center">

**Track Claude Code costs by folder, session, and date — at any depth in your project hierarchy.**

[![Python 3.11+](https://img.shields.io/badge/python-3.11%2B-blue?logo=python&logoColor=white)](https://python.org)
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Install with pipx](https://img.shields.io/badge/install-pipx-orange)](https://pipx.pypa.io)
[![Works with Claude Code](https://img.shields.io/badge/Claude%20Code-compatible-blueviolet)](https://claude.ai/code)

</div>

```
$ lazyburn --all

  Folder              Sess   Turns   Time    Cache W   Cache R   Output       Cost
 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Documents/acme        18     163   148.4h    10.2M    237.3M   920.5k   $125.74
  Documents/globex       5      52    24.1h     1.8M     59.5M    95.9k    $26.87
  Documents/initech      4      10     8.3h     1.1M     10.3M    54.1k     $8.56
  Documents/lab          3       6     2.1h   190.3k      2.2M     5.7k     $1.82
 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  TOTAL                 30                                                 $162.99
```

- **Folder-first view** — costs roll up by the directory structure you already have on disk
- **Drill into any depth** — `--path acme` shows sub-projects; `--depth 3` goes a level deeper
- **Auto-scopes to cwd** — run `lazyburn` from inside any project and it filters automatically, like `git`
- **Correct 4-bucket pricing** — splits cache writes into 5-minute and 1-hour tiers; most tools collapse these and get the math wrong
- **Session-level detail** — named slug, date, wall-clock duration, last prompt, and full token breakdown per session
- **Date filtering + CSV export** — slice by date range or pipe results to a spreadsheet
- **Pure Python** — no Node.js, no Rust; just `pip install` and go

---

## Install

```sh
curl -sSf https://raw.githubusercontent.com/joshsgoldstein/lazyburn/main/install.sh | sh
```

Requires Python 3.11+ and `pipx` (or `pip`).

**Or install directly:**

```sh
pipx install git+https://github.com/joshsgoldstein/lazyburn.git
```

**Or with pip:**

```sh
pip install --user git+https://github.com/joshsgoldstein/lazyburn.git
```

---

## Usage

### Quick start

```sh
# all projects, grouped by top-level folder
lazyburn --all

# filter to a specific folder (any path substring)
lazyburn --path acme

# run from inside a project — auto-scopes to that directory
cd ~/Documents/acme/project-alpha
lazyburn

# session-level breakdown for the current directory
lazyburn sessions

# session breakdown for a specific folder
lazyburn sessions --path acme
```

### Folder grouping

```sh
# default depth — groups at top-level folders
lazyburn --all

# go deeper — sub-projects within a folder
lazyburn --path acme --depth 3

# filter to a path substring and auto-drill one level below it
lazyburn --path acme
```

### Date filtering

```sh
# sessions after a date
lazyburn --all --since 2026-04-01

# sessions within a range
lazyburn --path acme --since 2026-04-01 --until 2026-04-30

# same filters work on sessions view
lazyburn sessions --since 2026-04-01
```

### Export

```sh
# export folder summary to CSV
lazyburn --all --export costs.csv

# export session detail to CSV
lazyburn sessions --export sessions.csv
lazyburn sessions --path acme --export acme-sessions.csv
```

---

## Output examples

### All projects

```
$ lazyburn --all

  Folder              Sess   Turns   Time    Cache W   Cache R   Output       Cost
 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Documents/acme        18     163   148.4h    10.2M    237.3M   920.5k   $125.74
  Documents/globex       5      52    24.1h     1.8M     59.5M    95.9k    $26.87
  Documents/initech      4      10     8.3h     1.1M     10.3M    54.1k     $8.56
  Documents/lab          3       6     2.1h   190.3k      2.2M     5.7k     $1.82
 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  TOTAL                 30                                                 $162.99
```

### Drilling into a folder

```
$ lazyburn --path acme

Documents/acme/
  Folder            Sess   Turns   Time    Cache W    Cache R   Output       Cost
 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  alpha-platform      13     147   112.3h     8.2M    214.5M   774.4k   $108.62
  api-service          3      14    18.7h     2.0M     20.4M   132.3k    $15.86
  data-pipeline        2       2     4.2h    89.0k      1.7M    13.0k     $1.05
 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  TOTAL               18                                                 $125.53
```

The common folder prefix is shown as a dim header above the table so names stay compact.

### Session breakdown

```
$ lazyburn sessions --path alpha-platform

  Session               Project          Date        Time   Turns   Cache W    Cache R   Output       Cost   Last Prompt
 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  brave-ancient-reef    alpha-platform   2026-04-22   6.3h      22    3.1M     82.4M   312.1k    $43.21   implement the auth flow…
  sleepy-golden-tide    alpha-platform   2026-04-19   4.1h      18    2.8M     71.2M   268.5k    $37.44   fix the dashboard load…
  clever-rushing-wind   alpha-platform   2026-04-15   2.9h      11    1.4M     39.7M   122.8k    $20.18   add export to CSV…
```

Session names come from Claude Code's own slug system (`brave-ancient-reef`, etc.). The **Last Prompt** column shows the opening message of each session so you can identify what you were working on without opening the file.

---

## How it works

lazyburn reads Claude Code's local session files from `~/.claude/projects/`. Each subfolder maps to a directory where you ran `claude`. Each `.jsonl` file inside is one session.

### Token buckets

Token costs are split across four buckets. Most tools only track three — collapsing the two cache write tiers into one — which produces incorrect cost estimates.

| Token type | Field in session data | Price vs input |
|---|---|---|
| Input | `input_tokens` | 1× |
| Cache write (5m) | `cache_creation.ephemeral_5m_input_tokens` | 1.25× |
| Cache write (1h) | `cache_creation.ephemeral_1h_input_tokens` | 2× |
| Cache read | `cache_read_input_tokens` | 0.1× |
| Output | `output_tokens` | 5× |

### Deduplication

Claude Code logs assistant messages multiple times per request due to streaming. lazyburn deduplicates by `requestId` (falling back to `sessionId`) so the token counts match what Anthropic actually bills.

### Project path

The real project path is read from the `cwd` field in each message — not decoded from the encoded folder name, which can be ambiguous.

### Duration

Wall-clock time from the first message timestamp to the last in each session. This is the elapsed time of the session, not active API processing time.

> Cost estimates reflect API token pricing. If you're on a Pro or Max subscription, token counts are accurate but dollar amounts represent the API equivalent — not your actual subscription cost.

---

## Pricing reference

| Model | Input | Cache Write 5m | Cache Write 1h | Cache Read | Output |
|---|---|---|---|---|---|
| Sonnet 4.6 | $3.00 | $3.75 | $6.00 | $0.30 | $15.00 |
| Opus 4.7 | $5.00 | $6.25 | $10.00 | $0.50 | $25.00 |
| Haiku 4.5 | $1.00 | $1.25 | $2.00 | $0.10 | $5.00 |

Per million tokens. Unknown models fall back to Sonnet 4.6 pricing. Source: [Anthropic pricing docs](https://platform.claude.com/docs/en/about-claude/pricing)

---

## Contributing

Pull requests are welcome. For significant changes, open an issue first to discuss the approach.

---

## License

MIT — see [LICENSE](LICENSE)

---

<div align="center">

[⭐ Star this repo](https://github.com/joshsgoldstein/lazyburn) · [🐛 Report a bug](https://github.com/joshsgoldstein/lazyburn/issues) · [💡 Request a feature](https://github.com/joshsgoldstein/lazyburn/issues)

</div>
