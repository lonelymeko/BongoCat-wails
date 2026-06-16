// Bridge for `@tauri-apps/api/tray`. The real tray now lives in Go (main.go);
// these stubs keep useTray.ts compiling while doing nothing on the JS side.

export interface TrayIconOptions {
  [k: string]: any
}

export class TrayIcon {
  static async getById(_id: string): Promise<TrayIcon | null> {
    return null
  }

  static async new(_options: TrayIconOptions): Promise<TrayIcon> {
    return new TrayIcon()
  }

  async setMenu(_menu: any): Promise<void> {}

  async setVisible(_visible: boolean): Promise<void> {}

  async setTooltip(_tooltip: string): Promise<void> {}
}
