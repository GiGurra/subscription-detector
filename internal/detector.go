package internal

import (
	"math"
	"sort"
	"strings"
	"time"
)

// DetectSubscriptions analyzes transactions to find recurring monthly subscriptions.
// It uses filteredTxs (from complete months) for pattern detection,
// and allTxs to determine the full lifecycle including current month.
// tolerance is the max allowed price change between consecutive months (e.g., 0.35 = 35%).
func DetectSubscriptions(filteredTxs []Transaction, allTxs []Transaction, dateRange DateRange, tolerance float64) []Subscription {
	// Group filtered transactions by payee name (case-insensitive)
	byName := make(map[string][]Transaction)
	displayNames := make(map[string]string) // lowercase -> display name (most recent)
	for _, tx := range filteredTxs {
		key := strings.ToLower(tx.Text)
		byName[key] = append(byName[key], tx)
		displayNames[key] = tx.Text // keeps updating to most recent
	}

	// Also group all transactions to check latest month
	allByName := make(map[string][]Transaction)
	for _, tx := range allTxs {
		key := strings.ToLower(tx.Text)
		allByName[key] = append(allByName[key], tx)
		displayNames[key] = tx.Text
	}

	var subscriptions []Subscription

	for key, txs := range byName {
		name := displayNames[key]
		// Need at least 2 occurrences to be a subscription
		if len(txs) < 2 {
			continue
		}

		// Only consider expenses (negative amounts)
		expenses := FilterExpenses(txs)
		if len(expenses) < 2 {
			continue
		}

		// Sort by date
		sort.Slice(expenses, func(i, j int) bool {
			return expenses[i].Date.Before(expenses[j].Date)
		})

		// Get all transactions for this subscription (including current month)
		allExpenses := FilterExpenses(allByName[key])
		sort.Slice(allExpenses, func(i, j int) bool {
			return allExpenses[i].Date.Before(allExpenses[j].Date)
		})

		// Check for monthly pattern using ALL transactions
		// If there are ever 2+ payments in any month, it's not a subscription
		if !IsMonthlyPattern(allExpenses) {
			continue
		}

		// Check if amounts are within tolerance of each other (using complete months data)
		if !AmountsWithinTolerance(expenses, tolerance) {
			continue
		}

		// Calculate statistics
		avgAmount := CalculateAverageAmount(expenses)
		minAmount, maxAmount := CalculateAmountRange(expenses)
		typicalDay := CalculateTypicalDay(expenses)

		startDate := allExpenses[0].Date
		lastDate := allExpenses[len(allExpenses)-1].Date
		latestAmount := allExpenses[len(allExpenses)-1].Amount

		// Determine status
		status := DetermineStatus(lastDate, typicalDay, dateRange.End)

		subscriptions = append(subscriptions, Subscription{
			Name:         name,
			AvgAmount:    avgAmount,
			LatestAmount: latestAmount,
			MinAmount:    minAmount,
			MaxAmount:    maxAmount,
			Transactions: allExpenses,
			StartDate:    startDate,
			LastDate:     lastDate,
			TypicalDay:   typicalDay,
			Status:       status,
		})
	}

	// Sort: active first, then by amount (highest first)
	sort.Slice(subscriptions, func(i, j int) bool {
		if subscriptions[i].Status != subscriptions[j].Status {
			return subscriptions[i].Status == StatusActive
		}
		return math.Abs(subscriptions[i].AvgAmount) > math.Abs(subscriptions[j].AvgAmount)
	})

	return subscriptions
}

// FilterExpenses returns only transactions with negative amounts (expenses).
func FilterExpenses(txs []Transaction) []Transaction {
	var expenses []Transaction
	for _, tx := range txs {
		if tx.Amount < 0 {
			expenses = append(expenses, tx)
		}
	}
	return expenses
}

// IsMonthlyPattern checks if transactions occur exactly once per calendar month.
func IsMonthlyPattern(txs []Transaction) bool {
	// Group by year-month
	byMonth := make(map[string]int)
	for _, tx := range txs {
		key := tx.Date.Format("2006-01")
		byMonth[key]++
	}

	// Each month should have exactly 1 payment
	for _, count := range byMonth {
		if count != 1 {
			return false
		}
	}

	return true
}

