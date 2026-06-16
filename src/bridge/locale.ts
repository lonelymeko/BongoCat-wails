// Bridge for `tauri-plugin-locale-api`. Proxies to AppService.

import { call } from './_call'

export async function getLocale(): Promise<string> {
  return call<string>('AppService', 'GetLocale')
}
