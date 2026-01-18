# Installation

## From Source

Requires Go 1.21 or later.

```bash
# Clone the repository
git clone https://github.com/GiGurra/subscription-detector.git
cd subscription-detector

# Build
go build .

# Run
./subscription-detector --help
```

## Using Go Install

```bash
go install github.com/GiGurra/subscription-detector@latest
```

## Verify Installation

```bash
subscription-detector --help
```

You should see:

```
Analyzes bank transaction data to identify recurring monthly subscriptions

Usage:
  subscription-detector [flags]

Flags:
      --config string      Path to config file (YAML)
  -h, --help              help for subscription-detector
      --init-config string Generate config template and save to path
      --output string      Output format (default "table")
      --show string        Which subscriptions to show (default "active")
      --sort string        Sort field for output (default "name")
      --sort-dir string    Sort direction (default "asc")
      --source string      Default format
      --suggest-groups     Analyze and suggest potential transaction groups
      --tags strings       Filter by tags
      --tolerance float    Max price change between months (default 0.35)
```
