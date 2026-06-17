<script setup lang="ts">
import type { MotionInfo } from 'easy-live2d'

import { convertFileSrc } from '@tauri-apps/api/core'
import { PhysicalSize } from '@tauri-apps/api/dpi'
import { Menu, PredefinedMenuItem } from '@tauri-apps/api/menu'
import { sep } from '@tauri-apps/api/path'
import { getCurrentWebviewWindow } from '@tauri-apps/api/webviewWindow'
import { exists, readDir, readTextFile } from '@tauri-apps/plugin-fs'
import { useDebounceFn, useEventListener } from '@vueuse/core'
import { round } from 'es-toolkit'
import { nth } from 'es-toolkit/compat'
import { onMounted, onUnmounted, ref, watch } from 'vue'

import type { MverManifest } from '@/utils/mver'

import MverStage from '@/components/mver-stage/index.vue'
import { useAppMenu } from '@/composables/useAppMenu'
import { useDevice } from '@/composables/useDevice'
import { useGamepad } from '@/composables/useGamepad'
import { useModel } from '@/composables/useModel'
import { useTauriListen } from '@/composables/useTauriListen'
import { LISTEN_KEY } from '@/constants'
import { hideWindow, setAlwaysOnTop, setTaskbarVisibility, showWindow } from '@/plugins/window'
import { useCatStore } from '@/stores/cat'
import { useGeneralStore } from '@/stores/general.ts'
import { useModelStore } from '@/stores/model'
import { isImage } from '@/utils/is'
import live2d from '@/utils/live2d'
import { join } from '@/utils/path'
import { isWindows } from '@/utils/platform'
import { clearObject } from '@/utils/shared'

const { startListening } = useDevice()
const appWindow = getCurrentWebviewWindow()
const { modelSize, modelLayout, handleLoad, handleDestroy, handleResize, handleKeyChange } = useModel()
const catStore = useCatStore()
const { getBaseMenu, getExitMenu } = useAppMenu()
const modelStore = useModelStore()
const generalStore = useGeneralStore()
const resizing = ref(false)
const backgroundImagePath = ref<string>()
// The window/scene size is defined by the background image (the desk/keyboard),
// which the Live2D model + full-frame keycaps are all designed to overlay. For
// BongoCat presets bg == model size (no change); for converted Bongo-Cat-Mver
// models the bg/keycaps are larger than the Live2D canvas, so sizing the window
// to the bg keeps the character and the keypress hand at the same scale.
const sceneSize = ref<{ width: number, height: number }>()
// Set when the current model is a legacy Bongo-Cat-Mver package; the dedicated
// MverStage renderer takes over and the BongoCat keycap path is skipped.
const mverManifest = ref<MverManifest>()
const { stickActive } = useGamepad()

function imageSize(url: string) {
  return new Promise<{ width: number, height: number } | undefined>((resolve) => {
    const img = new Image()
    img.onload = () => resolve({ width: img.naturalWidth, height: img.naturalHeight })
    img.onerror = () => resolve(void 0)
    img.src = url
  })
}

onMounted(() => {
  void (appWindow as any).setBackgroundColour?.(0, 0, 0, 0)
  startListening()
})

onUnmounted(handleDestroy)

const debouncedResize = useDebounceFn(async () => {
  await handleResize()

  resizing.value = false
}, 100)

useEventListener('resize', () => {
  resizing.value = true

  debouncedResize()
})

watch(() => modelStore.currentModel, async (model) => {
  if (!model) return

  // Legacy Bongo-Cat-Mver model: hand off to the dedicated MverStage renderer
  // and skip the BongoCat Live2D + keycap path entirely.
  const mverPath = join(model.path, 'mver.json')

  if (await exists(mverPath)) {
    mverManifest.value = JSON.parse(await readTextFile(mverPath)) as MverManifest

    clearObject([modelStore.supportKeys, modelStore.pressedKeys])
    backgroundImagePath.value = void 0

    // The standard mode's Live2D body + procedural arm are a follow-up; for now
    // MverStage draws the sprite layers (keyboard/gamepad are fully reproduced).
    handleDestroy()

    modelStore.modelReady = true
    return
  }

  mverManifest.value = void 0

  await handleLoad()

  const path = join(model.path, 'resources', 'background.png')
  const existed = await exists(path)

  backgroundImagePath.value = existed ? convertFileSrc(path) : void 0

  // Size the window/scene to the background image, not the Live2D canvas.
  sceneSize.value = existed ? await imageSize(backgroundImagePath.value!) : void 0

  clearObject([modelStore.supportKeys, modelStore.pressedKeys])

  const resourcePath = join(model.path, 'resources')
  const groups = ['left-keys', 'right-keys']

  for await (const groupName of groups) {
    const groupDir = join(resourcePath, groupName)
    const files = await readDir(groupDir).catch(() => [])
    const imageFiles = files.filter(file => isImage(file.name))

    for (const file of imageFiles) {
      const fileName = file.name.split('.')[0]

      modelStore.supportKeys[fileName] = join(groupDir, file.name)
    }
  }

  modelStore.modelReady = true
}, { deep: true, immediate: true })

