package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// SimpleJSONFormat is a minimal JSON format for importing transactions
// Example:
//
//	{
//	  "transactions": [
//	    {"date": "2025-01-15", "text": "Netflix", "amount": -99.00},
//	    {"date": "2025-02-15", "text": "Netflix", "amount": -99.00}
//	  ]
//	}
//
// This format is easy to convert to from any bank export or data source.
type SimpleJSONFormat struct {
	Transactions []SimpleJSONTransaction `json:"transactions"`
}

type SimpleJSONTransaction struct {
	Date   string  `json:"date"`   // YYYY-MM-DD format
	Text   string  `json:"text"`   // Payee/description
	Amount float64 `json:"amount"` // Negative for expenses
}

// ParseSimpleJSON parses a JSON file in the simple JSON format
func ParseSimpleJSON(path string) ([]Transaction, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var jsonData SimpleJSONFormat
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	var transactions []Transaction
	for _, tx := range jsonData.Transactions {
		date, err := time.Parse("2006-01-02", tx.Date)
		if err != nil {
			return nil, fmt.Errorf("parsing date %q: %w", tx.Date, err)
		}
		transactions = append(transactions, Transaction{
			Date:   date,
			Text:   tx.Text,
			Amount: tx.Amount,
		})
	}

	return transactions, nil
}

func init() {
	RegisterParser("simple-json", ParserFunc(ParseSimpleJSON))
}