// AmountsWithinTolerance checks if consecutive amounts are within the given tolerance.
// This handles currency fluctuations better than comparing to an average.
func AmountsWithinTolerance(txs []Transaction, tolerance float64) bool {
	if len(txs) < 2 {
		return len(txs) == 1 // single transaction is valid
	}

	for i := 1; i < len(txs); i++ {
		prev := math.Abs(txs[i-1].Amount)
		curr := math.Abs(txs[i].Amount)
		diff := math.Abs(curr-prev) / prev
		if diff > tolerance {
			return false
		}
	}
	return true
}

// CalculateAverageAmount returns the average amount across all transactions.
func CalculateAverageAmount(txs []Transaction) float64 {
	if len(txs) == 0 {
		return 0
	}
	sum := 0.0
	for _, tx := range txs {
		sum += tx.Amount
	}
	return sum / float64(len(txs))
}

// CalculateAmountRange returns the min and max absolute amounts.
func CalculateAmountRange(txs []Transaction) (min, max float64) {
	if len(txs) == 0 {
		return 0, 0
	}
	min = math.Abs(txs[0].Amount)
	max = math.Abs(txs[0].Amount)
	for _, tx := range txs[1:] {
		amt := math.Abs(tx.Amount)
		if amt < min {
			min = amt
		}
		if amt > max {
			max = amt
		}
	}
	return min, max
}

// CalculateTypicalDay returns the average day of month for payments.
func CalculateTypicalDay(txs []Transaction) int {
	if len(txs) == 0 {
		return 0
	}
	sum := 0
	for _, tx := range txs {
		sum += tx.Date.Day()
	}
	return sum / len(txs)
}

