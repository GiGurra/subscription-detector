# Configuration

Configuration is stored in YAML format. Default location: `~/.subscription-detector/config.yaml`

## Full Example

```yaml
# Custom descriptions for subscriptions
descriptions:
  "NETFLIX.COM": "Netflix - Family Plan"
  "K*svd.se": "Svenska Dagbladet"

# Tags for categorization and filtering
tags:
  "Netflix": ["entertainment", "streaming"]
  "Spotify": ["entertainment", "music"]
  "Google Workspace": ["productivity", "work"]

# Group transactions with varying names
groups:
  - name: "Google Workspace"
    patterns:
      - "GOOGLE\\*GSUITE"
      - "Google GSUITE_"
      - "Google Workspa"
  - name: "Spotify"
    patterns:
      - "^Spotify"
    tolerance: 0.50  # Custom tolerance for this group

# Disable built-in known subscriptions (Netflix, Spotify, etc.)
use_default_known: false

# Known subscriptions - detected immediately (even with 1 occurrence)
known:
  - pattern: "MyCustomService"
  - pattern: "PremiumApp"
    min_amount: 49
    max_amount: 99
  - pattern: "OldService"
    after: "2024-01-01"
    before: "2025-06-01"

# Exclude patterns from detection
exclude:
  - "Tokyo Ramen"           # Simple pattern
  - pattern: "A J Städ"     # With time bounds
    before: "2026-01-01"

# Currency for amount formatting (auto-detected from locale if not set)
currency: USD
```

## Sections

### descriptions

Map subscription names to human-readable descriptions:

```yaml
descriptions:
  "NETFLIX.COM": "Netflix - Family Plan"
  "SPOTIFY": "Spotify Premium"
```

### tags

Assign tags for categorization and filtering:

```yaml
tags:
  "Netflix": ["entertainment", "streaming"]
  "Spotify": ["entertainment", "music"]
```

Use with `--tags` flag: `./subscription-detector --tags entertainment`

### groups

Combine transactions with different names into a single subscription:

```yaml
groups:
  - name: "Google Workspace"
    patterns:
      - "GOOGLE\\*GSUITE"
      - "Google GSUITE_"
      - "Google Workspa"
    tolerance: 0.50  # Optional: custom tolerance for this group
```

Patterns are regex (case-insensitive).

### use_default_known

Controls whether built-in known subscription patterns are used. Default: `true`

```yaml
use_default_known: false  # Disable all built-in patterns
```

Built-in patterns include 70+ common services: Netflix, Spotify, Disney+, HBO Max, YouTube, GitHub, Adobe, Dropbox, and many more.

### known

Define patterns that are immediately detected as subscriptions, even with just 1 occurrence:

```yaml
known:
  # Simple pattern
  - pattern: "MyService"

  # With amount range
  - pattern: "PremiumApp"
    min_amount: 49
    max_amount: 99

  # With date filters
  - pattern: "OldService"
    after: "2024-01-01"
    before: "2025-06-01"
```

Options:

| Field | Description |
|-------|-------------|
| `pattern` | Regex pattern (case-insensitive) |
| `min_amount` | Minimum amount (absolute value) |
| `max_amount` | Maximum amount (absolute value) |
| `before` | Only match before this date (YYYY-MM-DD) |
| `after` | Only match after this date (YYYY-MM-DD) |

### exclude

Exclude patterns from subscription detection:

```yaml
exclude:
  # Always exclude
  - "Tokyo Ramen"

  # Exclude with time bounds
  - pattern: "A J Städ"
    before: "2026-01-01"  # Only exclude before this date
```

### currency

Set the currency code for amount formatting:

```yaml
currency: USD
```

If not specified, the currency is auto-detected from system locale environment variables (LC_MONETARY, LC_ALL, LANG), falling back to USD.

**Supported currencies:**

| Code | Symbol | Format Example |
|------|--------|----------------|
| SEK  | kr     | 1 234 kr       |
| USD  | $      | $1,234         |
| EUR  | €      | 1.234 €        |
| GBP  | £      | £1,234         |
| NOK  | kr     | 1 234 kr       |
| DKK  | kr     | 1 234 kr       |
| CHF  | CHF    | 1.234 CHF      |
| JPY  | ¥      | ¥1,234         |
| CAD  | $      | $1,234         |
| AUD  | $      | $1,234         |

You can also override via CLI: `--currency EUR`
