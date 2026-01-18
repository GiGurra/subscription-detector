package main

import (
	"fmt"
	"os"

	"github.com/GiGurra/boa/pkg/boa"
	"github.com/gigurra/subscription-detector/internal"
	"github.com/spf13/cobra"
)

type Params struct {
	Source        string   `descr:"Default format (or use format:path syntax)" alts:"handelsbanken-xlsx,simple-json" optional:"true"`
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

	var transactions []internal.Transaction
	for _, fileArg := range params.Files {
		format, filePath := internal.ParseFileArg(fileArg)
		if format == "" {
			format = params.Source // Fall back to --source flag
		}
		if format == "" {
			fmt.Fprintf(os.Stderr, "Error: no format specified for %s (use format:path or --source)\n", filePath)
			os.Exit(1)
		}

		parser, err := internal.GetParser(format)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		txs, err := parser.Parse(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing file %s: %v\n", filePath, err)
			os.Exit(1)
		}
		info("Loaded %d transactions from %s\n", len(txs), filePath)
		transactions = append(transactions, txs...)
	}

	info("Total: %d transactions from %d file(s)\n", len(transactions), len(params.Files))

	// Load config (from provided path or default location)
	var cfg *internal.Config
	configPath := params.Config
	if configPath == "" {
		// Try default config path
		defaultPath := internal.DefaultConfigPath()
		if _, err := os.Stat(defaultPath); err == nil {
			configPath = defaultPath
		}
	}
	if configPath != "" {
		var err error
		cfg, err = internal.LoadConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		info("Loaded config from %s\n", configPath)
	} else {
		// No config file - use default config with built-in known subscriptions
		var err error
		cfg, err = internal.NewDefaultConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating default config: %v\n", err)
			os.Exit(1)
		}
	}

	// Apply grouping from config (combines transactions with different names into one)
	transactions, _ = cfg.ApplyGroups(transactions)

	// Check data coverage
	completeMonths, dateRange := internal.AnalyzeDataCoverage(transactions)
	info("Data range: %s to %s\n", dateRange.Start.Format("2006-01-02"), dateRange.End.Format("2006-01-02"))
	info("Complete months: %d\n\n", len(completeMonths))

	if len(completeMonths) < 3 {
		fmt.Fprintf(os.Stderr, "Warning: Less than 3 complete months of data. Subscription detection may be unreliable.\n\n")
	}

	// Detect known subscriptions first (these can match even with 1 occurrence)
	knownSubs, matchedTexts := internal.DetectKnownSubscriptions(transactions, dateRange, cfg)

	// Filter out transactions that matched known subscriptions from regular detection
	regularTxs := internal.FilterOutMatched(transactions, matchedTexts)

	// Filter to only complete months for pattern detection
	filtered := internal.FilterToCompleteMonths(regularTxs, completeMonths)
	subscriptions := internal.DetectSubscriptions(filtered, regularTxs, dateRange, params.Tolerance)

	// Merge known and detected subscriptions
	subscriptions = append(knownSubs, subscriptions...)

	// Apply exclusion filters from config
	subscriptions = internal.FilterByExclusions(subscriptions, cfg)

	// Generate config template if requested
	if params.InitConfig != "" {
		template := internal.GenerateConfigTemplate(subscriptions)
		if err := template.Save(params.InitConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config template: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Config template saved to %s\n", params.InitConfig)
		return
	}

	// Suggest groups if requested
	if params.SuggestGroups {
		suggestions := internal.SuggestGroups(transactions, params.Tolerance)
		internal.PrintGroupSuggestions(suggestions)
		return
	}

	if len(subscriptions) == 0 {
		if params.Output == "json" {
			internal.PrintSubscriptionsJSON(os.Stdout, nil, cfg)
		} else {
			fmt.Println("No subscriptions detected.")
		}
		return
	}

	// Filter by status for display (but show total counts first)
	displaySubs := internal.FilterByStatus(subscriptions, params.Show)

	// Filter by tags if specified
	if len(params.Tags) > 0 {
		displaySubs = internal.FilterByTags(displaySubs, params.Tags, cfg)
	}

	if params.Output == "json" {
		internal.PrintSubscriptionsJSON(os.Stdout, displaySubs, cfg)
	} else {
		opts := internal.OutputOptions{
			ShowFilter: params.Show,
			TagFilter:  params.Tags,
			SortField:  params.Sort,
			SortDir:    params.SortDir,
		}
		internal.PrintSubscriptionsTable(os.Stdout, subscriptions, displaySubs, opts, cfg)
	}
}
