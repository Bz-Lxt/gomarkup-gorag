<template>
  <div class="min-h-full flex items-center justify-center px-4">
    <div class="w-full max-w-md border border-line bg-mist/80 p-8">
      <p class="font-mono text-xs tracking-[0.3em] text-cadmium mb-3">MINI MILVUS</p>
      <h1 class="font-display text-4xl mb-2">GoRag</h1>
      <p class="text-paper/70 mb-8">银盐暗房里的多模态检索台</p>
      <form class="space-y-4" @submit.prevent="onSubmit">
        <label class="block">
          <span class="text-sm">用户名 *</span>
          <input v-model="username" class="mt-1 w-full bg-ink border border-line px-3 py-2" />
          <span v-if="errors.username" class="text-cadmium text-xs">{{ errors.username }}</span>
        </label>
        <label class="block">
          <span class="text-sm">密码 *</span>
          <input v-model="password" type="password" class="mt-1 w-full bg-ink border border-line px-3 py-2" />
          <span v-if="errors.password" class="text-cadmium text-xs">{{ errors.password }}</span>
        </label>
        <button type="submit" class="w-full py-3 bg-cadmium text-ink font-display text-lg">进入工作台</button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { login } from '../api'
import { useToast } from '../stores/toast'

const username = ref('admin')
const password = ref('gorag123')
const errors = reactive({ username: '', password: '' })
const router = useRouter()
const toast = useToast()

async function onSubmit() {
  errors.username = username.value.trim() ? '' : '必填'
  errors.password = password.value ? '' : '必填'
  if (errors.username || errors.password) {
    toast.push('请先补全登录信息', 'err')
    return
  }
  try {
    await login(username.value, password.value)
    await router.push('/')
  } catch (e) {
    toast.push((e as Error).message, 'err')
  }
}
</script>
