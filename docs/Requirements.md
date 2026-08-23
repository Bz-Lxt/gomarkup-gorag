# Requirements.md — GoRag 多模态混合向量检索与图文知识库系统

> **权威性**：本文件定义 **WHAT**（做什么、边界在哪、如何验收）。**WHEN**（阶段顺序）由 `docs/Roadmap.md` 定义。
> **SSOT**：原始需求见 `docs/.meta/original_prompt.md`，不得被本文件覆盖或改写。
> **版本**：v1.0 · 2026-08-23 (GMT+8) · PM Agent (Alkaid-SOP v13.0)
> **代号**：GoRag（内部定位：Mini Milvus + RAG 闭环）

---

## 1. 立项评估结论

### 1.1 废弃评估（Discard Criteria）

| # | 判据 | 结论 | 依据 |
|---|---|---|---|
| 1 | 不完整 / 模糊 | ✅ 通过 | 业务背景、前端形态、后端算法、代码量均已明确，无缺失附件依赖 |
| 2 | Windows 独占 | ✅ 通过 | Go + Vue 全平台；开发机为 darwin/arm64，Go 1.25.12 已就绪 |
| 3 | 规模评估（分层） | ✅ **ACCEPT（中型档）** | 后端 7000–9500 行 + 前端 ≈2000–3000 行 ≈ **10k–12.5k LoC**，落在 10k–40k 区间。**强制要求**：分阶段 Roadmap（MVP / V1 / V2）必须先行，见 §9 |
| 4 | 外部依赖（智能检查） | ✅ **ACCEPT（Scenario A）** | 涉及 Embedding / CLIP / LLM 三类 AI 生成型 API，均属"可模拟"。强制 Mock Provider + `README.md` §7 真假开关。详见 §6 |
| 5 | 专业 / 付费软件 | ✅ 通过 | 全链路开源，无商业授权依赖 |

**立项结论：ACCEPT（中型项目，分阶段交付）**

### 1.2 Docker 交付标准兼容性

- 非微信小程序，**Docker 检查生效**。
- 后端 Go 编译为静态二进制（`CGO_ENABLED=0`），前端构建为静态资源经 Nginx 托管，`docker compose up --build -d` 一键启动，`localhost` 可访问 Web 工作台。✅ 满足
- 跨平台：`golang:1.25-alpine`、`node:22-alpine`、`nginx:1.27-alpine` 均提供 `linux/arm64` 与 `linux/amd64` 官方镜像。✅ 满足
- **CGO 禁令（衍生约束）**：为保证 alpine 静态构建与双架构，**禁止引入任何需要 CGO 的依赖**。这直接排除 `gojieba`（中文分词）与 `mattn/go-sqlite3`。替代方案见 §7.2。

---

## 2. 需求矛盾与技术裁决（Contradiction Detection）

> 以下 6 项为原始 Prompt 中存在的内在冲突或技术上不可直接落地之处。**每项均已裁决，后续 Agent 不得推翻**（Anti-Flip-Flop）。

### C-1｜「64MB 触发 Segment 落盘」与可演示性冲突 —— 阈值参数化

**冲突**：1024 维 `float32` 向量单条约 4KB，64MB ≈ 16384 条。演示场景上传数十张图片/文档永远无法触达阈值，导致 Segment 管道、异步索引构建、持久化恢复三条核心链路**在验收中不可观测**。

**裁决**：Segment 封口采用**三重触发**，全部可配置：

| 参数 | 开发/演示默认 | 生产建议 | 说明 |
|---|---|---|---|
| `segment.max_bytes` | `4MB` | `64MB` | 原始需求值作为生产默认，写入配置注释 |
| `segment.max_rows` | `1000` | `16384` | 行数兜底 |
| `segment.max_idle` | `30s` | `300s` | 空闲超时封口，防止小流量下 Buffer 永不落盘 |

额外提供 `POST /api/v1/admin/flush` 手动强制封口，供 QA 在测试中确定性验证落盘与恢复。

### C-2｜「悬停高亮匹配的特征区域」与全局向量检索冲突 —— 双层 Patch 索引

**冲突**：一张图片压成一个全局 embedding 后，**空间信息已丢失**，数学上无法反推"哪个区域匹配"。若前端随便画个框，即构成 Redline 4 意义上的**伪造**。

**裁决**：实现 **Parent-Child 双层索引**，让高亮区域成为真实计算结果。

