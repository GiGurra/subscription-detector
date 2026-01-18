package internal

import (
	"os"
	"testing"

	"golang.org/x/text/language"
)

// resetDetectedLocale resets the global detectedLocale for testing
func resetDetectedLocale() {
	detectedLocale = language.Und
}

func TestGetCurrency_KnownCurrencies(t *testing.T) {
	resetDetectedLocale()
	codes := []string{"SEK", "USD", "EUR", "GBP", "NOK", "DKK", "CHF", "JPY", "CAD", "AUD", "BRL"}

	for _, code := range codes {
		t.Run(code, func(t *testing.T) {
			c := GetCurrency(code)
			if c.Code != code {
				t.Errorf("Code = %q, want %q", c.Code, code)
			}
			// Verify it can format without panicking
			_ = c.Format(1234)
			_ = c.FormatRange(100, 200)
		})
	}
}

func TestGetCurrency_CaseInsensitive(t *testing.T) {
	resetDetectedLocale()
	tests := []string{"sek", "Sek", "SEK", "seK"}
	for _, code := range tests {
		c := GetCurrency(code)
		if c.Code != "SEK" {
			t.Errorf("GetCurrency(%q).Code = %q, want SEK", code, c.Code)
		}
	}
}

func TestGetCurrency_Unknown(t *testing.T) {
	resetDetectedLocale()
	c := GetCurrency("XYZ")
	if c.Code != "XYZ" {
		t.Errorf("Code = %q, want XYZ", c.Code)
	}
	// Unknown currency should use code as symbol
	formatted := c.Format(100)
	if formatted != "100 XYZ" {
		t.Errorf("Format(100) = %q, want %q", formatted, "100 XYZ")
	}
}

func TestCurrency_Format(t *testing.T) {
	resetDetectedLocale()
	// Note: x/text uses non-breaking space (U+00A0) for Swedish/Norwegian thousand separators
	// and fullwidth yen (￥) for Japanese
	nbsp := "\u00a0" // non-breaking space

	tests := []struct {
		name   string
		code   string
		amount float64
		want   string
	}{
		{"SEK small", "SEK", 100, "100 kr"},
		{"SEK thousands", "SEK", 1234, "1" + nbsp + "234 kr"},
		{"SEK large", "SEK", 12345, "12" + nbsp + "345 kr"},
		{"SEK very large", "SEK", 1234567, "1" + nbsp + "234" + nbsp + "567 kr"},
		{"USD small", "USD", 100, "$100"},
		{"USD thousands", "USD", 1234, "$1,234"},
		{"USD large", "USD", 12345, "$12,345"},
		{"EUR small", "EUR", 100, "100 €"},
		{"EUR thousands", "EUR", 1234, "1.234 €"},
		{"GBP small", "GBP", 100, "£100"},
		{"GBP thousands", "GBP", 1234, "£1,234"},
		{"CHF small", "CHF", 100, "100 CHF"},
		{"CHF thousands", "CHF", 1234, "1.234 CHF"},
		{"JPY thousands", "JPY", 1000, "￥1,000"},
		{"JPY large", "JPY", 123456, "￥123,456"},
		{"BRL small", "BRL", 100, "100 R$"},
		{"BRL thousands", "BRL", 1234, "1.234 R$"},
		{"Unknown small", "XYZ", 100, "100 XYZ"},
		{"Unknown thousands", "XYZ", 1234, "1,234 XYZ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := GetCurrency(tt.code)
			got := c.Format(tt.amount)
			if got != tt.want {
				t.Errorf("Format(%v) = %q, want %q", tt.amount, got, tt.want)
			}
		})
	}
}

func TestCurrency_FormatRange(t *testing.T) {
	resetDetectedLocale()
	nbsp := "\u00a0" // non-breaking space

	tests := []struct {
		name string
		code string
		min  float64
		max  float64
		want string
	}{
		{"SEK small range", "SEK", 100, 150, "100-150 kr"},
		{"SEK thousands range", "SEK", 1000, 1500, "1" + nbsp + "000-1" + nbsp + "500 kr"},
		{"USD small range", "USD", 100, 150, "$100-$150"},
		{"USD thousands range", "USD", 1000, 1500, "$1,000-$1,500"},
		{"EUR small range", "EUR", 50, 75, "50-75 €"},
		{"EUR thousands range", "EUR", 1000, 2000, "1.000-2.000 €"},
		{"BRL small range", "BRL", 100, 200, "100-200 R$"},
		{"BRL thousands range", "BRL", 1000, 2000, "1.000-2.000 R$"},
		{"Unknown small range", "XYZ", 10, 20, "10-20 XYZ"},
		{"Unknown thousands range", "XYZ", 1000, 2000, "1,000-2,000 XYZ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := GetCurrency(tt.code)
			got := c.FormatRange(tt.min, tt.max)
			if got != tt.want {
				t.Errorf("FormatRange(%v, %v) = %q, want %q", tt.min, tt.max, got, tt.want)
			}
		})
	}
}

