// Bridge for `@tauri-apps/plugin-global-shortcut`. DEFERRED stub: satisfies the
// imports in useKeyPress.ts without registering real global shortcuts.
// TODO M2: real global shortcuts

export type ShortcutHandler = (event: {
  shortcut: string
  id: number
  state: 'Pressed' | 'Released'
}) => void

export async function register(_shortcut: string, _handler: ShortcutHandler): Promise<void> {
  // no-op
}

export async function unregister(_shortcut: string): Promise<void> {
  // no-op
}

export async function isRegistered(_shortcut: string): Promise<boolean> {
  return false
}
