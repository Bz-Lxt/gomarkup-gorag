# DesignSpec — GoRag 工作台

> 美学方向：**银盐暗房 / 观测台**。不是科技紫渐变，不是默认仪表盘。

## 调色

| Token | 值 | 用途 |
|---|---|---|
| `--ink` | `#12110f` | 近黑底 |
| `--paper` | `#efe6d6` | 正文与卡片纸色 |
| `--cadmium` | `#e4572e` | 唯一强调：检索、分数、CTA |
| `--cyan` | `#7ec8c3` | 向量通道 |
| `--gold` | `#d4a017` | 关键词通道 |
| `--mist` | `#2a2824` | 次级面板 |
| `--line` | `#3a3732` | 细线 |

## 字体

- Display：`Fraunces`（衬线，标题与 Logo）
- Body：`Source Serif 4`
- Mono：`IBM Plex Mono`（Score、ID、调试）

## 组件

- **SearchDock**：居中宽输入，纸色描边，拖入图片时镉橙虚线框。
- **MasonryWall**：绝对定位瀑布流，ResizeObserver 重排，图片懒加载。
- **EvidenceOverlay**：半透明镉橙遮罩，opacity 跟 patch score 走。
- **ChannelPills**：vector=cyan，keyword=gold，rrf=paper。
- **CostChip**：触发计费前显示预估 ¥。
- **Modal / Toast**：禁止原生 alert；Toast 可关闭 + 5s 消失。

## 响应式

- 768px：侧栏收为顶栏。
- 480px：单列瀑布流，表单单列。

## 日期

用户可见时间一律 `yyyy-MM-dd HH:mm:ss`（GMT+8）。
