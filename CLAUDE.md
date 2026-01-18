# Subscription Detector

CLI tool to detect recurring monthly subscriptions from bank transaction exports.

## Project Structure

```
├── main.go                  # CLI entry point (boa direct API), output formatting (go-pretty)
├── types.go                 # Common types: Transaction, Subscription, DateRange
├── detector.go              # Detection logic (bank-agnostic)
├── detector_test.go         # Tests for detection logic
├── parser_handelsbanken.go  # Handelsbanken XLSX parser (regular + credit card formats)
├── config.go                # YAML config: descriptions, groups, exclude rules
├── suggest.go               # Group suggestion algorithm (--suggest-groups)
```

## Build & Test

```bash
go build .
go test -v .
```

## Usage

```bash
# Basic detection (auto-loads config from ~/.subscription-detector/config.yaml)
./subscription-detector --source handelsbanken-xlsx transactions.xlsx

# Multiple files
./subscription-detector --source handelsbanken-xlsx account.xlsx creditcard.xlsx

# Show all including stopped
./subscription-detector --source handelsbanken-xlsx tx.xlsx --show all

# Custom tolerance (default 35%)
./subscription-detector --source handelsbanken-xlsx tx.xlsx --tolerance 0.10

# Suggest groups for transactions with varying names
./subscription-detector --source handelsbanken-xlsx tx.xlsx --suggest-groups

# Generate config template
./subscription-detector --source handelsbanken-xlsx tx.xlsx --init-config config.yaml
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

exclude:
  - "Tokyo Ramen"           # Simple pattern (always)
  - pattern: "A J Städ"     # Time-based exclusion
    before: "2026-01-01"
```

## Detection Algorithm

1. Parse transactions from bank export(s)
2. Load config, apply groups (rename matching transactions)
3. Filter to complete months only (incomplete current month excluded from pattern detection)
4. Group by payee name (case-insensitive)
5. Require 2+ occurrences, expenses only (negative amounts)
6. Check monthly pattern: exactly 1 payment per calendar month (across ALL data)
7. Check amount tolerance: configurable % between consecutive payments (default 35%)
8. Determine status: ACTIVE if payment in current month or within 5-day grace period, otherwise STOPPED

## Key Dependencies

- `github.com/GiGurra/boa` - CLI framework (declarative cobra wrapper, direct API)
- `github.com/jedib0t/go-pretty/v6` - Table rendering with colors
- `github.com/xuri/excelize/v2` - XLSX parsing
- `gopkg.in/yaml.v3` - Config file parsing

## Notes

- Handelsbanken truncates payee names to ~14-16 chars in their export
- Source data uses Swedish column names: Reskontradatum, Transaktionsdatum, Text, Belopp, Saldo
- Credit card exports have slightly different format (no Saldo column)
- Grouping patterns are regex (case-insensitive)
- Env var enrichment is disabled (clean CLI without env bindings)
