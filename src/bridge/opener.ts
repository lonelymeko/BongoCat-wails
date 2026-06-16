// Bridge for `@tauri-apps/plugin-opener`. Proxies to AppService.

import { call } from './_call'

export async function openUrl(url: string): Promise<void> {
  await call('AppService', 'OpenUrl', url)
}

export async function openPath(path: string): Promise<void> {
  await call('AppService', 'OpenPath', path)
}

export async function revealItemInDir(path: string): Promise<void> {
  await call('AppService', 'RevealItemInDir', path)
}
