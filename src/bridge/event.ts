// Bridge for `@tauri-apps/api/event`, backed by the Wails `Events` runtime.
// Also tracks the last-known cursor position from the continuous
// "device-changed" stream, used by the window bridge's cursorPosition fallback.

import { Events } from '@wailsio/runtime'

export interface Event<T> {
  event: string
  id: number
  payload: T
}

export type EventCallback<T> = (event: Event<T>) => void

export type UnlistenFn = () => void

export function listen<T>(event: string, handler: EventCallback<T>): Promise<UnlistenFn> {
  // Events.On returns a synchronous unsubscribe fn; we wrap in a resolved
  // Promise to match Tauri's async `listen` signature.
  const unlisten = Events.On(event, (e) => {
    handler({ event, id: 0, payload: e.data as T })
  })

  return Promise.resolve(unlisten)
}

export async function emit(event: string, payload?: any): Promise<void> {
  await Events.Emit(event, payload)
}

// --- last-known cursor position --------------------------------------------
// Go emits "device-changed" continuously; on MouseMove the value is {x,y}.

let lastCursor = { x: 0, y: 0 }

Events.On('device-changed', (e) => {
  const data = e.data as { kind?: string, value?: { x: number, y: number } }

  if (data?.kind === 'MouseMove' && data.value) {
    lastCursor = { x: data.value.x, y: data.value.y }
  }
})

export function getLastCursor(): { x: number, y: number } {
  return lastCursor
}
