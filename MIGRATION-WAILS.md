# BongoCat — Tauri → Wails v3 迁移 (M1 / MVP)

本文档记录把 BongoCat 后端从 **Tauri (Rust)** 迁移到 **Wails v3 (Go)** 的 M1 成果、如何运行/构建,以及还没做的部分。

> ⚠️ **构建平台**:目标是 **macOS / Windows**。迁移开发是在无显示器的 Linux 上完成的,**Go 的 cgo/GUI 部分无法在该环境编译**,因此首次编译与界面验证必须在你本机的 macOS/Windows 上进行。前端 (`vite build`) 已在此验证通过。

---

## 架构

```
┌─────────────────────────────────────┐     ┌──────────────────────────────┐
│  Vue3 前端 (几乎零改动)               │     │  Go 后端 (services/)          │
│                                     │     │                              │
│  组件 import '@tauri-apps/*'  ──┐    │     │  AppService   (app/os/path/  │
│                                │    │     │                fs/dialog/...) │
│  vite.config.ts resolve.alias  │    │ IPC │  DeviceService(gohook 键鼠)   │
│        │  把每个 tauri 模块映射 │ ◄──┼─────┤  WindowService(显示/置顶/穿透)│
│        ▼  到 src/bridge/*.ts   │    │     │  StoreService (JSON 持久化)   │
│  src/bridge/* ── @wailsio/runtime ──┘    │  main.go      (两窗口/托盘)    │
└─────────────────────────────────────┘     └──────────────────────────────┘
```

**核心思路:不重写组件**。前端仍 `import` 原来的 `@tauri-apps/*`,由 `vite.config.ts` 的 `resolve.alias` 把这些导入重定向到 `src/bridge/` 下的兼容层 shim,shim 底层走 Wails 运行时 (`@wailsio/runtime`) 和我们的 Go 服务。

### Go 服务 → 前端调用约定

前端通过 `Call.ByName('bongocat/services.<Service>.<Method>', ...args)` 调用 Go(封装在 `src/bridge/_call.ts`)。

| 服务            | 替代的 Tauri 能力                                                                                              |
| --------------- | -------------------------------------------------------------------------------------------------------------- |
| `AppService`    | app 信息、os、path、process、opener、fs、dialog、clipboard、autostart、locale、admin-status、macos-permissions |
| `DeviceService` | `start_device_listening` + `device-changed` 事件(`github.com/robotn/gohook`)                                   |
| `WindowService` | custom-window 插件:显示/隐藏、置顶、穿透 (`setIgnoreCursorEvents`)、可见性                                     |
| `StoreService`  | `@tauri-store/pinia` 的持久化(`src/bridge/storePinia.ts` 实现 pinia 插件)                                      |

### 关键文件

- `main.go` — 应用入口:两个窗口(透明无边框置顶的 `main` + 设置窗 `preference`)、系统托盘、单实例、资源中间件。
- `services/*.go` — 后端服务(见上表);`keymap.go` 是 gohook 键码 → rdev 名的映射表。
- `src/bridge/*.ts` — 23 个 Tauri 兼容 shim。
- `vite.config.ts` — `resolve.alias` 把 19 个 tauri 包指到 bridge。
- `Taskfile.yml` / `build/config.yml` — 构建配置。

### 资源加载(Live2D 模型)

Live2D 模型在运行时从磁盘读取。`convertFileSrc(absPath)` 返回 `/_bongo/resource?path=...`,由 `main.go` 的 `resourceMiddleware` 提供文件(等价于 Tauri 的 `asset://` 协议)。`resolveResource('assets/models')` 由 `AppService.ResolveResource` 解析:

- 开发态:自动指向仓库内的 `./src-tauri`(其下已有 `assets/models`),**无需移动文件**。
- 生产态:把 `src-tauri/assets` 拷到二进制同目录,或设环境变量 `BONGOCAT_RESOURCES=<含 assets 的目录>`。

---

## 运行 / 构建

### 1) 开发(不依赖 wails3 CLI,推荐先用这个跑通)

需要两个终端:

```bash
# 终端 1:前端 dev server (Vite, :1420)
pnpm install
pnpm dev          # 或 task dev:frontend

# 终端 2:Go/Wails app,连上 dev server
FRONTEND_DEVSERVER_URL=http://localhost:1420 go run .   # 或 task dev:app
```

app 在 dev 构建下读取 `FRONTEND_DEVSERVER_URL` 直接用 Vite(带热更新);未设置时回退到内嵌 `dist`。

### 2) 生产二进制(前端内嵌)

```bash
task build        # = pnpm exec vite build && go build -tags production -o bin/BongoCat .
./bin/BongoCat
```

### 3) 完整打包 (dmg / nsis / 图标) —— 需要 wails3 CLI

