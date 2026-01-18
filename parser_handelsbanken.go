package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ParseHandelsbankenXLSX reads transactions from a Handelsbanken Excel export.
// Expected format:
// - Row 8: Headers (Reskontradatum, Transaktionsdatum, Text, Belopp, Saldo)
// - Row 9+: Transaction data
func ParseHandelsbankenXLSX(path string) ([]Transaction, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("reading sheet: %w", err)
	}

	var transactions []Transaction
	dataStarted := false

	for _, row := range rows {
		if len(row) >= 5 && row[0] == "Reskontradatum" {
			dataStarted = true
			continue
		}

		if !dataStarted || len(row) < 4 {
			continue
		}

		date, err := time.Parse("2006-01-02", row[0])
		if err != nil {
			continue
		}

		amountStr := strings.ReplaceAll(row[3], ",", ".")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			continue
		}

		transactions = append(transactions, Transaction{
			Date:   date,
			Text:   row[2],
			Amount: amount,
		})
	}

	return transactions, nil
}
