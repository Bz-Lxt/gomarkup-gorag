<template>
  <div class="min-h-full flex flex-col md:flex-row">
    <aside class="md:w-56 shrink-0 border-b md:border-b-0 md:border-r border-line p-5 space-y-6">
      <div>
        <p class="font-mono text-[10px] tracking-[0.25em] text-cadmium">GORAG</p>
        <h1 class="font-display text-2xl">银盐检索台</h1>
      </div>
      <nav class="flex md:flex-col gap-3 text-sm">
        <RouterLink to="/" class="hover:text-cadmium">混合检索</RouterLink>
        <RouterLink to="/ask" class="hover:text-cadmium">知识问答</RouterLink>
        <RouterLink to="/library" class="hover:text-cadmium">数据管理</RouterLink>
        <RouterLink to="/debug" class="hover:text-cadmium">检索调试</RouterLink>
      </nav>
      <button type="button" class="text-xs text-paper/50" @click="out">退出</button>
    </aside>
    <main class="flex-1 w-full p-4 md:p-8">
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { clearToken } from '../api'

const router = useRouter()
function out() {
  clearToken()
  router.push('/login')
}
</script>
