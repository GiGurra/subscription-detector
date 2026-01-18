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
	Source     string   `descr:"Data source type" alts:"handelsbanken-xlsx" strict:"true"`
	Files      []string `descr:"Path(s) to transaction file(s)" positional:"true"`
	Config     string   `descr:"Path to config file (YAML)" optional:"true"`
	InitConfig string   `descr:"Generate config template and save to path" optional:"true"`
}

func main() {
	boa.NewCmdT[Params]("subscription-detector").
		WithShort("Detect ongoing subscriptions from bank transactions").
		WithLong("Analyzes bank transaction data to identify recurring monthly subscriptions based on similar amounts (±10%) and recurring payee names.").
		WithRunFunc(func(params *Params) {
			var transactions []Transaction
			for _, file := range params.Files {
				txs, err := ParseHandelsbankenXLSX(file)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error parsing file %s: %v\n", file, err)
					os.Exit(1)
				}
				fmt.Printf("Loaded %d transactions from %s\n", len(txs), file)
				transactions = append(transactions, txs...)
			}

			fmt.Printf("Total: %d transactions from %d file(s)\n", len(transactions), len(params.Files))

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

			// Load config (from provided path or default location)
			var cfg *Config
			configPath := params.Config
			if configPath == "" {
				// Try default config path
				defaultPath := DefaultConfigPath()
				if _, err := os.Stat(defaultPath); err == nil {
					configPath = defaultPath
				}
			}
			if configPath != "" {
				var err error
				cfg, err = LoadConfig(configPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("Loaded config from %s\n\n", configPath)
			}

			// Apply exclusion filters from config
			if cfg != nil {
				subscriptions = filterSubscriptions(subscriptions, cfg)
			}

			// Generate config template if requested
			if params.InitConfig != "" {
				template := GenerateConfigTemplate(subscriptions)
				if err := template.Save(params.InitConfig); err != nil {
					fmt.Fprintf(os.Stderr, "Error saving config template: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("Config template saved to %s\n", params.InitConfig)
				return
			}

			if len(subscriptions) == 0 {
				fmt.Println("No subscriptions detected.")
				return
			}

			printSubscriptionSummary(subscriptions, cfg)
		}).
		Run()
}

func filterSubscriptions(subs []Subscription, cfg *Config) []Subscription {
	var result []Subscription
	for _, sub := range subs {
		if !cfg.ShouldExclude(sub) {
			result = append(result, sub)
		}
	}
	return result
}

func printSubscriptionSummary(subscriptions []Subscription, cfg *Config) {
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

	// Check if any subscription has a description
	hasDescriptions := false
	if cfg != nil {
		for _, sub := range subscriptions {
			if cfg.GetDescription(sub.Name) != "" {
				hasDescriptions = true
				break
			}
		}
	}

	if hasDescriptions {
		t.AppendHeader(table.Row{"Name", "Description", "Status", "Day", "Started", "Last Seen", "Monthly", "Yearly"})
	} else {
		t.AppendHeader(table.Row{"Name", "Status", "Day", "Started", "Last Seen", "Monthly", "Yearly"})
	}

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

		if hasDescriptions {
			desc := ""
			if cfg != nil {
				desc = cfg.GetDescription(sub.Name)
			}
			t.AppendRow(table.Row{
				sub.Name,
				desc,
				status,
				dayStr,
				sub.StartDate.Format("2006-01-02"),
				sub.LastDate.Format("2006-01-02"),
				monthlyStr,
				yearlyStr,
			})
		} else {
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
	}

	t.AppendSeparator()
	if hasDescriptions {
		t.AppendFooter(table.Row{
			"",
			"",
			"",
			"",
			"",
			text.Bold.Sprint("Total (active)"),
			text.Bold.Sprintf("%.0f kr", totalMonthlyCost),
			text.Bold.Sprintf("%.0f kr", totalYearlyCost),
		})
	} else {
		t.AppendFooter(table.Row{
			"",
			"",
			"",
			"",
			text.Bold.Sprint("Total (active)"),
			text.Bold.Sprintf("%.0f kr", totalMonthlyCost),
			text.Bold.Sprintf("%.0f kr", totalYearlyCost),
		})
	}

	t.SetStyle(table.StyleRounded)
	t.Style().Format.Header = text.FormatDefault
	if hasDescriptions {
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 7, Align: text.AlignRight},
			{Number: 8, Align: text.AlignRight},
		})
	} else {
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 6, Align: text.AlignRight},
			{Number: 7, Align: text.AlignRight},
		})
	}

	t.Render()
}
