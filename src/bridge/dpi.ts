// Bridge for `@tauri-apps/api/dpi`. These classes are used as VALUES
// (constructed with `new`), so they must be real classes.
// Note: existing code constructs PhysicalSize and LogicalSize with BOTH an
// object arg `{ width, height }` (main/index.vue, useModel.ts) and the
// positional `(w, h)` form (useWindowState.ts) — both are supported.

export class PhysicalPosition {
  x: number
  y: number

  constructor(x: number, y: number) {
    this.x = x
    this.y = y
  }

  toLogical(scaleFactor: number): { x: number, y: number } {
    return { x: this.x / scaleFactor, y: this.y / scaleFactor }
  }
}

export class PhysicalSize {
  width: number
  height: number

  constructor(arg: { width: number, height: number } | number, h?: number) {
    if (typeof arg === 'object') {
      this.width = arg.width
      this.height = arg.height
    } else {
      this.width = arg
      this.height = h!
    }
  }
}

export class LogicalSize {
  width: number
  height: number

  constructor(arg: { width: number, height: number } | number, h?: number) {
    if (typeof arg === 'object') {
      this.width = arg.width
      this.height = arg.height
    } else {
      this.width = arg
      this.height = h!
    }
  }
}
