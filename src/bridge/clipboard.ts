// Bridge for `@tauri-apps/plugin-clipboard-manager`. Proxies to AppService.

import { call } from './_call'

export async function writeText(text: string): Promise<void> {
  await call('AppService', 'WriteText', text)
}
