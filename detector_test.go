package main

import (
	"testing"
	"time"
)

func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestFilterExpenses(t *testing.T) {
	txs := []Transaction{
		{Date: date("2025-01-15"), Text: "Expense", Amount: -100},
		{Date: date("2025-01-16"), Text: "Income", Amount: 500},
		{Date: date("2025-01-17"), Text: "Expense2", Amount: -50},
	}

	expenses := FilterExpenses(txs)

	if len(expenses) != 2 {
		t.Errorf("expected 2 expenses, got %d", len(expenses))
	}
	if expenses[0].Amount != -100 || expenses[1].Amount != -50 {
		t.Errorf("unexpected expense amounts")
	}
}

func TestIsMonthlyPattern(t *testing.T) {
	tests := []struct {
		name     string
		txs      []Transaction
		expected bool
	}{
		{
			name: "valid monthly pattern",
			txs: []Transaction{
				{Date: date("2025-01-15"), Amount: -100},
				{Date: date("2025-02-15"), Amount: -100},
				{Date: date("2025-03-15"), Amount: -100},
			},
			expected: true,
		},
		{
			name: "two payments in same month",
			txs: []Transaction{
				{Date: date("2025-01-15"), Amount: -100},
				{Date: date("2025-01-20"), Amount: -100},
				{Date: date("2025-02-15"), Amount: -100},
			},
			expected: false,
		},
		{
			name: "single payment",
			txs: []Transaction{
				{Date: date("2025-01-15"), Amount: -100},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMonthlyPattern(tt.txs)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAmountsWithinTolerance(t *testing.T) {
	tests := []struct {
		name      string
		txs       []Transaction
		tolerance float64
		expected  bool
	}{
		{
			name: "identical amounts",
			txs: []Transaction{
				{Amount: -100},
				{Amount: -100},
				{Amount: -100},
			},
			tolerance: 0.10,
			expected:  true,
		},
		{
			name: "within 10% tolerance",
			txs: []Transaction{
				{Amount: -100},
				{Amount: -105},
				{Amount: -95},
			},
			tolerance: 0.10,
			expected:  true,
		},
		{
			name: "outside 10% tolerance - consecutive diff",
			txs: []Transaction{
				{Amount: -100},
				{Amount: -115}, // 15% diff from previous
			},
			tolerance: 0.10,
			expected:  false,
		},
		{
			name: "gradual drift within tolerance",
			txs: []Transaction{
				{Amount: -100},
				{Amount: -105}, // 5% diff
				{Amount: -110}, // 4.7% diff
				{Amount: -115}, // 4.5% diff - each step ok, total drift 15%
			},
			tolerance: 0.10,
			expected:  true,
		},
		{
			name:      "empty list",
			txs:       []Transaction{},
			tolerance: 0.10,
			expected:  false,
		},
		{
			name: "single transaction",
			txs: []Transaction{
				{Amount: -100},
			},
			tolerance: 0.10,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AmountsWithinTolerance(tt.txs, tt.tolerance)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCalculateAverageAmount(t *testing.T) {
	txs := []Transaction{
		{Amount: -100},
		{Amount: -200},
		{Amount: -300},
	}

	avg := CalculateAverageAmount(txs)
	if avg != -200 {
		t.Errorf("expected -200, got %f", avg)
	}

	// Empty list
	avg = CalculateAverageAmount([]Transaction{})
	if avg != 0 {
		t.Errorf("expected 0 for empty list, got %f", avg)
	}
}

func TestCalculateAmountRange(t *testing.T) {
	txs := []Transaction{
		{Amount: -150},
		{Amount: -100},
		{Amount: -200},
	}

	min, max := CalculateAmountRange(txs)
	if min != 100 || max != 200 {
		t.Errorf("expected min=100, max=200, got min=%f, max=%f", min, max)
	}
}

func TestCalculateTypicalDay(t *testing.T) {
	txs := []Transaction{
		{Date: date("2025-01-10")},
		{Date: date("2025-02-12")},
		{Date: date("2025-03-14")},
	}

	day := CalculateTypicalDay(txs)
	if day != 12 { // (10 + 12 + 14) / 3 = 12
		t.Errorf("expected 12, got %d", day)
	}
}

func TestDetermineStatus(t *testing.T) {
	tests := []struct {
		name        string
		lastPayment time.Time
		typicalDay  int
		dataEndDate time.Time
		expected    SubscriptionStatus
	}{
		{
			name:        "payment in current month - active",
			lastPayment: date("2025-03-15"),
			typicalDay:  15,
			dataEndDate: date("2025-03-20"),
			expected:    StatusActive,
		},
		{
			name:        "last month, within grace period - active",
			lastPayment: date("2025-02-15"),
			typicalDay:  15,
			dataEndDate: date("2025-03-18"), // 3 days after expected
			expected:    StatusActive,
		},
		{
			name:        "last month, past grace period - stopped",
			lastPayment: date("2025-02-15"),
			typicalDay:  15,
			dataEndDate: date("2025-03-25"), // 10 days after expected
			expected:    StatusStopped,
		},
		{
			name:        "two months ago - stopped",
			lastPayment: date("2025-01-15"),
			typicalDay:  15,
			dataEndDate: date("2025-03-10"),
			expected:    StatusStopped,
		},
		{
			name:        "typical day past end of month",
			lastPayment: date("2025-01-31"),
			typicalDay:  31,
			dataEndDate: date("2025-03-05"), // Feb doesn't have 31 days
			expected:    StatusStopped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineStatus(tt.lastPayment, tt.typicalDay, tt.dataEndDate)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAnalyzeDataCoverage(t *testing.T) {
	tests := []struct {
		name                   string
		txs                    []Transaction
		expectedCompleteMonths int
		expectedStartDate      string
		expectedEndDate        string
	}{
		{
			name: "four months, last incomplete",
			txs: []Transaction{
				{Date: date("2025-09-15")},
				{Date: date("2025-10-15")},
				{Date: date("2025-11-15")},
				{Date: date("2025-12-15")},
				{Date: date("2026-01-10")}, // incomplete month
			},
			expectedCompleteMonths: 4, // Sep, Oct, Nov, Dec
			expectedStartDate:      "2025-09-15",
			expectedEndDate:        "2026-01-10",
		},
		{
			name: "month ends on last day - complete",
			txs: []Transaction{
				{Date: date("2025-01-15")},
				{Date: date("2025-01-31")}, // last day of Jan
			},
			expectedCompleteMonths: 1,
			expectedStartDate:      "2025-01-15",
			expectedEndDate:        "2025-01-31",
		},
		{
			name:                   "empty transactions",
			txs:                    []Transaction{},
			expectedCompleteMonths: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			months, dateRange := AnalyzeDataCoverage(tt.txs)
			if len(months) != tt.expectedCompleteMonths {
				t.Errorf("expected %d complete months, got %d: %v", tt.expectedCompleteMonths, len(months), months)
			}
			if tt.expectedStartDate != "" && dateRange.Start.Format("2006-01-02") != tt.expectedStartDate {
				t.Errorf("expected start %s, got %s", tt.expectedStartDate, dateRange.Start.Format("2006-01-02"))
			}
			if tt.expectedEndDate != "" && dateRange.End.Format("2006-01-02") != tt.expectedEndDate {
				t.Errorf("expected end %s, got %s", tt.expectedEndDate, dateRange.End.Format("2006-01-02"))
			}
		})
	}
}

func TestFilterToCompleteMonths(t *testing.T) {
	txs := []Transaction{
		{Date: date("2025-01-15"), Text: "Jan"},
		{Date: date("2025-02-15"), Text: "Feb"},
		{Date: date("2025-03-15"), Text: "Mar"},
	}

	filtered := FilterToCompleteMonths(txs, []string{"2025-01", "2025-03"})

	if len(filtered) != 2 {
		t.Errorf("expected 2 transactions, got %d", len(filtered))
	}
	if filtered[0].Text != "Jan" || filtered[1].Text != "Mar" {
		t.Errorf("unexpected filtered transactions")
	}
}

func TestDetectSubscriptions(t *testing.T) {
	// Create test data for a subscription: Netflix with monthly payments
	allTxs := []Transaction{
		{Date: date("2025-01-15"), Text: "Netflix", Amount: -99},
		{Date: date("2025-02-15"), Text: "Netflix", Amount: -99},
		{Date: date("2025-03-15"), Text: "Netflix", Amount: -99},
		{Date: date("2025-04-10"), Text: "Netflix", Amount: -99}, // current month
		// Non-subscription: one-time purchase
		{Date: date("2025-02-20"), Text: "Amazon", Amount: -500},
		// Non-subscription: varying amounts
		{Date: date("2025-01-10"), Text: "Grocery", Amount: -150},
		{Date: date("2025-02-12"), Text: "Grocery", Amount: -300},
		{Date: date("2025-03-08"), Text: "Grocery", Amount: -200},
	}

	// Complete months: Jan, Feb, Mar (April is current/incomplete)
	filteredTxs := FilterToCompleteMonths(allTxs, []string{"2025-01", "2025-02", "2025-03"})
	dateRange := DateRange{Start: date("2025-01-10"), End: date("2025-04-10")}

	subs := DetectSubscriptions(filteredTxs, allTxs, dateRange, 0.10)

	if len(subs) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(subs))
	}

	netflix := subs[0]
	if netflix.Name != "Netflix" {
		t.Errorf("expected Netflix, got %s", netflix.Name)
	}
	if netflix.Status != StatusActive {
		t.Errorf("expected active status, got %s", netflix.Status)
	}
	if netflix.AvgAmount != -99 {
		t.Errorf("expected avg -99, got %f", netflix.AvgAmount)
	}
	if len(netflix.Transactions) != 4 {
		t.Errorf("expected 4 transactions (including current month), got %d", len(netflix.Transactions))
	}
}

func TestDetectSubscriptions_Stopped(t *testing.T) {
	// Subscription that stopped
	allTxs := []Transaction{
		{Date: date("2025-01-15"), Text: "Spotify", Amount: -59},
		{Date: date("2025-02-15"), Text: "Spotify", Amount: -59},
		// Stopped after Feb - no March or April payments
		{Date: date("2025-04-20"), Text: "Other", Amount: -10}, // just to set date range
	}

	filteredTxs := FilterToCompleteMonths(allTxs, []string{"2025-01", "2025-02", "2025-03"})
	dateRange := DateRange{Start: date("2025-01-15"), End: date("2025-04-20")}

	subs := DetectSubscriptions(filteredTxs, allTxs, dateRange, 0.10)

	if len(subs) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(subs))
	}

	spotify := subs[0]
	if spotify.Status != StatusStopped {
		t.Errorf("expected stopped status, got %s", spotify.Status)
	}
	if spotify.LastDate.Format("2006-01-02") != "2025-02-15" {
		t.Errorf("expected last date 2025-02-15, got %s", spotify.LastDate.Format("2006-01-02"))
	}
}
