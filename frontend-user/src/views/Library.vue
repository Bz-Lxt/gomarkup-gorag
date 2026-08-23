<template>
  <section class="w-full space-y-8">
    <h2 class="font-display text-4xl">数据管理</h2>
    <div class="grid lg:grid-cols-2 gap-6">
      <form class="border border-line p-5 space-y-3" @submit.prevent="addDoc">
        <h3 class="font-display text-xl">入库文档</h3>
        <input v-model="title" class="w-full bg-ink border border-line px-3 py-2" placeholder="标题 *" />
        <p v-if="err.title" class="text-cadmium text-xs">{{ err.title }}</p>
        <textarea v-model="content" rows="6" class="w-full bg-ink border border-line px-3 py-2" placeholder="正文 *" />
        <p v-if="err.content" class="text-cadmium text-xs">{{ err.content }}</p>
        <input v-model="tags" class="w-full bg-ink border border-line px-3 py-2" placeholder="标签，逗号分隔" />
        <button class="px-4 py-2 bg-cadmium text-ink" type="submit">写入切块</button>
      </form>
      <form class="border border-line p-5 space-y-3" @submit.prevent="addImg">
        <h3 class="font-display text-xl">入库图片</h3>
        <input type="file" accept="image/*" @change="onFile" />
        <p v-if="err.file" class="text-cadmium text-xs">{{ err.file }}</p>
        <input v-model="caption" class="w-full bg-ink border border-line px-3 py-2" placeholder="caption（以文搜图依赖它）" />
        <input v-model="imgtags" class="w-full bg-ink border border-line px-3 py-2" placeholder="tags" />
        <button class="px-4 py-2 border border-cadmium text-cadmium" type="submit">提取特征并入库</button>
      </form>
    </div>
    <div class="border border-line p-5 space-y-3">
      <div class="flex justify-between items-center">
        <h3 class="font-display text-xl">索引统计</h3>
        <button type="button" class="text-sm border border-line px-3 py-1" @click="confirmFlush = true">手动 Flush</button>
      </div>
      <pre class="font-mono text-xs whitespace-pre-wrap text-paper/80">{{ pretty }}</pre>
    </div>
    <Modal :open="confirmFlush" title="强制封口" text="将当前 Growing Buffer 异步落盘。确认继续？" @ok="doFlush" @cancel="confirmFlush = false" />
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import Modal from '../components/Modal.vue'
import { flush, ingestDoc, stats, uploadImage } from '../api'
import { useToast } from '../stores/toast'

const title = ref('')
const content = ref('')
const tags = ref('rag')
const caption = ref('')
const imgtags = ref('')
const file = ref<File | null>(null)
const pretty = ref('loading…')
const confirmFlush = ref(false)
const err = reactive({ title: '', content: '', file: '' })
const toast = useToast()

async function refresh() {
  pretty.value = JSON.stringify(await stats(), null, 2)
}

onMounted(refresh)

function onFile(e: Event) {
  file.value = (e.target as HTMLInputElement).files?.[0] || null
}

async function addDoc() {
  err.title = title.value.trim() ? '' : '标题必填'
  err.content = content.value.trim() ? '' : '正文必填'
  if (err.title || err.content) {
    toast.push('请修正表单', 'err')
    return
  }
  await ingestDoc(content.value, title.value, tags.value.split(',').map((s) => s.trim()).filter(Boolean))
  toast.push('文档已切块入库')
  await refresh()
}

async function addImg() {
  err.file = file.value ? '' : '请选择图片'
  if (err.file) {
    toast.push('请选择图片', 'err')
    return
  }
  await uploadImage(file.value!, caption.value, imgtags.value)
  toast.push('图片特征已写入')
  await refresh()
}

async function doFlush() {
  confirmFlush.value = false
  await flush()
  toast.push('已触发 Flush')
  setTimeout(refresh, 800)
}
</script>
