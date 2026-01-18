package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/GiGurra/boa/pkg/boa"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

type Params struct {
	Source        string   `descr:"Data source type" alts:"handelsbanken-xlsx,testdata-json" strict:"true"`
	Files         []string `descr:"Path(s) to transaction file(s)" positional:"true"`
	Config        string   `descr:"Path to config file (YAML)" optional:"true"`
	InitConfig    string   `descr:"Generate config template and save to path" optional:"true"`
	Show          string   `descr:"Which subscriptions to show" default:"active" alts:"active,stopped,all" strict:"true"`
	Sort          string   `descr:"Sort field for output" default:"name" alts:"name,description,amount" strict:"true"`
	SortDir       string   `descr:"Sort direction" default:"asc" alts:"asc,desc" strict:"true"`
	Output        string   `descr:"Output format" default:"table" alts:"table,json" strict:"true"`
	Tolerance     float64  `descr:"Max price change between months (0.35 = 35%)" default:"0.35"`
	SuggestGroups bool     `descr:"Analyze and suggest potential transaction groups" optional:"true"`
	Tags          []string `descr:"Filter by tags (e.g., entertainment, insurance)" optional:"true"`
}

func main() {
	boa.CmdT[Params]{
		Use:   "subscription-detector",
		Short: "Detect ongoing subscriptions from bank transactions",
		Long:  "Analyzes bank transaction data to identify recurring monthly subscriptions based on similar amounts and recurring payee names.",
		ParamEnrich: boa.ParamEnricherCombine(
			boa.ParamEnricherName,
			boa.ParamEnricherShort,
			boa.ParamEnricherBool,
		),
		RunFunc: run,
	}.Run()
}

