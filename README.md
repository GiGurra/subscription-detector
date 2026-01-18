# Subscription Detector

A CLI tool that analyzes bank transaction exports to detect recurring monthly subscriptions. It identifies active and stopped subscriptions, calculates monthly/yearly costs, and helps you understand your recurring expenses.

> **📚 Full documentation:** [gigurra.github.io/subscription-detector](https://gigurra.github.io/subscription-detector/)
>
> This README is a quick-start summary. See the docs for detailed configuration, the detection algorithm, and how to add custom parsers.

```
╭──────────────────┬──────────────────┬──────────────────────────┬─────────┬─────┬────────────┬────────────┬─────────╮
│ Name             │ Description      │ Tags                     │ Status  │ Day │ Last Seen  │ Monthly    │ Yearly  │
├──────────────────┼──────────────────┼──────────────────────────┼─────────┼─────┼────────────┼────────────┼─────────┤
│ Google Workspace │ Google Workspace │ work, productivity       │ ACTIVE  │ ~2  │ 2025-06-02 │   72-76 kr │  912 kr │
│ GYM MEMBERSHIP   │ Fitness Center   │ health                   │ STOPPED │ ~20 │ 2025-03-20 │     399 kr │       - │
│ NETFLIX.COM      │ Netflix          │ entertainment, streaming │ ACTIVE  │ ~5  │ 2025-06-05 │     199 kr │ 2388 kr │
│ Spotify          │                  │ entertainment, music     │ ACTIVE  │ ~12 │ 2025-06-12 │ 169-179 kr │ 2148 kr │
├──────────────────┼──────────────────┼──────────────────────────┼─────────┼─────┼────────────┼────────────┼─────────┤
│                  │                  │                          │         │     │            │ 454 KR     │ 5448 KR │
╰──────────────────┴──────────────────┴──────────────────────────┴─────────┴─────┴────────────┴────────────┴─────────╯
```

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
      --currency string      Currency code (e.g., USD, EUR, SEK)
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

# Display amounts in USD instead of auto-detected currency
./subscription-detector --currency USD simple-json:transactions.json

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

### Currency

Set the currency for amount formatting. If not specified, it's auto-detected from your system locale, falling back to SEK.

```yaml
currency: USD  # or EUR, GBP, SEK, NOK, DKK, CHF, JPY, CAD, AUD, BRL, etc.
```

Currency affects how amounts are displayed:
- **SEK/NOK/DKK**: `1 234 kr` (space separator, suffix)
- **USD/GBP/CAD/AUD**: `$1,234` (comma separator, prefix)
- **EUR**: `1.234 €` (period separator, suffix)
- **JPY**: `¥1,234` (comma separator, prefix)
- **BRL**: `1.234 R$` (period separator, suffix)

You can also override via CLI: `--currency USD`

#### Cross-Platform Locale Detection

The tool automatically detects your system currency based on locale:

| Platform | Detection Method |
|----------|------------------|
| **Linux/Unix** | `LC_MONETARY` → `LC_ALL` → `LANG` environment variables |
| **macOS** | Environment variables, then `AppleLocale` system preference |
| **Windows** | `GetUserDefaultLocaleName` API |

Any locale with a region code is supported (e.g., `pt_BR` → BRL, `ja_JP` → JPY, `de_DE` → EUR).

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
$ ./subscription-detector simple-json:examples/transactions.json --config examples/config.yaml --show all

Loaded 27 transactions from examples/transactions.json
Total: 27 transactions from 1 file(s)
Data range: 2025-01-02 to 2025-06-12
Complete months: 5

Found 4 subscriptions (3 active, 1 stopped)
Showing: all

╭──────────────────┬──────────────────┬──────────────────────────┬─────────┬─────┬────────────┬────────────────┬────────────┬─────────╮
│ Name             │ Description      │ Tags                     │ Status  │ Day │ Started    │ Last Seen      │ Monthly    │ Yearly  │
├──────────────────┼──────────────────┼──────────────────────────┼─────────┼─────┼────────────┼────────────────┼────────────┼─────────┤
│ Google Workspace │ Google Workspace │ work, productivity       │ ACTIVE  │ ~2  │ 2025-01-02 │ 2025-06-02     │   72-76 kr │  912 kr │
│ GYM MEMBERSHIP   │ Fitness Center   │ health                   │ STOPPED │ ~20 │ 2025-01-20 │ 2025-03-20     │     399 kr │       - │
│ NETFLIX.COM      │ Netflix          │ entertainment, streaming │ ACTIVE  │ ~5  │ 2025-01-05 │ 2025-06-05     │     199 kr │ 2388 kr │
│ Spotify          │                  │ entertainment, music     │ ACTIVE  │ ~12 │ 2025-01-12 │ 2025-06-12     │ 169-179 kr │ 2148 kr │
├──────────────────┼──────────────────┼──────────────────────────┼─────────┼─────┼────────────┼────────────────┼────────────┼─────────┤
│                  │                  │                          │         │     │            │ TOTAL (ACTIVE) │ 454 KR     │ 5448 KR │
╰──────────────────┴──────────────────┴──────────────────────────┴─────────┴─────┴────────────┴────────────────┴────────────┴─────────╯
```

## Development

This tool was hacked together in a couple of hours using [Claude Code](https://claude.com/product/claude-code) with Claude Opus 4.5. From initial idea to working CLI with tests, CI/CD, and documentation - all built through conversation.

## License

MIT
