package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// OutputOptions controls how subscriptions are displayed
type OutputOptions struct {
	ShowFilter string
	TagFilter  []string
	SortField  string
	SortDir    string
	Currency   Currency
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
	Currency     string  `json:"currency"`
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

// PrintSubscriptionsJSON outputs subscriptions in JSON format
func PrintSubscriptionsJSON(w io.Writer, subs []Subscription, cfg *Config, currency Currency) {
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
			Currency:     currency.Code,
		},
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(output)
}

// PrintSubscriptionsTable outputs subscriptions as a formatted table
func PrintSubscriptionsTable(w io.Writer, allSubs []Subscription, displaySubs []Subscription, opts OutputOptions, cfg *Config) {
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

	fmt.Fprintf(w, "Found %d subscriptions (%d active, %d stopped)\n",
		len(allSubs), activeCount, stoppedCount)
	showingStr := opts.ShowFilter
	if len(opts.TagFilter) > 0 {
		showingStr += fmt.Sprintf(", tags: %s", strings.Join(opts.TagFilter, ", "))
	}
	fmt.Fprintf(w, "Showing: %s\n\n", showingStr)

	// Sort displayed subscriptions
	sort.Slice(displaySubs, func(i, j int) bool {
		var less bool
		switch opts.SortField {
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
		if opts.SortDir == "desc" {
			return !less
		}
		return less
	})

	t := table.NewWriter()
	t.SetOutputMirror(w)

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

		monthlyStr := opts.Currency.Format(math.Abs(sub.AvgAmount))
		if sub.MinAmount != sub.MaxAmount {
			monthlyStr = opts.Currency.FormatRange(sub.MinAmount, sub.MaxAmount)
		}

		yearlyAmount := math.Abs(sub.LatestAmount) * 12
		yearlyStr := opts.Currency.Format(yearlyAmount)
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
	footer = append(footer, "", "", "", text.Bold.Sprint("Total (active)"), text.Bold.Sprint(opts.Currency.Format(totalMonthlyCost)), text.Bold.Sprint(opts.Currency.Format(totalYearlyCost)))
	t.AppendFooter(footer)

	t.SetStyle(table.StyleRounded)
	t.Style().Format.Header = text.FormatDefault
	t.Style().Format.Footer = text.FormatDefault

	// Right-align Monthly and Yearly columns (last two)
	colCount := len(header)
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: colCount - 1, Align: text.AlignRight},
		{Number: colCount, Align: text.AlignRight},
	})

	t.Render()
}

// FilterByStatus filters subscriptions by status (active/stopped/all)
func FilterByStatus(subs []Subscription, show string) []Subscription {
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

// FilterByTags filters subscriptions to only those with matching tags
func FilterByTags(subs []Subscription, tags []string, cfg *Config) []Subscription {
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

// FilterByExclusions removes subscriptions matching exclusion rules
func FilterByExclusions(subs []Subscription, cfg *Config) []Subscription {
	if cfg == nil {
		return subs
	}
	var result []Subscription
	for _, sub := range subs {
		if !cfg.ShouldExclude(sub) {
			result = append(result, sub)
		}
	}
	return result
}
