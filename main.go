package main

import (
	"fmt"
	"math"
	"os"

	"github.com/GiGurra/boa/pkg/boa"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
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
	var totalMonthlyCost float64

	for _, sub := range subscriptions {
		if sub.Status == StatusActive {
			activeCount++
			totalMonthlyCost += math.Abs(sub.AvgAmount)
		} else {
			stoppedCount++
		}
	}
	totalYearlyCost := totalMonthlyCost * 12

	fmt.Printf("Found %d subscriptions (%d active, %d stopped)\n\n",
		len(subscriptions), activeCount, stoppedCount)

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Name", "Status", "Day", "Started", "Last Seen", "Monthly", "Yearly"})

	for _, sub := range subscriptions {
		status := text.FgGreen.Sprint("ACTIVE")
		if sub.Status == StatusStopped {
			status = text.FgRed.Sprint("STOPPED")
		}

		monthlyStr := fmt.Sprintf("%.0f kr", math.Abs(sub.AvgAmount))
		if sub.MinAmount != sub.MaxAmount {
			monthlyStr = fmt.Sprintf("%.0f-%.0f kr", sub.MinAmount, sub.MaxAmount)
		}

		yearlyAmount := math.Abs(sub.AvgAmount) * 12
		yearlyStr := fmt.Sprintf("%.0f kr", yearlyAmount)
		if sub.Status == StatusStopped {
			yearlyStr = text.FgHiBlack.Sprint("-")
		}

		dayStr := fmt.Sprintf("~%d", sub.TypicalDay)

		t.AppendRow(table.Row{
			sub.Name,
			status,
			dayStr,
			sub.StartDate.Format("2006-01-02"),
			sub.LastDate.Format("2006-01-02"),
			monthlyStr,
			yearlyStr,
		})
	}

	t.AppendSeparator()
	t.AppendFooter(table.Row{
		"",
		"",
		"",
		"",
		text.Bold.Sprint("Total (active)"),
		text.Bold.Sprintf("%.0f kr", totalMonthlyCost),
		text.Bold.Sprintf("%.0f kr", totalYearlyCost),
	})

	t.SetStyle(table.StyleRounded)
	t.Style().Format.Header = text.FormatDefault
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, AutoMerge: false},
		{Number: 6, Align: text.AlignRight},
		{Number: 7, Align: text.AlignRight},
	})

	t.Render()
}
