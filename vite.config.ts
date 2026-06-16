import vue from '@vitejs/plugin-vue'
import { resolve } from 'node:path'
import { env } from 'node:process'
import UnoCSS from 'unocss/vite'
import { defineConfig } from 'vite'
import vitePluginDayjs from 'vite-plugin-dayjs'

const host = env.TAURI_DEV_HOST

// Tauri-compat bridge: every `@tauri-apps/*` (and friends) import the frontend
// still uses is redirected to a local shim under `src/bridge/` that is backed by
// the Wails v3 runtime + our Go services. This lets the Vue code migrate from
// Tauri to Wails with (almost) no component changes. See src/bridge/*.
function bridge(file: string) {
  return resolve(__dirname, 'src/bridge', file)
}

const tauriBridgeAlias = {
  '@tauri-apps/api/core': bridge('core.ts'),
  '@tauri-apps/api/event': bridge('event.ts'),
  '@tauri-apps/api/dpi': bridge('dpi.ts'),
  '@tauri-apps/api/webviewWindow': bridge('webviewWindow.ts'),
  '@tauri-apps/api/window': bridge('window.ts'),
  '@tauri-apps/api/path': bridge('path.ts'),
  '@tauri-apps/api/app': bridge('app.ts'),
  '@tauri-apps/api/menu': bridge('menu.ts'),
  '@tauri-apps/api/tray': bridge('tray.ts'),
  '@tauri-apps/plugin-os': bridge('os.ts'),
  '@tauri-apps/plugin-process': bridge('process.ts'),
  '@tauri-apps/plugin-opener': bridge('opener.ts'),
  '@tauri-apps/plugin-fs': bridge('fs.ts'),
  '@tauri-apps/plugin-dialog': bridge('dialog.ts'),
  '@tauri-apps/plugin-clipboard-manager': bridge('clipboard.ts'),
  '@tauri-apps/plugin-autostart': bridge('autostart.ts'),
  '@tauri-apps/plugin-global-shortcut': bridge('globalShortcut.ts'),
  '@tauri-apps/plugin-updater': bridge('updater.ts'),
  '@tauri-apps/plugin-log': bridge('log.ts'),
  '@tauri-store/pinia': bridge('storePinia.ts'),
  'tauri-plugin-macos-permissions-api': bridge('macosPermissions.ts'),
  'tauri-plugin-locale-api': bridge('locale.ts'),
}

// https://vitejs.dev/config/
export default defineConfig(async () => ({
  plugins: [vue(), UnoCSS(), vitePluginDayjs()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
      ...tauriBridgeAlias,
    },
  },
  // Don't obscure backend (Go/Wails) errors.
  clearScreen: false,
  // Wails dev expects a fixed port; fail if it's unavailable.
  server: {
    port: 1420,
    strictPort: true,
    host: host || false,
    hmr: host
      ? {
          protocol: 'ws',
          host,
          port: 1421,
        }
      : undefined,
    watch: {
      // Ignore the Go backend and (legacy) Tauri sources.
      ignored: ['**/src-tauri/**', '**/services/**', '**/build/bin/**'],
    },
  },
}))
