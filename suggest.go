package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// GroupSuggestion represents a suggested grouping of transactions
type GroupSuggestion struct {
	Prefix       string
	Pattern      string
	Names        []string
	MonthCount   int
	Transactions []Transaction
}

// SuggestGroups analyzes transactions to find potential groupings
// based on common prefixes with monthly payment patterns
func SuggestGroups(txs []Transaction, tolerance float64) []GroupSuggestion {
	// Only look at expenses
	expenses := FilterExpenses(txs)

	// Group by exact name first
	byName := make(map[string][]Transaction)
	for _, tx := range expenses {
		byName[tx.Text] = append(byName[tx.Text], tx)
	}

	// Find names that appear only 1-2 times (not enough for subscription detection alone)
	// These are candidates for grouping
	var orphanNames []string
	for name, txList := range byName {
		if len(txList) <= 2 {
			orphanNames = append(orphanNames, name)
		}
	}

	// Try to find common prefixes among orphan names
	prefixGroups := findPrefixGroups(orphanNames, byName)

	// Filter to only groups that look like subscriptions
	var suggestions []GroupSuggestion
	for _, group := range prefixGroups {
		if isLikelySubscription(group.Transactions, tolerance) {
			suggestions = append(suggestions, group)
		}
	}

	// Deduplicate: if two suggestions cover the same transactions, prefer shorter/cleaner prefix
	suggestions = deduplicateSuggestions(suggestions)

	// Sort by number of months (descending)
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].MonthCount > suggestions[j].MonthCount
	})

	return suggestions
}

// findPrefixGroups groups transaction names by common prefixes
func findPrefixGroups(names []string, txsByName map[string][]Transaction) []GroupSuggestion {
	// Track word-based vs character-based prefixes separately
	wordPrefixes := make(map[string][]string)  // word-based prefixes (preferred)
	charPrefixes := make(map[string][]string)  // character-based prefixes (fallback)

	for _, name := range names {
		words := strings.Fields(name)
		if len(words) > 0 {
			// First word as prefix (preferred)
			firstWord := words[0]
			if len(firstWord) >= 3 {
				wordPrefixes[firstWord] = append(wordPrefixes[firstWord], name)
			}
			// Also try first two words for multi-word vendor names
			if len(words) > 1 {
				twoWords := words[0] + " " + words[1]
				wordPrefixes[twoWords] = append(wordPrefixes[twoWords], name)
			}
		}

		// Character-based prefixes for names without spaces (like "GOOGLE*GSUITE")
		if !strings.Contains(name, " ") {
			for _, prefixLen := range []int{6, 8, 10, 12} {
				if len(name) > prefixLen {
					prefix := name[:prefixLen]
					charPrefixes[prefix] = append(charPrefixes[prefix], name)
				}
			}
		}
	}

	// Combine: word prefixes first (sorted by length), then char prefixes
	var sortedPrefixes []string

	var wordKeys []string
	for k := range wordPrefixes {
		wordKeys = append(wordKeys, k)
	}
	sort.Slice(wordKeys, func(i, j int) bool {
		if len(wordKeys[i]) != len(wordKeys[j]) {
			return len(wordKeys[i]) < len(wordKeys[j])
		}
		return wordKeys[i] < wordKeys[j]
	})
	sortedPrefixes = append(sortedPrefixes, wordKeys...)

	var charKeys []string
	for k := range charPrefixes {
		charKeys = append(charKeys, k)
	}
	sort.Slice(charKeys, func(i, j int) bool {
		if len(charKeys[i]) != len(charKeys[j]) {
			return len(charKeys[i]) < len(charKeys[j])
		}
		return charKeys[i] < charKeys[j]
	})
	sortedPrefixes = append(sortedPrefixes, charKeys...)

	// Merge the maps for lookup
	prefixCandidates := make(map[string][]string)
	for k, v := range wordPrefixes {
		prefixCandidates[k] = v
	}
	for k, v := range charPrefixes {
		prefixCandidates[k] = append(prefixCandidates[k], v...)
	}

	// Convert to GroupSuggestions, only keeping groups with 3+ unique names
	var groups []GroupSuggestion
	seen := make(map[string]bool) // avoid duplicate suggestions

	for _, prefix := range sortedPrefixes {
		matchedNames := prefixCandidates[prefix]
		if len(matchedNames) < 3 {
			continue
		}

		// Deduplicate names
		uniqueNames := uniqueStrings(matchedNames)
		if len(uniqueNames) < 3 {
			continue
		}

		// Create a key to avoid duplicates (sorted names)
		sortedNames := make([]string, len(uniqueNames))
		copy(sortedNames, uniqueNames)
		sort.Strings(sortedNames)
		key := strings.Join(sortedNames, "|")
		if seen[key] {
			continue
		}
		seen[key] = true

		// Collect all transactions for these names
		var allTxs []Transaction
		for _, name := range uniqueNames {
			allTxs = append(allTxs, txsByName[name]...)
		}

		// Count unique months
		months := make(map[string]bool)
		for _, tx := range allTxs {
			months[tx.Date.Format("2006-01")] = true
		}

		// Generate a regex pattern
		pattern := generatePattern(prefix)

		groups = append(groups, GroupSuggestion{
			Prefix:       prefix,
			Pattern:      pattern,
			Names:        uniqueNames,
			MonthCount:   len(months),
			Transactions: allTxs,
		})
	}

	return groups
}

