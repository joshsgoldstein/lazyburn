# lazyburn

Claude Code cost tracker by folder. See how much each client, project, or session is burning — right from your terminal.

Built for consultants who use Claude Code across multiple clients and need to know where the spend is going.

## Install

```sh
curl -sSf https://raw.githubusercontent.com/joshsgoldstein/lazyburn/main/install.sh | sh
```

Requires Python 3.11+ and `pipx` (or `pip`).

## Usage

```sh
# all projects grouped by client (depth 2 by default)
lazyburn --all

# drill into a specific client or folder
lazyburn acme
lazyburn acme/project-alpha

# session-level breakdown for current directory
cd ~/Documents/acme/project-alpha
lazyburn

# session-level breakdown from anywhere
lazyburn sessions
lazyburn sessions --path acme

# filter by date
lazyburn --all --since 2026-04-01
lazyburn acme --since 2026-04-01 --until 2026-04-30

# adjust grouping depth
lazyburn --all --depth 3    # sub-project level
lazyburn --all --depth 4    # repo level

# export to CSV
lazyburn --all --export costs.csv
lazyburn sessions --path acme --export sessions.csv
```

## How it works

Reads Claude Code's local session files from `~/.claude/projects/`. Each file is one Claude Code session (one time you ran `claude` in a directory). Token costs are calculated using the official Anthropic pricing for each model, with the full four-bucket breakdown:

| Token type | What it is |
|---|---|
| Input | Raw context not in cache |
| Cache write (5m) | Writing new context to the prompt cache |
| Cache write (1h) | Writing to the longer-lived cache |
| Cache read | Reading from cache (much cheaper) |
| Output | Claude's response |

Duplicate messages are filtered using `requestId` to match actual API billing.

> **Note:** Cost estimates reflect API token pricing. If you're on a Pro or Max subscription, the token counts are accurate but dollar amounts represent the API equivalent, not your actual subscription cost.

## Options

```
lazyburn [PATH] [OPTIONS]

Arguments:
  PATH    Filter to paths containing this string (e.g. acme)

Options:
  --depth INTEGER     Folder depth to group by [default: 2]
  --all               Show all projects, ignore current directory
  --since YYYY-MM-DD  Only include sessions after this date
  --until YYYY-MM-DD  Only include sessions before this date
  --export FILE       Export results to CSV

Commands:
  sessions            Session-level breakdown (use --path to filter)
```

## Pricing

Prices per million tokens as of April 2026:

| Model | Input | Cache Write 5m | Cache Write 1h | Cache Read | Output |
|---|---|---|---|---|---|
| Sonnet 4.6 | $3.00 | $3.75 | $6.00 | $0.30 | $15.00 |
| Opus 4.7 | $5.00 | $6.25 | $10.00 | $0.50 | $25.00 |
| Haiku 4.5 | $1.00 | $1.25 | $2.00 | $0.10 | $5.00 |

Source: [Anthropic pricing docs](https://platform.claude.com/docs/en/about-claude/pricing)
