<template>
  <section class="w-full space-y-8">
    <header class="space-y-2">
      <h2 class="font-display text-4xl md:text-5xl">以文搜图，或以图搜图</h2>
      <p class="text-paper/70 max-w-3xl">拖一张本地图片到输入区，或输入文字。未配置 CLIP 时，以文搜图走 caption / 标签的真实倒排通道，不会假装跨模态对齐。</p>
    </header>

    <div
      class="border border-dashed p-4 md:p-6 transition-colors"
      :class="over ? 'border-cadmium bg-cadmium/5' : 'border-line'"
      @dragover.prevent="over = true"
      @dragleave="over = false"
      @drop.prevent="onDrop"
    >
      <div class="flex flex-col md:flex-row gap-3">
        <input
          v-model="query"
          class="flex-1 bg-ink border border-line px-4 py-3 font-serif"
          placeholder="输入查询，或把图片拖进来…"
          @keydown.enter="run"
        />
        <select v-model="modality" class="bg-ink border border-line px-3 py-3">
          <option value="">全部模态</option>
          <option value="text">仅文档</option>
          <option value="image">仅图片</option>
        </select>
        <button type="button" class="px-6 py-3 bg-cadmium text-ink font-display" @click="run">检索</button>
      </div>
      <div v-if="file" class="mt-3 text-sm text-cyan">已选图片：{{ file.name }} <button class="underline" type="button" @click="file = null">清除</button></div>
      <input ref="picker" type="file" accept="image/*" class="hidden" @change="onPick" />
      <button type="button" class="mt-3 text-xs text-paper/50" @click="picker?.click()">或点击选择图片</button>
    </div>

    <div v-if="note" class="border border-gold/40 bg-gold/10 px-4 py-2 text-sm">{{ note }}</div>
    <div v-if="resp" class="flex flex-wrap gap-4 text-xs font-mono text-paper/60">
      <span>{{ resp.took_ms }} ms</span>
      <span>channels: {{ resp.channels.join(' + ') || '—' }}</span>
      <span>cross_modal: {{ resp.cross_modal }}</span>
    </div>

    <MasonryWall :hits="hits" />
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import MasonryWall from '../components/MasonryWall.vue'
import { searchHybrid, searchImage, searchText, type SearchHit, type SearchResp } from '../api'
import { useToast } from '../stores/toast'

const query = ref('')
const modality = ref('')
const file = ref<File | null>(null)
const over = ref(false)
const picker = ref<HTMLInputElement | null>(null)
const hits = ref<SearchHit[]>([])
const resp = ref<SearchResp | null>(null)
const note = ref('')
const toast = useToast()

function onDrop(e: DragEvent) {
  over.value = false
  const f = e.dataTransfer?.files?.[0]
  if (f) file.value = f
}

function onPick(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (f) file.value = f
}

async function run() {
  try {
    if (file.value) {
      resp.value = await searchImage(file.value, {
        top_k: '16',
        metric: 'cosine',
        index_type: 'hnsw',
      })
    } else if (query.value.trim()) {
      const body = {
        query: query.value,
        top_k: 16,
        modality: modality.value,
        compare_flat: false,
      }
      resp.value = modality.value === 'image' ? await searchHybrid(body) : await searchText(body)
    } else {
      toast.push('请输入文字或放入图片', 'err')
      return
    }
    hits.value = resp.value.hits || []
    note.value = resp.value.degrade_note || ''
  } catch (e) {
    toast.push((e as Error).message, 'err')
  }
}
</script>
