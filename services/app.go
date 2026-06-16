package services

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// AppService is the catch-all backend service replacing the assorted Tauri
// core APIs and plugins the frontend used: app info, os, path, process,
// opener, fs, dialog, clipboard, autostart, locale and admin status.
//
// All public methods are exposed to the frontend as
// "bongocat/services.AppService.<Method>" (see the bridge's call helper).
type AppService struct {
	app *application.App

	name        string
	version     string
	resourceDir string // base dir for ResolveResource (contains "assets/...")
	dataDir     string // per-user app data dir
	logDir      string // per-user log dir
}

// NewAppService builds the service. resourceDir is the directory that contains
// the bundled "assets" folder; dataDir/logDir are per-user writable locations.
func NewAppService(app *application.App, name, version, resourceDir, dataDir, logDir string) *AppService {
	return &AppService{
		app:         app,
		name:        name,
		version:     version,
		resourceDir: resourceDir,
		dataDir:     dataDir,
		logDir:      logDir,
	}
}

// ---- app ----

func (s *AppService) GetName() string    { return s.name }
func (s *AppService) GetVersion() string { return s.version }

// GetTauriVersion is kept for the About page; returns the Wails runtime version.
func (s *AppService) GetTauriVersion() string { return "wails-v3.0.0-alpha2.103" }

// ---- os ----

// Platform returns a tauri-plugin-os style platform string.
func (s *AppService) Platform() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	default:
		return runtime.GOOS // "windows" | "linux"
	}
}

// Arch returns a tauri-plugin-os style arch string.
func (s *AppService) Arch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		return runtime.GOARCH
	}
}

// OSVersion is display-only on the About page. Best-effort; empty if unknown.
func (s *AppService) OSVersion() string { return "" }

// GetLocale returns the system locale (e.g. "zh-CN"), used by the general store.
func (s *AppService) GetLocale() string { return systemLocale() }

// ---- path ----

func (s *AppService) Sep() string { return string(os.PathSeparator) }

// ResolveResource resolves a bundled-resource-relative path to an absolute one,
// matching tauri's resolveResource('assets/models', ...).
func (s *AppService) ResolveResource(rel string) string {
	return filepath.Join(s.resourceDir, rel)
}

func (s *AppService) AppDataDir() string { return s.dataDir }
func (s *AppService) AppLogDir() string  { return s.logDir }

// ---- process ----

func (s *AppService) Exit(code int) { s.app.Quit() }

// Relaunch restarts the application. A best-effort cross-platform re-exec.
func (s *AppService) Relaunch() { relaunchSelf(); s.app.Quit() }

// ---- opener ----

func (s *AppService) OpenUrl(url string) error { return s.app.Browser.OpenURL(url) }
func (s *AppService) OpenPath(path string) error {
	return s.app.Browser.OpenFile(path)
}

// RevealItemInDir opens the system file manager with the item selected.
func (s *AppService) RevealItemInDir(path string) error {
	return s.app.Env.OpenFileManager(path, true)
}

// ---- fs ----

// FsEntry mirrors the subset of tauri's plugin-fs DirEntry the frontend reads.
type FsEntry struct {
	Name        string `json:"name"`
	IsDirectory bool   `json:"isDirectory"`
	IsFile      bool   `json:"isFile"`
}

func (s *AppService) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *AppService) ReadDir(path string) ([]FsEntry, error) {
	items, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	entries := make([]FsEntry, 0, len(items))
	for _, it := range items {
		entries = append(entries, FsEntry{
			Name:        it.Name(),
			IsDirectory: it.IsDir(),
			IsFile:      !it.IsDir(),
		})
	}
	return entries, nil
}

func (s *AppService) ReadTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *AppService) Remove(path string) error { return os.RemoveAll(path) }

// CopyDir recursively copies a directory tree (replaces the copy_dir command).
func (s *AppService) CopyDir(from, to string) error { return copyDir(from, to) }

// ---- clipboard ----

func (s *AppService) WriteText(text string) { s.app.Clipboard.SetText(text) }

// ---- dialog ----

// Confirm shows a native question dialog and blocks until the user answers,
// returning true for OK/Yes. Replaces plugin-dialog confirm().
func (s *AppService) Confirm(message, title string) bool {
	result := make(chan bool, 1)

	dialog := s.app.Dialog.Question()
	dialog.SetTitle(title)
	dialog.SetMessage(message)

	ok := dialog.AddButton("OK")
	ok.OnClick(func() { result <- true })

	cancel := dialog.AddButton("Cancel")
	cancel.SetAsCancel()
	cancel.OnClick(func() { result <- false })

	dialog.Show()

	return <-result
}

// OpenFileDialog shows a file/directory picker. directory=true picks folders.
// Returns the selected absolute paths (empty if cancelled).
func (s *AppService) OpenFileDialog(directory, multiple bool) ([]string, error) {
	dialog := s.app.Dialog.OpenFile()
	dialog.CanChooseFiles(!directory)
	dialog.CanChooseDirectories(directory)

	if multiple {
		return dialog.PromptForMultipleSelection()
	}

	selected, err := dialog.PromptForSingleSelection()
	if err != nil {
		return nil, err
	}
	if selected == "" {
		return nil, nil
	}
	return []string{selected}, nil
}

// ---- autostart ----

func (s *AppService) AutostartEnable() error   { return s.app.Autostart.Enable() }
func (s *AppService) AutostartDisable() error  { return s.app.Autostart.Disable() }
func (s *AppService) AutostartIsEnabled() bool { ok, _ := s.app.Autostart.IsEnabled(); return ok }

// ---- macos permissions (stubbed for MVP) ----

// CheckInputMonitoring reports whether input monitoring is granted. The real
// macOS check is a TODO (cgo/ObjC); MVP assumes granted. On macOS the user must
// grant "Input Monitoring" to BongoCat in System Settings for key/mouse events.
func (s *AppService) CheckInputMonitoring() bool { return true }

// RequestInputMonitoring is a no-op stub for MVP.
func (s *AppService) RequestInputMonitoring() bool { return true }

// ---- admin status ----

// IsRunningAsAdministrator is implemented per-platform (admin_windows.go /
// admin_other.go).
func (s *AppService) IsRunningAsAdministrator() bool { return isElevated() }