// isLikelySubscription checks if transactions look like a subscription
func isLikelySubscription(txs []Transaction, tolerance float64) bool {
	if len(txs) < 3 {
		return false
	}

	// Sort by date
	sorted := make([]Transaction, len(txs))
	copy(sorted, txs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Date.Before(sorted[j].Date)
	})

	// Check monthly pattern (max 1 per month)
	if !IsMonthlyPattern(sorted) {
		return false
	}

	// Check amounts are within tolerance
	if !AmountsWithinTolerance(sorted, tolerance) {
		return false
	}

	return true
}

// generatePattern creates a regex pattern from a prefix
func generatePattern(prefix string) string {
	// Escape special regex characters in prefix
	escaped := regexp.QuoteMeta(prefix)
	return "^" + escaped
}

// deduplicateSuggestions removes redundant suggestions that cover the same transactions
// preferring shorter/cleaner prefixes
func deduplicateSuggestions(suggestions []GroupSuggestion) []GroupSuggestion {
	if len(suggestions) <= 1 {
		return suggestions
	}

	// Sort by prefix length (shorter first) so we prefer cleaner names
	sort.Slice(suggestions, func(i, j int) bool {
		return len(suggestions[i].Prefix) < len(suggestions[j].Prefix)
	})

	var result []GroupSuggestion
	coveredNames := make(map[string]bool)

	for _, s := range suggestions {
		// Check if this suggestion's names are already covered
		newNames := 0
		for _, name := range s.Names {
			if !coveredNames[name] {
				newNames++
			}
		}

		// Only keep if it adds significant new coverage (>50% new names)
		if float64(newNames)/float64(len(s.Names)) > 0.5 {
			result = append(result, s)
			for _, name := range s.Names {
				coveredNames[name] = true
			}
		}
	}

	return result
}

// uniqueStrings returns unique strings from a slice
func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// PrintGroupSuggestions displays suggested groups in a user-friendly format
func PrintGroupSuggestions(suggestions []GroupSuggestion) {
	if len(suggestions) == 0 {
		fmt.Println("No group suggestions found.")
		return
	}

	fmt.Printf("Found %d potential group(s):\n\n", len(suggestions))

	for _, s := range suggestions {
		fmt.Printf("  \"%s\" (%d months, %d transactions)\n", s.Prefix, s.MonthCount, len(s.Transactions))
		fmt.Printf("    Names: %s\n", strings.Join(truncateStrings(s.Names, 3), ", "))
		if len(s.Names) > 3 {
			fmt.Printf("           ... and %d more\n", len(s.Names)-3)
		}
		fmt.Println()
		fmt.Println("    Add to config:")
		fmt.Printf("      - name: \"%s\"\n", s.Prefix)
		fmt.Println("        patterns:")
		fmt.Printf("          - \"%s\"\n", s.Pattern)
		fmt.Println()
	}
}

// truncateStrings returns at most n strings from the slice
func truncateStrings(strs []string, n int) []string {
	if len(strs) <= n {
		return strs
	}
	return strs[:n]
}
