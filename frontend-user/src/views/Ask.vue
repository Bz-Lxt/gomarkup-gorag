<template>
  <section class="w-full space-y-6">
    <header>
      <h2 class="font-display text-4xl">知识库问答</h2>
      <p class="text-paper/70 mt-2">检索增强生成。Mock 模式逐字吐出抽取式回答，并带 [MOCK] 标识。</p>
    </header>
    <div class="flex flex-col md:flex-row gap-3">
      <input v-model="q" class="flex-1 bg-ink border border-line px-4 py-3" placeholder="问一个关于知识库的问题…" @keydown.enter="ask" />
      <button type="button" class="px-6 py-3 bg-cadmium text-ink font-display" @click="ask">
        提问
        <span v-if="estimate > 0" class="ml-2 font-mono text-xs">约 ¥{{ estimate.toFixed(3) }}</span>
        <span v-else class="ml-2 font-mono text-xs">¥0 Mock</span>
      </button>
    </div>
    <article class="border border-line bg-mist/50 p-6 min-h-40 font-serif leading-relaxed whitespace-pre-wrap">{{ answer || '回答将出现在这里。' }}</article>
    <div class="grid md:grid-cols-2 gap-3">
      <button
        v-for="c in cites"
        :key="c.index"
        type="button"
        class="text-left border border-line p-3 hover:border-cadmium"
        @click="focus = c"
      >
        <p class="font-mono text-xs text-cadmium">[{{ c.index }}] score {{ c.hit.score.toFixed(3) }}</p>
        <p class="text-sm mt-1">{{ c.hit.title || c.hit.caption || c.hit.content?.slice(0, 80) }}</p>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { meta, token } from '../api'
import { useToast } from '../stores/toast'

const q = ref('什么是混合检索？')
const answer = ref('')
const estimate = ref(0)
const cites = ref<{ index: number; hit: { score: number; title?: string; caption?: string; content?: string } }[]>([])
const focus = ref<(typeof cites.value)[0] | null>(null)
const toast = useToast()

onMounted(async () => {
  try {
    const m = await meta()
    estimate.value = m.estimate_rag_cny
  } catch {
    /* ignore */
  }
})

async function ask() {
  if (!q.value.trim()) {
    toast.push('问题不能为空', 'err')
    return
  }
  answer.value = ''
  cites.value = []
  const res = await fetch('/api/v1/rag/query', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token()}` },
    body: JSON.stringify({ question: q.value, top_k: 6 }),
  })
  if (!res.ok || !res.body) {
    toast.push('问答失败', 'err')
    return
  }
  const reader = res.body.getReader()
  const dec = new TextDecoder()
  let buf = ''
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buf += dec.decode(value, { stream: true })
    const parts = buf.split('\n\n')
    buf = parts.pop() || ''
    for (const block of parts) {
      const ev = /event: (\w+)/.exec(block)?.[1]
      const dataLine = block.split('\n').find((l) => l.startsWith('data:'))
      if (!dataLine) continue
      const data = JSON.parse(dataLine.slice(5).trim() || '{}')
      if (ev === 'meta') cites.value = data.citations || []
      if (ev === 'token' && data.text) answer.value += data.text
      if (ev === 'error') toast.push(data.error || 'stream error', 'err')
    }
  }
}
</script>
