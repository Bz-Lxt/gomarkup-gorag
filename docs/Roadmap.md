# Roadmap.md — GoRag 多模态混合向量检索与图文知识库

> **权威性**：本文件定义 **WHEN**（阶段顺序与完成定义）。**WHAT** 见 `docs/Requirements.md`。
> **版本**：v1.0 · 2026-08-23 (GMT+8) · Chief Architect
> **预估规模**：后端 7000–9500 行 / 40+ Go 文件；前端 2000–3000 行；合计 10k–12.5k LoC（中型档）。

---

## 0. Phase Order Decision

**决策：Logic-First（交换 SOP 默认的 Phase 2 UI 与 Phase 3 Logic）**

**理由**：工作台的结果卡片形态、悬停遮罩坐标映射、通道徽标与 `cross_modal` 降级提示，全部派生于后端统一的 `evidence` / `channels` 响应结构；schema 不定型则 UI 无法落地。先完成引擎契约与真实 bbox / char_range 计算，再按契约画界面。

执行顺序：Phase 1 架构 → Phase 3 后端引擎 → Phase 2 前端工作台 → Phase 4 QA → Phase 5 审计。

---

## 1. 目录结构（冻结）

```
GoRag/
├── backend/                      # Go 1.25 · CGO_ENABLED=0
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── api/                  # HTTP 路由 / 中间件 / 处理器
│   │   ├── config/               # 环境变量配置
│   │   ├── cost/                 # 不可变成本流水 + 预算熔断
│   │   ├── engine/               # 写入 / 检索 / 评估编排
│   │   ├── feature/              # 文本 hashing + 图像 HSV/pHash/边缘
│   │   ├── filter/               # 标量过滤表达式 AST
│   │   ├── hybrid/               # 查询规划 + RRF
│   │   ├── index/flat|hnsw/      # 向量索引
│   │   ├── invert/               # 倒排 + BM25
│   │   ├── metric/               # Cosine / L2
│   │   ├── model/                # 领域类型与错误码
│   │   ├── provider/             # Embedding / CLIP / LLM（Mock+Real）
│   │   ├── rag/                  # 检索增强问答
│   │   ├── segment/              # Buffer / 封口 / 编解码 / Compaction
│   │   ├── store/                # Manifest + 资产落盘
│   │   ├── tokenize/             # ASCII + CJK bigram
│   │   └── wal/                  # 写前日志
│   ├── pkg/logger|timeutil
│   └── testdata/
├── frontend-user/                # Vue 3 + TS + Vite + Tailwind + Pinia
├── frontend-admin/               # 占位（本项目无独立后台，管理并入工作台）
├── frontend-mp/                  # 占位（非小程序）
├── tests/                        # Playwright E2E + API smoke
├── docker-compose.yml            # Dev 随机端口 19281 / 19282
└── docs/
```

---

## 2. 阶段边界

### MVP — 可 curl 的文本向量闭环

| Task | 内容 | 完成定义 |
|---|---|---|
| M-01 | 统一 Logger / 北京时间 / 配置加载 | 生产屏蔽 debug；时间戳 GMT+8 |
| M-02 | Collection CRUD + 维度强校验 | 错误码 40001 维度不匹配 |
| M-03 | FLAT 并发暴力检索（Cosine/L2） | 单元测试覆盖正确性 + worker 加速 |
| M-04 | 本地确定性文本 embedding（1024 维） | 同输入同输出 |
| M-05 | 文本入库切块 + 向量检索 API | `POST /documents` + `POST /search/text` |
| M-06 | `/healthz` `/readyz` `/stats` | Docker healthcheck 可用 |

**可演示**：`curl` 入库一段中文文档并按语义/关键词检索。

### V1 — 混合检索 + 图文证据 + 工作台

| Task | 内容 | 完成定义 |
|---|---|---|
| V1-01 | 手写 HNSW（M / efConstruction / efSearch） | Top-10 召回 ≥ 0.95 vs FLAT |
| V1-02 | 自研分词 + 倒排 + BM25 | 中英混合查询有命中 |
| V1-03 | RRF 融合（k=60）+ 通道权重 | 双通道分数可复现 |
| V1-04 | 标量过滤 AST（白名单） | `tag == "cat" && score > 0.5` |
| V1-05 | 图像 HSV + pHash + 边缘特征 → 1024 维 | 纯 Go，无 CGO |
| V1-06 | Parent-Child patch 双层索引 | 响应含真实 bbox + score |
| V1-07 | 以图搜图 / 以文搜图 / hybrid | `cross_modal` 字段诚实 |
| V1-08 | Vue 工作台：瀑布流 + 悬停高亮 + 调试面板 | 对照 `evidence` 渲染 |

**可演示**：Web 端拖图搜图，悬停看到真实 patch 遮罩。

### V2 — 持久化 / RAG / 交付

| Task | 内容 | 完成定义 |
|---|---|---|
| V2-01 | Segment 三重触发 + 异步建索引 + 二进制落盘 | magic + version + CRC32 |
| V2-02 | WAL + 启动恢复零丢失 | 写入→flush→重启结果一致 |
| V2-03 | Compaction（合并小段 / 清 tombstone） | 统计可见段数下降 |
| V2-04 | RAG SSE + Mock/Real LLM + 引用溯源 | `[MOCK]` 标识 |
| V2-05 | 成本流水 + 预算熔断 + 窄化重试 | 4xx 不重试 |
| V2-06 | `/eval/recall` + 手动 flush | 召回率可观测 |
| V2-07 | Docker 双架构 + E2E Mock ¥0 | `compose up` 即用 |

**可演示**：`docker compose up --build -d` 全功能。

---

## 3. 技术冻结

- Go 1.25 · `CGO_ENABLED=0` · 标准库 `net/http` ServeMux
- 主索引 **HNSW**，基线 **FLAT**；**不实现 IVF-FLAT**
- 维度 **1024** · 度量 Cosine / L2
- Segment 演示阈值 4MB / 1000 行 / 30s；生产注释 64MB
- 模态锁定 **text + image**；`Modality` 枚举预留 audio/video 但不实现提取器
- Provider 默认：`EMBEDDING_PROVIDER=local` · `VISION_PROVIDER=local` · `LLM_PROVIDER=mock`
- Dev 端口：前端 **19281** · 后端 **19282**（交付阶段再改为 8081/8082）

---

## 4. 进度勾选

- [x] Phase 1 架构（本文件 + 骨架 + compose）
- [x] MVP（M-01 … M-06）
- [x] V1（V1-01 … V1-08）
- [x] V2（V2-01 … V2-07）
- [x] Phase 4 QA
- [x] Phase 5 审计
