//go:build !darwin

package services

func mouseButtonDown(_ string) (bool, bool) {
	return false, false
}
