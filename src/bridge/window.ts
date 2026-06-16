// Bridge for `@tauri-apps/api/window`. Provides monitor/cursor helpers backed
// by the Wails `Screens` runtime + the cursor cache from the event bridge.

import { Screens } from '@wailsio/runtime'

import { PhysicalPosition } from './dpi'
import { getLastCursor } from './event'

export type { PhysicalPosition } from './dpi'

// Shape of the Wails runtime Screen object (defined locally to avoid relying
// on the alpha package's type exports).
interface Screen {
  ID: string
  Name: string
  ScaleFactor: number
  X: number
  Y: number
  Size: { Width: number, Height: number }
  Bounds: { X: number, Y: number, Width: number, Height: number }
  WorkArea: { X: number, Y: number, Width: number, Height: number }
  IsPrimary: boolean
}

export interface Monitor {
  name: string | null
  size: { width: number, height: number }
  position: { x: number, y: number }
  scaleFactor: number
}

function toMonitor(s: Screen): Monitor {
  return {
    name: s.Name ?? null,
    size: { width: s.Size.Width, height: s.Size.Height },
    position: { x: s.X, y: s.Y },
    scaleFactor: s.ScaleFactor,
  }
}

export async function availableMonitors(): Promise<Monitor[]> {
  const screens = await Screens.GetAll()
  return screens.map(toMonitor)
}

// x,y are LOGICAL coords (caller passes cursor.toLogical). Screen bounds are
// physical; this is a best-effort hit-test against each screen's Bounds. We
// fall back to the primary screen when no bounds contain the point.
export async function monitorFromPoint(x: number, y: number): Promise<Monitor | null> {
  const screens = await Screens.GetAll()

  const hit = screens.find((s) => {
    const b = s.Bounds
    return x >= b.X && x <= b.X + b.Width && y >= b.Y && y <= b.Y + b.Height
  })

  if (hit) return toMonitor(hit)

  const primary = screens.find(s => s.IsPrimary) ?? screens[0]
  return primary ? toMonitor(primary) : null
}

export async function cursorPosition(): Promise<PhysicalPosition> {
  const last = getLastCursor()
  return new PhysicalPosition(last.x, last.y)
}
