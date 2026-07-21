package services

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework ApplicationServices -framework CoreFoundation
#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdbool.h>

extern void bongoMousePress(int button);
extern void bongoMouseRelease(int button);
extern void bongoMouseMove(double x, double y);
extern void bongoKeyPress(int keycode);
extern void bongoKeyRelease(int keycode);
extern void bongoFlagsChanged(int keycode, unsigned long flags);

static bool bongo_emit_current_mouse_location(void) {
	CGEventRef event = CGEventCreate(NULL);
	if (event == NULL) {
		return false;
	}

	CGPoint point = CGEventGetLocation(event);
	CFRelease(event);
	bongoMouseMove(point.x, point.y);
	return true;
}

static CGEventRef bongo_input_tap_callback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *refcon) {
	CGPoint point = CGEventGetLocation(event);

	switch (type) {
	case kCGEventKeyDown:
		bongoKeyPress((int)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode));
		break;
	case kCGEventKeyUp:
		bongoKeyRelease((int)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode));
		break;
	case kCGEventFlagsChanged:
		bongoFlagsChanged(
			(int)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode),
			(unsigned long)CGEventGetFlags(event)
		);
		break;
	case kCGEventLeftMouseDown:
		bongoMousePress(0);
		break;
	case kCGEventRightMouseDown:
		bongoMousePress(1);
		break;
	case kCGEventOtherMouseDown:
		bongoMousePress((int)CGEventGetIntegerValueField(event, kCGMouseEventButtonNumber));
		break;
	case kCGEventLeftMouseUp:
		bongoMouseRelease(0);
		break;
	case kCGEventRightMouseUp:
		bongoMouseRelease(1);
		break;
	case kCGEventOtherMouseUp:
		bongoMouseRelease((int)CGEventGetIntegerValueField(event, kCGMouseEventButtonNumber));
		break;
	case kCGEventMouseMoved:
	case kCGEventLeftMouseDragged:
	case kCGEventRightMouseDragged:
	case kCGEventOtherMouseDragged:
		bongoMouseMove(point.x, point.y);
		break;
	case kCGEventTapDisabledByTimeout:
	case kCGEventTapDisabledByUserInput:
		CGEventTapEnable((CFMachPortRef)refcon, true);
		break;
	default:
		break;
	}

	return event;
}

static bool bongo_start_input_tap(void) {
	CGEventMask mask =
		CGEventMaskBit(kCGEventKeyDown) |
		CGEventMaskBit(kCGEventKeyUp) |
		CGEventMaskBit(kCGEventFlagsChanged) |
		CGEventMaskBit(kCGEventLeftMouseDown) |
		CGEventMaskBit(kCGEventLeftMouseUp) |
		CGEventMaskBit(kCGEventRightMouseDown) |
		CGEventMaskBit(kCGEventRightMouseUp) |
		CGEventMaskBit(kCGEventOtherMouseDown) |
		CGEventMaskBit(kCGEventOtherMouseUp) |
		CGEventMaskBit(kCGEventMouseMoved) |
		CGEventMaskBit(kCGEventLeftMouseDragged) |
		CGEventMaskBit(kCGEventRightMouseDragged) |
		CGEventMaskBit(kCGEventOtherMouseDragged);

	CFMachPortRef tap = CGEventTapCreate(
		kCGSessionEventTap,
		kCGHeadInsertEventTap,
		kCGEventTapOptionListenOnly,
		mask,
		bongo_input_tap_callback,
		NULL
	);
	if (tap == NULL) {
		return false;
	}

	CFRunLoopSourceRef source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, tap, 0);
	if (source == NULL) {
		CFRelease(tap);
		return false;
	}

	CFRunLoopAddSource(CFRunLoopGetCurrent(), source, kCFRunLoopCommonModes);
	CGEventTapEnable(tap, true);
	CFRunLoopRun();

	CFRelease(source);
	CFRelease(tap);
	return true;
}
*/
import "C"

var darwinInputService *DeviceService
var darwinModifierState = map[string]bool{}

func startPlatformInputListener(s *DeviceService) {
	darwinInputService = s

	C.bongo_emit_current_mouse_location()

	go C.bongo_start_input_tap()
}

// platformUsesGohook reports whether the gohook run loop should be started.
// macOS uses the native CGEventTap above instead, and starting gohook here
// only adds a crash-prone second run loop, so it is disabled.
func platformUsesGohook() bool {
	return false
}

func handleHookKeyboardEvents() bool {
	return false
}

func handleHookMouseEvents() bool {
	return false
}

//export bongoMousePress
func bongoMousePress(button C.int) {
	if darwinInputService == nil {
		return
	}

	name := cgMouseButtonName(int(button))
	darwinInputService.emitMousePress(name)
}

//export bongoMouseRelease
func bongoMouseRelease(button C.int) {
	if darwinInputService == nil {
		return
	}

	name := cgMouseButtonName(int(button))
	darwinInputService.emitMouseRelease(name)
}

//export bongoMouseMove
func bongoMouseMove(x, y C.double) {
	if darwinInputService == nil {
		return
	}

	darwinInputService.handleMouseMove(float64(x), float64(y))
}

