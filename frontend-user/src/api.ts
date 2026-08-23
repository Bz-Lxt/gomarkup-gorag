import { log } from './logger'

export interface Envelope<T> {
  code: number
  message: string
  data: T
  trace_id: string
}

export interface SearchHit {
  id: number
  score: number
  modality: 'text' | 'image'
  channels: { vector: number; keyword: number; rrf: number }
  cross_modal: boolean
  evidence: {
    bbox: { box: [number, number, number, number]; score: number }[]
    char_ranges: { start: number; end: number; kind: string }[]
  }
  content?: string
  caption?: string
  asset_url?: string
  tags?: string[]
  collection: string
  title?: string
}

export interface SearchResp {
  hits: SearchHit[]
  flat_hits?: SearchHit[]
  recall_at_k?: number
  cross_modal: boolean
  degrade_note?: string
  took_ms: number
  channels: string[]
}

const TOKEN_KEY = 'gorag_token'

export function token(): string {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function setToken(t: string) {
  localStorage.setItem(TOKEN_KEY, t)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

async function parse<T>(res: Response): Promise<T> {
  const env = (await res.json()) as Envelope<T>
  if (!res.ok || env.code !== 0) {
    throw new Error(env.message || `http ${res.status}`)
  }
  return env.data
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (token()) headers.set('Authorization', `Bearer ${token()}`)
  if (init.body && !(init.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  const res = await fetch(path, { ...init, headers })
  return parse<T>(res)
}

export async function login(username: string, password: string) {
  const data = await api<{ token: string; username: string }>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
  setToken(data.token)
  log.info('login.ok')
  return data
}

export function searchText(body: Record<string, unknown>) {
  return api<SearchResp>('/api/v1/search/text', { method: 'POST', body: JSON.stringify(body) })
}

export function searchHybrid(body: Record<string, unknown>) {
  return api<SearchResp>('/api/v1/search/hybrid', { method: 'POST', body: JSON.stringify(body) })
}

export async function searchImage(file: File, extra: Record<string, string>) {
  const fd = new FormData()
  fd.append('file', file)
  Object.entries(extra).forEach(([k, v]) => fd.append(k, v))
  const headers = new Headers()
  if (token()) headers.set('Authorization', `Bearer ${token()}`)
  const res = await fetch('/api/v1/search/image', { method: 'POST', body: fd, headers })
  return parse<SearchResp>(res)
}

export async function uploadImage(file: File, caption: string, tags: string) {
  const fd = new FormData()
  fd.append('file', file)
  fd.append('caption', caption)
  fd.append('tags', tags)
  const headers = new Headers()
  if (token()) headers.set('Authorization', `Bearer ${token()}`)
  const res = await fetch('/api/v1/images', { method: 'POST', body: fd, headers })
  return parse<unknown>(res)
}

export function ingestDoc(content: string, title: string, tags: string[]) {
  return api('/api/v1/documents', {
    method: 'POST',
    body: JSON.stringify({ content, title, tags }),
  })
}

export function stats() {
  return api<Record<string, unknown>>('/api/v1/stats')
}

export function meta() {
  return api<{ providers: Record<string, string>; cross_modal: boolean; estimate_rag_cny: number }>('/api/v1/meta')
}

export function evalRecall(n = 800) {
  return api<Record<string, unknown>>(`/api/v1/eval/recall?n=${n}&queries=24&k=10`)
}

export function flush() {
  return api('/api/v1/admin/flush', { method: 'POST', body: '{}' })
}

export function assetSrc(url?: string) {
  if (!url) return ''
  return url
}
