package internal

import (
	"os"
	"strings"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// Currency represents a currency with its formatting rules
type Currency struct {
	Code    string // "SEK", "USD", "EUR"
	unit    currency.Unit
	tag     language.Tag
	printer *message.Printer
}

// currencyToLocale maps currency codes to their "home" locale for formatting
var currencyToLocale = map[string]language.Tag{
	"SEK": language.Swedish,
	"USD": language.AmericanEnglish,
	"EUR": language.German, // Uses space as thousand separator
	"GBP": language.BritishEnglish,
	"NOK": language.Norwegian,
	"DKK": language.Danish,
	"CHF": language.German,
	"JPY": language.Japanese,
	"CAD": language.CanadianFrench, // Uses space as thousand separator
	"AUD": language.MustParse("en-AU"),
}

// Country to currency code mapping for locale detection
var countryToCurrency = map[string]string{
	"SE": "SEK", // Sweden
	"US": "USD", // United States
	"DE": "EUR", // Germany
	"FR": "EUR", // France
	"IT": "EUR", // Italy
	"ES": "EUR", // Spain
	"NL": "EUR", // Netherlands
	"AT": "EUR", // Austria
	"FI": "EUR", // Finland
	"GB": "GBP", // United Kingdom
	"NO": "NOK", // Norway
	"DK": "DKK", // Denmark
	"CH": "CHF", // Switzerland
	"JP": "JPY", // Japan
	"CA": "CAD", // Canada
	"AU": "AUD", // Australia
}

// symbolOverrides provides custom symbols where x/text defaults aren't ideal
var symbolOverrides = map[string]string{
	"SEK": "kr",
	"NOK": "kr",
	"DKK": "kr",
}

// GetCurrency returns the Currency for a given code.
func GetCurrency(code string) Currency {
	code = strings.ToUpper(code)

	// Get the currency unit (validates the code)
	unit, err := currency.ParseISO(code)
	isUnknown := err != nil
	if isUnknown {
		unit = currency.USD // fallback unit for number formatting only
	}

	// Get the locale for this currency
	tag, ok := currencyToLocale[code]
	if !ok {
		tag = language.English // default to English formatting
	}

	c := Currency{
		Code:    code,
		unit:    unit,
		tag:     tag,
		printer: message.NewPrinter(tag),
	}

	// For unknown currencies, override the symbol to use the code
	if isUnknown {
		symbolOverrides[code] = code
	}

	return c
}

// DetectSystemCurrency attempts to detect the system currency from locale environment variables.
// Checks LC_MONETARY, LC_ALL, and LANG in that order.
// Returns empty string if detection fails.
func DetectSystemCurrency() string {
	// Check environment variables in priority order
	for _, envVar := range []string{"LC_MONETARY", "LC_ALL", "LANG"} {
		locale := os.Getenv(envVar)
		if locale == "" || locale == "C" || locale == "POSIX" {
			continue
		}

		// Parse locale format: language_COUNTRY.encoding or language_COUNTRY
		// Examples: sv_SE.UTF-8, en_US.UTF-8, de_DE
		country := parseCountryFromLocale(locale)
		if country != "" {
			if curr, ok := countryToCurrency[country]; ok {
				return curr
			}
		}
	}
	return ""
}

// parseCountryFromLocale extracts the country code from a locale string.
// Examples: "sv_SE.UTF-8" -> "SE", "en_US" -> "US"
func parseCountryFromLocale(locale string) string {
	// Remove encoding suffix (everything after .)
	if idx := strings.Index(locale, "."); idx != -1 {
		locale = locale[:idx]
	}

	// Remove modifier suffix (everything after @)
	if idx := strings.Index(locale, "@"); idx != -1 {
		locale = locale[:idx]
	}

	// Find underscore separating language and country
	if idx := strings.Index(locale, "_"); idx != -1 && idx+1 < len(locale) {
		country := locale[idx+1:]
		// Country codes are 2 uppercase letters
		if len(country) >= 2 {
			return strings.ToUpper(country[:2])
		}
	}

	return ""
}

// getSymbol returns the currency symbol, using overrides where needed
func (c Currency) getSymbol() string {
	if sym, ok := symbolOverrides[c.Code]; ok {
		return sym
	}
	// Use x/text to get the narrow symbol
	return c.printer.Sprint(currency.NarrowSymbol(c.unit))
}

// isPrefix returns true if this currency symbol should be placed before the amount
func (c Currency) isPrefix() bool {
	switch c.Code {
	case "USD", "GBP", "JPY", "CAD", "AUD":
		return true
	default:
		return false
	}
}

// Format formats a single amount with the currency symbol
func (c Currency) Format(amount float64) string {
	// Use x/text/number for proper locale-aware formatting
	formatted := c.printer.Sprint(number.Decimal(amount, number.MaxFractionDigits(0)))
	symbol := c.getSymbol()

	if c.isPrefix() {
		return symbol + formatted
	}
	return formatted + " " + symbol
}

// FormatRange formats a range of amounts (min-max) with the currency symbol
func (c Currency) FormatRange(min, max float64) string {
	minStr := c.printer.Sprint(number.Decimal(min, number.MaxFractionDigits(0)))
	maxStr := c.printer.Sprint(number.Decimal(max, number.MaxFractionDigits(0)))
	symbol := c.getSymbol()

	if c.isPrefix() {
		return symbol + minStr + "-" + symbol + maxStr
	}
	return minStr + "-" + maxStr + " " + symbol
}
