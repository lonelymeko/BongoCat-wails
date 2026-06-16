// Bridge for `@tauri-apps/api/path`.
// NOTE: `sep()` is used SYNCHRONOUSLY (utils/path.ts, useModel.ts do
// `path.split(sep())`), so it must return a string synchronously. We derive it
// from the runtime `System` (sync) rather than a backend call.

import { System } from '@wailsio/runtime'

import { call } from './_call'

export function sep(): string {
  return System.IsWindows() ? '\\' : '/'
}

export async function resolveResource(rel: string): Promise<string> {
  return call<string>('AppService', 'ResolveResource', rel)
}

export async function appDataDir(): Promise<string> {
  return call<string>('AppService', 'AppDataDir')
}

export async function appLogDir(): Promise<string> {
  return call<string>('AppService', 'AppLogDir')
}
