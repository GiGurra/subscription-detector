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
$ ./subscription-detector handelsbanken-xlsx:transactions.xlsx --show all

Loaded 27 transactions from transactions.xlsx
Data range: 2025-01-02 to 2025-06-12
Complete months: 5

Found 4 subscriptions (3 active, 1 stopped)

+------------------+-------------+---------------+---------+-----+------------+------------+---------+
| Name             | Description | Tags          | Status  | Day | Last Seen  | Monthly    | Yearly  |
+------------------+-------------+---------------+---------+-----+------------+------------+---------+
| Google Workspace |             | work          | ACTIVE  | ~2  | 2025-06-02 |  72-76 kr  |  912 kr |
| GYM MEMBERSHIP   |             | health        | STOPPED | ~20 | 2025-03-20 |    399 kr  |       - |
| NETFLIX.COM      | Netflix     | entertainment | ACTIVE  | ~5  | 2025-06-05 |    199 kr  | 2388 kr |
| Spotify          |             | music         | ACTIVE  | ~12 | 2025-06-12 | 169-179 kr | 2148 kr |
+------------------+-------------+---------------+---------+-----+------------+------------+---------+
|                  |             |               |         |     |      TOTAL |    454 KR  | 5448 KR |
+------------------+-------------+---------------+---------+-----+------------+------------+---------+
```

## How It Works

1. Parse transactions from bank export files
2. Apply grouping rules (combine similar transaction names)
3. Detect known subscriptions immediately (Netflix, Spotify, etc.)
4. Analyze remaining transactions for monthly patterns
5. Calculate statistics and determine active/stopped status

## Getting Started

See [Installation](installation.md) to get started, or jump to [Usage](usage.md) if you already have it installed.
