# Subscription Detector

CLI tool to detect recurring monthly subscriptions from bank transaction exports.

## Project Structure

```
├── main.go                  # CLI entry point (boa), output formatting (go-pretty)
├── types.go                 # Common types: Transaction, Subscription, DateRange
├── detector.go              # Detection logic (bank-agnostic)
├── detector_test.go         # Tests for detection logic
├── parser_handelsbanken.go  # Handelsbanken XLSX parser
├── config.go                # YAML config: descriptions, exclude rules
```

## Build & Test

```bash
go build .
go test -v .
```

## Usage

```bash
# Basic detection
./subscription-detector --source handelsbanken-xlsx transactions.xlsx

# With config file
./subscription-detector --source handelsbanken-xlsx transactions.xlsx --config config.yaml

# Generate config template
./subscription-detector --source handelsbanken-xlsx transactions.xlsx --init-config config.yaml
```

## Config File Format (YAML)

```yaml
descriptions:
  "K*svd.se": "Svenska Dagbladet"
  "COMVIQ": "Mobil - Natalie"

exclude:
  - "^SBC$"           # Exact match
  - "Överf Mobil"     # Partial match (internal transfers)
```

## Detection Algorithm

1. Parse transactions from bank export
2. Filter to complete months only (incomplete current month excluded)
3. Group by payee name (Text field)
4. Require 2+ occurrences, expenses only (negative amounts)
5. Check monthly pattern: exactly 1 payment per calendar month
6. Check amount tolerance: ±10% between consecutive payments (handles currency fluctuations)
7. Determine status: ACTIVE if payment in current month or within 5-day grace period, otherwise STOPPED

## Key Dependencies

- `github.com/GiGurra/boa` - CLI framework (declarative cobra wrapper)
- `github.com/jedib0t/go-pretty/v6` - Table rendering with colors
- `github.com/xuri/excelize/v2` - XLSX parsing
- `gopkg.in/yaml.v3` - Config file parsing

## Notes

- Handelsbanken truncates payee names to ~14-16 chars in their export
- Source data uses Swedish column names: Reskontradatum, Transaktionsdatum, Text, Belopp, Saldo