本仓库**没有**包含 `wails3 init` 生成的整套平台脚手架(`build/<os>/Taskfile.yml`、`Info.plist`、nsis 等),因为它和 CLI 版本强绑定。在你本机这样补齐:

```bash
# 1. 用与你 CLI 匹配的版本生成模板到临时目录
wails3 init -n BongoCat -t vue -q -d /tmp/bc-scaffold
# 2. 把脚手架拷进本仓库(不要覆盖我们的 main.go / services / 前端)
cp /tmp/bc-scaffold/Taskfile.yml .            # 如需,合并而非覆盖
cp -r /tmp/bc-scaffold/build/darwin build/    # 各平台子目录
cp -r /tmp/bc-scaffold/build/windows build/
# 3. build/config.yml 已按 BongoCat 填好;把前端目录/命令指到仓库根(pnpm)
# 4. 之后即可:
wails3 dev        # 开发热重载
wails3 build      # 构建
wails3 package    # 出 .app / installer
```

安装 CLI:`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`(Linux 上需 `pkg-config` 和 webkitgtk;macOS/Windows 按官方文档装依赖)。

---

## 已知缺口 / MVP 限制(后续里程碑)

| 项                                    | 状态   | 说明                                                                                                                                                                        |
| ------------------------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **手柄 (gamepad)**                    | 未做   | `useGamepad` 的 `start/stop_gamepad_listing` 在 bridge 里是 no-op。Go 生态无好用的跨平台手柄库,后续可上 SDL2(cgo)。                                                         |
| **键码映射**                          | 部分   | `services/keymap.go` 覆盖字母/数字/常用键/方向键/修饰键。**未覆盖**:CapsLock(gohook 规范码无此项,需 per-OS rawcode)、左右 Ctrl 区分(只发 `ControlLeft`)。需在真机逐项核对。 |
| **macOS 全屏覆盖/穿透**               | 基础版 | 用 Wails 内置 `Mac.WindowLevel=Floating` + `CollectionBehavior`,未做原版的自定义 NSPanel。"盖在全屏 App 上 + 完全穿透"体验可能打折。                                        |
| **Windows 激进置顶**                  | 简化   | 用 Wails 内置 `SetAlwaysOnTop`,未复刻原版 16ms `SetWindowPos(HWND_TOPMOST)` 线程。若被其它置顶窗遮挡再补 `window_windows.go`。                                              |
| **自动更新**                          | stub   | `plugin-updater` 的 `check()` 返回 null。Wails v3 有 `app.Updater`,后续可接。                                                                                               |
| **全局快捷键**                        | stub   | `useKeyPress` 的 register/unregister 是 no-op,行为快捷键暂不可用。                                                                                                          |
| **macOS 权限检查**                    | stub   | `CheckInputMonitoring` 恒返回 true。**首次在 macOS 运行需手动到「系统设置 → 隐私与安全性 → 输入监控」给 BongoCat 授权**,否则收不到键鼠事件。                                |
| **skip-taskbar 运行时切换**           | 固定   | 主窗创建时即隐藏于任务栏;Wails alpha 无运行时切换接口,`SetTaskbarVisibility` 为 no-op。                                                                                     |
| **窗口缩放变化事件**                  | no-op  | `onScaleChanged` 暂不触发(显示器 DPI 改变时不重算)。                                                                                                                        |
| `OSVersion()` / `os.arch()/version()` | 空     | 仅「关于」页展示用,返回空字符串。                                                                                                                                           |
| **持久化首帧竞态**                    | 已知   | store 从磁盘异步回填,极早的读取可能拿到默认值(随后即被填充)。                                                                                                               |

---

## 本机验证清单(Mac / Windows)

1. `task build` 能出二进制 / `task dev:*` 能起窗。
2. 透明无边框的猫显示,且**置顶**、不在任务栏。
3. 打字 → 手部动画;鼠标移动 → 头/眼跟随;左右键 → 对应动作。
4. 托盘菜单:Preferences 打开设置窗、Show/Hide Cat、Quit 生效。
5. 穿透模式开启后鼠标可穿过猫点到下层窗口。
6. 缩放/透明度等设置改完**重启仍保留**(StoreService 持久化)。
7. macOS:确认已授权「输入监控」;确认猫能浮在全屏应用之上。
8. Windows:键码映射是否齐全(尤其修饰键/CapsLock 这类已知缺口)。
9. dev 模式下确认 `/_bongo/resource?path=...` 能加载到模型贴图(资源中间件在 dev 下是否生效需重点验证)。

---

## 旧 Tauri 代码

`src-tauri/` 暂时保留(其 `assets/` 在开发态被复用为资源目录)。迁移验证通过后可删除其 Rust 部分;`assets/` 建议保留或迁到稳定的资源目录。
