// Bridge for `@tauri-apps/api/core`. Maps legacy Tauri `invoke` command names
// onto our Go services, and reimplements `convertFileSrc` against the Go asset
// middleware (`/_bongo/resource`).

import { call, currentLabel } from './_call'

export function invoke<T = any>(cmd: string, args?: Record<string, any>): Promise<T> {
  const a = args ?? {}

  switch (cmd) {
    // utils: copy a model directory. Original call site passes { fromPath, toPath }.
    case 'copy_dir':
      return call('AppService', 'CopyDir', a.fromPath ?? a.from, a.toPath ?? a.to) as Promise<T>

    // device listener (rdev) — fire-and-forget on the Go side.
    case 'start_device_listening':
      return call('DeviceService', 'StartListening') as Promise<T>

    // gamepad support is deferred to a later milestone; resolve as no-op.
    case 'start_gamepad_listing':
    case 'stop_gamepad_listing':
      return Promise.resolve(undefined as unknown as T)

    // custom-window plugin commands (src/plugins/window.ts). These operate on
    // the current window, so we resolve the label synchronously.
    case 'plugin:custom-window|show_window':
      return call('WindowService', 'ShowWindow', currentLabel()) as Promise<T>
    case 'plugin:custom-window|hide_window':
      return call('WindowService', 'HideWindow', currentLabel()) as Promise<T>
    case 'plugin:custom-window|set_always_on_top':
      return call('WindowService', 'SetAlwaysOnTop', !!a.alwaysOnTop) as Promise<T>
    case 'plugin:custom-window|set_taskbar_visibility':
      return call('WindowService', 'SetTaskbarVisibility', !!a.visible) as Promise<T>

    // admin-status plugin (src/plugins/adminStatus.ts).
    case 'plugin:admin-status|is_running_as_administrator':
      return call('AppService', 'IsRunningAsAdministrator') as Promise<T>

    default:
      return Promise.reject(new Error(`[bridge/core] Unsupported invoke command: ${cmd}`))
  }
}

export function convertFileSrc(filePath: string, _protocol?: string): string {
  // Served by our Go asset middleware.
  return `/_bongo/resource?path=${encodeURIComponent(filePath)}`
}
