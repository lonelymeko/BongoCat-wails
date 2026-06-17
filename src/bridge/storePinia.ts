// Bridge for `@tauri-store/pinia`. Reimplements the persistent-pinia plugin on
// top of our StoreService (Load/Save) PLUS cross-window sync.
//
// @tauri-store/pinia does two things the app relies on:
//   1. persistence  — hydrate each store from a JSON blob keyed by store id,
//      and save changes back to disk.
//   2. cross-window sync — a change in one window (e.g. picking a model or
//      changing the scale in the preferences window) is reflected live in the
//      other window (the cat). Without this, selecting a model in preferences
//      never reaches the main window: it never renders and the "switching…"
//      spinner hangs forever.
//
// Sync is implemented by broadcasting the filtered state over a Wails event
// (delivered to every window). Each window ignores its own broadcasts (via a
// per-window clientId) and patches under a guard so applying a remote change
// doesn't echo back.

import type { PiniaPlugin, PiniaPluginContext } from 'pinia'

import { call } from './_call'
import { emit, listen } from './event'

interface TauriStoreOptions {
  filterKeys?: string[]
  filterKeysStrategy?: 'omit' | 'pick'
}

interface CreatePluginOptions {
  saveOnChange?: boolean
}

interface SyncMessage {
  clientId: string
  state: Record<string, any>
}

// Unique per window/webview, so a window can ignore the events it emits itself.
const clientId = Math.random().toString(36).slice(2)

function filterState(
  state: Record<string, any>,
  options?: TauriStoreOptions,
): Record<string, any> {
  const keys = options?.filterKeys ?? []

  if (!keys.length) return { ...state }

  const strategy = options?.filterKeysStrategy ?? 'omit'

  if (strategy === 'pick') {
    const picked: Record<string, any> = {}
    for (const key of keys) {
      if (key in state) picked[key] = state[key]
    }
    return picked
  }

  // 'omit': drop the configured keys.
  const result = { ...state }
  for (const key of keys) {
    delete result[key]
  }
  return result
}

export function createPlugin(_options?: CreatePluginOptions): PiniaPlugin {
  return (context: PiniaPluginContext) => {
    const { store, options } = context

    // Custom store option, e.g. `tauri: { filterKeys: ['pressedKeys'] }`.
    const tauriOptions = (options as any).tauri as TauriStoreOptions | undefined

    const syncEvent = `store-sync:${store.$id}`

    // Set while applying a remote change so the $subscribe handler doesn't
    // persist/rebroadcast it.
    let applyingRemote = false

    // --- hydrate from persisted state ---------------------------------------
    const start = async () => {
      try {
        const json = await call<string>('StoreService', 'Load', store.$id)
        if (json) {
          try {
            applyingRemote = true
            store.$patch(JSON.parse(json))
          } catch {
            // ignore malformed persisted state
          } finally {
            applyingRemote = false
          }
        }
      } catch {
        // ignore load failures; store keeps its default state
      }
    }

    ;(store as any).$tauri = { start }

    void start()

    // --- live cross-window sync ---------------------------------------------
    void listen<SyncMessage>(syncEvent, ({ payload }) => {
      if (!payload || payload.clientId === clientId) return

      applyingRemote = true
      try {
        store.$patch(payload.state)
      } finally {
        applyingRemote = false
      }
    })

    // --- persist + broadcast on change (debounced) --------------------------
    let timer: ReturnType<typeof setTimeout> | undefined

    const flush = () => {
      const filtered = filterState(store.$state as Record<string, any>, tauriOptions)
      void call('StoreService', 'Save', store.$id, JSON.stringify(filtered)).catch(() => {})
      void emit(syncEvent, { clientId, state: filtered })
    }

    // flush: 'sync' so `applyingRemote` reliably brackets remote $patch calls.
    store.$subscribe(() => {
      if (applyingRemote) return

      if (timer) clearTimeout(timer)
      timer = setTimeout(flush, 120)
    }, { flush: 'sync' })
  }
}
