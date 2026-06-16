//go:build windows

package services

import "golang.org/x/sys/windows"

// isElevated reports whether the process is running with an elevated
// (administrator) token. Mirrors the original admin-status plugin which queried
// TokenElevation via the Win32 API.
func isElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}
