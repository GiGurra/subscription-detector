# Examples

This directory contains example data and configuration for subscription-detector.

## Files

- `transactions.json` - Sample transaction data in simple-json format
- `config.yaml` - Example configuration with descriptions, tags, groups, and exclusions

## Running the Examples

### Basic usage (no config)

```bash
subscription-detector --source simple-json examples/transactions.json --config /dev/null
```

Output:
```
Found 4 subscriptions (2 active, 2 stopped)
Showing: active

╭──────────────────┬────────┬─────┬────────────┬────────────────┬─────────┬─────────╮
│ Name             │ Status │ Day │ Started    │ Last Seen      │ Monthly │ Yearly  │
├──────────────────┼────────┼─────┼────────────┼────────────────┼─────────┼─────────┤
│ Google Workspace │ ACTIVE │ ~2  │ 2025-03-02 │ 2025-06-02     │   76 kr │  912 kr │
│ NETFLIX.COM      │ ACTIVE │ ~5  │ 2025-01-05 │ 2025-06-05     │  199 kr │ 2388 kr │
├──────────────────┼────────┼─────┼────────────┼────────────────┼─────────┼─────────┤
│                  │        │     │            │ TOTAL (ACTIVE) │ 275 KR  │ 3300 KR │
╰──────────────────┴────────┴─────┴────────────┴────────────────┴─────────┴─────────╯
```

Note: Without config, Spotify isn't detected (varying transaction IDs), and the early Google transactions with different names aren't grouped.

### With configuration

The example config (`examples/config.yaml`):

```yaml
descriptions:
  NETFLIX.COM: "Netflix"
  GYM MEMBERSHIP: "Fitness Center"
  Google Workspace: "Google Workspace"

tags:
  NETFLIX.COM: [entertainment, streaming]
  Spotify: [entertainment, music]
  Google Workspace: [work, productivity]
  GYM MEMBERSHIP: [health]

groups:
  - name: "Spotify"
    patterns:
      - "^Spotify"  # Matches "Spotify P3A8AC", "Spotify P3B5D9", etc.
  - name: "Google Workspace"
    patterns:
      - "GOOGLE\\*GSUITE"
      - "Google GSUITE_"
      - "Google Workspace"

exclude:
  - "McDonald's"
  - "Flowers Shop"
  - "Birthday Gift"
```

Run with config:

```bash
subscription-detector --source simple-json examples/transactions.json --config examples/config.yaml
```

Output:
```
Found 4 subscriptions (3 active, 1 stopped)
Showing: active

╭──────────────────┬──────────────────┬──────────────────────────┬────────┬─────┬────────────┬────────────────┬────────────┬─────────╮
│ Name             │ Description      │ Tags                     │ Status │ Day │ Started    │ Last Seen      │ Monthly    │ Yearly  │
├──────────────────┼──────────────────┼──────────────────────────┼────────┼─────┼────────────┼────────────────┼────────────┼─────────┤
│ Google Workspace │ Google Workspace │ work, productivity       │ ACTIVE │ ~2  │ 2025-01-02 │ 2025-06-02     │   72-76 kr │  912 kr │
│ NETFLIX.COM      │ Netflix          │ entertainment, streaming │ ACTIVE │ ~5  │ 2025-01-05 │ 2025-06-05     │     199 kr │ 2388 kr │
│ Spotify          │                  │ entertainment, music     │ ACTIVE │ ~12 │ 2025-01-12 │ 2025-06-12     │ 169-179 kr │ 2148 kr │
├──────────────────┼──────────────────┼──────────────────────────┼────────┼─────┼────────────┼────────────────┼────────────┼─────────┤
│                  │                  │                          │        │     │            │ TOTAL (ACTIVE) │ 454 KR     │ 5448 KR │
╰──────────────────┴──────────────────┴──────────────────────────┴────────┴─────┴────────────┴────────────────┴────────────┴─────────╯
```

The config:
- Groups Spotify transactions (varying IDs) into one subscription
- Groups Google Workspace transactions (varying names) into one
- Adds descriptions and tags
- Excludes McDonald's and one-time purchases

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

### Currency options

```bash
# Auto-detect from system locale (default)
subscription-detector simple-json:examples/transactions.json

# Override with specific currency
subscription-detector --currency USD simple-json:examples/transactions.json
subscription-detector --currency EUR simple-json:examples/transactions.json
subscription-detector --currency BRL simple-json:examples/transactions.json
```

## Testing on Windows

On Windows, clone the repo and build:

```powershell
git clone https://github.com/GiGurra/subscription-detector
cd subscription-detector
go build .
```

Run with examples:

```powershell
# Auto-detects currency from Windows locale settings
.\subscription-detector.exe simple-json:examples/transactions.json

# Override currency
.\subscription-detector.exe --currency USD simple-json:examples/transactions.json

# With config
.\subscription-detector.exe --config examples/config.yaml simple-json:examples/transactions.json
```

The tool automatically detects your Windows locale via `GetUserDefaultLocaleName` API and sets the appropriate currency. For example:
- US English locale → USD ($1,234)
- Swedish locale → SEK (1 234 kr)
- German locale → EUR (1.234 €)
- Brazilian Portuguese locale → BRL (1.234 R$)