func run(params *Params, _ *cobra.Command, _ []string) {
	// Helper to print info messages (suppressed in JSON mode)
	info := func(format string, args ...any) {
		if params.Output != "json" {
			fmt.Printf(format, args...)
		}
	}

	parser, err := GetParser(params.Source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var transactions []Transaction
	for _, file := range params.Files {
		txs, err := parser.Parse(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing file %s: %v\n", file, err)
			os.Exit(1)
		}
		info("Loaded %d transactions from %s\n", len(txs), file)
		transactions = append(transactions, txs...)
	}

	info("Total: %d transactions from %d file(s)\n", len(transactions), len(params.Files))

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
		info("Loaded config from %s\n", configPath)
	}

	// Apply grouping from config (combines transactions with different names into one)
	transactions, _ = cfg.ApplyGroups(transactions)

	// Check data coverage
	completeMonths, dateRange := AnalyzeDataCoverage(transactions)
	info("Data range: %s to %s\n", dateRange.Start.Format("2006-01-02"), dateRange.End.Format("2006-01-02"))
	info("Complete months: %d\n\n", len(completeMonths))

	if len(completeMonths) < 3 {
		fmt.Fprintf(os.Stderr, "Warning: Less than 3 complete months of data. Subscription detection may be unreliable.\n\n")
	}

	// Filter to only complete months for pattern detection
	filtered := FilterToCompleteMonths(transactions, completeMonths)
	subscriptions := DetectSubscriptions(filtered, transactions, dateRange, params.Tolerance)

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

	// Suggest groups if requested
	if params.SuggestGroups {
		suggestions := SuggestGroups(transactions, params.Tolerance)
		PrintGroupSuggestions(suggestions)
		return
	}

	if len(subscriptions) == 0 {
		if params.Output == "json" {
			printSubscriptionsJSON(nil, cfg)
		} else {
			fmt.Println("No subscriptions detected.")
		}
		return
	}

	// Filter by status for display (but show total counts first)
	displaySubs := filterByStatus(subscriptions, params.Show)

	// Filter by tags if specified
	if len(params.Tags) > 0 {
		displaySubs = filterByTags(displaySubs, params.Tags, cfg)
	}

	if params.Output == "json" {
		printSubscriptionsJSON(displaySubs, cfg)
	} else {
		printSubscriptionSummary(subscriptions, displaySubs, params.Show, params.Tags, params.Sort, params.SortDir, cfg)
	}
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

func filterByStatus(subs []Subscription, show string) []Subscription {
	if show == "all" {
		return subs
	}
	var result []Subscription
	for _, sub := range subs {
		if show == "active" && sub.Status == StatusActive {
			result = append(result, sub)
		} else if show == "stopped" && sub.Status == StatusStopped {
			result = append(result, sub)
		}
	}
	return result
}

func filterByTags(subs []Subscription, tags []string, cfg *Config) []Subscription {
	if cfg == nil || len(tags) == 0 {
		return subs
	}
	var result []Subscription
	for _, sub := range subs {
		subTags := cfg.GetTags(sub.Name)
		if hasAnyTag(subTags, tags) {
			result = append(result, sub)
		}
	}
	return result
}

func hasAnyTag(subTags []string, filterTags []string) bool {
	for _, ft := range filterTags {
		for _, st := range subTags {
			if strings.EqualFold(st, ft) {
				return true
			}
		}
	}
	return false
}

func printSubscriptionSummary(allSubs []Subscription, displaySubs []Subscription, showFilter string, tagFilter []string, sortField string, sortDir string, cfg *Config) {
	// Count from all subscriptions (for summary line)
	activeCount := 0
	stoppedCount := 0
	for _, sub := range allSubs {
		if sub.Status == StatusActive {
			activeCount++
		} else {
			stoppedCount++
		}
	}

	// Calculate totals from displayed subscriptions only (using latest amount)
	var totalMonthlyCost float64
	for _, sub := range displaySubs {
		if sub.Status == StatusActive {
			totalMonthlyCost += math.Abs(sub.LatestAmount)
		}
	}
	totalYearlyCost := totalMonthlyCost * 12

	fmt.Printf("Found %d subscriptions (%d active, %d stopped)\n",
		len(allSubs), activeCount, stoppedCount)
	showingStr := showFilter
	if len(tagFilter) > 0 {
		showingStr += fmt.Sprintf(", tags: %s", strings.Join(tagFilter, ", "))
	}
	fmt.Printf("Showing: %s\n\n", showingStr)

	// Sort displayed subscriptions
	sort.Slice(displaySubs, func(i, j int) bool {
		var less bool
		switch sortField {
		case "amount":
			less = math.Abs(displaySubs[i].AvgAmount) < math.Abs(displaySubs[j].AvgAmount)
		case "description":
			iName := displaySubs[i].Name
			jName := displaySubs[j].Name
			if cfg != nil {
				if desc := cfg.GetDescription(iName); desc != "" {
					iName = desc
				}
				if desc := cfg.GetDescription(jName); desc != "" {
					jName = desc
				}
			}
			less = strings.ToLower(iName) < strings.ToLower(jName)
		default: // "name"
			less = strings.ToLower(displaySubs[i].Name) < strings.ToLower(displaySubs[j].Name)
		}
		if sortDir == "desc" {
			return !less
		}
		return less
	})

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	// Check which optional columns to show
	hasDescriptions := false
	hasTags := false
	if cfg != nil {
		for _, sub := range displaySubs {
			if cfg.GetDescription(sub.Name) != "" {
				hasDescriptions = true
			}
			if len(cfg.GetTags(sub.Name)) > 0 {
				hasTags = true
			}
			if hasDescriptions && hasTags {
				break
			}
		}
	}

	// Build header dynamically
	header := table.Row{"Name"}
	if hasDescriptions {
		header = append(header, "Description")
	}
	if hasTags {
		header = append(header, "Tags")
	}
	header = append(header, "Status", "Day", "Started", "Last Seen", "Monthly", "Yearly")
	t.AppendHeader(header)

	for _, sub := range displaySubs {
		status := text.FgGreen.Sprint("ACTIVE")
		if sub.Status == StatusStopped {
			status = text.FgRed.Sprint("STOPPED")
		}

		monthlyStr := fmt.Sprintf("%.0f kr", math.Abs(sub.AvgAmount))
		if sub.MinAmount != sub.MaxAmount {
			monthlyStr = fmt.Sprintf("%.0f-%.0f kr", sub.MinAmount, sub.MaxAmount)
		}

		yearlyAmount := math.Abs(sub.LatestAmount) * 12
		yearlyStr := fmt.Sprintf("%.0f kr", yearlyAmount)
		if sub.Status == StatusStopped {
			yearlyStr = text.FgHiBlack.Sprint("-")
		}

		dayStr := fmt.Sprintf("~%d", sub.TypicalDay)

		// Build row dynamically
		row := table.Row{sub.Name}
		if hasDescriptions {
			desc := ""
			if cfg != nil {
				desc = cfg.GetDescription(sub.Name)
			}
			row = append(row, desc)
		}
		if hasTags {
			tagsStr := ""
			if cfg != nil {
				tags := cfg.GetTags(sub.Name)
				tagsStr = strings.Join(tags, ", ")
			}
			row = append(row, tagsStr)
		}
		row = append(row, status, dayStr, sub.StartDate.Format("2006-01-02"), sub.LastDate.Format("2006-01-02"), monthlyStr, yearlyStr)
		t.AppendRow(row)
	}

	t.AppendSeparator()

	// Build footer dynamically (empty cells for optional columns)
	footer := table.Row{""}
	if hasDescriptions {
		footer = append(footer, "")
	}
	if hasTags {
		footer = append(footer, "")
	}
	footer = append(footer, "", "", "", text.Bold.Sprint("Total (active)"), text.Bold.Sprintf("%.0f kr", totalMonthlyCost), text.Bold.Sprintf("%.0f kr", totalYearlyCost))
	t.AppendFooter(footer)

	t.SetStyle(table.StyleRounded)
	t.Style().Format.Header = text.FormatDefault

	// Right-align Monthly and Yearly columns (last two)
	colCount := len(header)
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: colCount - 1, Align: text.AlignRight},
		{Number: colCount, Align: text.AlignRight},
	})

	t.Render()
}

