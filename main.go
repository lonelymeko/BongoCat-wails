package main

import (
	"embed"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"bongocat/services"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Frontend production assets. In `wails3 dev` the Vite dev server is used
// instead; this embed is only consulted for packaged builds. A placeholder
// dist/ exists so the module compiles before the first `pnpm build`.
//
//go:embed all:dist
var assets embed.FS

// System tray icon (template image). Replaces src-tauri/assets/tray.png.
//
//go:embed build/tray.png
var trayIcon []byte

const (
	appName    = "BongoCat"
	appVersion = "1.1.0"

	// resourceQueryPath is the URL the frontend's convertFileSrc() points at to
	// load arbitrary on-disk files (Live2D models/textures), replacing tauri's
	// asset:// protocol. See bridge/core.ts.
	resourceURLPath = "/_bongo/resource"
)

// appRef holds the running application so the single-instance callback (defined
// in the options, before New returns) can reach it.
var appRef *application.App

func main() {
	resourceDir := resolveResourceDir()
	dataDir := userDir(appName)
	logDir := filepath.Join(dataDir, "logs")

	app := application.New(application.Options{
		Name:        appName,
		Description: "A cute Live2D desktop pet that reacts to your keyboard, mouse and gamepad.",
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(assets),
			Middleware: resourceMiddleware,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.ayangweb.BongoCat",
			// A second launch surfaces the preferences window instead of
			// starting a new process (mirrors the Tauri single-instance plugin).
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				if appRef == nil {
					return
				}
				if win, ok := appRef.Window.GetByName(services.PreferenceWindowLabel); ok {
					win.Show()
					win.Focus()
				}
			},
		},
		// Keep the app alive in the tray when all windows are hidden/closed —
		// the cat lives in the menu bar / tray, not a normal window lifecycle.
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	// Services. The frontend bridge calls these via
	// "bongocat/services.<Service>.<Method>".
	deviceService := services.NewDeviceService(app)
	windowService := services.NewWindowService(app)
	storeService := services.NewStoreService(dataDir)
	appService := services.NewAppService(app, appName, appVersion, resourceDir, dataDir, logDir)

	app.RegisterService(application.NewService(deviceService))
	app.RegisterService(application.NewService(windowService))
	app.RegisterService(application.NewService(storeService))
	app.RegisterService(application.NewService(appService))

	appRef = app

	mainWindow := newMainWindow(app)
	newPreferenceWindow(app)

	forwardWindowState(mainWindow, services.MainWindowLabel)

	setupTray(app)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// newMainWindow creates the transparent, frameless, always-on-top, click-able
// cat window (hidden from the taskbar). Mirrors the "main" window in
// tauri.conf.json plus the macOS NSPanel level/collection-behaviour tweaks.
func newMainWindow(app *application.App) *application.WebviewWindow {
	return app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             services.MainWindowLabel,
		Title:            appName,
		URL:              "/#/",
		Width:            300,
		Height:           300,
		Frameless:        true,
		AlwaysOnTop:      true,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		DisableResize:    false,
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: true,
		},
		Mac: application.MacWindow{
			DisableShadow: true,
			Backdrop:      application.MacBackdropTransparent,
			WindowLevel:   application.MacWindowLevelFloating,
			CollectionBehavior: application.MacWindowCollectionBehaviorCanJoinAllSpaces |
				application.MacWindowCollectionBehaviorFullScreenAuxiliary,
		},
	})
}

// newPreferenceWindow creates the standard settings window, hidden at startup
// (shown via the tray or second-instance launch).
func newPreferenceWindow(app *application.App) application.Window {
	return app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:            services.PreferenceWindowLabel,
		Title:           appName,
		URL:             "/#/preference",
		Width:           900,
		Height:          700,
		MinWidth:        800,
		MinHeight:       600,
		Hidden:          true,
		InitialPosition: application.WindowCentered,
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: true,
		},
	})
}

// forwardWindowState re-emits Wails move/resize events as custom events the
// frontend bridge (webview-window.ts onMoved/onResized) subscribes to. Replaces
// the manual emit in the original macOS setup.rs.
func forwardWindowState(win *application.WebviewWindow, label string) {
	win.OnWindowEvent(events.Common.WindowDidMove, func(*application.WindowEvent) {
		x, y := win.Position()
		win.EmitEvent(label+":moved", map[string]int{"x": x, "y": y})
	})

	win.OnWindowEvent(events.Common.WindowDidResize, func(*application.WindowEvent) {
		w, h := win.Size()
		win.EmitEvent(label+":resized", map[string]int{"width": w, "height": h})
	})
}

// setupTray builds a minimal system tray (Preferences / Show cat / Hide cat /
// Open source / Quit). The richer, state-driven menu from useTray.ts/useAppMenu.ts
// is deferred to a later milestone.
func setupTray(app *application.App) {
	tray := app.SystemTray.New()
	if len(trayIcon) > 0 {
		tray.SetIcon(trayIcon)
	}
	tray.SetLabel(appName)

	menu := application.NewMenu()

	menu.Add("Preferences").OnClick(func(*application.Context) {
		if win, ok := app.Window.GetByName(services.PreferenceWindowLabel); ok {
			win.Show()
			win.Focus()
		}
	})

	menu.AddSeparator()

	menu.Add("Show Cat").OnClick(func(*application.Context) {
		if win, ok := app.Window.GetByName(services.MainWindowLabel); ok {
			win.Show()
		}
	})
	menu.Add("Hide Cat").OnClick(func(*application.Context) {
		if win, ok := app.Window.GetByName(services.MainWindowLabel); ok {
			win.Hide()
		}
	})

	menu.AddSeparator()

	menu.Add("Open Source").OnClick(func(*application.Context) {
		_ = app.Browser.OpenURL("https://github.com/ayangweb/BongoCat")
	})
	menu.Add("Quit").OnClick(func(*application.Context) {
		app.Quit()
	})

	tray.SetMenu(menu)
}

// resourceMiddleware serves arbitrary on-disk files requested via convertFileSrc
// (resourceURLPath?path=<abs path>). This is the Wails equivalent of tauri's
// asset:// protocol and is how the Live2D models/textures are loaded.
func resourceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != resourceURLPath {
			next.ServeHTTP(w, r)
			return
		}

		path := r.URL.Query().Get("path")
		if path == "" {
			http.Error(w, "missing path", http.StatusBadRequest)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		http.ServeFile(w, r, path)
	})
}

// resolveResourceDir locates the directory that contains the bundled "assets"
// folder. Order: $BONGOCAT_RESOURCES, ./src-tauri (dev), the executable's dir.
func resolveResourceDir() string {
	if dir := os.Getenv("BONGOCAT_RESOURCES"); dir != "" {
		return dir
	}

	if wd, err := os.Getwd(); err == nil {
		dev := filepath.Join(wd, "src-tauri")
		if _, err := os.Stat(filepath.Join(dev, "assets")); err == nil {
			return dev
		}
	}

	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}

	return "."
}

// userDir returns a per-user writable config directory for the app.
func userDir(name string) string {
	base, err := os.UserConfigDir()
	if err != nil {
		base, _ = os.UserHomeDir()
	}
	return filepath.Join(base, name)
}
