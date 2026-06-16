// Bridge for `@tauri-store/pinia`. Reimplements the persistent-pinia plugin on
// top of our StoreService (Load/Save). Behavior mapping:
//   - @tauri-store/pinia hydrates each store from a persisted JSON blob keyed by
//     the store id, and persists state changes back. We mirror that:
//       * on init  -> StoreService.Load(store.$id) -> JSON.parse -> $patch
//       * on change -> debounced StoreService.Save(store.$id, JSON.stringify(state))
//   - Stores may declare a custom option `tauri: { filterKeys, filterKeysStrategy }`
//     to control which state keys are persisted (default strategy: 'omit').

import type { PiniaPlugin, PiniaPluginContext } from 'pinia'

import { call } from './_call'

interface TauriStoreOptions {
  filterKeys?: string[]
  filterKeysStrategy?: 'omit' | 'pick'
}

interface CreatePluginOptions {
  saveOnChange?: boolean
}

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

    // --- hydrate from persisted state ---------------------------------------
    const start = async () => {
      try {
        const json = await call<string>('StoreService', 'Load', store.$id)
        if (json) {
          try {
            const saved = JSON.parse(json)
            store.$patch(saved)
          } catch {
            // ignore malformed persisted state
          }
        }
      } catch {
        // ignore load failures; store keeps its default state
      }
    }

    ;(store as any).$tauri = {
      start,
    }

    void start()

    // --- persist on change (debounced ~200ms) -------------------------------
    let timer: ReturnType<typeof setTimeout> | undefined

    const persist = () => {
      if (timer) clearTimeout(timer)
      timer = setTimeout(() => {
        try {
          const filtered = filterState(store.$state as Record<string, any>, tauriOptions)
          void call('StoreService', 'Save', store.$id, JSON.stringify(filtered)).catch(() => {})
        } catch {
          // ignore serialization/save failures
        }
      }, 200)
    }

    store.$subscribe(() => {
      persist()
    })
  }
}