- **图片**：入库时切分 `N×N` 网格（默认 3×3，可配），全局向量（parent）+ 9 个 patch 向量（child）同时入库。检索时全局向量做粗排，对命中的图片再在其 patch 集合内计算与 query 向量的相似度，返回 **Top-K patch 的归一化 bbox `[x, y, w, h]` 及各自 score**。前端悬停时按 score 渲染半透明高亮遮罩。
- **文本**：切块时记录每个 chunk 内 term 的**字符偏移区间**与句级子向量。检索返回命中关键词的 `char_ranges` + 最相似子句区间，前端用 `<mark>` 精确高亮。
- 该设计为 late-interaction（类 ColBERT）思想的简化实现，`bbox` 与 `char_range` 全部来自后端真实计算，响应体中携带 `evidence` 字段承载。

### C-3｜业务背景提「音视频」，功能需求仅「图 + 文」—— 范围锁定

**冲突**：背景段落提及音视频多模态，但功能清单（以图搜图 / 以文搜图 / 知识库问答）不含音视频。若擅自实现即 Scope Drift。

**裁决**：**交付范围锁定为 文本 + 图片双模态**。代码层预留 `Modality` 枚举与 `FeatureExtractor` 接口以证明架构可扩展性，但**不实现**音视频提取器。`README.md` 明示该边界。

### C-4｜「IVF-FLAT 或 HNSW」二选一 —— 选定 HNSW + FLAT 基线

**冲突**：原文为"如基于 IVF-FLAT 或 HNSW 核心概念"，二者架构差异大，必须定型。

**裁决**：手写 **HNSW** 作为主索引，同时保留 **FLAT 暴力检索**。

- **理由**：IVF-FLAT 需要 k-means 训练阶段才能建立倒排桶，与 C-1 的"流式 Buffer 实时写入 + 增量封口"管道天然冲突；HNSW 支持增量插入，无训练期，契合本系统写入模型。
- **FLAT 的双重用途**：① 小 Segment（< 阈值）直接暴力检索更快；② 作为 **Ground Truth 基线**，使 §5.1 的召回率成为可自动化度量的验收指标——这是本项目最关键的可验证点。
- IVF-FLAT **不实现**，避免过度工程。

### C-5｜「以文搜图」的跨模态对齐能力边界 —— 诚实分通道

**冲突**：真正的"以文搜图"需要 CLIP 类图文对齐模型将文本与图像投影到同一语义空间。本地纯 Go 手写的颜色/纹理特征与文本 hashing embedding **处于不同向量空间，余弦相似度无任何语义含义**。假装能算即为技术欺骗。

**裁决**：以文搜图**明确分为两条通道**，并在 UI 与文档中标注当前生效通道：

| 通道 | 触发条件 | 原理 | 可用性 |
|---|---|---|---|
| **标量通道**（默认，无需外部服务） | 始终启用 | 图片的 caption / tag / OCR 文本入倒排索引，走 BM25 关键词检索 | ✅ 真实可用，非 mock |
| **跨模态向量通道** | `VISION_PROVIDER=clip_api` 且配置密钥 | 调用 CLIP 兼容服务将文本与图像映射到同一空间 | ⚠️ 需外部服务，Contract Gate 验证 |

两通道结果经 RRF 融合。未配置 CLIP 时，API 响应显式返回 `"cross_modal": false` 与降级说明，**前端必须可见地告知用户当前为标量匹配**。

### C-6｜「千维向量」的确定值 —— 定为 1024 维

**裁决**：统一 `dim = 1024`（float32，L2 归一化后存储）。理由：与主流 BGE-large / text-embedding-3 对齐，切换真实 Provider 时无需重建索引；单向量 4KB，便于 §C-1 阈值换算。维度写入 Collection 元数据，插入时强校验（呼应 Global 记忆：外部数据必须校验结构完整性）。

---

## 3. 范围边界

### 3.1 交付范围（IN）

**前端工作台**
1. 混合搜索主页：单一输入框支持①纯文字查询 ②拖拽/粘贴本地图片 ③图文组合查询（图 + 文字修饰词）。
2. 多模态结果墙：瀑布流布局，按 Score 降序，卡片区分图片/文档段落两种形态，显示 Score、来源、命中通道徽标（向量/关键词/RRF）。
3. 悬停证据高亮：图片显示 patch 遮罩，文档段落显示关键词与最相似子句高亮（真实数据，见 C-2）。
4. 知识库问答页：输入问题 → 流式返回答案 + 引用卡片（可点击回溯原文/原图），标注 Mock/Real 模型来源。
5. 数据管理页：上传文档与图片、查看索引统计（Segment 数量、向量总数、内存占用、Flush 历史）、手动 Flush。
6. 检索调试面板：可调 `topK`、`metric(cosine|l2)`、`index(hnsw|flat)`、`rrf_k`、通道权重，并**并排展示 HNSW 与 FLAT 结果差异及召回率**——这是系统专业性的核心展示。

