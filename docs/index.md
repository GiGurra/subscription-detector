# Subscription Detector

A CLI tool to detect recurring monthly subscriptions from bank transaction exports.

## Features

- **Automatic detection** - Analyzes transaction patterns to find recurring monthly payments
- **Multiple bank formats** - Supports Handelsbanken XLSX and simple JSON (extensible)
- **Built-in known subscriptions** - 70+ common services (Netflix, Spotify, etc.) detected immediately
- **Smart grouping** - Combine transactions with varying names into single subscriptions
- **Configurable** - YAML config for descriptions, tags, exclusions, and custom patterns
- **Active/Stopped status** - Tracks which subscriptions are still active

## Quick Example

```bash
$ subscription-detector --source handelsbanken-xlsx account.xlsx creditcard.xlsx --show all

Loaded 115 transactions from account.xlsx
Loaded 688 transactions from creditcard.xlsx
Total: 803 transactions from 2 file(s)
Loaded config from /home/user/.subscription-detector/config.yaml
Data range: 2025-01-02 to 2026-01-15
Complete months: 12

Found 17 subscriptions (12 active, 5 stopped)
Showing: all

+------------------+----------------------+---------------+---------+-----+------------+----------------+------------+----------+
| Name             | Description          | Tags          | Status  | Day | Started    | Last Seen      | Monthly    | Yearly   |
+------------------+----------------------+---------------+---------+-----+------------+----------------+------------+----------+
| NETFLIX.COM      | Netflix              | entertainment | ACTIVE  | ~15 | 2025-01-15 | 2026-01-15     |     229 kr |  2748 kr |
| Spotify          | Spotify Family       | music         | ACTIVE  | ~1  | 2025-01-01 | 2026-01-01     | 189-199 kr |  2388 kr |
| Google Workspace |                      | work          | ACTIVE  | ~2  | 2025-01-02 | 2026-01-02     |   72-76 kr |   912 kr |
| GYM MEMBERSHIP   | Fitness Center       | health        | STOPPED | ~20 | 2025-01-20 | 2025-09-20     |     399 kr |        - |
| ...              |                      |               |         |     |            |                |            |          |
+------------------+----------------------+---------------+---------+-----+------------+----------------+------------+----------+
|                  |                      |               |         |     |            | TOTAL (ACTIVE) |   1852 KR  | 22228 KR |
+------------------+----------------------+---------------+---------+-----+------------+----------------+------------+----------+
```

## How It Works

1. Parse transactions from bank export files
2. Apply grouping rules (combine similar transaction names)
3. Detect known subscriptions immediately (Netflix, Spotify, etc.)
4. Analyze remaining transactions for monthly patterns
5. Calculate statistics and determine active/stopped status

## Development

This tool was hacked together in a couple of hours using [Claude Code](https://claude.ai/claude-code) with Claude Opus 4.5. From initial idea to working CLI with tests, CI/CD, and documentation - all built through conversation.

## Getting Started

See [Installation](installation.md) to get started, or jump to [Usage](usage.md) if you already have it installed.
