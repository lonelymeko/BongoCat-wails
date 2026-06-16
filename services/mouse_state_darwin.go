package services

/*
#cgo darwin LDFLAGS: -framework CoreGraphics
#include <CoreGraphics/CoreGraphics.h>

static bool bongo_mouse_button_down(int button) {
	return CGEventSourceButtonState(kCGEventSourceStateCombinedSessionState, (CGMouseButton)button);
}
*/
import "C"

func mouseButtonDown(name string) (bool, bool) {
	switch name {
	case "Left":
		return bool(C.bongo_mouse_button_down(0)), true
	case "Right":
		return bool(C.bongo_mouse_button_down(1)), true
	case "Middle":
		return bool(C.bongo_mouse_button_down(2)), true
	default:
		return false, false
	}
}