**后端引擎**
1. 向量引擎：手写 HNSW（多层图、可配 `M` / `efConstruction` / `efSearch`）+ FLAT 暴力检索；Cosine 与 L2 双度量；分片并发计算（worker pool，按 CPU 核数）。
2. 标量引擎：自研倒排索引 + BM25 打分；混合分词器（ASCII 词边界 + CJK bigram + 停用词表），纯 Go 无 CGO。
3. 混合检索路由：查询规划器决定走标量/向量/双通道，RRF（`k=60`）融合，支持通道权重。
4. Segment 写入管道：内存 Buffer → 三重触发封口 → Goroutine 异步建索引 → 二进制落盘；WAL 保证崩溃不丢数据；启动时恢复 + 只读 Segment 与可写 Buffer 统一检索视图。
5. 特征提取：图片纯 Go 真实实现（HSV 颜色直方图 + pHash/DCT + 边缘方向直方图，拼接归一化至 1024 维）；文本本地确定性 embedding（hashing trick + n-gram）或外部 Provider。
6. RAG 闭环：检索 → 重排 → 上下文装配 → 生成（Mock/Real 双 Provider）→ 引用溯源，SSE 流式输出。
7. 统一 Logger（分级、生产屏蔽 debug）、成本记录、预算上限、窄化重试。

### 3.2 排除范围（OUT）

- ❌ 音视频模态（C-3）
- ❌ IVF-FLAT / PQ / 量化压缩（C-4）
- ❌ 分布式集群、多副本、Raft 元数据（单机内存 + 本地磁盘）
- ❌ 用户注册体系（仅内置演示账号）
- ❌ 真实 CLIP 模型权重本地推理（仅接口对接）
- ❌ GPU 加速

---

## 4. 功能需求清单（FR）

| ID | 模块 | 需求 | 优先级 |
|---|---|---|---|
| FR-01 | Collection | 创建/删除/列举集合，声明 `dim`、`metric`、`index_type`；插入时强校验维度与字段完整性 | P0 |
| FR-02 | 写入 | 文本文档入库：切块 → 分词入倒排 → embedding 入向量索引，返回 chunk 数 | P0 |
| FR-03 | 写入 | 图片入库（multipart）：解码 → 全局特征 + N×N patch 特征 → 入库；校验格式/尺寸/大小上限 | P0 |
| FR-04 | 检索 | 以文搜文：BM25 + 向量双通道 RRF 融合，返回 evidence.char_ranges | P0 |
| FR-05 | 检索 | 以图搜图：全局向量粗排 + patch 精定位，返回 evidence.bbox 列表 | P0 |
| FR-06 | 检索 | 以文搜图：标量通道（默认）/ 跨模态向量通道（可选），响应标注 `cross_modal` | P0 |
| FR-07 | 检索 | 混合查询：标量过滤表达式（如 `tag == "cat" && score > 0.5`）+ 向量 ANN 联合执行 | P1 |
| FR-08 | 索引 | HNSW 增量插入、删除标记（tombstone）、`efSearch` 运行时可调 | P0 |
| FR-09 | 索引 | FLAT 并发暴力检索，作为 Ground Truth 与召回率评估基线 | P0 |
| FR-10 | 存储 | Segment 三重触发封口 + 异步建索引 + 二进制落盘 + 元数据清单 | P0 |
| FR-11 | 存储 | WAL 写前日志；进程重启后数据零丢失恢复；提供恢复耗时指标 | P0 |
| FR-12 | 存储 | Compaction：合并小 Segment、清理 tombstone | P2 |
| FR-13 | RAG | 检索增强问答：SSE 流式、引用溯源、Mock/Real 双 Provider | P0 |
| FR-14 | 运维 | `/healthz`、`/readyz`、`/api/v1/stats`（Segment/向量数/内存/QPS/延迟分位）、`/api/v1/admin/flush` | P0 |
| FR-15 | 成本 | 每次外部 API 调用记录 token 与费用；预算上限熔断；窄化重试（仅重试瞬时错误，**绝不重试鉴权与参数校验错误**） | P0 |
| FR-16 | 前端 | 瀑布流结果墙 + 悬停证据高亮 + 通道徽标 | P0 |
| FR-17 | 前端 | 检索调试面板：HNSW vs FLAT 并排对比 + 实时召回率显示 | P1 |
| FR-18 | 前端 | 成本可见性：任何触发计费 API 的控件，激活前显示预估费用 | P0 |

