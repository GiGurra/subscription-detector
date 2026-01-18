//go:build !windows && !darwin

package internal

import (
	"os"
	"os/exec"
	"strings"
)

// skipSystemLocale can be set to true in tests to skip OS-level locale detection
var skipSystemLocale = false

// detectSystemLocale returns the system locale string on Unix-like systems.
// Priority:
// 1. Environment variables: LC_MONETARY > LC_ALL > LANG
// 2. `locale` command output for LC_MONETARY
// 3. WSL: Windows locale via PowerShell
// Returns empty string if no valid locale is found.
func detectSystemLocale() string {
	// First check env vars
	for _, envVar := range []string{"LC_MONETARY", "LC_ALL", "LANG"} {
		locale := os.Getenv(envVar)
		if isValidLocale(locale) {
			return locale
		}
	}

	if skipSystemLocale {
		return ""
	}

	// Try `locale` command to get LC_MONETARY (may have region even if env var doesn't)
	if locale := getLocaleFromCommand(); locale != "" {
		return locale
	}

	// Check if running in WSL and get Windows locale
	if locale := getWSLWindowsLocale(); locale != "" {
		return locale
	}

	return ""
}

// isValidLocale checks if a locale string is usable for currency detection.
// Filters out empty, "C", "POSIX", and locales without region (like "C.UTF-8").
func isValidLocale(locale string) bool {
	if locale == "" || locale == "C" || locale == "POSIX" {
		return false
	}
	// Filter out C.UTF-8 and similar (no region code)
	if strings.HasPrefix(locale, "C.") {
		return false
	}
	return true
}

// getLocaleFromCommand runs `locale` and extracts LC_MONETARY value.
func getLocaleFromCommand() string {
	out, err := exec.Command("locale").Output()
	if err != nil {
		return ""
	}

	// Parse output like: LC_MONETARY="sv_SE.UTF-8" or LC_MONETARY=sv_SE.UTF-8
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "LC_MONETARY=") {
			locale := strings.TrimPrefix(line, "LC_MONETARY=")
			locale = strings.Trim(locale, "\"")
			if isValidLocale(locale) {
				return locale
			}
		}
	}
	return ""
}

// getWSLWindowsLocale detects if running in WSL and gets Windows locale via PowerShell.
func getWSLWindowsLocale() string {
	// Check if running in WSL by looking for "microsoft" or "WSL" in /proc/version
	version, err := os.ReadFile("/proc/version")
	if err != nil {
		return ""
	}
	versionLower := strings.ToLower(string(version))
	if !strings.Contains(versionLower, "microsoft") && !strings.Contains(versionLower, "wsl") {
		return ""
	}

	// Get Windows locale via PowerShell (returns "en-US", "sv-SE", etc.)
	// Try multiple paths since powershell.exe may not be in PATH
	var out []byte
	for _, ps := range []string{
		"powershell.exe",
		"/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe",
	} {
		out, err = exec.Command(ps, "-NoProfile", "-c", "(Get-Culture).Name").Output()
		if err == nil {
			break
		}
	}
	if err != nil {
		return ""
	}

	locale := strings.TrimSpace(string(out))
	if locale == "" {
		return ""
	}

	// PowerShell returns BCP 47 format (en-US), convert to POSIX format (en_US)
	// for consistency with parseCurrencyFromLocale
	return strings.Replace(locale, "-", "_", 1)
}
