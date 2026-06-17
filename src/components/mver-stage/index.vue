<script setup lang="ts">
import { convertFileSrc } from '@tauri-apps/api/core'
import { computed, reactive } from 'vue'

import type { MverManifest } from '@/utils/mver'

import { useTauriListen } from '@/composables/useTauriListen'
import { LISTEN_KEY } from '@/constants'
import { expandKeyName } from '@/utils/mver'
import { join } from '@/utils/path'

// Faithful renderer for legacy Bongo-Cat-Mver models. Replicates Mver's draw
// semantics (drawkeyboard = stack every matching combo; drawhand = the most
// recently pressed combo wins, with an idle fallback) using its original
// full-frame sprite layers. Keyboard / gamepad modes are fully reproduced here;
// the standard mode's procedural arm + mouse-following device are a follow-up
// (TODO: see utils/mver.ts setRightHand / cursorToDevice).

const props = defineProps<{
  path: string
  manifest: MverManifest
}>()

interface DeviceEvent {
  kind: 'KeyboardPress' | 'KeyboardRelease' | 'MousePress' | 'MouseRelease' | 'MouseMove'
  value: string | { x: number, y: number }
}

// `down` is a reactive Set, so reading it inside the computed below tracks it
// and the layers re-evaluate on every key/mouse state change.
const down = reactive(new Set<string>())
const pressSeq = new Map<string, number>()
let seq = 0

function press(name: string) {
  for (const expanded of expandKeyName(name)) {
    down.add(expanded)
    pressSeq.set(expanded, ++seq)
  }
}

function release(name: string) {
  for (const expanded of expandKeyName(name)) {
    down.delete(expanded)
  }
}

useTauriListen<DeviceEvent>(LISTEN_KEY.DEVICE_CHANGED, ({ payload }) => {
  const { kind, value } = payload

  switch (kind) {
    case 'KeyboardPress':
    case 'MousePress':
      press(value as string)
      break
    case 'KeyboardRelease':
    case 'MouseRelease':
      release(value as string)
      break
  }
})

function src(rel: string) {
  return convertFileSrc(join(props.path, rel))
}

function comboActive(keys: string[]) {
  return keys.length > 0 && keys.every(key => down.has(key))
}

function comboRecency(keys: string[]) {
  return Math.max(0, ...keys.map(key => pressSeq.get(key) ?? 0))
}

// Stacked categories: every currently-pressed combo is shown.
function stackImages(category: string): string[] {
  const layers = props.manifest.layers[category]
  if (!layers) return []
  return layers.filter(layer => comboActive(layer.keys)).map(layer => src(layer.img))
}

// Hand categories: only the most-recently-pressed matching combo, else idle.
function handImage(category: string): string | undefined {
  const layers = props.manifest.layers[category]
  if (!layers) {
    return idleImage(category)
  }

  let best: { img: string, recency: number } | undefined
  for (const layer of layers) {
    if (!comboActive(layer.keys)) continue
    const recency = comboRecency(layer.keys)
    if (!best || recency > best.recency) {
      best = { img: layer.img, recency }
    }
  }

  return best ? src(best.img) : idleImage(category)
}

function idleImage(category: string): string | undefined {
  const rel = props.manifest.idle?.[category]
  return rel ? src(rel) : undefined
}

// Ordered list of layer image URLs, bottom → top, matching Mver's draw order.
const orderedImages = computed<string[]>(() => {
  const images: (string | undefined)[] = []

  if (props.manifest.background) {
    images.push(src(props.manifest.background))
  }

  if (props.manifest.mode === 'standard') {
    // bg → (arm/device TODO) → left hand → keyboard → face
    images.push(handImage('hand'))
    images.push(...stackImages('keyboard'))
  } else {
    // bg → keyboard → right hand → left hand → face
    images.push(...stackImages('keyboard'))
    images.push(handImage('righthand'))
    images.push(handImage('lefthand'))
  }

  images.push(...stackImages('face'))

  return images.filter(Boolean) as string[]
})
</script>

<template>
  <div class="absolute size-full">
    <img
      v-for="(image, index) in orderedImages"
      :key="`${index}-${image}`"
      class="absolute size-full object-cover"
      :src="image"
      :style="{ zIndex: index }"
    >
  </div>
</template>
