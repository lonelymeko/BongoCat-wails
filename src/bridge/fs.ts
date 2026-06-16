// Bridge for `@tauri-apps/plugin-fs`. Proxies to AppService. Symlink detection
// is not surfaced by the Go ReadDir contract, so `isSymlink` is always false.

import { call } from './_call'

export interface DirEntry {
  name: string
  isDirectory: boolean
  isFile: boolean
  isSymlink: boolean
}

export async function exists(path: string): Promise<boolean> {
  return call<boolean>('AppService', 'Exists', path)
}

export async function readDir(path: string): Promise<DirEntry[]> {
  const entries = await call<Array<{ name: string, isDirectory: boolean, isFile: boolean }>>(
    'AppService',
    'ReadDir',
    path,
  )

  return (entries ?? []).map(e => ({
    name: e.name,
    isDirectory: e.isDirectory,
    isFile: e.isFile,
    isSymlink: false,
  }))
}

export async function readTextFile(path: string): Promise<string> {
  return call<string>('AppService', 'ReadTextFile', path)
}

export async function remove(path: string, _options?: any): Promise<void> {
  await call('AppService', 'Remove', path)
}
