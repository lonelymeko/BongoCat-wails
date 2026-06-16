// Bridge for `@tauri-apps/plugin-os`.
// NOTE: `platform()` is used SYNCHRONOUSLY at module top-level
// (utils/platform.ts does `platform() === 'macos'`), so it must be sync. We
// derive from the runtime `System`. `arch()` and `version()` are display-only
// (About page) and returned as sync best-effort empty strings.

import { System } from '@wailsio/runtime'

export function platform(): 'macos' | 'windows' | 'linux' {
  return System.IsMac() ? 'macos' : System.IsWindows() ? 'windows' : 'linux'
}

// Display-only; not reliably derivable synchronously in the webview.
export function arch(): string {
  return ''
}

// Display-only; not reliably derivable synchronously in the webview.
export function version(): string {
  return ''
}