//export bongoKeyPress
func bongoKeyPress(keycode C.int) {
	if darwinInputService == nil {
		return
	}

	darwinInputService.emitKeyboardPress(darwinKeyName(int(keycode)))
}

//export bongoKeyRelease
func bongoKeyRelease(keycode C.int) {
	if darwinInputService == nil {
		return
	}

	darwinInputService.emitKeyboardRelease(darwinKeyName(int(keycode)))
}

//export bongoFlagsChanged
func bongoFlagsChanged(keycode C.int, flags C.ulong) {
	if darwinInputService == nil {
		return
	}

	name, pressed := darwinModifierStateForKey(int(keycode), uint64(flags))
	if name == "" || pressed == darwinModifierState[name] {
		return
	}

	if pressed {
		darwinInputService.emitKeyboardPress(name)
	} else {
		darwinInputService.emitKeyboardRelease(name)
	}
	darwinModifierState[name] = pressed
}

func cgMouseButtonName(button int) string {
	switch button {
	case 0:
		return "Left"
	case 1:
		return "Right"
	case 2:
		return "Middle"
	default:
		return "Unknown"
	}
}

func darwinKeyName(keycode int) string {
	switch keycode {
	case 10:
		return "BackQuote"
	case 0:
		return "KeyA"
	case 1:
		return "KeyS"
	case 2:
		return "KeyD"
	case 3:
		return "KeyF"
	case 4:
		return "KeyH"
	case 5:
		return "KeyG"
	case 6:
		return "KeyZ"
	case 7:
		return "KeyX"
	case 8:
		return "KeyC"
	case 9:
		return "KeyV"
	case 11:
		return "KeyB"
	case 12:
		return "KeyQ"
	case 13:
		return "KeyW"
	case 14:
		return "KeyE"
	case 15:
		return "KeyR"
	case 16:
		return "KeyY"
	case 17:
		return "KeyT"
	case 18:
		return "Num1"
	case 19:
		return "Num2"
	case 20:
		return "Num3"
	case 21:
		return "Num4"
	case 22:
		return "Num6"
	case 23:
		return "Num5"
	case 25:
		return "Num9"
	case 26:
		return "Num7"
	case 28:
		return "Num8"
	case 29:
		return "Num0"
	case 31:
		return "KeyO"
	case 32:
		return "KeyU"
	case 34:
		return "KeyI"
	case 35:
		return "KeyP"
	case 37:
		return "KeyL"
	case 38:
		return "KeyJ"
	case 40:
		return "KeyK"
	case 45:
		return "KeyN"
	case 46:
		return "KeyM"
	case 44:
		return "Slash"
	case 47:
		return "Dot"
	case 43:
		return "Comma"
	case 24:
		return "Equal"
	case 27:
		return "Minus"
	case 33:
		return "LeftBracket"
	case 30:
		return "RightBracket"
	case 42:
		return "BackSlash"
	case 41:
		return "SemiColon"
	case 39:
		return "Quote"
	case 48:
		return "Tab"
	case 49:
		return "Space"
	case 51:
		return "Backspace"
	case 53:
		return "Escape"
	case 55:
		return "Meta"
	case 56:
		return "ShiftLeft"
	case 57:
		return "CapsLock"
	case 58:
		return "Alt"
	case 59:
		return "ControlLeft"
	case 60:
		return "ShiftRight"
	case 61:
		return "AltGr"
	case 62:
		return "ControlRight"
	case 63:
		return "Fn"
	case 76, 36:
		return "Return"
	case 96:
		return "F5"
	case 97:
		return "F6"
	case 98:
		return "F7"
	case 99:
		return "F3"
	case 100:
		return "F8"
	case 101:
		return "F9"
	case 103:
		return "F11"
	case 109:
		return "F10"
	case 111:
		return "F12"
	case 115:
		return "Home"
	case 116:
		return "PageUp"
	case 117:
		return "Delete"
	case 119:
		return "End"
	case 120:
		return "F2"
	case 121:
		return "PageDown"
	case 122:
		return "F1"
	case 123:
		return "LeftArrow"
	case 124:
		return "RightArrow"
	case 125:
		return "DownArrow"
	case 126:
		return "UpArrow"
	default:
		return ""
	}
}

func darwinModifierStateForKey(keycode int, flags uint64) (string, bool) {
	const (
		maskAlphaShift  = 1 << 16
		maskShift       = 1 << 17
		maskControl     = 1 << 18
		maskAlternate   = 1 << 19
		maskCommand     = 1 << 20
		maskSecondaryFn = 1 << 23
	)

	switch keycode {
	case 55, 54:
		return "Meta", flags&maskCommand != 0
	case 56:
		return "ShiftLeft", flags&maskShift != 0
	case 60:
		return "ShiftRight", flags&maskShift != 0
	case 57:
		return "CapsLock", flags&maskAlphaShift != 0
	case 58:
		return "Alt", flags&maskAlternate != 0
	case 61:
		return "AltGr", flags&maskAlternate != 0
	case 59:
		return "ControlLeft", flags&maskControl != 0
	case 62:
		return "ControlRight", flags&maskControl != 0
	case 63:
		return "Fn", flags&maskSecondaryFn != 0
	default:
		return "", false
	}
}
