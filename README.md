<div align="center">

# BongoCat · Wails v3 版

一只透明、置顶、会跟着你打字和挪鼠标动起来的 Live2D 桌宠。

**本仓库是 [ayangweb/BongoCat](https://github.com/ayangweb/BongoCat) 的 Wails v3 (Go) 改造版** —— 后端从 Tauri / Rust 迁移到 Wails v3 / Go,前端 Vue 几乎原样保留。

<p>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go&logoColor=white" />
  <img alt="Wails" src="https://img.shields.io/badge/Wails-v3--alpha-DF0000?style=flat-square" />
  <img alt="Vue" src="https://img.shields.io/badge/Vue-3-42b883?style=flat-square&logo=vuedotjs&logoColor=white" />
  <img alt="Vite" src="https://img.shields.io/badge/Vite-6-646CFF?style=flat-square&logo=vite&logoColor=white" />
  <a href="./LICENSE"><img alt="License" src="https://img.shields.io/badge/License-MIT-green?style=flat-square" /></a>
</p>

</div>

> ⚠️ **状态:M1 / MVP**。核心(透明猫窗 + 键鼠监听 + Live2D + 偏好持久化 + 托盘)已成型;手柄、自动更新、全局快捷键等仍是 stub。完整的迁移说明、架构细节与待办见 **[MIGRATION-WAILS.md](./MIGRATION-WAILS.md)**。
>
> 目标平台 **macOS / Windows**。代码在 headless Linux 上完成,**Go 的 cgo/GUI 部分需在 macOS/Windows 真机首次编译**。

---

## ✨ 功能

- 透明、无边框、置顶、可穿透的桌宠窗口
- 跟随键盘 / 鼠标(手柄待补)实时联动 Live2D 动作
- 支持导入自定义模型
- 偏好设置本地持久化,离线运行,不收集数据

## 🧩 技术栈

| 层       | 用的东西                                                                                                                                    |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| 前端     | Vue 3 · Vite 6 · Pinia · Live2D(pixi.js + easy-live2d)                                                                                      |
| 后端     | Go · [Wails v3](https://v3.wails.io)(alpha)                                                                                                 |
| 桥接     | `src/bridge/*` 把 `@tauri-apps/*` 导入经 Vite alias 重定向到 [`@wailsio/runtime`](https://www.npmjs.com/package/@wailsio/runtime) + Go 服务 |
| 原生能力 | 全局键鼠:[`robotn/gohook`](https://github.com/robotn/gohook) · 窗口/托盘/单实例:Wails                                                       |

简版架构:

```
Vue 组件  ──import '@tauri-apps/*'──►  vite alias  ──►  src/bridge/*.ts
                                                              │
                                  @wailsio/runtime (Events/Call/Window)
                                                              │
                                                        Go services
                              AppService · DeviceService · WindowService · StoreService
                                                              │
                                                      main.go(两窗口 + 托盘)
```

> 详见 [MIGRATION-WAILS.md](./MIGRATION-WAILS.md)。

## 📦 环境要求

- **Go ≥ 1.24**
- **Node ≥ 20** + **pnpm**(`corepack enable` 即可)
- 平台原生工具链(因为用到 cgo —— 系统 webview + 全局键鼠 hook):
  - **macOS**:Xcode Command Line Tools(`xcode-select --install`)
  - **Windows**:一个 C 编译器(MSYS2/MinGW 的 `gcc`)—— gohook 的全局 hook 需要
  - **Linux(x11)**:`webkit2gtk` / `gtk` 开发包 + `libX11/libXtst/libxcb*` 开发头
- 可选:[`wails3` CLI](https://v3.wails.io)(仅打包 dmg/installer 时需要)
  `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`

## 🚀 快速开始

```bash
git clone https://github.com/lonelymeko/BongoCat-wails.git
cd BongoCat-wails

pnpm install            # 装前端依赖
pnpm exec vite build    # ❗首次必须先构建一次前端:main.go 的 //go:embed 需要 dist/ 存在
```

## 🛠 开发

> macOS 首次运行前,请到 **系统设置 → 隐私与安全性 → 输入监控** 给 BongoCat 授权,否则收不到键鼠事件,猫不动。

### 方式 A:不依赖 wails3 CLI(推荐先用这个跑通)

开两个终端:

```bash
# 终端 1:Vite dev server(热更新,:1420)
pnpm dev                # 或 task dev:frontend

# 终端 2:Go/Wails app 连上 dev server
FRONTEND_DEVSERVER_URL=http://localhost:1420 go run .   # 或 task dev:app
```

app 在开发构建下读取 `FRONTEND_DEVSERVER_URL` 直接用 Vite;未设置时回退到内嵌的 `dist`。

### 方式 B:wails3 CLI(热重载更完整)

需要先补齐平台脚手架,见 [MIGRATION-WAILS.md](./MIGRATION-WAILS.md) 的「完整打包」一节,然后:

```bash
wails3 dev
```

## 🏗 构建

```bash
# 生产二进制(前端内嵌进可执行文件)
task build              # = pnpm exec vite build && go build -tags production -o bin/BongoCat .
./bin/BongoCat

# 不用 task 的等价命令:
pnpm exec vite build
go build -tags production -ldflags "-w -s" -o bin/BongoCat .
```

打包成 `.app` / `.dmg` / Windows 安装包需要 `wails3 package`,步骤见 [MIGRATION-WAILS.md](./MIGRATION-WAILS.md)。

> **资源(Live2D 模型)**:开发态自动复用仓库内的 `src-tauri/assets/models`;生产态需把 `src-tauri/assets` 拷到二进制同目录,或设环境变量 `BONGOCAT_RESOURCES=<含 assets 的目录>`。

## 📂 项目结构

```
.
├── main.go                 # 入口:两窗口、托盘、单实例、资源中间件
├── services/               # Go 后端服务
│   ├── app.go              #   app/os/path/fs/dialog/clipboard/opener/...
│   ├── device.go           #   全局键鼠监听(gohook)→ "device-changed" 事件
│   ├── window.go           #   显示/隐藏/置顶/穿透
│   ├── store.go            #   设置持久化(JSON)
│   └── keymap.go           #   gohook 键码 → rdev 名映射表
├── src/                    # Vue 前端(沿用原版)
│   └── bridge/             # Tauri 兼容层(23 个 shim)
├── vite.config.ts          # resolve.alias:@tauri-apps/* → src/bridge/*
├── Taskfile.yml            # task install / dev:* / build
├── build/config.yml        # wails3 CLI 构建配置
└── MIGRATION-WAILS.md      # 迁移详解 + 已知缺口 + 验证清单
```

## 🚧 已知缺口(M2 待办)

手柄、自动更新、全局快捷键、macOS NSPanel/权限真检查、键码映射(CapsLock、左右修饰键区分)、Windows 激进置顶。完整清单见 [MIGRATION-WAILS.md](./MIGRATION-WAILS.md#已知缺口--mvp-限制后续里程碑)。

## 🙏 致谢

- 原项目 [ayangweb/BongoCat](https://github.com/ayangweb/BongoCat) —— 本仓库是其 Wails 改造 fork。
- 灵感来源 [MMmmmoko/Bongo-Cat-Mver](https://github.com/MMmmmoko/Bongo-Cat-Mver)。
- 更多模型:[Awesome-BongoCat](https://github.com/ayangweb/Awesome-BongoCat)。

## 📄 License

[MIT](./LICENSE) —— 沿用原项目协议。
