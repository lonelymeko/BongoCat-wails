package services

import "github.com/wailsapp/wails/v3/pkg/application"

// Window labels, matching the frontend constants (WINDOW_LABEL).
const (
	MainWindowLabel       = "main"
	PreferenceWindowLabel = "preference"
)

// WindowService replaces the custom-window Tauri plugin. The transparent,
// frameless, always-on-top and click-through behaviour is configured at window
// creation in main.go; this service exposes the runtime toggles the frontend
// needs.
//
// NOTE (MVP): The original Windows backend ran a 16ms SetWindowPos(HWND_TOPMOST)
// loop to stay above other top-most windows. Here we rely on Wails' built-in
// SetAlwaysOnTop. If the cat ends up behind other always-on-top windows on
// Windows, reintroduce the aggressive loop in a window_windows.go (see plan).
type WindowService struct {
	app *application.App
}

func NewWindowService(app *application.App) *WindowService {
	return &WindowService{app: app}
}

func (s *WindowService) window(label string) (application.Window, bool) {
	return s.app.Window.GetByName(label)
}

// ShowWindow shows (and focuses) the window with the given label.
func (s *WindowService) ShowWindow(label string) {
	if win, ok := s.window(label); ok {
		win.Show()
		win.UnMinimise()
		win.Focus()
	}
}

// HideWindow hides the window with the given label.
func (s *WindowService) HideWindow(label string) {
	if win, ok := s.window(label); ok {
		win.Hide()
	}
}

// SetAlwaysOnTop toggles always-on-top for the main window.
func (s *WindowService) SetAlwaysOnTop(alwaysOnTop bool) {
	if win, ok := s.window(MainWindowLabel); ok {
		win.SetAlwaysOnTop(alwaysOnTop)
	}
}

// SetTaskbarVisibility is a no-op in this MVP: the main window is hidden from
// the taskbar at creation (Windows.HiddenOnTaskbar). The Wails alpha exposes no
// runtime toggle. Kept so the frontend call resolves.
func (s *WindowService) SetTaskbarVisibility(visible bool) {}

// SetIgnoreCursorEvents toggles click-through for a window (mouse pass-through).
// Replaces webviewWindow.setIgnoreCursorEvents, which the Wails JS runtime does
// not expose.
func (s *WindowService) SetIgnoreCursorEvents(label string, ignore bool) {
	if win, ok := s.window(label); ok {
		win.SetIgnoreMouseEvents(ignore)
	}
}

// IsVisible reports whether a window is currently visible. Replaces
// webviewWindow.isVisible, absent from the Wails JS runtime.
func (s *WindowService) IsVisible(label string) bool {
	if win, ok := s.window(label); ok {
		return win.IsVisible()
	}
	return false
}
