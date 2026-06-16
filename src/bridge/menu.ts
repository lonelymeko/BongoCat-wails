// Bridge for `@tauri-apps/api/menu`. The tray/app menu is now built in Go, so
// these are minimal, type-safe stubs that simply retain their options.
// DEFERRED: no JS-side rendering.

export class Menu {
  opts: any

  constructor(opts?: any) {
    this.opts = opts
  }

  static async new(opts?: any): Promise<Menu> {
    return new Menu(opts)
  }
}

export class MenuItem {
  opts: any

  constructor(opts?: any) {
    this.opts = opts
  }

  static async new(opts?: any): Promise<MenuItem> {
    return new MenuItem(opts)
  }
}

export class CheckMenuItem {
  opts: any

  constructor(opts?: any) {
    this.opts = opts
  }

  static async new(opts?: any): Promise<CheckMenuItem> {
    return new CheckMenuItem(opts)
  }
}

export class PredefinedMenuItem {
  opts: any

  constructor(opts?: any) {
    this.opts = opts
  }

  static async new(opts?: any): Promise<PredefinedMenuItem> {
    return new PredefinedMenuItem(opts)
  }
}

export class Submenu {
  opts: any

  constructor(opts?: any) {
    this.opts = opts
  }

  static async new(opts?: any): Promise<Submenu> {
    return new Submenu(opts)
  }
}
