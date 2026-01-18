//go:build windows

package internal

import (
	"os"
	"syscall"
	"unsafe"
)

// skipSystemLocale can be set to true in tests to skip OS-level locale detection
var skipSystemLocale = false

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGetUserDefaultLocale = kernel32.NewProc("GetUserDefaultLocaleName")
)

// detectSystemLocale returns the system locale string on Windows.
// First checks environment variables (for testing and WSL compatibility),
// then falls back to GetUserDefaultLocaleName API.
// Returns empty string if detection fails.
func detectSystemLocale() string {
	// Check env vars first (for testing and cross-platform consistency)
	// Priority: LC_MONETARY (most specific) > LC_ALL > LANG
	for _, envVar := range []string{"LC_MONETARY", "LC_ALL", "LANG"} {
		locale := os.Getenv(envVar)
		if locale != "" && locale != "C" && locale != "POSIX" {
			return locale
		}
	}

	// Fall back to Windows API (skip in tests)
	if skipSystemLocale {
		return ""
	}

	const maxLen = 85 // LOCALE_NAME_MAX_LENGTH

	buf := make([]uint16, maxLen)
	ret, _, _ := procGetUserDefaultLocale.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(maxLen),
	)

	if ret == 0 {
		return ""
	}

	// Convert UTF-16 to string
	return syscall.UTF16ToString(buf)
}
