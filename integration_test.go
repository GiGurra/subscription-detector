package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gigurra/subscription-detector/internal"
	"github.com/xuri/excelize/v2"
)

// runCLI runs the subscription-detector CLI with the given args and returns stdout
// It uses an empty config to avoid interference from user's config
func runCLI(t *testing.T, args ...string) string {
	t.Helper()

	// Create empty config to override user's default config
	tmpDir := t.TempDir()
	emptyConfigPath := filepath.Join(tmpDir, "empty-config.yaml")
	os.WriteFile(emptyConfigPath, []byte(""), 0644)

	fullArgs := append([]string{"--config", emptyConfigPath}, args...)
	cmd := exec.Command("go", append([]string{"run", "."}, fullArgs...)...)

	// Capture stdout only (stderr has go download messages)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("CLI failed: %v\nStderr: %s", err, exitErr.Stderr)
		}
		t.Fatalf("CLI failed: %v", err)
	}
	return string(output)
}

// runCLIJSON runs the CLI with JSON output and parses the result
func runCLIJSON(t *testing.T, args ...string) internal.JSONOutput {
	t.Helper()
	fullArgs := append(args, "--output", "json")
	output := runCLI(t, fullArgs...)

	var result internal.JSONOutput
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
	}
	return result
}

// runCLIWithConfig runs the CLI with a custom config file
func runCLIWithConfig(t *testing.T, configContent string, args ...string) string {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	fullArgs := append([]string{"--config", configPath}, args...)
	cmd := exec.Command("go", append([]string{"run", "."}, fullArgs...)...)

	// Capture stdout only (stderr has go download messages)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("CLI failed: %v\nStderr: %s", err, exitErr.Stderr)
		}
		t.Fatalf("CLI failed: %v", err)
	}
	return string(output)
}

// runCLIWithConfigJSON runs the CLI with config and JSON output
func runCLIWithConfigJSON(t *testing.T, configContent string, args ...string) internal.JSONOutput {
	t.Helper()
	fullArgs := append(args, "--output", "json")
	output := runCLIWithConfig(t, configContent, fullArgs...)

	var result internal.JSONOutput
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
	}
	return result
}

func TestCLI_BasicDetection(t *testing.T) {
	result := runCLIJSON(t, "--source", "simple-json", "testdata/sample.json")

	if result.Summary.Count != 2 {
		t.Errorf("expected 2 subscriptions, got %d", result.Summary.Count)
	}

	names := make(map[string]bool)
	for _, sub := range result.Subscriptions {
		names[sub.Name] = true
	}

	if !names["Netflix"] {
		t.Error("expected Netflix subscription")
	}
	if !names["Spotify"] {
		t.Error("expected Spotify subscription")
	}
	if names["Grocery Store"] {
		t.Error("Grocery Store should not be detected as subscription")
	}
}

func TestCLI_Summary(t *testing.T) {
	result := runCLIJSON(t, "--source", "simple-json", "testdata/sample.json")

	// Netflix: 99, Spotify: 129 (latest)
	expectedMonthly := 99.0 + 129.0
	if result.Summary.MonthlyTotal != expectedMonthly {
		t.Errorf("expected monthly total %.0f, got %.0f", expectedMonthly, result.Summary.MonthlyTotal)
	}
	if result.Summary.YearlyTotal != expectedMonthly*12 {
		t.Errorf("expected yearly total %.0f, got %.0f", expectedMonthly*12, result.Summary.YearlyTotal)
	}
}

func TestCLI_ShowAll(t *testing.T) {
	output := runCLI(t, "--source", "simple-json", "testdata/sample.json", "--show", "all")

	if !strings.Contains(output, "Showing: all") {
		t.Errorf("expected 'Showing: all', got: %s", output)
	}
}

func TestCLI_Tags(t *testing.T) {
	config := `
tags:
  Netflix: [entertainment]
  Spotify: [entertainment, music]
`
	result := runCLIWithConfigJSON(t, config, "--source", "simple-json", "testdata/sample.json")

	for _, sub := range result.Subscriptions {
		if sub.Name == "Netflix" && len(sub.Tags) == 0 {
			t.Error("expected Netflix to have tags")
		}
		if sub.Name == "Spotify" && len(sub.Tags) != 2 {
			t.Errorf("expected Spotify to have 2 tags, got %d", len(sub.Tags))
		}
	}
}

func TestCLI_TagFilter(t *testing.T) {
	config := `
tags:
  Netflix: [entertainment]
  Spotify: [music]
`
	result := runCLIWithConfigJSON(t, config,
		"--source", "simple-json", "testdata/sample.json",
		"--tags", "entertainment")

	if result.Summary.Count != 1 {
		t.Errorf("expected 1 subscription after tag filter, got %d", result.Summary.Count)
	}
	if result.Subscriptions[0].Name != "Netflix" {
		t.Errorf("expected Netflix (has entertainment tag), got %s", result.Subscriptions[0].Name)
	}
}

