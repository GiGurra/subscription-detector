//go:build !windows && !darwin

package internal

import "os"

// skipSystemLocale can be set to true in tests to skip OS-level locale detection
var skipSystemLocale = false

// detectSystemLocale returns the system locale string on Unix-like systems.
// For currency detection, priority is: LC_MONETARY (most specific), LC_ALL, LANG.
// Returns empty string if no valid locale is found.
func detectSystemLocale() string {
	for _, envVar := range []string{"LC_MONETARY", "LC_ALL", "LANG"} {
		locale := os.Getenv(envVar)
		if locale != "" && locale != "C" && locale != "POSIX" {
			return locale
		}
	}
	return ""
}
