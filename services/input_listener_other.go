//go:build !darwin

package services

func startPlatformInputListener(_ *DeviceService) {}

func handleHookKeyboardEvents() bool {
	return true
}

func handleHookMouseEvents() bool {
	return true
}