func TestCLI_Descriptions(t *testing.T) {
	config := `
descriptions:
  Netflix: "Video Streaming"
`
	result := runCLIWithConfigJSON(t, config, "--source", "simple-json", "testdata/sample.json")

	for _, sub := range result.Subscriptions {
		if sub.Name == "Netflix" && sub.Description != "Video Streaming" {
			t.Errorf("expected Netflix description 'Video Streaming', got '%s'", sub.Description)
		}
	}
}

func TestCLI_Exclusions(t *testing.T) {
	config := `
exclude:
  - Netflix
`
	result := runCLIWithConfigJSON(t, config, "--source", "simple-json", "testdata/sample.json")

	if result.Summary.Count != 1 {
		t.Errorf("expected 1 subscription after exclusion, got %d", result.Summary.Count)
	}
	for _, sub := range result.Subscriptions {
		if sub.Name == "Netflix" {
			t.Error("Netflix should be excluded")
		}
	}
}

func TestCLI_Groups(t *testing.T) {
	// Create test data with varying names
	tmpDir := t.TempDir()
	testData := `{
  "transactions": [
    {"date": "2025-01-15", "text": "SPOTIFY AB123", "amount": -99.00},
    {"date": "2025-02-15", "text": "SPOTIFY CD456", "amount": -99.00},
    {"date": "2025-03-15", "text": "SPOTIFY EF789", "amount": -99.00},
    {"date": "2025-04-15", "text": "SPOTIFY GH012", "amount": -99.00}
  ]
}`
	dataPath := filepath.Join(tmpDir, "data.json")
	if err := os.WriteFile(dataPath, []byte(testData), 0644); err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}

	config := `
groups:
  - name: "Spotify"
    patterns:
      - "^SPOTIFY"
`
	result := runCLIWithConfigJSON(t, config, "--source", "simple-json", dataPath)

	if result.Summary.Count != 1 {
		t.Errorf("expected 1 grouped subscription, got %d", result.Summary.Count)
	}
	if result.Subscriptions[0].Name != "Spotify" {
		t.Errorf("expected grouped name 'Spotify', got '%s'", result.Subscriptions[0].Name)
	}
}

func TestCLI_SortByAmount(t *testing.T) {
	result := runCLIJSON(t, "--source", "simple-json", "testdata/sample.json",
		"--sort", "amount", "--sort-dir", "desc")

	if len(result.Subscriptions) < 2 {
		t.Fatal("expected at least 2 subscriptions")
	}

	// Spotify (129) should come before Netflix (99) when sorted by amount desc
	if result.Subscriptions[0].Name != "Spotify" {
		t.Errorf("expected Spotify first when sorted by amount desc, got %s", result.Subscriptions[0].Name)
	}
}

func TestCLI_PriceRange(t *testing.T) {
	result := runCLIJSON(t, "--source", "simple-json", "testdata/sample.json")

	for _, sub := range result.Subscriptions {
		if sub.Name == "Spotify" {
			if sub.MinAmount != 119 || sub.MaxAmount != 129 {
				t.Errorf("expected Spotify price range 119-129, got %.0f-%.0f", sub.MinAmount, sub.MaxAmount)
			}
		}
	}
}

func TestCLI_Tolerance(t *testing.T) {
	// With very strict tolerance, Spotify price change (119->129 = ~8%) should still pass
	result := runCLIJSON(t, "--source", "simple-json", "testdata/sample.json", "--tolerance", "0.10")

	if result.Summary.Count != 2 {
		t.Errorf("expected 2 subscriptions with 10%% tolerance, got %d", result.Summary.Count)
	}

	// With extremely strict tolerance (1%), Spotify should be rejected
	result = runCLIJSON(t, "--source", "simple-json", "testdata/sample.json", "--tolerance", "0.01")

	if result.Summary.Count != 1 {
		t.Errorf("expected 1 subscription with 1%% tolerance (Spotify rejected), got %d", result.Summary.Count)
	}
}

func TestCLI_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	data1 := `{"transactions": [
		{"date": "2025-01-15", "text": "ServiceA", "amount": -50.00},
		{"date": "2025-02-15", "text": "ServiceA", "amount": -50.00},
		{"date": "2025-03-15", "text": "ServiceA", "amount": -50.00}
	]}`
	data2 := `{"transactions": [
		{"date": "2025-04-15", "text": "ServiceA", "amount": -50.00},
		{"date": "2025-05-15", "text": "ServiceA", "amount": -50.00},
		{"date": "2025-06-15", "text": "ServiceA", "amount": -50.00}
	]}`

	path1 := filepath.Join(tmpDir, "file1.json")
	path2 := filepath.Join(tmpDir, "file2.json")
	os.WriteFile(path1, []byte(data1), 0644)
	os.WriteFile(path2, []byte(data2), 0644)

	result := runCLIJSON(t, "--source", "simple-json", path1, path2)

	if result.Summary.Count != 1 {
		t.Errorf("expected 1 subscription from combined files, got %d", result.Summary.Count)
	}
	if result.Subscriptions[0].Name != "ServiceA" {
		t.Errorf("expected ServiceA subscription, got %s", result.Subscriptions[0].Name)
	}
}

