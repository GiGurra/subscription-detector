# Subscription Detector

CLI tool to detect recurring monthly subscriptions from bank transaction exports.

## Project Structure

```
├── main.go                           # CLI entry point (boa direct API)
├── internal/
│   ├── types.go                      # Common types: Transaction, Subscription, DateRange
│   ├── detector.go                   # Detection logic (bank-agnostic)
│   ├── detector_test.go              # Tests for detection logic
│   ├── parser.go                     # Parser registry and format:path parsing
│   ├── parser_handelsbanken.go       # Handelsbanken XLSX parser
│   ├── parser_simple_json.go         # Simple JSON parser
│   ├── config.go                     # YAML config: descriptions, groups, known, exclude
│   ├── currency.go                   # Currency formatting with locale support (x/text)
│   ├── suggest.go                    # Group suggestion algorithm (--suggest-groups)
│   └── output.go                     # Output formatting (table, JSON)
```

## Build & Test

```bash
go build .
go test -v .
```

## Usage

```bash
# Basic detection with format prefix (auto-loads config from ~/.subscription-detector/config.yaml)
./subscription-detector handelsbanken-xlsx:transactions.xlsx

# Or use --source flag for all files
./subscription-detector --source handelsbanken-xlsx transactions.xlsx

# Mix different formats in one command
./subscription-detector handelsbanken-xlsx:bank.xlsx simple-json:other.json

# Multiple files with same format
./subscription-detector --source handelsbanken-xlsx account.xlsx creditcard.xlsx

# Show all including stopped
./subscription-detector --source handelsbanken-xlsx tx.xlsx --show all

# Custom tolerance (default 35%)
./subscription-detector --source handelsbanken-xlsx tx.xlsx --tolerance 0.10

# Suggest groups for transactions with varying names
./subscription-detector --source handelsbanken-xlsx tx.xlsx --suggest-groups

# Generate config template
./subscription-detector --source handelsbanken-xlsx tx.xlsx --init-config config.yaml

# Use a specific currency (overrides locale detection)
./subscription-detector --currency USD simple-json:transactions.json
```

## Config File Format (YAML)

Default location: `~/.subscription-detector/config.yaml`

```yaml
descriptions:
  "NETFLIX.COM": "Netflix"
  "K*svd.se": "Svenska Dagbladet"

groups:
  - name: "Google Workspace"
    patterns:
      - "GOOGLE\\*GSUITE"
      - "Google GSUITE_"
      - "Google Workspa"
  - name: "Spotify"
    patterns:
      - "^Spotify"

# Disable built-in known subscription patterns (Netflix, Spotify, etc.)
# Default: true (built-in patterns are included)
use_default_known: false

# Known subscriptions are detected immediately (even with 1 occurrence, including current month)
# These are added to the built-in defaults (unless use_default_known: false)
known:
  - pattern: "NewStreamingService"     # Just a name pattern
  - pattern: "PremiumApp"              # With optional amount range
    min_amount: 49
    max_amount: 99
  - pattern: "OldService"              # With date filters
    after: "2024-01-01"
    before: "2025-06-01"

exclude:
  - "Tokyo Ramen"           # Simple pattern (always)
  - pattern: "A J Städ"     # Time-based exclusion
    before: "2026-01-01"

# Currency code for formatting (auto-detected from system locale if not set)
# Supported: SEK, USD, EUR, GBP, NOK, DKK, CHF, JPY, CAD, AUD
currency: SEK
```

## Detection Algorithm

1. Parse transactions from bank export(s)
2. Load config, apply groups (rename matching transactions)
3. **Detect known subscriptions first** (from `known` config - matches even with 1 occurrence, includes current month)
4. Filter remaining transactions to complete months only (incomplete current month excluded from pattern detection)
5. Group by payee name (case-insensitive)
6. Require 2+ occurrences, expenses only (negative amounts)
7. Check monthly pattern: exactly 1 payment per calendar month (across ALL data)
8. Check amount tolerance: configurable % between consecutive payments (default 35%)
9. Determine status: ACTIVE if payment in current month or within 5-day grace period, otherwise STOPPED

## Key Dependencies

- `github.com/GiGurra/boa` - CLI framework (declarative cobra wrapper, direct API)
- `github.com/jedib0t/go-pretty/v6` - Table rendering with colors
- `github.com/xuri/excelize/v2` - XLSX parsing
- `gopkg.in/yaml.v3` - Config file parsing
- `golang.org/x/text` - Locale-aware currency formatting

## Notes

- **Built-in known subscriptions**: Includes 70+ common services (Netflix, Spotify, Disney+, HBO Max, YouTube, GitHub, Adobe, etc.). Disable with `use_default_known: false` in config.
- Handelsbanken truncates payee names to ~14-16 chars in their export
- Source data uses Swedish column names: Reskontradatum, Transaktionsdatum, Text, Belopp, Saldo
- Credit card exports have slightly different format (no Saldo column)
- Grouping patterns are regex (case-insensitive)
- Env var enrichment is disabled (clean CLI without env bindings)
