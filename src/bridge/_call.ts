// Shared helper for calling Go services via the Wails v3 runtime.
// Replaces the Tauri IPC layer; every bridge module routes through `call()`.

import { Call } from '@wailsio/runtime'

const PKG = 'bongocat/services'

export function call<T = any>(service: string, method: string, ...args: any[]): Promise<T> {
  return Call.ByName(`${PKG}.${service}.${method}`, ...args) as unknown as Promise<T>
}

// The app uses hash routing: main window URL is `#/`, preference window is
// `#/preference`. We need the current window's logical label synchronously.
export function currentLabel(): 'main' | 'preference' {
  return location.hash.replace(/^#\/?/, '').split('/')[0] === 'preference' ? 'preference' : 'main'
}
