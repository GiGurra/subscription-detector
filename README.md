# Subscription Detector

A CLI tool that analyzes bank transaction exports to detect recurring monthly subscriptions. It identifies active and stopped subscriptions, calculates monthly/yearly costs, and helps you understand your recurring expenses.

## Features

- **Automatic Detection**: Identifies subscriptions based on recurring monthly payments with similar amounts
- **Multiple Accounts**: Combine transactions from multiple bank export files
- **Smart Grouping**: Group transactions with varying names (e.g., "Spotify P3E460", "Spotify P3D49A") into a single subscription
- **Time-based Exclusions**: Exclude transactions only within specific date ranges
- **Configurable Tolerance**: Adjust how much price variation is allowed between payments (default: 35%)
- **Status Tracking**: Shows which subscriptions are ACTIVE vs STOPPED
- **Cost Summary**: Displays monthly and yearly costs with totals

## Installation

```bash
go install github.com/gigurra/subscription-detector@latest
```

Or build from source:

```bash
git clone https://github.com/gigurra/subscription-detector
cd subscription-detector
go build .
```

## Usage

### Basic Usage

```bash
# Analyze a single transaction file
./subscription-detector --source handelsbanken-xlsx transactions.xlsx

# Analyze multiple files (e.g., checking account + credit card)
./subscription-detector --source handelsbanken-xlsx account.xlsx creditcard.xlsx
```

### Options

```
Flags:
  -s, --source string        Data source type (required)
  -c, --config string        Path to config file (YAML)
  -i, --init-config string   Generate config template and save to path
      --show string          Which subscriptions to show: active, stopped, all (default "active")
  -t, --tolerance float      Max price change between months, e.g., 0.35 = 35% (default 0.35)
      --suggest-groups       Analyze and suggest potential transaction groups
  -h, --help                 help for subscription-detector
```

### Examples

```bash
# Show all subscriptions including stopped ones
./subscription-detector --source handelsbanken-xlsx tx.xlsx --show all

# Use stricter tolerance (10% max price change)
./subscription-detector --source handelsbanken-xlsx tx.xlsx --tolerance 0.10

# Generate a config template from detected subscriptions
./subscription-detector --source handelsbanken-xlsx tx.xlsx --init-config config.yaml

# Find potential groupings for transactions with varying names
./subscription-detector --source handelsbanken-xlsx tx.xlsx --suggest-groups
```

## Configuration

The tool automatically loads config from `~/.subscription-detector/config.yaml` if it exists.

### Config File Format

```yaml
# Custom descriptions for subscriptions
descriptions:
  NETFLIX.COM: "Netflix"
  GOOGLE *YouTub: "YouTube Premium"
  K*svd.se: "Svenska Dagbladet"

# Group transactions with different names into one subscription
groups:
  - name: "Google Workspace"
    patterns:
      - "GOOGLE\\*GSUITE"
      - "Google GSUITE_"
      - "Google Workspa"
  - name: "Spotify"
    patterns:
      - "^Spotify"

# Exclude transactions from detection
exclude:
  # Simple patterns (always excluded)
  - "Tokyo Ramen"
  - "McDonald"

  # Time-based exclusions
  - pattern: "A J Städ"
    before: "2026-01-01"  # Only exclude before this date
```

### Grouping

Some services append transaction IDs or change their billing name over time. Use groups to combine them:

```yaml
groups:
  - name: "Spotify"
    patterns:
      - "^Spotify"  # Matches "Spotify P3E460", "Spotify P3D49A", etc.
```

The `--suggest-groups` flag can help identify these automatically:

```bash
./subscription-detector --source handelsbanken-xlsx tx.xlsx --suggest-groups
```

Output:
```
Found 1 potential group(s):

  "Spotify" (13 months, 13 transactions)
    Names: Spotify P3A8AC, Spotify P3B5D9, Spotify P34103
           ... and 10 more

    Add to config:
      - name: "Spotify"
        patterns:
          - "^Spotify"
```

### Time-based Exclusions

Exclude transactions only within a specific time period:

```yaml
exclude:
  # Exclude only transactions before 2026
  - pattern: "A J Städ"
    before: "2026-01-01"

  # Exclude only transactions after a date
  - pattern: "Old Service"
    after: "2025-06-01"
```

## Detection Algorithm

1. **Parse**: Read transactions from bank export files
2. **Group**: Combine transactions by payee name (case-insensitive), applying custom groups from config
3. **Filter**: Keep only expenses (negative amounts) with 2+ occurrences
4. **Pattern Check**: Verify exactly 1 payment per calendar month
5. **Amount Check**: Ensure consecutive payments are within tolerance (default 35%)
6. **Status**: Mark as ACTIVE if paid in current month or within 5-day grace period, otherwise STOPPED

## Supported Banks

Currently supported:
- **Handelsbanken** (Sweden) - XLSX export format

The parser handles both regular accounts and credit card exports which have slightly different column layouts.

## Output Example

```
Found 12 subscriptions (12 active, 0 stopped)

╭──────────────────┬────────────────────┬────────┬─────┬────────────┬────────────────┬────────────┬──────────╮
│ Name             │ Description        │ Status │ Day │ Started    │ Last Seen      │ Monthly    │ Yearly   │
├──────────────────┼────────────────────┼────────┼─────┼────────────┼────────────────┼────────────┼──────────┤
│ NETFLIX.COM      │ Netflix            │ ACTIVE │ ~4  │ 2025-01-07 │ 2026-01-05     │     199 kr │  2388 kr │
│ Spotify          │                    │ ACTIVE │ ~14 │ 2025-01-15 │ 2026-01-12     │ 169-219 kr │  2198 kr │
│ Google Workspace │ Google Workspace   │ ACTIVE │ ~2  │ 2025-01-03 │ 2026-01-02     │   64-76 kr │   807 kr │
├──────────────────┼────────────────────┼────────┼─────┼────────────┼────────────────┼────────────┼──────────┤
│                  │                    │        │     │            │ TOTAL (ACTIVE) │   4637 KR  │ 55641 KR │
╰──────────────────┴────────────────────┴────────┴─────┴────────────┴────────────────┴────────────┴──────────╯
```

## License

MIT
