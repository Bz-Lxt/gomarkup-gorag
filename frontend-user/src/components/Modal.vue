<template>
  <div v-if="open" class="fixed inset-0 z-40 flex items-center justify-center bg-black/60 px-4">
    <div class="w-full max-w-md border border-line bg-ink p-6">
      <h3 class="font-display text-xl mb-3">{{ title }}</h3>
      <p class="text-paper/80 mb-6">{{ text }}</p>
      <div class="flex justify-end gap-3">
        <button type="button" class="px-4 py-2 border border-line" @click="$emit('cancel')">取消</button>
        <button type="button" class="px-4 py-2 bg-cadmium text-ink" @click="$emit('ok')">确认</button>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
defineProps<{ open: boolean; title: string; text: string }>()
defineEmits(['ok', 'cancel'])
</script>