---

## 5. 非功能需求与可度量验收基线

### 5.1 性能与质量基线（硬性，QA 自动化断言）

| 指标 | 基线 | 测量方法 |
|---|---|---|
| HNSW Top-10 召回率 | **≥ 0.95** | 10,000 条 1024 维随机+聚簇混合向量，以 FLAT 结果为 Ground Truth |
| 单查询延迟 P99（10k 向量 / HNSW） | **< 50 ms** | 500 次查询打点统计 |
| FLAT 并发加速比 | **≥ 3.0×** | 多 worker vs 单 goroutine，8 核环境 |
| 批量写入吞吐 | **≥ 2,000 vec/s** | 10k 向量顺序插入计时 |
| Segment 恢复正确性 | **零丢失** | 写入 → flush → kill → 重启 → 检索结果完全一致 |
| RRF 常数 | `k = 60` | 业界标准值，可配置 |
| 后端核心引擎单测覆盖率 | **≥ 70%** | `go test -cover`（引擎、索引、存储、分词、RRF 包） |
| 前端首屏 LCP | **< 2 s** | 本地 Docker 环境 |
| 瀑布流滚动流畅度 | **≥ 50 fps**（500 卡片） | 虚拟化/懒加载 |
| QA 单轮成本 | **¥0** | 全程 Mock 模式，禁止测试触达计费 API |

### 5.2 工程规范（继承 Global 记忆，硬性）

1. **统一 Logger**：后端 `pkg/logger` 分级（debug/info/warn/error），前端统一 log 工具；**禁止散落 `fmt.Println` / `console.log`**；生产环境自动屏蔽 debug。
2. **API 文档**：独立 `docs/API.md`，每个端点提供请求/响应示例、参数类型、**错误码表**。
3. **测试代码**：后端覆盖 CRUD + 核心引擎单元测试；E2E 用 Playwright 覆盖关键路径（上传→搜索→高亮→问答）。
4. **外部数据校验**：所有入站 JSON / multipart / Segment 反序列化必须校验字段存在性、类型、边界值，不得仅依赖调用方；Segment 文件带 magic number + version + CRC32 校验。
5. **时区**：全链路 GMT+8（`TZ=Asia/Shanghai` 写入 compose 与镜像），日志与时间戳统一北京时间。
6. **错误处理**：所有导出函数错误链路完整（`fmt.Errorf("...: %w", err)`），禁止吞异常。

### 5.3 安全与合规

- 上传文件校验 MIME + 魔术字节 + 尺寸上限（默认 10MB）+ 像素上限，防解压炸弹。
- 路径穿越防护：资产文件名一律以内容哈希重命名。
- 密钥仅经环境变量注入，禁止入库/入日志/入前端；日志中密钥自动脱敏。
- 标量过滤表达式（FR-07）使用自研安全解析器（白名单 AST），**禁止 `eval` 类动态执行**。
- Mock 生成内容在 UI 与 API 响应中带 `[MOCK]` 标识（合规标识）。

---

## 6. 外部 API 与 Mock 合法性策略（Redline 4）

三类外部依赖全部属 **Scenario A（可模拟）**，均须满足 Mock 合法性双条件：**真实路径已接线** + **切换开关文档化于 `README.md` §7**。

| Provider | 环境变量 | Mock 实现 | Real 实现 | 默认 |
|---|---|---|---|---|
| 文本 Embedding | `EMBEDDING_PROVIDER=local\|openai` | 本地确定性 hashing embedding（1024维，同输入必同输出，可复现） | OpenAI 兼容 `/v1/embeddings`（BGE / text-embedding-3） | `local` |
| 图像特征 | `VISION_PROVIDER=local\|clip_api` | **无需 Mock**：本地 HSV 直方图 + pHash + 边缘方向直方图为真实算法实现 | CLIP 兼容图文对齐服务（启用跨模态通道） | `local` |
| LLM 问答 | `LLM_PROVIDER=mock\|openai` | 基于检索片段的抽取式模板回答，带 `[MOCK]` 前缀，SSE 逐字吐出 | OpenAI 兼容 `/v1/chat/completions` 流式 | `mock` |

