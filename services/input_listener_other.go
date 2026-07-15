//go:build !darwin

package services

func startPlatformInputListener(_ *DeviceService) {}

// platformUsesGohook reports whether the gohook run loop should be started.
// On Windows/Linux gohook is the input backend, so it must run.
func platformUsesGohook() bool {
	return true
}

func handleHookKeyboardEvents() bool {
	return true
}

func handleHookMouseEvents() bool {
	return true
}
