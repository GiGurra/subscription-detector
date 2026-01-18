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
# Analyze a single transaction file (format prefix syntax)
./subscription-detector handelsbanken-xlsx:transactions.xlsx

# Or use --source flag
./subscription-detector --source handelsbanken-xlsx transactions.xlsx

# Mix different formats in one command
./subscription-detector handelsbanken-xlsx:bank.xlsx simple-json:other.json

# Multiple files with same format
./subscription-detector --source handelsbanken-xlsx account.xlsx creditcard.xlsx
```

### Options

```
Flags:
  -s, --source string        Default format (or use format:path syntax)
  -c, --config string        Path to config file (YAML)
  -i, --init-config string   Generate config template and save to path
      --show string          Which subscriptions to show: active, stopped, all (default "active")
      --sort string          Sort field: name, description, amount (default "name")
      --sort-dir string      Sort direction: asc, desc (default "asc")
      --tags strings         Filter by tags (e.g., entertainment, insurance)
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

# Tags for categorizing subscriptions
tags:
  NETFLIX.COM: [entertainment]
  GOOGLE *YouTub: [entertainment]
  Spotify: [entertainment]
  K*svd.se: [news]

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

### Tags

Categorize subscriptions with tags and filter by them:

```yaml
tags:
  NETFLIX.COM: [entertainment]
  Spotify: [entertainment, music]
  K*svd.se: [news]
```

Filter by tags:

```bash
# Show only entertainment subscriptions
./subscription-detector --source handelsbanken-xlsx tx.xlsx --tags entertainment

# Show multiple tag categories
./subscription-detector --source handelsbanken-xlsx tx.xlsx --tags entertainment --tags insurance
```

Tags are displayed in a dedicated column when any subscription has tags configured.

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

## Supported Formats

| Source | Description |
|--------|-------------|
| `handelsbanken-xlsx` | Handelsbanken (Sweden) XLSX export. Supports both regular accounts and credit cards. |
| `simple-json` | Simple JSON format, easy to convert to from any source. |

### Simple JSON Format

The `simple-json` format is useful for importing from custom sources or converted data:

```json
{
  "transactions": [
    {"date": "2025-01-15", "text": "Netflix", "amount": -99.00},
    {"date": "2025-02-15", "text": "Netflix", "amount": -99.00}
  ]
}
```

Usage:
```bash
./subscription-detector simple-json:transactions.json
```

### Adding a Custom Format

To add support for a new bank or data source, create a parser file in `internal/`:

```go
// internal/parser_mybank.go
package internal

import (
    "encoding/csv"
    "os"
    "strconv"
    "time"
)

func ParseMyBank(path string) ([]Transaction, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    reader := csv.NewReader(f)
    records, err := reader.ReadAll()
    if err != nil {
        return nil, err
    }

    var transactions []Transaction
    for _, row := range records[1:] { // skip header
        date, _ := time.Parse("2006-01-02", row[0])
        amount, _ := strconv.ParseFloat(row[2], 64)
        transactions = append(transactions, Transaction{
            Date:   date,
            Text:   row[1],  // payee/description
            Amount: amount,  // negative for expenses
        })
    }
    return transactions, nil
}

func init() {
    RegisterParser("mybank-csv", ParserFunc(ParseMyBank))
}
```

The `Transaction` struct requires:
- `Date` - transaction date (`time.Time`)
- `Text` - payee name or description (`string`)
- `Amount` - transaction amount, negative for expenses (`float64`)

After adding the file, rebuild and use. Since the file is in the `internal` package (already imported by `main.go`), the `init()` function runs automatically - no additional imports needed.

```bash
go build .
./subscription-detector mybank-csv:export.csv
```

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
