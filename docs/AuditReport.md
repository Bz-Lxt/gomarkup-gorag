# AuditReport

## Iteration 1 · 2026-08-23 16:05 (GMT+8)

依据 `audit-rules.md` 与 `docs/.meta/original_prompt.md`。此前无审核记录。

### 1. 硬性门槛 — PASS

`docker compose up --build -d` 可一键启动，`localhost:19281` 工作台与 `19282` API 均可访问，未改核心代码。主题为多模态混合检索 + RAG，未跑偏。

### 2. 交付完整性 — PASS

覆盖手写 HNSW/FLAT、倒排+BM25、RRF、Segment 三重触发与 WAL、以图搜图/以文搜图、证据 bbox 与 char_ranges、Vue 瀑布流工作台。Mock 合法性：local hashing / Mock LLM / CLIP HTTP 均已接线；`README.md` §7 写明切换。无密钥的真实 Provider 在 `docs/.meta/api_contracts.md` 标 UNVERIFIED。

### 3. 工程架构 — PASS

backend 按 index / invert / hybrid / segment / wal / feature / provider / engine / api 分包，非单文件堆叠。Modality 预留音视频但不实现，符合范围锁定。

### 4. 工程细节 — PASS

统一 slog、业务错误码、入站与 Segment magic/CRC 校验、过滤 AST 白名单。存在一处经验：异步 Flush 必须 WaitGroup，否则测试目录清理竞态（已修）。

### 5. 需求适配 — PASS

六项矛盾裁决均落实：阈值参数化、双层 patch、HNSW 而非 IVF、以文搜图诚实分通道、维度 1024、音视频不交付。未发现与裁决相反的实现。

### 6. 美观度 — PASS

银盐暗房风格（炭黑 / 纸色 / 镉橙），通道徽标与分数可读，登录与工作台分区清楚。窄视口侧栏改顶栏符合 768 断点。瀑布流高度为估算值，长卡片可能留白偏大，但不构成功能缺失。

### 7. 成本可控性 — PASS（适用）

问答按钮激活前显示「¥0 Mock」或预估费用；`BUDGET_LIMIT_CNY` + Ledger；4xx 不重试。QA 实测 cost_cny=0。

### 8. 异步可靠性 — 不适用

无面向用户的 >30s 后台作业。Segment 异步建索引通常毫秒到数秒，且 Close 等待落盘完成。WAL 保证崩溃恢复。

### 9. 合规标识 — PASS（适用）

Mock 回答带 `[MOCK]` 前缀，问答页文案与按钮同时标明 Mock。

**Decision: PASS**

### Knowledge Harvest

已写入 `knowledge-base/server.md`：
- `[Go][HNSW]` 评测向量须高斯单位+聚簇，禁止用高度相关模板向量。
- `[Go][Segment]` 异步 persist 必须 WaitGroup，否则测试目录清理竞态。
