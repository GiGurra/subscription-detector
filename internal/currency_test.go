package internal

import (
	"os"
	"testing"
)

func TestGetCurrency_KnownCurrencies(t *testing.T) {
	codes := []string{"SEK", "USD", "EUR", "GBP", "NOK", "DKK", "CHF", "JPY", "CAD", "AUD"}

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
	tests := []string{"sek", "Sek", "SEK", "seK"}
	for _, code := range tests {
		c := GetCurrency(code)
		if c.Code != "SEK" {
			t.Errorf("GetCurrency(%q).Code = %q, want SEK", code, c.Code)
		}
	}
}

func TestGetCurrency_Unknown(t *testing.T) {
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

func TestParseCountryFromLocale(t *testing.T) {
	tests := []struct {
		locale string
		want   string
	}{
		{"sv_SE.UTF-8", "SE"},
		{"en_US.UTF-8", "US"},
		{"de_DE", "DE"},
		{"fr_FR.ISO-8859-1", "FR"},
		{"en_GB.UTF-8@euro", "GB"},
		{"C", ""},
		{"POSIX", ""},
		{"en", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			got := parseCountryFromLocale(tt.locale)
			if got != tt.want {
				t.Errorf("parseCountryFromLocale(%q) = %q, want %q", tt.locale, got, tt.want)
			}
		})
	}
}

func TestDetectSystemCurrency(t *testing.T) {
	// Save original env vars
	origMonetary := os.Getenv("LC_MONETARY")
	origAll := os.Getenv("LC_ALL")
	origLang := os.Getenv("LANG")

	// Cleanup after test
	defer func() {
		os.Setenv("LC_MONETARY", origMonetary)
		os.Setenv("LC_ALL", origAll)
		os.Setenv("LANG", origLang)
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
