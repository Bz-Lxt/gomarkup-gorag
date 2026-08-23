import { test, expect } from '@playwright/test'

const base = process.env.E2E_BASE || 'http://127.0.0.1:19281'

test.describe('GoRag workbench', () => {
  test('login search ask library', async ({ page }) => {
    await page.goto(base + '/login')
    await page.getByRole('heading', { name: 'GoRag' }).waitFor()
    await page.locator('input').nth(0).fill('admin')
    await page.locator('input[type="password"]').fill('gorag123')
    await page.getByRole('button', { name: '进入工作台' }).click()
    await expect(page.getByRole('heading', { name: /以文搜图/ })).toBeVisible()

    await page.getByPlaceholder(/输入查询/).fill('向量检索')
    await page.getByRole('button', { name: '检索' }).click()
    await expect(page.locator('article').first()).toBeVisible({ timeout: 15000 })

    await page.getByRole('link', { name: '知识问答' }).click()
    await page.getByRole('button', { name: /提问/ }).click()
    await expect(page.getByText('[MOCK]')).toBeVisible({ timeout: 15000 })

    await page.getByRole('link', { name: '数据管理' }).click()
    await expect(page.getByRole('heading', { name: '数据管理' })).toBeVisible()
    await expect(page.getByText('索引统计')).toBeVisible()
  })
})
