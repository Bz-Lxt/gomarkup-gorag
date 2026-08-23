<template>
  <section class="w-full space-y-6">
    <h2 class="font-display text-4xl">检索调试</h2>
    <p class="text-paper/70">并排对比 HNSW 与 FLAT Ground Truth，观察召回率。评测会在内存中生成向量，不调用计费 API。</p>
    <div class="grid md:grid-cols-3 gap-3">
      <label>TopK
        <input v-model.number="topK" type="number" min="1" max="50" class="w-full bg-ink border border-line px-3 py-2" />
      </label>
      <label>Metric
        <select v-model="metric" class="w-full bg-ink border border-line px-3 py-2">
          <option value="cosine">cosine</option>
          <option value="l2">l2</option>
        </select>
      </label>
      <label>Index
        <select v-model="indexType" class="w-full bg-ink border border-line px-3 py-2">
          <option value="hnsw">hnsw</option>
          <option value="flat">flat</option>
        </select>
      </label>
    </div>
    <div class="flex gap-3">
      <input v-model="q" class="flex-1 bg-ink border border-line px-3 py-2" />
      <button type="button" class="px-4 py-2 bg-cadmium text-ink" @click="compare">对比检索</button>
      <button type="button" class="px-4 py-2 border border-line" @click="bench">合成评测</button>
    </div>
    <p v-if="recall !== null" class="font-mono text-cyan">在线对比 Recall@K = {{ recall.toFixed(3) }}</p>
    <div class="grid md:grid-cols-2 gap-4">
      <div>
        <h3 class="font-display mb-2">HNSW / 当前索引</h3>
        <ul class="space-y-2">
          <li v-for="h in left" :key="h.id" class="border border-line p-2 text-sm">
            <span class="font-mono text-cadmium">{{ h.score.toFixed(3) }}</span> · {{ h.title || h.caption || h.content?.slice(0, 48) }}
          </li>
        </ul>
      </div>
      <div>
        <h3 class="font-display mb-2">FLAT Ground Truth</h3>
        <ul class="space-y-2">
          <li v-for="h in right" :key="h.id" class="border border-line p-2 text-sm">
            <span class="font-mono text-cyan">{{ h.score.toFixed(3) }}</span> · {{ h.title || h.caption || h.content?.slice(0, 48) }}
          </li>
        </ul>
      </div>
    </div>
    <pre v-if="benchOut" class="font-mono text-xs border border-line p-4 overflow-auto">{{ benchOut }}</pre>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { evalRecall, searchText, type SearchHit } from '../api'
import { useToast } from '../stores/toast'

const q = ref('向量检索')
const topK = ref(10)
const metric = ref('cosine')
const indexType = ref('hnsw')
const left = ref<SearchHit[]>([])
const right = ref<SearchHit[]>([])
const recall = ref<number | null>(null)
const benchOut = ref('')
const toast = useToast()

async function compare() {
  try {
    const r = await searchText({
      query: q.value, top_k: topK.value, metric: metric.value, index_type: indexType.value, compare_flat: true,
    })
    left.value = r.hits
    right.value = r.flat_hits || []
    recall.value = r.recall_at_k ?? null
  } catch (e) {
    toast.push((e as Error).message, 'err')
  }
}

async function bench() {
  try {
    benchOut.value = JSON.stringify(await evalRecall(800), null, 2)
  } catch (e) {
    toast.push((e as Error).message, 'err')
  }
}
</script>
