# GoRag — Mini Milvus + RAG 闭环

## 1. 如何启动

```bash
docker compose up --build -d
```

浏览器打开 `http://localhost:19281`。后端 API 在 `http://localhost:19282`。时区 `Asia/Shanghai`（GMT+8）。

## 2. 使用说明

登录后进入银盐检索台。混合检索支持文字、拖图、图文组合；结果墙按 Score 排序，悬停图片显示真实 patch 遮罩，文档显示关键词高亮。知识问答走 RAG SSE。数据管理可入库文档/图片并手动 Flush。检索调试并排对比 HNSW 与 FLAT。

交付范围锁定为文本 + 图片。音视频仅预留 `Modality` 枚举，不实现提取器。本地图像特征表达视觉相似，不是语义相似。

## 3. 服务列表及API说明

| 服务 | 地址 |
|---|---|
| 工作台 | http://localhost:19281 |
| 后端 API | http://localhost:19282 |
| 健康检查 | http://localhost:19282/healthz |

完整契约见 `docs/API.md`。

## 4. 测试账号

- 用户名：`admin`
- 密码：`gorag123`

## 5. 题目内容

用 Go 实现多模态混合向量检索与图文知识库（Mini Milvus + RAG）：手写 HNSW + FLAT、倒排 BM25、RRF 融合、Segment 写入管道、以图搜图 / 以文搜图 / 知识库问答工作台。

## 6. 项目结构

```
backend/           Go 引擎与 API
frontend-user/     Vue 3 工作台
frontend-admin/    占位（管理并入工作台）
frontend-mp/       占位
tests/             API smoke + Playwright
docs/              Requirements / Roadmap / API / DesignSpec
```

## 7. API 模拟与切换指南

默认全部离线，QA 成本 ¥0。

| 能力 | 环境变量 | 默认 | 真实路径 |
|---|---|---|---|
| 文本向量 | `EMBEDDING_PROVIDER=local\|openai` | `local` 确定性 hashing（1024 维） | 配置 `OPENAI_API_KEY` + `OPENAI_BASE_URL` 后走 `/v1/embeddings` |
| 图像特征 | `VISION_PROVIDER=local\|clip_api` | `local` HSV+pHash+边缘（真实算法，非 mock） | `clip_api` 需 `CLIP_BASE_URL` + `CLIP_API_KEY`，启用跨模态 |
| 问答 | `LLM_PROVIDER=mock\|openai` | `mock` 抽取式 SSE，回答带 `[MOCK]` | `openai` 走 `/v1/chat/completions` 流式 |

预算：`BUDGET_LIMIT_CNY`（默认 10）。触顶熔断并降级 Mock。4xx 鉴权/校验错误不重试。契约状态见 `docs/.meta/api_contracts.md`（无密钥时为 UNVERIFIED）。
