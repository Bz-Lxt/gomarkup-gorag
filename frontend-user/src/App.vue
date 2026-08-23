<template>
  <div class="min-h-full grain">
    <RouterView />
    <div class="fixed right-4 top-4 z-50 space-y-2 w-80">
      <div
        v-for="t in toast.items"
        :key="t.id"
        class="border px-3 py-2 text-sm flex justify-between gap-3"
        :class="t.kind === 'err' ? 'border-cadmium bg-ink text-cadmium' : 'border-line bg-mist text-paper'"
      >
        <span>{{ t.text }}</span>
        <button type="button" @click="toast.dismiss(t.id)" aria-label="关闭">×</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useToast } from './stores/toast'
const toast = useToast()
</script>
