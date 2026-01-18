package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// ExcludeRule represents an exclusion rule with optional time bounds
type ExcludeRule struct {
	Pattern string `yaml:"pattern"`
	Before  string `yaml:"before,omitempty"` // Exclude only before this date (YYYY-MM-DD)
	After   string `yaml:"after,omitempty"`  // Exclude only after this date (YYYY-MM-DD)

	// compiled fields
	regex      *regexp.Regexp `yaml:"-"`
	beforeDate time.Time      `yaml:"-"`
	afterDate  time.Time      `yaml:"-"`
}

// Group allows grouping multiple transaction patterns under a single name
type Group struct {
	Name      string   `yaml:"name"`
	Patterns  []string `yaml:"patterns"`
	Tolerance *float64 `yaml:"tolerance,omitempty"` // Optional custom tolerance for this group

	// compiled patterns
	regexes []*regexp.Regexp `yaml:"-"`
}

type Config struct {
	// Descriptions maps subscription names to custom descriptions
	Descriptions map[string]string `yaml:"descriptions,omitempty"`

	// Tags maps subscription names to a list of tags (e.g., "entertainment", "utilities")
	Tags map[string][]string `yaml:"tags,omitempty"`

	// Groups allows combining multiple transaction patterns into one subscription
	Groups []Group `yaml:"groups,omitempty"`

	// Exclude is a list of exclusion rules (can be strings or objects with time bounds)
	Exclude []yaml.Node `yaml:"exclude,omitempty"`

	// compiled exclusion rules (not serialized)
	excludeRules []ExcludeRule `yaml:"-"`
}

// DefaultConfigPath returns the default config file path (~/.subscription-detector/config.yaml)
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".subscription-detector", "config.yaml")
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Compile group patterns
	for i := range cfg.Groups {
		for _, pattern := range cfg.Groups[i].Patterns {
			re, err := regexp.Compile("(?i)" + pattern) // case-insensitive
			if err != nil {
				return nil, fmt.Errorf("invalid group pattern %q: %w", pattern, err)
			}
			cfg.Groups[i].regexes = append(cfg.Groups[i].regexes, re)
		}
	}

	// Parse exclude rules (supports both strings and objects)
	for _, node := range cfg.Exclude {
		var rule ExcludeRule

		if node.Kind == yaml.ScalarNode {
			// Simple string pattern
			rule.Pattern = node.Value
		} else if node.Kind == yaml.MappingNode {
			// Object with pattern and optional time bounds
			if err := node.Decode(&rule); err != nil {
				return nil, fmt.Errorf("parsing exclude rule: %w", err)
			}
		} else {
			return nil, fmt.Errorf("invalid exclude rule format")
		}

		// Compile regex
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude pattern %q: %w", rule.Pattern, err)
		}
		rule.regex = re

		// Parse time bounds
		if rule.Before != "" {
			t, err := time.Parse("2006-01-02", rule.Before)
			if err != nil {
				return nil, fmt.Errorf("invalid 'before' date %q: %w", rule.Before, err)
			}
			rule.beforeDate = t
		}
		if rule.After != "" {
			t, err := time.Parse("2006-01-02", rule.After)
			if err != nil {
				return nil, fmt.Errorf("invalid 'after' date %q: %w", rule.After, err)
			}
			rule.afterDate = t
		}

		cfg.excludeRules = append(cfg.excludeRules, rule)
	}

	return &cfg, nil
}

func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// ShouldExclude returns true if the subscription matches any exclude rule
// considering time bounds against the subscription's date range
func (c *Config) ShouldExclude(sub Subscription) bool {
	if c == nil {
		return false
	}
	for _, rule := range c.excludeRules {
		if !rule.regex.MatchString(sub.Name) {
			continue
		}

		// Check time bounds - exclude if subscription falls within the rule's time window
		// before: exclude subscriptions that ended before this date
		// after: exclude subscriptions that started after this date
		if !rule.beforeDate.IsZero() && !sub.LastDate.Before(rule.beforeDate) {
			continue // Subscription extends past the "before" date, don't exclude
		}
		if !rule.afterDate.IsZero() && sub.StartDate.Before(rule.afterDate) {
			continue // Subscription started before the "after" date, don't exclude
		}

		return true
	}
	return false
}

// GetDescription returns the custom description for a subscription, or empty string
func (c *Config) GetDescription(name string) string {
	if c == nil || c.Descriptions == nil {
		return ""
	}
	return c.Descriptions[name]
}

// GetTags returns the tags for a subscription, or nil if none
func (c *Config) GetTags(name string) []string {
	if c == nil || c.Tags == nil {
		return nil
	}
	return c.Tags[name]
}

// ApplyGroups transforms transactions by replacing names that match group patterns
// with the group name. Returns the transformed transactions and a map of group tolerances.
func (c *Config) ApplyGroups(txs []Transaction) ([]Transaction, map[string]float64) {
	tolerances := make(map[string]float64)
	if c == nil || len(c.Groups) == 0 {
		return txs, tolerances
	}

	result := make([]Transaction, len(txs))
	for i, tx := range txs {
		result[i] = tx
		for _, group := range c.Groups {
			for _, re := range group.regexes {
				if re.MatchString(tx.Text) {
					result[i].Text = group.Name
					if group.Tolerance != nil {
						tolerances[group.Name] = *group.Tolerance
					}
					break
				}
			}
		}
	}
	return result, tolerances
}

// GenerateFromSubscriptions creates a config template from detected subscriptions
func GenerateConfigTemplate(subscriptions []Subscription) *Config {
	cfg := &Config{
		Descriptions: make(map[string]string),
	}

	for _, sub := range subscriptions {
		cfg.Descriptions[sub.Name] = "" // Empty description as placeholder
	}

	return cfg
}
