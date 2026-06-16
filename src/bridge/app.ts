// Bridge for `@tauri-apps/api/app`. Proxies to AppService.

import { call } from './_call'

export async function getName(): Promise<string> {
  return call<string>('AppService', 'GetName')
}

export async function getVersion(): Promise<string> {
  return call<string>('AppService', 'GetVersion')
}

export async function getTauriVersion(): Promise<string> {
  return call<string>('AppService', 'GetTauriVersion')
}