// DetermineStatus checks if a subscription is active or stopped based on payment history.
func DetermineStatus(lastPayment time.Time, typicalDay int, dataEndDate time.Time) SubscriptionStatus {
	// Calculate how many months since last payment
	lastPaymentStart := time.Date(lastPayment.Year(), lastPayment.Month(), 1, 0, 0, 0, 0, time.UTC)
	currentMonthStart := time.Date(dataEndDate.Year(), dataEndDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	// If last payment is in current month - active
	if lastPaymentStart.Equal(currentMonthStart) {
		return StatusActive
	}

	// Calculate months difference
	monthsDiff := (currentMonthStart.Year()-lastPaymentStart.Year())*12 + int(currentMonthStart.Month()-lastPaymentStart.Month())

	// If more than 1 month has passed completely, it's stopped
	if monthsDiff > 1 {
		return StatusStopped
	}

	// Last payment was last month - check if we're past expected date + 5 days
	expectedDay := typicalDay
	lastDayOfMonth := time.Date(dataEndDate.Year(), dataEndDate.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if expectedDay > lastDayOfMonth {
		expectedDay = lastDayOfMonth
	}

	expectedDate := time.Date(dataEndDate.Year(), dataEndDate.Month(), expectedDay, 0, 0, 0, 0, time.UTC)
	gracePeriodEnd := expectedDate.AddDate(0, 0, 5)

	if dataEndDate.After(gracePeriodEnd) {
		return StatusStopped
	}

	// Still within grace period - consider active
	return StatusActive
}

// AnalyzeDataCoverage returns complete months and the date range of transactions.
func AnalyzeDataCoverage(transactions []Transaction) ([]string, DateRange) {
	if len(transactions) == 0 {
		return nil, DateRange{}
	}

	// Find date range
	minDate := transactions[0].Date
	maxDate := transactions[0].Date
	for _, tx := range transactions {
		if tx.Date.Before(minDate) {
			minDate = tx.Date
		}
		if tx.Date.After(maxDate) {
			maxDate = tx.Date
		}
	}

	// Determine complete months
	// A month is complete if:
	// - It's a past month (not the month of maxDate), OR
	// - maxDate is the last day of that month
	var completeMonths []string

	// Iterate through all months in range
	current := time.Date(minDate.Year(), minDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	endMonth := time.Date(maxDate.Year(), maxDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	for !current.After(endMonth) {
		monthKey := current.Format("2006-01")
		lastDayOfMonth := current.AddDate(0, 1, -1).Day()

		if current.Before(endMonth) {
			// Past month - complete
			completeMonths = append(completeMonths, monthKey)
		} else if maxDate.Day() == lastDayOfMonth {
			// Current month but includes last day
			completeMonths = append(completeMonths, monthKey)
		}

		current = current.AddDate(0, 1, 0)
	}

	return completeMonths, DateRange{Start: minDate, End: maxDate}
}

// FilterToCompleteMonths returns only transactions from complete months.
func FilterToCompleteMonths(transactions []Transaction, completeMonths []string) []Transaction {
	monthSet := make(map[string]bool)
	for _, m := range completeMonths {
		monthSet[m] = true
	}

	var filtered []Transaction
	for _, tx := range transactions {
		if monthSet[tx.Date.Format("2006-01")] {
			filtered = append(filtered, tx)
		}
	}
	return filtered
}

// FilterOutMatched returns transactions whose text (case-insensitive) is not in the matched set.
func FilterOutMatched(transactions []Transaction, matchedTexts map[string]bool) []Transaction {
	if len(matchedTexts) == 0 {
		return transactions
	}

	var filtered []Transaction
	for _, tx := range transactions {
		if !matchedTexts[strings.ToLower(tx.Text)] {
			filtered = append(filtered, tx)
		}
	}
	return filtered
}

// DetectKnownSubscriptions finds subscriptions based on configured known patterns.
// Unlike regular detection, these can match even with a single occurrence and
// include transactions from the current (incomplete) month.
// Returns known subscriptions and the set of transaction texts that matched (to exclude from regular detection).
func DetectKnownSubscriptions(allTxs []Transaction, dateRange DateRange, cfg *Config) ([]Subscription, map[string]bool) {
	matchedTexts := make(map[string]bool) // tracks which transaction texts matched known patterns

	if cfg == nil || len(cfg.Known) == 0 {
		return nil, matchedTexts
	}

	// Group matching transactions by the known subscription pattern
	type matchGroup struct {
		pattern string
		txs     []Transaction
	}
	byPattern := make(map[string]*matchGroup)

	for _, tx := range allTxs {
		// Only consider expenses
		if tx.Amount >= 0 {
			continue
		}

		known := cfg.MatchesKnown(tx)
		if known == nil {
			continue
		}

		// Mark this text as matched (case-insensitive key)
		matchedTexts[strings.ToLower(tx.Text)] = true

		if byPattern[known.Pattern] == nil {
			byPattern[known.Pattern] = &matchGroup{pattern: known.Pattern}
		}
		byPattern[known.Pattern].txs = append(byPattern[known.Pattern].txs, tx)
	}

	var subscriptions []Subscription
	for _, group := range byPattern {
		if len(group.txs) == 0 {
			continue
		}

		// Sort by date
		sort.Slice(group.txs, func(i, j int) bool {
			return group.txs[i].Date.Before(group.txs[j].Date)
		})

		// Use the most recent transaction text as the display name
		name := group.txs[len(group.txs)-1].Text

		// Calculate statistics
		avgAmount := CalculateAverageAmount(group.txs)
		minAmount, maxAmount := CalculateAmountRange(group.txs)
		typicalDay := CalculateTypicalDay(group.txs)

		startDate := group.txs[0].Date
		lastDate := group.txs[len(group.txs)-1].Date
		latestAmount := group.txs[len(group.txs)-1].Amount

		// Determine status
		status := DetermineStatus(lastDate, typicalDay, dateRange.End)

		subscriptions = append(subscriptions, Subscription{
			Name:         name,
			AvgAmount:    avgAmount,
			LatestAmount: latestAmount,
			MinAmount:    minAmount,
			MaxAmount:    maxAmount,
			Transactions: group.txs,
			StartDate:    startDate,
			LastDate:     lastDate,
			TypicalDay:   typicalDay,
			Status:       status,
		})
	}

	// Sort: active first, then by amount
	sort.Slice(subscriptions, func(i, j int) bool {
		if subscriptions[i].Status != subscriptions[j].Status {
			return subscriptions[i].Status == StatusActive
		}
		return math.Abs(subscriptions[i].AvgAmount) > math.Abs(subscriptions[j].AvgAmount)
	})

	return subscriptions, matchedTexts
}
