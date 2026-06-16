// Bridge for `@tauri-apps/plugin-updater`. DEFERRED stub: reports no update
// available so the update flow becomes a no-op.
// TODO M2: real updater backed by Go.

export interface Update {
  version: string
  currentVersion: string
  date?: string
  body?: string
  downloadAndInstall: (cb?: any) => Promise<void>
}

export async function check(): Promise<Update | null> {
  return null
}
