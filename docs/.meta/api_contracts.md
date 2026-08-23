# API Contracts — Contract Gate

记录时间：2026-08-23 (GMT+8)

| Provider | 环境变量 | 密钥 | 状态 | 说明 |
|---|---|---|---|---|
| 文本 Embedding | `EMBEDDING_PROVIDER=local\|openai` | `OPENAI_API_KEY` 未提供 | **UNVERIFIED** | 交付默认 `local` hashing embedding。OpenAI 兼容 `/v1/embeddings` 已接线，未做真实探测。 |
| 图像特征 | `VISION_PROVIDER=local\|clip_api` | `CLIP_API_KEY` 未提供 | **UNVERIFIED**（CLIP） / **本地真实** | 本地 HSV+pHash+边缘为真实算法。CLIP `/embed/text` 已接线未探测。 |
| LLM | `LLM_PROVIDER=mock\|openai` | 未提供 | **UNVERIFIED** | 默认 Mock 抽取式 SSE。OpenAI `/v1/chat/completions` stream 已接线。 |

禁止凭想象补全未验证字段。真实单价、限流头待拿到密钥后再回填。
