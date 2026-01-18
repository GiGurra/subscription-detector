# Examples

This directory contains example data and configuration for subscription-detector.

## Files

- `transactions.json` - Sample transaction data in simple-json format
- `config.yaml` - Example configuration with descriptions, tags, groups, and exclusions

## Running the Examples

### Basic usage (no config)

```bash
subscription-detector --source simple-json examples/transactions.json
```

Output:
```
Found 4 subscriptions (3 active, 1 stopped)
Showing: active

╭────────────────────────┬────────┬─────┬────────────┬────────────┬────────────┬─────────╮
│ Name                   │ Status │ Day │ Started    │ Last Seen  │ Monthly    │ Yearly  │
├────────────────────────┼────────┼─────┼────────────┼────────────┼────────────┼─────────┤
│ GOOGLE*GSUITE abc123   │ ACTIVE │ ~2  │ 2025-01-02 │ 2025-02-02 │      72 kr │  864 kr │
│ ...                    │        │     │            │            │            │         │
```

### With configuration

```bash
subscription-detector --source simple-json examples/transactions.json --config examples/config.yaml
```

The config will:
- Group Spotify transactions (varying IDs) into one subscription
- Group Google Workspace transactions (varying names) into one subscription
- Add descriptions and tags
- Exclude McDonald's and one-time purchases

### JSON output

```bash
subscription-detector --source simple-json examples/transactions.json --config examples/config.yaml --output json
```

### Filter by tags

```bash
# Show only entertainment subscriptions
subscription-detector --source simple-json examples/transactions.json --config examples/config.yaml --tags entertainment

# Show work-related subscriptions
subscription-detector --source simple-json examples/transactions.json --config examples/config.yaml --tags work
```

### Show all (including stopped)

```bash
subscription-detector --source simple-json examples/transactions.json --config examples/config.yaml --show all
```

### Suggest groups

Detect transactions that might need grouping:

```bash
subscription-detector --source simple-json examples/transactions.json --suggest-groups
```

### Sort by amount

```bash
subscription-detector --source simple-json examples/transactions.json --config examples/config.yaml --sort amount --sort-dir desc
```
