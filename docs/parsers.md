# Adding Custom Parsers

The subscription detector supports multiple bank export formats through a parser interface. You can add support for new formats by implementing a simple interface.

## Built-in Parsers

| Format | Description |
|--------|-------------|
| `handelsbanken-xlsx` | Handelsbanken bank export (XLSX) |
| `simple-json` | Simple JSON format |

## Parser Interface

```go
type Parser interface {
    Parse(filePath string) ([]Transaction, error)
}

type Transaction struct {
    Date   time.Time
    Text   string
    Amount float64
}
```

## Adding a New Parser

### 1. Create Parser File

Create a new file `internal/parser_mybank.go`:

```go
package internal

import (
    "encoding/csv"
    "os"
    "strconv"
    "time"
)

type MyBankParser struct{}

func (p *MyBankParser) Parse(filePath string) ([]Transaction, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    reader := csv.NewReader(file)
    records, err := reader.ReadAll()
    if err != nil {
        return nil, err
    }

    var transactions []Transaction
    for i, record := range records {
        if i == 0 {
            continue // Skip header
        }

        date, _ := time.Parse("2006-01-02", record[0])
        amount, _ := strconv.ParseFloat(record[2], 64)

        transactions = append(transactions, Transaction{
            Date:   date,
            Text:   record[1],
            Amount: amount,
        })
    }

    return transactions, nil
}
```

### 2. Register the Parser

Add your parser to `internal/parser.go`:

```go
var parsers = map[string]Parser{
    "handelsbanken-xlsx": &HandelsbankenParser{},
    "simple-json":        &SimpleJSONParser{},
    "mybank-csv":         &MyBankParser{},  // Add this line
}
```

### 3. Use Your Parser

```bash
./subscription-detector mybank-csv:export.csv
```

## Simple JSON Format

The `simple-json` format is useful for testing or converting from other formats:

```json
{
  "transactions": [
    {"date": "2025-01-15", "text": "Netflix", "amount": -99.00},
    {"date": "2025-02-15", "text": "Netflix", "amount": -99.00},
    {"date": "2025-03-15", "text": "Netflix", "amount": -99.00}
  ]
}
```

## Handelsbanken Format

The Handelsbanken parser handles their XLSX export format with Swedish column names:

- `Reskontradatum` - Booking date
- `Transaktionsdatum` - Transaction date
- `Text` - Description/payee
- `Belopp` - Amount
- `Saldo` - Balance (optional, for credit cards)

Both regular account and credit card exports are supported.
