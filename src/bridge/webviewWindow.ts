// Bridge for `@tauri-apps/api/webviewWindow`. Returns an object bound to the
// current window label that proxies to the Wails `Window` runtime + our
// WindowService. Move/resize events are emitted by Go as `<label>:moved` /
// `<label>:resized`.

import { Window } from '@wailsio/runtime'

import type { UnlistenFn } from './event'

import { call, currentLabel } from './_call'
import { listen } from './event'

export function getCurrentWebviewWindow() {
  const label = currentLabel()

  return {
    label,

    async setBackgroundColour(r: number, g: number, b: number, a: number): Promise<void> {
      await Window.SetBackgroundColour(r, g, b, a)
    },

    async isVisible(): Promise<boolean> {
      return call<boolean>('WindowService', 'IsVisible', label)
    },

    async scaleFactor(): Promise<number> {
      return (await Window.GetScreen()).ScaleFactor
    },

    // Scale-change events are deferred; return a resolved no-op unlisten.
    onScaleChanged(_cb: (e: any) => void): Promise<UnlistenFn> {
      return Promise.resolve(() => {})
    },

    async setIgnoreCursorEvents(ignore: boolean): Promise<void> {
      await call('WindowService', 'SetIgnoreCursorEvents', label, ignore)
    },

    async setPosition(pos: { x: number, y: number }): Promise<void> {
      await Window.SetPosition(pos.x, pos.y)
    },

    async setSize(size: { width: number, height: number }): Promise<void> {
      await Window.SetSize(size.width, size.height)
    },

    async size(): Promise<{ width: number, height: number }> {
      return Window.Size()
    },

    async outerSize(): Promise<{ width: number, height: number }> {
      return Window.Size()
    },

    async innerSize(): Promise<{ width: number, height: number }> {
      return Window.Size()
    },

    async outerPosition(): Promise<{ x: number, y: number }> {
      return Window.Position()
    },

    async isMinimized(): Promise<boolean> {
      return Window.IsMinimised()
    },

    async unminimize(): Promise<void> {
      await Window.UnMinimise()
    },

    async show(): Promise<void> {
      await Window.Show()
    },

    async hide(): Promise<void> {
      await Window.Hide()
    },

    async setFocus(): Promise<void> {
      await Window.Focus()
    },

    // Go emits `<label>:moved` with `{ x, y }`.
    onMoved(cb: (e: { payload: { x: number, y: number } }) => void): Promise<UnlistenFn> {
      return listen<{ x: number, y: number }>(`${label}:moved`, (e) => {
        cb({ payload: e.payload })
      })
    },

    // Go emits `<label>:resized` with `{ width, height }`.
    onResized(cb: (e: { payload: { width: number, height: number } }) => void): Promise<UnlistenFn> {
      return listen<{ width: number, height: number }>(`${label}:resized`, (e) => {
        cb({ payload: e.payload })
      })
    },

    // Window dragging is handled by Wails via the CSS `--wails-draggable: drag`
    // already set on the main container, so this is a no-op (kept for parity).
    async startDragging(): Promise<void> {},

    async setTitle(title: string): Promise<void> {
      await Window.SetTitle(title)
    },

    // Theme helpers — the Wails alpha has no JS theme API, so we derive the
    // system theme from the media query and let the app apply it.
    async theme(): Promise<'light' | 'dark'> {
      return matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
    },

    async setTheme(_theme?: 'light' | 'dark' | null): Promise<void> {},

    onThemeChanged(cb: (e: { payload: 'light' | 'dark' }) => void): Promise<UnlistenFn> {
      const mql = matchMedia('(prefers-color-scheme: dark)')
      const handler = () => cb({ payload: mql.matches ? 'dark' : 'light' })
      mql.addEventListener('change', handler)
      return Promise.resolve(() => mql.removeEventListener('change', handler))
    },

    // Native file drag-and-drop isn't wired in the Wails port yet (needs
    // EnableFileDrop + a runtime event); importing via the click/file-picker
    // works. This no-op keeps the upload component from throwing on mount.
    onDragDropEvent(_cb: (e: any) => void): Promise<UnlistenFn> {
      return Promise.resolve(() => {})
    },
  }
}
