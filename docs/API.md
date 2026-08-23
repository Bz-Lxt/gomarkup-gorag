# API.md — GoRag

统一包裹：`{ "code": 0, "message": "ok", "data": {}, "trace_id": "..." }`  
时间展示：`yyyy-MM-dd HH:mm:ss`（GMT+8）

## 错误码

| Code | HTTP | 含义 |
|---|---|---|
| 0 | 200/201 | 成功 |
| 40000 | 400 | 请求格式错误 |
| 40001 | 400 | 字段校验失败 |
| 40002 | 400 | 向量维度不匹配 |
| 40003 | 400 | 过滤表达式非法 |
| 40004 | 400 | 上传文件非法 |
| 40005 | 400 | 预算耗尽，已降级 Mock |
| 40006 | 400 | 能力未实现（如音视频） |
| 40100 | 401 | 未登录或凭证错误 |
| 40400 | 404 | 资源不存在 |
| 40900 | 409 | 集合重名 |
| 50000 | 500 | 内部错误 |
| 50200 | 502 | 上游 Provider 失败 |

演示账号：`admin` / `gorag123`

## 端点

### POST /api/v1/auth/login

请求：`{"username":"admin","password":"gorag123"}`  
响应：`{"token":"...","username":"admin","issued_at":"2026-08-23 15:04:05"}`

### GET /healthz · GET /readyz

无需鉴权。就绪时 `data.status=ready`。

### POST /api/v1/collections

`{"name":"default","dim":1024,"metric":"cosine","index_type":"hnsw"}`

### GET /api/v1/collections

返回集合数组。

### POST /api/v1/documents

`{"collection":"default","title":"混合检索","content":"...","tags":["rag"]}`  
响应：`{"doc_id":"doc-...","chunks":2,"entity_ids":[1,2]}`

### POST /api/v1/images  (multipart)

字段：`file` `caption` `tags` `collection`  
响应：`{"entity_id":3,"content_hash":"...","patches":9,"width":160,"height":160,"asset_url":"/api/v1/assets/..."}`

### POST /api/v1/search/text

```json
{
  "collection": "default",
  "query": "向量检索",
  "top_k": 10,
  "metric": "cosine",
  "index_type": "hnsw",
  "rrf_k": 60,
  "vector_weight": 1,
  "keyword_weight": 1,
  "filter": "tag == \"rag\" && score > 0.01",
  "compare_flat": true
}
```

响应 `hits[]` 含 `score` `channels` `evidence.char_ranges` `cross_modal`。

### POST /api/v1/search/image  (multipart)

`file` + `top_k` `metric` `index_type` `compare_flat`  
响应含 `evidence.bbox`（归一化 x,y,w,h + score，真实计算）。

### POST /api/v1/search/hybrid

与 text 相同，可设 `"modality":"image"`。未配 CLIP 时 `cross_modal=false` 且 `degrade_note` 非空。

### POST /api/v1/rag/query  (SSE)

`{"question":"什么是混合检索？","top_k":6}`  
事件：`meta`（citations + mock 标记）`token` `done` `error`

### GET /api/v1/eval/recall?n=2000&queries=40&k=10

HNSW vs FLAT。不走计费 API。

### GET /api/v1/stats · GET /api/v1/cost · GET /api/v1/meta

统计、成本流水、Provider 状态与问答预估费用。

### POST /api/v1/admin/flush · POST /api/v1/admin/compact

强制封口 / 合并小 Segment。

### GET /api/v1/assets/{content_hash}

图片字节。文件名由内容哈希生成，防路径穿越。
