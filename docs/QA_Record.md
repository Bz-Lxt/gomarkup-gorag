# QA_Record

## Round 1 · 2026-08-23 16:00 (GMT+8)

**Cost**: ¥0（`EMBEDDING_PROVIDER=local` `LLM_PROVIDER=mock` `VISION_PROVIDER=local`）

**环境**：`docker compose up --build -d`，端口 19281 / 19282。冒烟：`docker compose --profile qa run --rm gorag-qa`。

### 结果

| 检查 | 结论 | 日志摘要 |
|---|---|---|
| Docker Build | PASS | gorag-backend:dev / gorag-frontend:dev 构建成功 |
| Health Check | PASS | backend healthy，frontend 已启动 |
| API Smoke | PASS | `2 passed in 0.04s`（healthz / login / search / stats.cost_cny=0 / flush / rag SSE） |
| 核心单测 | PASS | `go test ./...` 全绿 |
| 浏览器主路径 | PASS | 登录 → 检索「向量检索」出现 Score/VEC/KEY/RRF 卡片 → 问答页按钮显示「¥0 Mock」且正文含 `[MOCK]` → 数据管理页可见入库与 Flush |
| Playwright 容器 | SKIP | QA 镜像仅含 pytest；E2E 脚本已写入 `tests/e2e_flow.spec.ts`，本轮用 Cursor 浏览器代替执行同等路径 |

**无失败项，不进入 Test Fixing 循环。**
