package services

// keymap.go maps robotn/gohook's normalized key/mouse codes to the rdev-style
// names the BongoCat frontend expects (the same names used for the model key
// image files, e.g. "KeyA", "ControlLeft", "Num1", "Space").
//
// The original Tauri backend used the `rdev` crate and emitted
// `format!("{:?}", key)` — the Debug name of the rdev Key enum. The frontend's
// `supportKeys` map and `assets/models/**/resources/{left,right}-keys/*.png`
// are keyed by those exact names. To stay drop-in compatible, gohook events are
// translated to the same strings here.
//
// Source of the gohook codes: github.com/vcaesar/keycode (Keycode / MouseMap).
//
// KNOWN MVP LIMITATIONS (see plan risk #1 — verify on real hardware per-OS):
//   - gohook's normalized keycode does NOT distinguish left/right Control
//     (single "ctrl" = 29). We emit "ControlLeft". The frontend's
//     getSupportedKey() collapses unsupported "ControlLeft" → "Control", so a
//     model exposing only "Control" still animates.
//   - CapsLock has no entry in gohook's normalized map; it needs a per-OS
//     Rawcode fallback (TODO). It is currently undetected.
//   - Scancode 14 (gohook "delete") is physically Backspace on PC layouts; we
//     map it to "Backspace". The forward-delete key is not in the normalized map.

// keycodeToName translates an event.Keycode (gohook normalized) to an rdev name.
var keycodeToName = map[uint16]string{
	// number row
	41: "BackQuote",
	2:  "Num1",
	3:  "Num2",
	4:  "Num3",
	5:  "Num4",
	6:  "Num5",
	7:  "Num6",
	8:  "Num7",
	9:  "Num8",
	10: "Num9",
	11: "Num0",
	12: "Minus",
	13: "Equal",

	// QWERTY row
	16: "KeyQ",
	17: "KeyW",
	18: "KeyE",
	19: "KeyR",
	20: "KeyT",
	21: "KeyY",
	22: "KeyU",
	23: "KeyI",
	24: "KeyO",
	25: "KeyP",
	26: "LeftBracket",
	27: "RightBracket",
	43: "BackSlash",

	// ASDF row
	30: "KeyA",
	31: "KeyS",
	32: "KeyD",
	33: "KeyF",
	34: "KeyG",
	35: "KeyH",
	36: "KeyJ",
	37: "KeyK",
	38: "KeyL",
	39: "SemiColon",
	40: "Quote",

	// ZXCV row
	44: "KeyZ",
	45: "KeyX",
	46: "KeyC",
	47: "KeyV",
	48: "KeyB",
	49: "KeyN",
	50: "KeyM",
	51: "Comma",
	52: "Dot",
	53: "Slash",

	// function keys
	59: "F1",
	60: "F2",
	61: "F3",
	62: "F4",
	63: "F5",
	64: "F6",
	65: "F7",
	66: "F8",
	67: "F9",
	68: "F10",
	69: "F11",
	70: "F12",

	// control / whitespace / modifiers
	1:    "Escape",
	14:   "Backspace",
	15:   "Tab",
	28:   "Return",
	29:   "ControlLeft",
	42:   "ShiftLeft",
	54:   "ShiftRight",
	57:   "Space",
	56:   "Alt",
	3640: "AltGr",
	3675: "MetaLeft",
	3676: "MetaRight",

	// arrows
	57416: "UpArrow",
	57424: "DownArrow",
	57419: "LeftArrow",
	57421: "RightArrow",

	// numeric keypad (mapped onto the main names so the model still reacts)
	79:   "Num1",
	80:   "Num2",
	81:   "Num3",
	75:   "Num4",
	76:   "Num5",
	77:   "Num6",
	71:   "Num7",
	72:   "Num8",
	73:   "Num9",
	82:   "Num0",
	74:   "Minus",
	3612: "Return",
	3637: "Slash",
}

// mouseButtonToName maps gohook MouseMap button codes to rdev button names.
var mouseButtonToName = map[uint16]string{
	1: "Left",
	2: "Right",
	3: "Middle",
}

// keyName returns the rdev-style name for a gohook keycode, or "" if unmapped.
func keyName(code uint16) string {
	return keycodeToName[code]
}

// mouseButtonName returns the rdev-style name for a gohook mouse button code.
// Unknown buttons fall back to the rdev "Unknown(n)" debug form.
func mouseButtonName(code uint16) string {
	if name, ok := mouseButtonToName[code]; ok {
		return name
	}
	return "Unknown"
}
