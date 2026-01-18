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
# Detect subscriptions from a bank export
./subscription-detector handelsbanken-xlsx:transactions.xlsx

# Output
┌──────────────────┬─────────────┬────────────┬────────┐
│ NAME             │ DESCRIPTION │ AMOUNT/MO  │ STATUS │
├──────────────────┼─────────────┼────────────┼────────┤
│ Netflix          │             │  99.00 kr  │ ACTIVE │
│ Spotify          │             │ 129.00 kr  │ ACTIVE │
│ Google Workspace │             │ 115.00 kr  │ ACTIVE │
└──────────────────┴─────────────┴────────────┴────────┘
```

## How It Works

1. Parse transactions from bank export files
2. Apply grouping rules (combine similar transaction names)
3. Detect known subscriptions immediately (Netflix, Spotify, etc.)
4. Analyze remaining transactions for monthly patterns
5. Calculate statistics and determine active/stopped status

## Getting Started

See [Installation](installation.md) to get started, or jump to [Usage](usage.md) if you already have it installed.
