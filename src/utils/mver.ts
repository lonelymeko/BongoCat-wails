// Types + helpers for the legacy Bongo-Cat-Mver renderer (see components/mver-stage).
//
// A converted Mver model ships a `mver.json` manifest (written by Go's
// services/model_import.go) that preserves Mver's original full-frame sprite
// layers. The frontend replicates Mver's own draw semantics rather than forcing
// the images into BongoCat's Live2D + keycap system.

export interface MverLayer {
  /** rdev-style key names; the combo is active when ALL are pressed. */
  keys: string[]
  /** image path relative to the model dir, e.g. "keyboard/0.png". */
  img: string
}

export interface MverManifest {
  renderer: 'mver'
  mode: 'standard' | 'keyboard' | 'gamepad'
  windowSize: [number, number]
  catModel?: string
  background?: string
  decoration: {
    l2dCorrect: number
    l2dFlip: boolean
    leftHanded: boolean
    offsetX: number
    offsetY: number
    scale: number
    handOffset: number[]
    armColor: number[]
  }
  device?: {
    base?: string
    left?: string
    right?: string
    side?: string
    arm?: string
    up?: string
  }
  layers: Record<string, MverLayer[]>
  idle?: Record<string, string>
}

// Categories drawn "stacked" (every matching combo shows at once), mirroring
// Mver's drawkeyboard().
export const STACK_CATEGORIES = ['keyboard', 'face'] as const
// Categories where only the most-recently-pressed matching combo shows, with an
// idle fallback, mirroring Mver's drawhand().
export const HAND_CATEGORIES = ['hand', 'lefthand', 'righthand'] as const

/**
 * Expand a device key name into the names it should satisfy in a combo. The
 * DeviceService emits side-specific modifiers ("ShiftLeft"), while Mver combos
 * use the base name ("Shift"), so we add the collapsed form too.
 */
export function expandKeyName(name: string): string[] {
  const names = [name]

  for (const base of ['Shift', 'Control', 'Alt', 'Meta']) {
    if (name.startsWith(base) && name !== base) {
      names.push(base)
    }
  }

  return names
}

// --- standard-mode procedural arm (kuvster's bongo.cat "magic"), ported
// verbatim from Bongo-Cat-Mver's catfunc.cpp setrighthand()/bezier(). -------

function binom(n: number, k: number): number {
  let result = 1
  for (let i = 0; i < k; i++) {
    result = (result * (n - i)) / (i + 1)
  }
  return result
}

function bezier(ratio: number, points: number[]): [number, number] {
  const n = points.length / 2 - 1
  let x = 0
  let y = 0
  for (let i = 0; i <= n; i++) {
    const b = binom(n, i) * ratio ** i * (1 - ratio) ** (n - i)
    x += points[2 * i] * b
    y += points[2 * i + 1] * b
  }
  return [x, y]
}

export interface RightHand {
  /** anchor position for the mouse/tablet device sprite. */
  mpos: [number, number]
  /** 26-point arm outline polygon (flat [x0,y0,x1,y1,...]). */
  poly: number[]
}

/**
 * Compute the procedural right arm reaching from the cat's shoulder to the
 * cursor-mapped point (x, y). Coordinates are in Mver's native window_size
 * space; the stage scales them to the actual window. dx/dy match the C++
 * defaults (-38, -50).
 */
export function setRightHand(x: number, y: number, dx = -38, dy = -50): RightHand {
  const oof = 6
  const pss: number[] = [211, 159]

  let dist = Math.hypot(211 - x, 159 - y)
  const cl0 = 211 - (0.7237 * dist) / 2
  const cl1 = 159 + (0.69 * dist) / 2
  for (let i = 1; i < oof; i++) {
    pss.push(...bezier(i / oof, [211, 159, cl0, cl1, x, y]))
  }
  pss.push(x, y)

  let a = y - cl1
  let b = cl0 - x
  const le0 = Math.hypot(a, b)
  a = x + (a / le0) * 60
  b = y + (b / le0) * 60

  const a1 = 258
  const a2 = 228
  dist = Math.hypot(a1 - a, a2 - b)
  const cr0 = a1 - (0.6 * dist) / 2
  const cr1 = a2 + (0.8 * dist) / 2

  const push = 20
  let s = x - cl0
  let t = y - cl1
  let le = Math.hypot(s, t)
  s *= push / le
  t *= push / le
  let s2 = a - cr0
  let t2 = b - cr1
  le = Math.hypot(s2, t2)
  s2 *= push / le
  t2 *= push / le

  for (let i = 1; i < oof; i++) {
    pss.push(...bezier(i / oof, [x, y, x + s, y + t, a + s2, b + t2, a, b]))
  }
  pss.push(a, b)

  for (let i = oof - 1; i > 0; i--) {
    pss.push(...bezier(i / oof, [a1, a2, cr0, cr1, a, b]))
  }
  pss.push(a1, a2)

  const mpos: [number, number] = [(a + x) / 2 - 67, (b + y) / 2 - 29]

  const iter = 25
  const poly: number[] = [pss[0] + dx, pss[1] + dy]
  for (let i = 1; i < iter; i++) {
    const [p0, p1] = bezier(i / iter, pss)
    poly.push(p0 + dx, p1 + dy)
  }
  poly.push(pss[36] + dx, pss[37] + dy)

  return { mpos, poly }
}

/**
 * Map a screen cursor ratio (fx, fy in [0,1]) to Mver's device anchor space,
 * matching mode98_live2d_standard.cpp: x = -97fx + 44fy + 184, y = -76fx - 40fy + 324.
 */
export function cursorToDevice(fx: number, fy: number, leftHanded: boolean): [number, number] {
  if (leftHanded) fx = 1 - fx
  fx = Math.min(Math.max(fx, 0), 1)
  fy = Math.min(Math.max(fy, 0), 1)
  return [-97 * fx + 44 * fy + 184, -76 * fx - 40 * fy + 324]
}