func TestParseCurrencyFromLocale(t *testing.T) {
	tests := []struct {
		locale       string
		wantCurrency string
		wantTag      string
	}{
		{"sv_SE.UTF-8", "SEK", "sv-SE"},
		{"en_US.UTF-8", "USD", "en-US"},
		{"pt_BR.UTF-8", "BRL", "pt-BR"},
		{"de_DE", "EUR", "de-DE"},
		{"ja_JP.UTF-8", "JPY", "ja-JP"},
		{"en_GB.UTF-8", "GBP", "en-GB"},
		{"C", "", ""},
		{"en", "", ""},  // No region
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			gotCurrency, gotTag := parseCurrencyFromLocale(tt.locale)
			if gotCurrency != tt.wantCurrency {
				t.Errorf("parseCurrencyFromLocale(%q) currency = %q, want %q", tt.locale, gotCurrency, tt.wantCurrency)
			}
			if tt.wantTag != "" && gotTag.String() != tt.wantTag {
				t.Errorf("parseCurrencyFromLocale(%q) tag = %q, want %q", tt.locale, gotTag.String(), tt.wantTag)
			}
		})
	}
}

func TestDetectSystemCurrency(t *testing.T) {
	// Save original env vars
	origMonetary := os.Getenv("LC_MONETARY")
	origAll := os.Getenv("LC_ALL")
	origLang := os.Getenv("LANG")

	// Skip OS-level locale detection so tests are predictable across platforms
	skipSystemLocale = true

	// Cleanup after test
	defer func() {
		os.Setenv("LC_MONETARY", origMonetary)
		os.Setenv("LC_ALL", origAll)
		os.Setenv("LANG", origLang)
		resetDetectedLocale()
		skipSystemLocale = false
	}()

	tests := []struct {
		name         string
		lcMonetary   string
		lcAll        string
		lang         string
		wantCurrency string
	}{
		{
			name:         "LC_MONETARY takes priority",
			lcMonetary:   "sv_SE.UTF-8",
			lcAll:        "en_US.UTF-8",
			lang:         "de_DE.UTF-8",
			wantCurrency: "SEK",
		},
		{
			name:         "LC_ALL when LC_MONETARY empty",
			lcMonetary:   "",
			lcAll:        "en_US.UTF-8",
			lang:         "de_DE.UTF-8",
			wantCurrency: "USD",
		},
		{
			name:         "LANG as fallback",
			lcMonetary:   "",
			lcAll:        "",
			lang:         "de_DE.UTF-8",
			wantCurrency: "EUR",
		},
		{
			name:         "Norwegian krone",
			lcMonetary:   "nb_NO.UTF-8",
			lcAll:        "",
			lang:         "",
			wantCurrency: "NOK",
		},
		{
			name:         "British pound",
			lcMonetary:   "en_GB.UTF-8",
			lcAll:        "",
			lang:         "",
			wantCurrency: "GBP",
		},
		{
			name:         "Brazilian real",
			lcMonetary:   "pt_BR.UTF-8",
			lcAll:        "",
			lang:         "",
			wantCurrency: "BRL",
		},
		{
			name:         "Japanese yen",
			lcMonetary:   "ja_JP.UTF-8",
			lcAll:        "",
			lang:         "",
			wantCurrency: "JPY",
		},
		{
			name:         "No detection when all empty",
			lcMonetary:   "",
			lcAll:        "",
			lang:         "",
			wantCurrency: "",
		},
		{
			name:         "Skip C locale",
			lcMonetary:   "C",
			lcAll:        "POSIX",
			lang:         "sv_SE.UTF-8",
			wantCurrency: "SEK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetDetectedLocale()
			os.Setenv("LC_MONETARY", tt.lcMonetary)
			os.Setenv("LC_ALL", tt.lcAll)
			os.Setenv("LANG", tt.lang)

			got := DetectSystemCurrency()
			if got != tt.wantCurrency {
				t.Errorf("DetectSystemCurrency() = %q, want %q", got, tt.wantCurrency)
			}
		})
	}
}

func TestDetectSystemCurrency_SetsLocaleForFormatting(t *testing.T) {
	// Save original env vars
	origMonetary := os.Getenv("LC_MONETARY")
	origAll := os.Getenv("LC_ALL")
	origLang := os.Getenv("LANG")

	// Skip OS-level locale detection so tests are predictable across platforms
	skipSystemLocale = true

	defer func() {
		os.Setenv("LC_MONETARY", origMonetary)
		os.Setenv("LC_ALL", origAll)
		os.Setenv("LANG", origLang)
		resetDetectedLocale()
		skipSystemLocale = false
	}()

	// Set Brazilian locale
	resetDetectedLocale()
	os.Setenv("LC_MONETARY", "pt_BR.UTF-8")
	os.Setenv("LC_ALL", "")
	os.Setenv("LANG", "")

	currCode := DetectSystemCurrency()
	if currCode != "BRL" {
		t.Fatalf("DetectSystemCurrency() = %q, want BRL", currCode)
	}

	// Now GetCurrency should use Brazilian formatting
	c := GetCurrency("BRL")
	formatted := c.Format(1234)
	// Brazilian Portuguese uses period as thousand separator
	if formatted != "1.234 R$" {
		t.Errorf("Format(1234) = %q, want %q", formatted, "1.234 R$")
	}
}
