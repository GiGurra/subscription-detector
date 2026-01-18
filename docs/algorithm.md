# Detection Algorithm

This page explains how the subscription detector identifies recurring payments from your transaction data.

## Overview

The detection process has two main paths:

1. **Known subscriptions** - Matched immediately by pattern (Netflix, Spotify, etc.)
2. **Pattern detection** - Analyzed for recurring monthly patterns

```
Transactions
     │
     ├─► Known subscription patterns ─► Immediate match
     │
     └─► Pattern detection ─► Monthly analysis ─► Tolerance check
```

## Step-by-Step Process

### 1. Parse Transactions

Transactions are loaded from bank export files. Each transaction has:

- **Date** - When the payment occurred
- **Text** - Payee/merchant name
- **Amount** - Payment amount (negative = expense)

### 2. Apply Grouping Rules

Before detection, grouping rules from your config are applied. This combines transactions with varying names into a single subscription:

```yaml
groups:
  - name: "Google Workspace"
    patterns:
      - "GOOGLE\\*GSUITE"
      - "Google GSUITE_"
      - "Google Workspa"
```

All matching transactions are renamed to "Google Workspace" before detection.

### 3. Detect Known Subscriptions

The tool has 70+ built-in patterns for common services. These are matched **immediately** and **bypass the normal detection rules**:

| Rule | Pattern Detection | Known Subscriptions |
|------|-------------------|---------------------|
| Minimum occurrences | 2+ required | **1 is enough** |
| Complete months only | Yes | **No - includes current month** |
| Tolerance check | Yes | No |
| Monthly pattern check | Yes | No |

**Built-in patterns include:**

- Video: Netflix, Disney+, HBO Max, Amazon Prime, etc.
- Music: Spotify, Apple Music, Tidal, YouTube Music, etc.
- Gaming: Xbox Game Pass, PlayStation Plus, etc.
- Productivity: Adobe, Microsoft 365, Dropbox, etc.
- And many more...

This means if you just subscribed to Netflix today, it will be detected immediately - no need to wait for 2+ months of history.

You can add your own patterns or disable defaults:

```yaml
use_default_known: false  # Disable built-in patterns

known:
  - pattern: "MyCustomService"
```

### 4. Analyze Data Coverage

The tool determines which months have complete data:

- **Complete month**: A past month, or current month if today is the last day
- **Incomplete month**: The current month (still accumulating transactions)

**Why exclude incomplete months from pattern detection?**

If today is January 15th and your subscription usually charges on the 20th, the algorithm shouldn't conclude "no payment this month" - the payment just hasn't happened yet. By excluding the current month, we avoid false negatives.

However, incomplete months **are** included when:

- Checking for known subscriptions (they match immediately)
- Determining if a subscription is ACTIVE (payment in current month = active)
- Calculating the latest payment amount

### 5. Pattern Detection

For remaining transactions, the algorithm looks for recurring patterns:

#### Grouping
Transactions are grouped by payee name (case-insensitive).

#### Minimum Occurrences
A group needs **2+ occurrences** to be considered a subscription.

#### Expenses Only
Only negative amounts (expenses) are analyzed. Income is ignored.

#### Monthly Pattern Check
Each calendar month should have **exactly 1 payment**:

```
✓ Valid pattern:
  Jan: 1 payment
  Feb: 1 payment
  Mar: 1 payment

✗ Invalid (multiple per month):
  Jan: 2 payments
  Feb: 1 payment
```

!!! note
    Gaps are allowed - a subscription doesn't need to appear every month, but when it does appear, it should be once per month.

#### Tolerance Check

Consecutive payments must be within the tolerance threshold (default: 35%):

```
Payment 1: 100 kr
Payment 2: 110 kr  → 10% change ✓ (within 35%)
Payment 3: 150 kr  → 36% change ✗ (exceeds 35%)
```

This catches subscriptions with minor price changes while filtering out variable expenses like groceries.

You can adjust the tolerance:

```bash
# Strict: only allow 10% variation
./subscription-detector --tolerance 0.10 ...

# Relaxed: allow 50% variation
./subscription-detector --tolerance 0.50 ...
```

### 6. Determine Status

Each subscription is marked as **ACTIVE** or **STOPPED**:

| Condition | Status |
|-----------|--------|
| Payment in current month | ACTIVE |
| Within 5-day grace period after expected date | ACTIVE |
| Last payment was 2+ months ago | STOPPED |
| Past grace period, no current month payment | STOPPED |

The **typical day** is calculated as the average day-of-month across all payments. This is used to determine the grace period.

### 7. Apply Exclusions

Finally, exclusion rules from your config are applied:

```yaml
exclude:
  - "Tokyo Ramen"           # Always exclude
  - pattern: "Old Service"
    before: "2025-01-01"    # Only exclude before this date
```

## Summary Statistics

After detection, the tool calculates:

| Statistic | Description |
|-----------|-------------|
| Average amount | Mean of all payments |
| Latest amount | Most recent payment (used for totals) |
| Min/Max amount | Price range across all payments |
| Typical day | Average day-of-month |
| Start date | First payment date |
| Last date | Most recent payment date |

## Example

Given these transactions:

```
2025-01-15  Netflix   -99 kr
2025-02-15  Netflix   -99 kr
2025-03-15  Netflix   -99 kr
2025-01-10  Grocery   -250 kr
2025-02-22  Grocery   -180 kr
2025-03-05  Grocery   -320 kr
```

**Netflix**: ✓ Detected as subscription

- Monthly pattern: 1 per month ✓
- Tolerance: 0% variation ✓

**Grocery**: ✗ Not detected

- Monthly pattern: 1 per month ✓
- Tolerance: 38% variation between 180→250 ✗
