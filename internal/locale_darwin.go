//go:build darwin

package internal

import (
	"os"
	"os/exec"
	"strings"
)

// detectSystemLocale returns the system locale string on macOS.
// First checks environment variables (for terminal overrides),
// then falls back to macOS defaults (AppleLocale).
// Returns empty string if no valid locale is found.
func detectSystemLocale() string {
	// First check env vars (user might have overridden in terminal)
	for _, envVar := range []string{"LC_ALL", "LC_MONETARY", "LANG"} {
		locale := os.Getenv(envVar)
		if locale != "" && locale != "C" && locale != "POSIX" {
			return locale
		}
	}

	// Fall back to macOS system preference
	out, err := exec.Command("defaults", "read", "-g", "AppleLocale").Output()
	if err != nil {
		return ""
	}

	locale := strings.TrimSpace(string(out))
	if locale == "" {
		return ""
	}

	// AppleLocale format is like "en_US" or "sv_SE" - already what we need
	return locale
}
