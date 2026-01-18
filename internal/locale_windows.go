//go:build windows

package internal

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procGetUserDefaultLocale  = kernel32.NewProc("GetUserDefaultLocaleName")
)

// detectSystemLocale returns the system locale string on Windows.
// Uses GetUserDefaultLocaleName which returns BCP 47 format like "en-US".
// Returns empty string if detection fails.
func detectSystemLocale() string {
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