watch([() => catStore.window.scale, sceneSize, modelSize], async ([scale, sceneSize, modelSize]) => {
  // Prefer the background/scene size so the Live2D model and the full-frame
  // keycaps share one coordinate space; fall back to the model size.
  const size = sceneSize ?? modelSize

  if (!size) return

  const { width, height } = size

  await appWindow.setSize(
    new PhysicalSize({
      width: Math.round(width * (scale / 100)),
      height: Math.round(height * (scale / 100)),
    }),
  )
}, { immediate: true })

// Mver models size the window from the manifest's native windowSize instead of
// the Live2D model size.
watch([() => catStore.window.scale, mverManifest], async ([scale, manifest]) => {
  if (!manifest?.windowSize) return

  const [width, height] = manifest.windowSize

  await appWindow.setSize(
    new PhysicalSize({
      width: Math.round(width * (scale / 100)),
      height: Math.round(height * (scale / 100)),
    }),
  )
}, { immediate: true })

watch([modelStore.pressedKeys, stickActive], ([keys, stickActive]) => {
  const dirs = Object.values(keys).map((path) => {
    return nth(path.split(sep()), -2)!
  })

  const hasLeft = dirs.some(dir => dir.startsWith('left'))
  const hasRight = dirs.some(dir => dir.startsWith('right'))

  handleKeyChange(true, stickActive.left || hasLeft)
  handleKeyChange(false, stickActive.right || hasRight)
}, { deep: true })

watch(() => catStore.window.visible, async (value) => {
  value ? showWindow() : hideWindow()
})

watch(() => catStore.window.passThrough, (value) => {
  appWindow.setIgnoreCursorEvents(value)
}, { immediate: true })

watch(() => catStore.window.alwaysOnTop, setAlwaysOnTop, { immediate: true })

watch(() => generalStore.app.taskbarVisible, setTaskbarVisibility, { immediate: true })

watch(() => catStore.model.motionSound, live2d.setMotionSoundEnabled, { immediate: true })

watch(() => catStore.model.maxFPS, live2d.setMaxFPS, { immediate: true })

useTauriListen<MotionInfo>(LISTEN_KEY.START_MOTION, ({ payload }) => {
  live2d.startMotion(payload)
})

useTauriListen<number>(LISTEN_KEY.SET_EXPRESSION, ({ payload }) => {
  live2d.setExpression(payload)
})

function handleMouseDown() {
  appWindow.startDragging()
}

async function handleContextmenu(event: MouseEvent) {
  event.preventDefault()

  if (event.shiftKey) return

  const menu = await Menu.new({
    items: [
      ...await getBaseMenu(),
      await PredefinedMenuItem.new({ item: 'Separator' }),
      ...await getExitMenu(),
    ],
  })

  // Temporarily disable always-on-top on Windows so the context menu is not covered
  if (isWindows && catStore.window.alwaysOnTop) {
    setAlwaysOnTop(false)
  }

  await menu.popup()

  // Restore always-on-top after the menu is closed
  if (!isWindows || !catStore.window.alwaysOnTop) return

  setAlwaysOnTop(true)
}

function handleMouseMove(event: MouseEvent) {
  const { buttons, shiftKey, movementX, movementY } = event

  if (buttons !== 2 || !shiftKey) return

  const delta = (movementX + movementY) * 0.5
  const nextScale = Math.max(10, Math.min(catStore.window.scale + delta, 500))

  catStore.window.scale = round(nextScale)
}
</script>

<template>
  <div
    class="bongo-drag-region relative size-screen overflow-hidden bg-transparent children:(absolute size-full)"
    :class="{ '-scale-x-100': catStore.model.mirror }"
    :style="{
      'opacity': catStore.window.opacity / 100,
      'borderRadius': `${catStore.window.radius}%`,
      '--wails-draggable': 'drag',
    }"
    @contextmenu="handleContextmenu"
    @mousedown="handleMouseDown"
    @mousemove="handleMouseMove"
  >
    <img
      v-if="backgroundImagePath"
      class="object-cover"
      :class="modelLayout?.behindBase ? 'z-20' : 'z-0'"
      :src="backgroundImagePath"
    >

    <canvas
      id="live2dCanvas"
      class="z-10"
    />

    <MverStage
      v-if="mverManifest && modelStore.currentModel"
      :key="modelStore.currentModel.path"
      class="z-10"
      :manifest="mverManifest"
      :path="modelStore.currentModel.path"
    />

    <img
      v-for="path in modelStore.pressedKeys"
      :key="path"
      class="z-30 object-cover"
      :src="convertFileSrc(path)"
    >

    <div
      v-show="resizing || !modelStore.modelReady"
      class="z-40 flex items-center justify-center bg-transparent"
    >
      <span class="text-center text-[10vw] text-[#fff]">
        {{ resizing ? $t('pages.main.hints.redrawing') : $t('pages.main.hints.switching') }}
      </span>
    </div>
  </div>
</template>
