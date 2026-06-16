// Bridge for `@tauri-apps/plugin-log`. Maps log levels to the console.
// (info/debug/trace fall back to console.warn — the project's lint rules only
// permit console.warn/error.)

export async function error(msg: string): Promise<void> {
  console.error(msg)
}

export async function warn(msg: string): Promise<void> {
  console.warn(msg)
}

export async function info(msg: string): Promise<void> {
  console.warn(msg)
}

export async function debug(msg: string): Promise<void> {
  console.warn(msg)
}

export async function trace(msg: string): Promise<void> {
  console.warn(msg)
}
