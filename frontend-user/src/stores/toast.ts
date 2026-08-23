import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useToast = defineStore('toast', () => {
  const items = ref<{ id: number; text: string; kind: 'ok' | 'err' }[]>([])
  let n = 1
  function push(text: string, kind: 'ok' | 'err' = 'ok') {
    const id = n++
    items.value.push({ id, text, kind })
    setTimeout(() => dismiss(id), 5000)
  }
  function dismiss(id: number) {
    items.value = items.value.filter((x) => x.id !== id)
  }
  return { items, push, dismiss }
})
