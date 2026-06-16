// Bridge for `@tauri-apps/plugin-dialog`. Proxies to AppService.
// `confirm` is called with an options object (e.g. { title, okLabel, kind });
// only the title is forwarded to the Go-side dialog.

import { call } from './_call'

export async function confirm(
  message: string,
  options?: string | { title?: string, [k: string]: any },
): Promise<boolean> {
  const title = typeof options === 'string' ? options : (options?.title ?? '')
  return call<boolean>('AppService', 'Confirm', message, title)
}

export async function open(
  options?: { directory?: boolean, multiple?: boolean },
): Promise<string | string[] | null> {
  const r = await call<string[]>(
    'AppService',
    'OpenFileDialog',
    !!options?.directory,
    !!options?.multiple,
  )

  if (!r || !r.length) return null

  return options?.multiple ? r : r[0]
}
