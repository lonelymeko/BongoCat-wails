//go:build !windows

package services

// isElevated is a no-op on non-Windows platforms. The original plugin returned
// true (always "administrator") off Windows, so behaviour is preserved.
func isElevated() bool { return true }