**Contract Gate 要求（Phase 3 强制）**：对每个启用的真实 Provider，先发起 **1 次最小真实调用**，记录鉴权方式、请求结构、响应 schema、错误格式、限流响应头、实际单价至 `docs/.meta/api_contracts.md`。无密钥时先实现 Mock 并将该 Provider 标记 `UNVERIFIED`，**禁止凭想象编写响应解析代码**。

**成本护栏**：全局预算上限（`BUDGET_LIMIT_CNY`，默认 `10.00`）触顶即熔断降级至 Mock；逐次调用记录不可变成本流水；重试仅针对 5xx / 超时 / 429（指数退避），4xx 鉴权与参数错误立即失败。

---

## 7. 技术栈决策

### 7.1 选型

| 层 | 选型 | 理由 |
|---|---|---|
| 后端语言 | **Go 1.25**（`CGO_ENABLED=0`） | 需求指定；静态二进制契合 alpine 双架构 |
| HTTP | 标准库 `net/http` + `ServeMux`（Go 1.22+ 路由） | 零重依赖，减少供应链风险 |
| 向量索引 | **自研 HNSW + FLAT** | 需求核心，禁止引入 `faiss` / `hnswlib` 绑定 |
| 倒排 / BM25 | **自研** | 需求核心（"混合倒排索引路由"） |
| 中文分词 | **自研混合分词器** | `gojieba` 需 CGO，违反 §1.2 约束 |
| 存储 | **自研 Segment 二进制格式 + WAL + manifest** | 需求核心；不引入外部 DB，Docker 依赖最小 |
| 图像处理 | Go 标准库 `image/*` + `golang.org/x/image` | 纯 Go 解码 jpeg/png/webp |
| 前端 | **Vue 3 + TypeScript + Vite + TailwindCSS + Pinia** | 需求指定 |
| 瀑布流 | **自研组件**（absolute 定位 + ResizeObserver + 虚拟化） | 需精确控制悬停遮罩坐标映射，第三方库难以透传 bbox |
| 网关 | Nginx（alpine） | 静态资源 + `/api` 反代，同源避免 CORS |
| E2E | Playwright | Global 记忆要求 |

### 7.2 服务拓扑

| 服务 | 内容 | Dev 端口 | 交付端口 |
|---|---|---|---|
| `gorag-backend` | Go API + 向量引擎 + 存储 | 随机高位端口 | `8082` |
| `gorag-frontend` | Nginx + Vue 静态资源 + API 反代 | 随机高位端口 | `8081`（用户入口） |

数据持久化：命名 volume 挂载 `/data`（`wal/`、`segments/`、`assets/`、`manifest.json`）。

---

## 8. 数据模型与 API 契约概览

### 8.1 核心实体

```
Collection { name, dim=1024, metric(cosine|l2), index_type(hnsw|flat), created_at }
Entity     { id, collection, modality(text|image), vector[1024], scalar_fields{...},
             source_ref, created_at, deleted(tombstone) }
TextChunk  { entity_id, doc_id, chunk_index, content, char_offset, terms[], sentence_vectors[] }
ImageAsset { entity_id, content_hash, width, height, caption, tags[],
             patches[{grid_pos, bbox, vector[1024]}] }
Segment    { id, state(growing|sealed|persisted), row_count, byte_size,
             index_type, file_path, crc32, min_ts, max_ts }
```

### 8.2 端点清单（详细契约见 Phase 3 产出的 `docs/API.md`）

```
POST   /api/v1/collections                创建集合
GET    /api/v1/collections                列举集合
POST   /api/v1/documents                  文本入库（切块+双索引）
POST   /api/v1/images                     图片入库（multipart，全局+patch 特征）
POST   /api/v1/search/text                以文检索（BM25 + 向量 + RRF）
POST   /api/v1/search/image               以图搜图（multipart，返回 bbox 证据）
POST   /api/v1/search/hybrid              混合查询（标量过滤 + 向量 ANN + 通道权重）
POST   /api/v1/rag/query                  RAG 问答（SSE 流式 + 引用溯源）
GET    /api/v1/eval/recall                HNSW vs FLAT 召回率评估
GET    /api/v1/stats                      索引/内存/延迟/成本统计
POST   /api/v1/admin/flush                强制封口落盘
GET    /api/v1/assets/{content_hash}      图片资产读取
GET    /healthz | /readyz                 存活/就绪探针
```

