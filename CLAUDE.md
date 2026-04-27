# lazyburn

Claude Code session cost tracker by folder. Python CLI using Click + Rich.

## Structure

- `lazyburn.py` — single-file CLI, all logic here
- `pyproject.toml` — package config, entry point: `lazyburn = "lazyburn:cli"`
- `install.sh` — curl installer using pipx

## Data source

Reads `~/.claude/projects/` — each subfolder is a project (directory where `claude` was run), each `.jsonl` inside is one session. Tool-result subfolders next to the JSONL files are ignored (they're just output blobs).

## Key concepts

**Deduplication:** Assistant messages appear multiple times per `requestId` due to streaming. Keep last occurrence per `req:{message.id}:{requestId}` (fall back to `session:{message.id}:{sessionId}`).

**Four token buckets** (not three — most tools get this wrong):
- `input_tokens` → base input price
- `cache_creation.ephemeral_5m_input_tokens` → 5-min cache write (1.25x input)
- `cache_creation.ephemeral_1h_input_tokens` → 1-hr cache write (2x input)
- `cache_read_input_tokens` → cache read (0.1x input)

**Project path:** Use `cwd` field from messages, not the encoded folder name.

**Duration:** `end_time - start_time` across all messages in the session (wall clock).

## Pricing

Source: https://platform.claude.com/docs/en/about-claude/pricing

| Model | Input | Cache 5m | Cache 1h | Cache Read | Output |
|---|---|---|---|---|---|
| claude-sonnet-4-6 | $3.00 | $3.75 | $6.00 | $0.30 | $15.00 |
| claude-opus-4-7 | $5.00 | $6.25 | $10.00 | $0.50 | $25.00 |
| claude-haiku-4-5 | $1.00 | $1.25 | $2.00 | $0.10 | $5.00 |

Unknown models fall back to Sonnet 4.6 pricing.

## Known issues

- Duration is wall-clock (start to end of session), not active API time.
- OpenCode sessions not included (different format, mostly free models).
