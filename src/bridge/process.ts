// Bridge for `@tauri-apps/plugin-process`. Proxies to AppService.

import { call } from './_call'

export async function exit(code = 0): Promise<void> {
  await call('AppService', 'Exit', code)
}

export async function relaunch(): Promise<void> {
  await call('AppService', 'Relaunch')
}
