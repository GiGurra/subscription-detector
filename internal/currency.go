package internal

import (
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

// symbolOverrides provides custom symbols where x/text defaults aren't ideal
var symbolOverrides = map[string]string{
	"SEK": "kr",
	"NOK": "kr",
	"DKK": "kr",
	"ISK": "kr",
}

// defaultLocaleForCurrency provides fallback locales when currency is specified
// without a system locale (e.g., --currency USD). Uses a "home" locale for each currency.
var defaultLocaleForCurrency = map[string]language.Tag{
	"SEK": language.Swedish,
	"USD": language.AmericanEnglish,
	"EUR": language.German,
	"GBP": language.BritishEnglish,
	"NOK": language.Norwegian,
	"DKK": language.Danish,
	"CHF": language.German,
	"JPY": language.Japanese,
	"CAD": language.CanadianFrench,
	"AUD": language.MustParse("en-AU"),
	"BRL": language.BrazilianPortuguese,
	"MXN": language.LatinAmericanSpanish,
	"INR": language.MustParse("en-IN"),
	"CNY": language.Chinese,
	"KRW": language.Korean,
	"PLN": language.Polish,
	"CZK": language.Czech,
	"HUF": language.Hungarian,
	"RUB": language.Russian,
	"TRY": language.Turkish,
	"ZAR": language.MustParse("en-ZA"),
	"NZD": language.MustParse("en-NZ"),
	"SGD": language.MustParse("en-SG"),
	"HKD": language.MustParse("zh-HK"),
	"THB": language.Thai,
}

// detectedLocale stores the system locale when auto-detected, so we can use it for formatting
var detectedLocale language.Tag

// GetCurrency returns the Currency for a given code.
func GetCurrency(code string) Currency {
	code = strings.ToUpper(code)

	// Get the currency unit (validates the code)
	unit, err := currency.ParseISO(code)
	isUnknown := err != nil
	if isUnknown {
		unit = currency.USD // fallback unit for number formatting only
	}

	// Determine the locale for formatting
	// Priority: detected system locale > default locale for currency > English
	var tag language.Tag
	if detectedLocale != language.Und {
		tag = detectedLocale
	} else if t, ok := defaultLocaleForCurrency[code]; ok {
		tag = t
	} else {
		tag = language.English
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

// GetCurrencyWithLocale returns a Currency with a specific locale for formatting.
func GetCurrencyWithLocale(code string, tag language.Tag) Currency {
	code = strings.ToUpper(code)

	unit, err := currency.ParseISO(code)
	isUnknown := err != nil
	if isUnknown {
		unit = currency.USD
	}

	c := Currency{
		Code:    code,
		unit:    unit,
		tag:     tag,
		printer: message.NewPrinter(tag),
	}

	if isUnknown {
		symbolOverrides[code] = code
	}

	return c
}

// DetectSystemCurrency attempts to detect the system currency from the OS locale.
// On Linux/Unix: checks LANGUAGE, LC_ALL, LC_MONETARY, LC_MESSAGES, LANG env vars
// On macOS: checks env vars first, then falls back to AppleLocale system preference
// On Windows: uses GetUserDefaultLocaleName API
// Returns empty string if detection fails.
// Also sets detectedLocale for use in formatting.
func DetectSystemCurrency() string {
	locale := detectSystemLocale()
	if locale == "" {
		return ""
	}

	// Try to get currency and locale from the locale string
	currCode, tag := parseCurrencyFromLocale(locale)
	if currCode != "" {
		detectedLocale = tag
		return currCode
	}
	return ""
}

// parseCurrencyFromLocale extracts currency code and language tag from a locale string.
// Examples: "sv_SE.UTF-8" -> ("SEK", sv-SE), "pt_BR.UTF-8" -> ("BRL", pt-BR)
func parseCurrencyFromLocale(locale string) (string, language.Tag) {
	// Remove encoding suffix (everything after .)
	base := locale
	if idx := strings.Index(base, "."); idx != -1 {
		base = base[:idx]
	}

	// Remove modifier suffix (everything after @)
	if idx := strings.Index(base, "@"); idx != -1 {
		base = base[:idx]
	}

	// Convert to BCP 47 format: "sv_SE" -> "sv-SE"
	tagStr := strings.Replace(base, "_", "-", 1)
	tag, err := language.Parse(tagStr)
	if err != nil {
		return "", language.Und
	}

	// Extract region and get currency for that region
	_, _, region := tag.Raw()
	if region.String() == "" || region.String() == "ZZ" {
		return "", language.Und
	}

	unit, ok := currency.FromRegion(region)
	if !ok {
		return "", language.Und
	}

	return unit.String(), tag
}

// getSymbol returns the currency symbol, using overrides where needed
func (c Currency) getSymbol() string {
	if sym, ok := symbolOverrides[c.Code]; ok {
		return sym
	}
	// Use x/text to get the narrow symbol
	return c.printer.Sprint(currency.NarrowSymbol(c.unit))
}

// isPrefix returns true if this currency symbol should be placed before the amount.
// Note: golang.org/x/text/currency doesn't implement symbol positioning from CLDR patterns
// (see TODO in x/text/internal/number/pattern.go for ¤ handling). Until that's fixed,
// we maintain this list of prefix currencies manually.
func (c Currency) isPrefix() bool {
	switch c.Code {
	case "USD", "GBP", "JPY", "CAD", "AUD", "MXN", "HKD", "SGD", "NZD", "ZAR":
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
