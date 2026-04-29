# ADR 007: 官方版本公告知识层

## Status

Accepted

## Implementation Status（实现状态）

Status: Done for MVP（最小可用版本已完成）

Checklist:

- [x] `scripts/update_cn_knowledge.py` 已支持 `--patch-note-url`。
- [x] 已生成 `tft/knowledge/data/patch_notes/17.1.json`。
- [x] knowledge loader（知识库加载器）已加载 patch notes（版本公告）。
- [x] contract（公共协议）已新增 `PatchNoteInsight`。
- [x] `QueryNLU` 已能返回相关版本公告片段。
- [x] Agent prompt（提示词）已展示“官方版本环境”。
- [x] 已补充单元测试。

Evidence（证据）:

- `scripts/update_cn_knowledge.py`
- `tft/knowledge/data/patch_notes/17.1.json`
- `tft/knowledge/models/patch_note.go`
- `tft/knowledge/patch_note_query.go`
- `tft/knowledge/loader.go`
- `tft/knowledge/contracts/query_nlu.go`
- `tft/agent/prompt.go`
- `tft/knowledge/internal_query_test.go`
- `tft/agent/prompt_test.go`

## Context

MetaTFT 统计数据能回答“当前谁强”，但不能完整解释“为什么环境发生变化”。例如 17.1 版本公告中包含新赛季机制、商店概率、战利品、装备、神器和强化符文轮换，这些信息会直接影响运营节奏、合装判断和阵容理解。

如果只依赖统计榜单，agent 只能给出结果；如果只依赖版本公告，agent 又容易把设计意图误当成强度结论。因此需要把官方公告作为单独知识层接入。

## Decision

新增 `Patch Notes Knowledge`（版本公告知识）层：

1. `scripts/update_cn_knowledge.py` 支持 `--patch-note-url` 参数。
2. 传入官方公告 URL 后，脚本会抓取页面、处理 GBK/UTF-8 编码、抽取标题、发布时间和正文。
3. 公告正文被拆成结构化章节，写入 `tft/knowledge/data/patch_notes/<patch>.json`。
4. `knowledge` 加载版本公告，并在 `QueryNLU` 时根据用户问题匹配相关章节。
5. `agent` 在 prompt 中新增“官方版本环境”，用于解释版本原因，但不能替代统计数据给强度结论。

## Consequences

好处：

- 统计数据和官方公告职责分离：统计数据负责强度，公告负责版本原因。
- 版本公告可以解释“坦装为什么变弱”“7 级搜牌为什么变了”“新机制为什么影响开局”等问题。
- 面试展示时，可以体现 agent 不只是 RAG（检索增强生成），而是能融合不同可靠性层级的知识源。

风险：

- 官方公告内容很长，自动切分不等于完美理解，后续可能需要人工校验或 LLM 摘要辅助。
- 公告是设计意图和规则变化，不应直接推导 T0/T1 强度。
- 腾讯公告页面可能存在编码和 HTML 结构变化，需要脚本保持容错。

## Implementation Notes

当前结构：

```text
MetaTFT 统计数据 -> 阵容强度、装备优先级、英雄画像
官方版本公告 -> 系统/装备/强化/机制变化
agent prompt -> 先给数据结论，再用官方公告解释原因
```

版本公告章节包含：

- `type`：章节类型，例如 `system`、`item`、`augment`。
- `summary`：章节摘要。
- `impact_tags`：影响标签，例如 `shop_odds`、`itemization`、`frontline`。
- `details`：关键细节。

## Learning Note

要记住：版本公告不是“强度榜”，而是“版本语义层”。它让 agent 更懂环境变化，但最终强度判断仍然要回到统计数据。
