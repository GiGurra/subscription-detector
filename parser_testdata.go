package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// TestDataFormat is a simple JSON format for testing
// Example:
//
//	{
//	  "transactions": [
//	    {"date": "2025-01-15", "text": "Netflix", "amount": -99.00},
//	    {"date": "2025-02-15", "text": "Netflix", "amount": -99.00}
//	  ]
//	}
type TestDataFormat struct {
	Transactions []TestDataTransaction `json:"transactions"`
}

type TestDataTransaction struct {
	Date   string  `json:"date"`
	Text   string  `json:"text"`
	Amount float64 `json:"amount"`
}

// ParseTestData parses a JSON file in the test data format
func ParseTestData(path string) ([]Transaction, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var testData TestDataFormat
	if err := json.Unmarshal(data, &testData); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	var transactions []Transaction
	for _, tx := range testData.Transactions {
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
	RegisterParser("testdata-json", ParserFunc(ParseTestData))
}
