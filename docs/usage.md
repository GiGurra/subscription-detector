# Usage

## Basic Usage

```bash
# Using format:path prefix syntax
./subscription-detector handelsbanken-xlsx:transactions.xlsx

# Using --source flag
./subscription-detector --source handelsbanken-xlsx transactions.xlsx

# Multiple files with same format
./subscription-detector --source handelsbanken-xlsx account.xlsx creditcard.xlsx

# Mix different formats
./subscription-detector handelsbanken-xlsx:bank.xlsx simple-json:other.json
```

## Output Options

### Show Filter

```bash
# Show only active subscriptions (default)
./subscription-detector --source simple-json data.json --show active

# Show only stopped subscriptions
./subscription-detector --source simple-json data.json --show stopped

# Show all subscriptions
./subscription-detector --source simple-json data.json --show all
```

### Sorting

```bash
# Sort by name (default)
./subscription-detector --source simple-json data.json --sort name

# Sort by amount
./subscription-detector --source simple-json data.json --sort amount --sort-dir desc

# Sort by description
./subscription-detector --source simple-json data.json --sort description
```

### Output Format

```bash
# Table output (default)
./subscription-detector --source simple-json data.json --output table

# JSON output
./subscription-detector --source simple-json data.json --output json
```

## Detection Tuning

### Tolerance

The tolerance controls how much price variation is allowed between consecutive payments. Default is 35%.

```bash
# Strict tolerance (10%) - reject subscriptions with >10% price changes
./subscription-detector --source simple-json data.json --tolerance 0.10

# Relaxed tolerance (50%)
./subscription-detector --source simple-json data.json --tolerance 0.50
```

### Tag Filtering

Filter subscriptions by tags defined in your config:

```bash
./subscription-detector --source simple-json data.json --tags entertainment
./subscription-detector --source simple-json data.json --tags entertainment,streaming
```

## Config File

By default, the tool loads config from `~/.subscription-detector/config.yaml` if it exists.

```bash
# Use a specific config file
./subscription-detector --config myconfig.yaml --source simple-json data.json

# Generate a config template from detected subscriptions
./subscription-detector --source simple-json data.json --init-config config.yaml
```

## Group Suggestions

Analyze transactions and suggest grouping patterns:

```bash
./subscription-detector --source simple-json data.json --suggest-groups
```

This helps identify transactions with varying names that should be grouped together (e.g., "GOOGLE*GSUITE", "Google GSUITE_", "Google Workspa" → "Google Workspace").
