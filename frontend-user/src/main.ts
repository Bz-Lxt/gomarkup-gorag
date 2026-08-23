import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import './style.css'
import Login from './views/Login.vue'
import Workbench from './views/Workbench.vue'
import SearchHome from './views/SearchHome.vue'
import Ask from './views/Ask.vue'
import Library from './views/Library.vue'
import Debug from './views/Debug.vue'
import { token } from './api'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: Login },
    {
      path: '/',
      component: Workbench,
      children: [
        { path: '', component: SearchHome },
        { path: 'ask', component: Ask },
        { path: 'library', component: Library },
        { path: 'debug', component: Debug },
      ],
    },
  ],
})

router.beforeEach((to) => {
  if (to.path !== '/login' && !token()) return '/login'
  if (to.path === '/login' && token()) return '/'
})

createApp(App).use(createPinia()).use(router).mount('#app')