// JSONOutput is the root JSON output object
type JSONOutput struct {
	Subscriptions []JSONSubscription `json:"subscriptions"`
	Summary       JSONSummary        `json:"summary"`
}

// JSONSummary contains aggregate statistics
type JSONSummary struct {
	Count        int     `json:"count"`
	MonthlyTotal float64 `json:"monthly_total"`
	YearlyTotal  float64 `json:"yearly_total"`
}

// JSONSubscription is the JSON output format for a subscription
type JSONSubscription struct {
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Status       string   `json:"status"`
	TypicalDay   int      `json:"typical_day"`
	StartDate    string   `json:"start_date"`
	LastDate     string   `json:"last_date"`
	LatestAmount float64  `json:"latest_amount"`
	MinAmount    float64  `json:"min_amount"`
	MaxAmount    float64  `json:"max_amount"`
	YearlyCost   float64  `json:"yearly_cost"`
}

func printSubscriptionsJSON(subs []Subscription, cfg *Config) {
	var subscriptions []JSONSubscription
	var monthlyTotal float64

	for _, sub := range subs {
		desc := ""
		var tags []string
		if cfg != nil {
			desc = cfg.GetDescription(sub.Name)
			tags = cfg.GetTags(sub.Name)
		}

		latestAmount := math.Abs(sub.LatestAmount)
		if sub.Status == StatusActive {
			monthlyTotal += latestAmount
		}

		subscriptions = append(subscriptions, JSONSubscription{
			Name:         sub.Name,
			Description:  desc,
			Tags:         tags,
			Status:       string(sub.Status),
			TypicalDay:   sub.TypicalDay,
			StartDate:    sub.StartDate.Format("2006-01-02"),
			LastDate:     sub.LastDate.Format("2006-01-02"),
			LatestAmount: latestAmount,
			MinAmount:    sub.MinAmount,
			MaxAmount:    sub.MaxAmount,
			YearlyCost:   latestAmount * 12,
		})
	}

	output := JSONOutput{
		Subscriptions: subscriptions,
		Summary: JSONSummary{
			Count:        len(subscriptions),
			MonthlyTotal: monthlyTotal,
			YearlyTotal:  monthlyTotal * 12,
		},
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(output)
}
