// Bridge for `@tauri-apps/plugin-autostart`. Proxies to AppService.

import { call } from './_call'

export async function enable(): Promise<void> {
  await call('AppService', 'AutostartEnable')
}

export async function disable(): Promise<void> {
  await call('AppService', 'AutostartDisable')
}

export async function isEnabled(): Promise<boolean> {
  return call<boolean>('AppService', 'AutostartIsEnabled')
}
