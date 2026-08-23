<template>
  <article
    class="absolute bg-mist border border-line overflow-hidden"
    :style="style"
    @mouseenter="hover = true"
    @mouseleave="hover = false"
  >
    <div v-if="hit.modality === 'image'" class="relative">
      <img :src="hit.asset_url" :alt="hit.caption || 'image'" class="w-full block" loading="lazy" />
      <div v-if="hover" class="absolute inset-0">
        <div
          v-for="(b, i) in hit.evidence.bbox"
          :key="i"
          class="absolute border border-cadmium"
          :style="boxStyle(b)"
        />
      </div>
    </div>
    <div class="p-3 space-y-2">
      <div class="flex items-center justify-between gap-2">
        <span class="font-mono text-cadmium text-xs">{{ hit.score.toFixed(3) }}</span>
        <div class="flex gap-1">
          <span v-if="hit.channels.vector" class="text-[10px] px-1.5 py-0.5 bg-cyan/20 text-cyan">VEC {{ hit.channels.vector }}</span>
          <span v-if="hit.channels.keyword" class="text-[10px] px-1.5 py-0.5 bg-gold/20 text-gold">KEY {{ hit.channels.keyword }}</span>
          <span class="text-[10px] px-1.5 py-0.5 border border-paper/20">RRF {{ hit.channels.rrf.toFixed(3) }}</span>
        </div>
      </div>
      <h3 class="font-display text-base leading-snug">{{ hit.title || hit.caption || '段落' }}</h3>
      <p v-if="hit.modality === 'text'" class="text-sm text-paper/85 leading-relaxed" v-html="highlighted" />
      <p v-else class="text-sm text-paper/70">{{ hit.caption }}</p>
      <div class="flex flex-wrap gap-1">
        <span v-for="t in hit.tags || []" :key="t" class="text-[10px] uppercase tracking-wider text-paper/50">#{{ t }}</span>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { SearchHit } from '../api'

const props = defineProps<{ hit: SearchHit; left: number; top: number; width: number }>()
const hover = ref(false)
const style = computed(() => ({
  left: props.left + 'px',
  top: props.top + 'px',
  width: props.width + 'px',
}))

function boxStyle(b: SearchHit['evidence']['bbox'][0]) {
  const [x, y, w, h] = b.box
  return {
    left: x * 100 + '%',
    top: y * 100 + '%',
    width: w * 100 + '%',
    height: h * 100 + '%',
    background: `rgba(228,87,46,${Math.min(0.45, 0.12 + b.score * 0.35)})`,
  }
}

const highlighted = computed(() => {
  const text = props.hit.content || ''
  const ranges = [...(props.hit.evidence.char_ranges || [])]
    .filter((r) => r.start >= 0 && r.end <= text.length && r.start < r.end)
    .sort((a, b) => a.start - b.start)
  if (!ranges.length) return escapeHtml(text)
  let out = ''
  let cur = 0
  for (const r of ranges) {
    if (r.start < cur) continue
    out += escapeHtml(text.slice(cur, r.start))
    out += `<mark>${escapeHtml(text.slice(r.start, r.end))}</mark>`
    cur = r.end
  }
  out += escapeHtml(text.slice(cur))
  return out
})

function escapeHtml(s: string) {
  return s.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c] as string))
}
</script>
