<template>
  <div ref="root" class="relative w-full" :style="{ height: height + 'px' }">
    <ResultCard
      v-for="item in placed"
      :key="item.hit.id"
      :hit="item.hit"
      :left="item.left"
      :top="item.top"
      :width="item.width"
    />
    <p v-if="!hits.length" class="text-paper/50 font-serif py-16 text-center">尚无结果。试着输入「向量检索」或拖入一张色块图。</p>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { SearchHit } from '../api'
import ResultCard from './ResultCard.vue'

const props = defineProps<{ hits: SearchHit[] }>()
const root = ref<HTMLElement | null>(null)
const height = ref(0)
const placed = ref<{ hit: SearchHit; left: number; top: number; width: number }[]>([])

function colsFor(w: number) {
  if (w < 480) return 1
  if (w < 768) return 2
  if (w < 1200) return 3
  return 4
}

function layout() {
  const el = root.value
  if (!el) return
  const w = el.clientWidth
  const cols = colsFor(w)
  const gap = 16
  const colW = (w - gap * (cols - 1)) / cols
  const colH = Array(cols).fill(0)
  const next: typeof placed.value = []
  props.hits.forEach((hit) => {
    let c = 0
    for (let i = 1; i < cols; i++) if (colH[i] < colH[c]) c = i
    const est = hit.modality === 'image' ? colW * 0.85 + 120 : 180 + Math.min(160, (hit.content?.length || 40) * 0.4)
    next.push({ hit, left: c * (colW + gap), top: colH[c], width: colW })
    colH[c] += est + gap
  })
  placed.value = next
  height.value = Math.max(0, ...colH)
}

let ro: ResizeObserver | null = null
onMounted(() => {
  ro = new ResizeObserver(() => layout())
  if (root.value) ro.observe(root.value)
  layout()
})
onBeforeUnmount(() => ro?.disconnect())
watch(() => props.hits, async () => { await nextTick(); layout() }, { deep: true })
</script>
