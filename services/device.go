package services

import (
	"sync"

	hook "github.com/robotn/gohook"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// DeviceEventName is the event the frontend listens on (see constants/index.ts
// LISTEN_KEY.DEVICE_CHANGED and composables/useDevice.ts).
const DeviceEventName = "device-changed"

// deviceEvent mirrors the payload shape emitted by the original Tauri backend
// (src-tauri/src/core/device.rs): { kind, value }.
//
//	kind:  "MousePress" | "MouseRelease" | "MouseMove" | "KeyboardPress" | "KeyboardRelease"
//	value: string (key/button name) | { x, y } (mouse move)
type deviceEvent struct {
	Kind  string `json:"kind"`
	Value any    `json:"value"`
}

type cursorPoint struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}

// DeviceService replaces start_device_listening. It hooks global keyboard and
// mouse input via robotn/gohook and forwards each event to the frontend.
type DeviceService struct {
	app       *application.App
	mu        sync.Mutex
	listening bool
}

// NewDeviceService creates the service bound to the running application.
func NewDeviceService(app *application.App) *DeviceService {
	return &DeviceService{app: app}
}

// StartListening begins global device listening. It is idempotent: calling it
// again while already listening is a no-op (matches the IS_LISTENING guard in
// the original Rust implementation).
func (s *DeviceService) StartListening() {
	s.mu.Lock()
	if s.listening {
		s.mu.Unlock()
		return
	}
	s.listening = true
	s.mu.Unlock()

	go s.loop()
}

func (s *DeviceService) loop() {
	// hook.Start returns a channel of raw events. The hook runs its own native
	// run loop in C, so consuming from a goroutine is safe.
	//
	// macOS note: this requires the "Input Monitoring" permission to be granted
	// to the app, otherwise no events are delivered.
	evChan := hook.Start()
	defer hook.End()

	for ev := range evChan {
		var de deviceEvent

		switch ev.Kind {
		case hook.KeyDown:
			name := keyName(ev.Keycode)
			if name == "" {
				continue
			}
			de = deviceEvent{Kind: "KeyboardPress", Value: name}

		case hook.KeyUp:
			name := keyName(ev.Keycode)
			if name == "" {
				continue
			}
			de = deviceEvent{Kind: "KeyboardRelease", Value: name}

		case hook.MouseDown:
			de = deviceEvent{Kind: "MousePress", Value: mouseButtonName(ev.Button)}

		case hook.MouseUp:
			de = deviceEvent{Kind: "MouseRelease", Value: mouseButtonName(ev.Button)}

		case hook.MouseMove, hook.MouseDrag:
			de = deviceEvent{Kind: "MouseMove", Value: cursorPoint{X: ev.X, Y: ev.Y}}

		default:
			// KeyHold / wheel / fake events are ignored — the frontend models
			// press/release pairs and (on Windows) auto-releases held keys.
			continue
		}

		s.app.Event.Emit(DeviceEventName, de)
	}
}