统一响应包裹：`{ "code": 0, "message": "ok", "data": {...}, "trace_id": "..." }`；错误码表纳入 `docs/API.md`。

检索结果项统一携带证据字段：

```json
{
  "id": "...", "score": 0.8731, "modality": "image",
  "channels": { "vector": 1, "keyword": 4, "rrf": 0.0312 },
  "cross_modal": false,
  "evidence": {
    "bbox": [ { "box": [0.33, 0.0, 0.33, 0.33], "score": 0.91 } ],
    "char_ranges": []
  }
}
```

---

## 9. 规模分层建议（移交 Chief Architect）

> 依据 §1.1 判据 3，10k–40k LoC 项目**必须**先有含 MVP / V1 / V2 边界的 `docs/Roadmap.md` 才可写代码。以下为 PM 建议切分，最终由 Phase 1 定稿。

| 阶段 | 范围 | 预估 LoC | 可演示成果 |
|---|---|---|---|
| **MVP** | Collection 管理、FLAT 并发检索、本地文本 embedding、文本入库/检索、统一 Logger、基础 API | ~2,800 | curl 完成文本入库与向量检索 |
| **V1** | HNSW 手写索引、倒排 + BM25 + 自研分词、RRF 融合、图片特征 + Patch 双层索引、Vue 工作台（瀑布流 + 悬停高亮） | ~5,200 | Web 端完成以图搜图/以文搜图 + 证据高亮 |
| **V2** | Segment 管道 + WAL + 恢复、RAG 问答闭环（SSE + 引用）、成本护栏、召回率评估端点、Compaction、Docker 交付、E2E | ~4,000 | `docker compose up` 全功能交付 |

**Phase Order 建议**：本系统的前端结构（结果墙卡片形态、悬停遮罩坐标、通道徽标）**完全派生于后端 `evidence` 数据结构**，属 Alkaid-SOP v13 §PHASE 1 中"UI 由数据模型派生"的情形。建议 Chief Architect 采纳 **Logic-First（Phase 2 与 Phase 3 交换）**，理由记入 Roadmap。

---

## 10. 验收清单（Definition of Done）

- [ ] `docker compose up --build -d` 一键启动，`localhost:8081` 可访问工作台，双架构镜像可拉取
- [ ] 以图搜图返回真实计算的 patch bbox，悬停高亮位置与 score 对应
- [ ] 以文搜图明示当前通道（标量 / 跨模态），未配 CLIP 时前端可见降级提示
- [ ] `/api/v1/eval/recall` 实测 HNSW Top-10 召回率 ≥ 0.95，P99 < 50ms
- [ ] 写入 → flush → 强杀进程 → 重启 → 检索结果完全一致（零丢失）
- [ ] `go test ./...` 全绿，核心引擎覆盖率 ≥ 70%
- [ ] Playwright E2E 全绿，Mock 模式运行，成本 ¥0
- [ ] `docs/API.md` 含全端点示例与错误码表；`README.md` 七大章节齐备，§7 明确 Mock/Real 切换
- [ ] 全链路无散落 `fmt.Println` / `console.log`；生产 debug 日志已屏蔽
- [ ] 时间显示为 GMT+8
- [ ] UI 达"Dribbble 标准"：现代、精致、响应式，无 Engineer UI

---

## 11. 风险登记

| 风险 | 等级 | 缓解 |
|---|---|---|
| 手写 HNSW 召回率不达 0.95 | 高 | 调参 `M=16 / efConstruction=200 / efSearch=64`；FLAT 基线可随时兜底并在调试面板暴露差异 |
| 本地图像特征在语义相似度上表现弱（同类不同色的图不相似） | 中 | 诚实定位为"视觉相似"而非"语义相似"，UI 文案与 README 明示；CLIP 通道为语义升级路径 |
| 自研 CJK 分词召回不足 | 中 | bigram 全覆盖保底 + 停用词表 + 可配自定义词典 |
| Segment 二进制格式演进破坏兼容 | 中 | magic + version + CRC32，version 不匹配拒绝加载并给出明确错误 |
| 前端 500+ 卡片瀑布流卡顿 | 中 | 虚拟化窗口 + 图片懒加载 + 缩略图预生成 |
| 无外部 API 密钥导致 Contract Gate 无法验证 | 中 | 允许标记 `UNVERIFIED`，Mock 优先交付，禁止臆造 schema |

---

**Requirements frozen. 输入 `/auto` 启动 Auto-Swarm（Phase 1 → 5 连续执行）。**