func TestCLI_EmptyResult(t *testing.T) {
	tmpDir := t.TempDir()
	testData := `{"transactions": [
		{"date": "2025-01-15", "text": "Random", "amount": -50.00}
	]}`
	dataPath := filepath.Join(tmpDir, "data.json")
	os.WriteFile(dataPath, []byte(testData), 0644)

	result := runCLIJSON(t, "--source", "simple-json", dataPath)

	if result.Summary.Count != 0 {
		t.Errorf("expected 0 subscriptions, got %d", result.Summary.Count)
	}
	if len(result.Subscriptions) != 0 {
		t.Error("expected empty subscriptions array")
	}
}

func TestCLI_FormatPrefixSyntax(t *testing.T) {
	// Test using format:path prefix syntax instead of --source flag
	result := runCLIJSON(t, "simple-json:testdata/sample.json")

	if result.Summary.Count != 2 {
		t.Errorf("expected 2 subscriptions with prefix syntax, got %d", result.Summary.Count)
	}
}

// createTestXLSX creates a minimal Handelsbanken-format xlsx file for testing
func createTestXLSX(t *testing.T, path string, transactions [][]string) {
	t.Helper()
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)

	// Header row
	f.SetCellValue(sheet, "A1", "Reskontradatum")
	f.SetCellValue(sheet, "B1", "Text")
	f.SetCellValue(sheet, "C1", "Belopp")

	// Data rows
	for i, tx := range transactions {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), tx[0]) // date
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), tx[1]) // text
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), tx[2]) // amount
	}

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to create test xlsx: %v", err)
	}
}

func TestCLI_MixedFormatPrefixes(t *testing.T) {
	// Test mixing two DIFFERENT formats: simple-json and handelsbanken-xlsx
	tmpDir := t.TempDir()

	// First file: ServiceA in simple-json format
	jsonData := `{"transactions": [
		{"date": "2025-01-15", "text": "ServiceA", "amount": -50.00},
		{"date": "2025-02-15", "text": "ServiceA", "amount": -50.00},
		{"date": "2025-03-15", "text": "ServiceA", "amount": -50.00}
	]}`
	jsonPath := filepath.Join(tmpDir, "source.json")
	os.WriteFile(jsonPath, []byte(jsonData), 0644)

	// Second file: ServiceB in handelsbanken-xlsx format
	xlsxPath := filepath.Join(tmpDir, "source.xlsx")
	createTestXLSX(t, xlsxPath, [][]string{
		{"2025-01-20", "ServiceB", "-75.00"},
		{"2025-02-20", "ServiceB", "-75.00"},
		{"2025-03-20", "ServiceB", "-75.00"},
	})

	// Mix different formats using prefix syntax
	result := runCLIJSON(t, "simple-json:"+jsonPath, "handelsbanken-xlsx:"+xlsxPath)

	if result.Summary.Count != 2 {
		t.Errorf("expected 2 subscriptions from mixed format files, got %d", result.Summary.Count)
	}

	names := make(map[string]bool)
	for _, sub := range result.Subscriptions {
		names[sub.Name] = true
	}

	if !names["ServiceA"] {
		t.Error("expected ServiceA subscription (from simple-json)")
	}
	if !names["ServiceB"] {
		t.Error("expected ServiceB subscription (from handelsbanken-xlsx)")
	}
}

func TestCLI_MixedPrefixAndSourceFlag(t *testing.T) {
	// Test mixing prefix syntax with --source fallback
	tmpDir := t.TempDir()

	// File with explicit prefix
	data1 := `{"transactions": [
		{"date": "2025-01-15", "text": "ServiceA", "amount": -50.00},
		{"date": "2025-02-15", "text": "ServiceA", "amount": -50.00},
		{"date": "2025-03-15", "text": "ServiceA", "amount": -50.00}
	]}`

	// File relying on --source fallback
	data2 := `{"transactions": [
		{"date": "2025-01-20", "text": "ServiceB", "amount": -75.00},
		{"date": "2025-02-20", "text": "ServiceB", "amount": -75.00},
		{"date": "2025-03-20", "text": "ServiceB", "amount": -75.00}
	]}`

	path1 := filepath.Join(tmpDir, "explicit.json")
	path2 := filepath.Join(tmpDir, "fallback.json")
	os.WriteFile(path1, []byte(data1), 0644)
	os.WriteFile(path2, []byte(data2), 0644)

	// First file uses prefix, second relies on --source
	result := runCLIJSON(t, "--source", "simple-json", "simple-json:"+path1, path2)

	if result.Summary.Count != 2 {
		t.Errorf("expected 2 subscriptions, got %d", result.Summary.Count)
	}
}
