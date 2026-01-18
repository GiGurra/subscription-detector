package main

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/GiGurra/boa/pkg/boa"
)

type Params struct {
	Source string `descr:"Data source type" alts:"handelsbanken-xlsx" strict:"true"`
	File   string `descr:"Path to the transaction file" positional:"true"`
}

func main() {
	boa.NewCmdT[Params]("subscription-detector").
		WithShort("Detect ongoing subscriptions from bank transactions").
		WithLong("Analyzes bank transaction data to identify recurring monthly subscriptions based on similar amounts (±10%) and recurring payee names.").
		WithRunFunc(func(params *Params) {
			transactions, err := ParseHandelsbankenXLSX(params.File)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Loaded %d transactions\n", len(transactions))

			// Check data coverage
			completeMonths, dateRange := AnalyzeDataCoverage(transactions)
			fmt.Printf("Data range: %s to %s\n", dateRange.Start.Format("2006-01-02"), dateRange.End.Format("2006-01-02"))
			fmt.Printf("Complete months: %d\n\n", len(completeMonths))

			if len(completeMonths) < 3 {
				fmt.Fprintf(os.Stderr, "Warning: Less than 3 complete months of data. Subscription detection may be unreliable.\n\n")
			}

			// Filter to only complete months for pattern detection
			filtered := FilterToCompleteMonths(transactions, completeMonths)
			subscriptions := DetectSubscriptions(filtered, transactions, dateRange)

			if len(subscriptions) == 0 {
				fmt.Println("No subscriptions detected.")
				return
			}

			printSubscriptionSummary(subscriptions)
		}).
		Run()
}

func printSubscriptionSummary(subscriptions []Subscription) {
	activeCount := 0
	stoppedCount := 0
	for _, sub := range subscriptions {
		if sub.Status == StatusActive {
			activeCount++
		} else {
			stoppedCount++
		}
	}

	fmt.Printf("Found %d subscriptions (%d active, %d stopped)\n\n", len(subscriptions), activeCount, stoppedCount)

	// Print header
	fmt.Printf("%-25s  %-8s  %-12s  %-12s  %-15s  %s\n",
		"NAME", "STATUS", "STARTED", "LAST SEEN", "AMOUNT", "OCCURRENCES")
	fmt.Println(strings.Repeat("-", 95))

	for _, sub := range subscriptions {
		status := "ACTIVE"
		if sub.Status == StatusStopped {
			status = "STOPPED"
		}

		amountStr := fmt.Sprintf("%.0f kr", math.Abs(sub.AvgAmount))
		if sub.MinAmount != sub.MaxAmount {
			amountStr = fmt.Sprintf("%.0f-%.0f kr", sub.MinAmount, sub.MaxAmount)
		}

		fmt.Printf("%-25s  %-8s  %-12s  %-12s  %-15s  %d\n",
			truncate(sub.Name, 25),
			status,
			sub.StartDate.Format("2006-01-02"),
			sub.LastDate.Format("2006-01-02"),
			amountStr,
			len(sub.Transactions))
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-2] + ".."
}
