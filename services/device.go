package services

import (
	"log"
	"sync"
	"time"

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
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// DeviceService replaces start_device_listening. It hooks global keyboard and
// mouse input via robotn/gohook and forwards each event to the frontend.
type DeviceService struct {
	app           *application.App
	mu            sync.Mutex
	listening     bool
	mouseUpCancel map[string]chan struct{}
	mousePressed  map[string]bool
}

// NewDeviceService creates the service bound to the running application.
func NewDeviceService(app *application.App) *DeviceService {
	return &DeviceService{
		app:           app,
		mouseUpCancel: make(map[string]chan struct{}),
		mousePressed:  make(map[string]bool),
	}
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

	startPlatformInputListener(s)

	// On macOS the native CGEventTap (startPlatformInputListener) is the input
	// backend and every gohook event is discarded by handleHook*Events(). Starting
	// gohook there only spins up a second, crash-prone native run loop (the source
	// of the SIGSEGV in gohook's C code), so skip it. On other platforms gohook IS
	// the backend and the loop must run.
	if platformUsesGohook() {
		go s.loop()
	}
}

func (s *DeviceService) loop() {
	// A panic in event handling (e.g. Event.Emit after the app is tearing down)
	// must not crash the whole process; recover and let the loop exit. Note this
	// only catches Go-side panics — a native SIGSEGV inside gohook's C run loop
	// cannot be recovered, which is why gohook is not started on macOS at all.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("device: recovered from panic in gohook loop: %v", r)
		}
	}()

	// hook.Start returns a channel of raw events. The hook runs its own native
	// run loop in C, so consuming from a goroutine is safe.
	evChan := hook.Start()
	defer hook.End()

	for ev := range evChan {
		var de deviceEvent

		switch ev.Kind {
		case hook.KeyDown:
			if !handleHookKeyboardEvents() {
				continue
			}
			name := keyName(ev.Keycode)
			if name == "" {
				continue
			}
			de = deviceEvent{Kind: "KeyboardPress", Value: name}

		case hook.KeyUp:
			if !handleHookKeyboardEvents() {
				continue
			}
			name := keyName(ev.Keycode)
			if name == "" {
				continue
			}
			de = deviceEvent{Kind: "KeyboardRelease", Value: name}

		case hook.MouseDown:
			if !handleHookMouseEvents() {
				continue
			}
			name := mouseButtonName(ev.Button)
			s.emitMousePress(name)
			continue

		case hook.MouseUp:
			if !handleHookMouseEvents() {
				continue
			}
			name := mouseButtonName(ev.Button)
			s.emitMouseRelease(name)
			continue

		case hook.MouseMove, hook.MouseDrag:
			if !handleHookMouseEvents() {
				continue
			}
			de = deviceEvent{Kind: "MouseMove", Value: cursorPoint{X: float64(ev.X), Y: float64(ev.Y)}}

		default:
			// KeyHold / wheel / fake events are ignored — the frontend models
			// press/release pairs and (on Windows) auto-releases held keys.
			continue
		}

		s.app.Event.Emit(DeviceEventName, de)
	}
}

func (s *DeviceService) handleMouseMove(x, y float64) {
	s.app.Event.Emit(DeviceEventName, deviceEvent{Kind: "MouseMove", Value: cursorPoint{X: x, Y: y}})
}

func (s *DeviceService) emitKeyboardPress(name string) {
	if name == "" {
		return
	}

	s.app.Event.Emit(DeviceEventName, deviceEvent{Kind: "KeyboardPress", Value: name})
}

func (s *DeviceService) emitKeyboardRelease(name string) {
	if name == "" {
		return
	}

	s.app.Event.Emit(DeviceEventName, deviceEvent{Kind: "KeyboardRelease", Value: name})
}

func (s *DeviceService) emitMousePress(name string) bool {
	if !s.markMousePressed(name) {
		return false
	}

	s.app.Event.Emit(DeviceEventName, deviceEvent{Kind: "MousePress", Value: name})
	s.watchMouseRelease(name)
	return true
}

func (s *DeviceService) emitMouseRelease(name string) bool {
	if !s.markMouseReleased(name) {
		return false
	}

	s.app.Event.Emit(DeviceEventName, deviceEvent{Kind: "MouseRelease", Value: name})
	s.cancelMouseReleaseWatch(name)
	return true
}

func (s *DeviceService) watchMouseRelease(name string) {
	_, ok := mouseButtonDown(name)
	if !ok {
		return
	}

	s.cancelMouseReleaseWatch(name)

	stop := make(chan struct{})

	s.mu.Lock()
	s.mouseUpCancel[name] = stop
	s.mu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("device: recovered from panic in mouse-release watcher: %v", r)
			}
		}()

		ticker := time.NewTicker(8 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				down, ok := mouseButtonDown(name)
				if !ok || down {
					continue
				}

				s.emitMouseRelease(name)
				return
			}
		}
	}()
}

func (s *DeviceService) cancelMouseReleaseWatch(name string) {
	s.mu.Lock()
	stop, ok := s.mouseUpCancel[name]
	if ok {
		delete(s.mouseUpCancel, name)
	}
	s.mu.Unlock()

	if ok {
		close(stop)
	}
}

func (s *DeviceService) markMousePressed(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mousePressed[name] {
		return false
	}

	s.mousePressed[name] = true
	return true
}

func (s *DeviceService) markMouseReleased(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.mousePressed[name] {
		return false
	}

	delete(s.mousePressed, name)
	return true
}
