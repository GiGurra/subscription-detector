package internal

import "testing"

func TestIsKnownParser(t *testing.T) {
	// Register a test parser
	RegisterParser("test-format", ParserFunc(func(path string) ([]Transaction, error) {
		return nil, nil
	}))

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"known parser", "test-format", true},
		{"built-in parser", "handelsbanken-xlsx", true},
		{"unknown parser", "unknown-format", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsKnownParser(tt.input)
			if got != tt.expected {
				t.Errorf("IsKnownParser(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseFileArg(t *testing.T) {
	// Register a test parser for these tests
	RegisterParser("test-format", ParserFunc(func(path string) ([]Transaction, error) {
		return nil, nil
	}))

	tests := []struct {
		name           string
		input          string
		expectedFormat string
		expectedPath   string
	}{
		{
			name:           "with known format prefix",
			input:          "test-format:data.json",
			expectedFormat: "test-format",
			expectedPath:   "data.json",
		},
		{
			name:           "with built-in format prefix",
			input:          "handelsbanken-xlsx:bank.xlsx",
			expectedFormat: "handelsbanken-xlsx",
			expectedPath:   "bank.xlsx",
		},
		{
			name:           "no prefix",
			input:          "data.json",
			expectedFormat: "",
			expectedPath:   "data.json",
		},
		{
			name:           "unknown prefix treated as path",
			input:          "unknown:data.json",
			expectedFormat: "",
			expectedPath:   "unknown:data.json",
		},
		{
			name:           "windows path with drive letter",
			input:          "C:\\Users\\test\\data.xlsx",
			expectedFormat: "",
			expectedPath:   "C:\\Users\\test\\data.xlsx",
		},
		{
			name:           "path with colon but not a parser",
			input:          "foo:bar:baz.json",
			expectedFormat: "",
			expectedPath:   "foo:bar:baz.json",
		},
		{
			name:           "format prefix with path containing spaces",
			input:          "test-format:path with spaces/file.json",
			expectedFormat: "test-format",
			expectedPath:   "path with spaces/file.json",
		},
		{
			name:           "format prefix with absolute path",
			input:          "test-format:/home/user/data.json",
			expectedFormat: "test-format",
			expectedPath:   "/home/user/data.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFormat, gotPath := ParseFileArg(tt.input)
			if gotFormat != tt.expectedFormat {
				t.Errorf("ParseFileArg(%q) format = %q, want %q", tt.input, gotFormat, tt.expectedFormat)
			}
			if gotPath != tt.expectedPath {
				t.Errorf("ParseFileArg(%q) path = %q, want %q", tt.input, gotPath, tt.expectedPath)
			}
		})
	}
}
