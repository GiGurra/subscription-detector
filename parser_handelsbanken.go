package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ParseHandelsbankenXLSX reads transactions from a Handelsbanken Excel export.
// Supports two layouts:
// - Regular account: Reskontradatum, Transaktionsdatum, Text, Belopp, Saldo
// - Credit card: Reskontradatum, Transaktionsdatum, Text, Belopp (no Saldo, may have empty first column)
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

	// Find header row and column indices
	var dateCol, textCol, amountCol int = -1, -1, -1
	var dataStartRow int = -1

	for i, row := range rows {
		for j, cell := range row {
			cell = strings.TrimSpace(cell)
			switch cell {
			case "Reskontradatum":
				dateCol = j
				dataStartRow = i + 1
			case "Transaktionsdatum":
				// Use transaction date if available, prefer over Reskontradatum
				if dateCol == -1 || j > dateCol {
					// Keep Reskontradatum as date column
				}
			case "Text":
				textCol = j
			case "Belopp":
				amountCol = j
			}
		}
		if dateCol >= 0 && textCol >= 0 && amountCol >= 0 {
			break
		}
	}

	if dateCol < 0 || textCol < 0 || amountCol < 0 {
		return nil, fmt.Errorf("could not find required columns (Reskontradatum, Text, Belopp)")
	}

	var transactions []Transaction
	for i := dataStartRow; i < len(rows); i++ {
		row := rows[i]

		// Ensure row has enough columns
		maxCol := max(dateCol, textCol, amountCol)
		if len(row) <= maxCol {
			continue
		}

		dateStr := strings.TrimSpace(row[dateCol])
		text := strings.TrimSpace(row[textCol])
		amountStr := strings.TrimSpace(row[amountCol])

		// Skip empty rows
		if dateStr == "" || text == "" || amountStr == "" {
			continue
		}

		// Parse date
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// Parse amount
		amountStr = strings.ReplaceAll(amountStr, ",", ".")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			continue
		}

		// Strip "Prel " prefix from pending transactions
		text = strings.TrimPrefix(text, "Prel ")

		transactions = append(transactions, Transaction{
			Date:   date,
			Text:   text,
			Amount: amount,
		})
	}

	return transactions, nil
}
