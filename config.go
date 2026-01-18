package main

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// Descriptions maps subscription names to custom descriptions
	Descriptions map[string]string `yaml:"descriptions,omitempty"`

	// Exclude is a list of regex patterns - matching subscriptions are excluded
	Exclude []string `yaml:"exclude,omitempty"`

	// compiled regex patterns (not serialized)
	excludePatterns []*regexp.Regexp `yaml:"-"`
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

	// Compile exclude patterns
	for _, pattern := range cfg.Exclude {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude pattern %q: %w", pattern, err)
		}
		cfg.excludePatterns = append(cfg.excludePatterns, re)
	}

	return &cfg, nil
}

func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// ShouldExclude returns true if the subscription name matches any exclude pattern
func (c *Config) ShouldExclude(name string) bool {
	if c == nil {
		return false
	}
	for _, re := range c.excludePatterns {
		if re.MatchString(name) {
			return true
		}
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
