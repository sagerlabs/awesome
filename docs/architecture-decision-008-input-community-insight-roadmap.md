# ADR 008: 输入体验与玩家评论注入路线

## Status

Accepted

## Implementation Status（实现状态）

Status: Roadmap（路线图，尚未实现）

Checklist:

- [ ] OCR（图片文字识别）输入尚未实现。
- [ ] `Input Adapter`（输入适配器）尚未抽象。
- [ ] `Community Insight Provider`（玩家评论洞察提供器）尚未实现。
- [ ] 玩家评论数据源尚未定义。
- [ ] 评论来源、版本、置信度等字段尚未进入 contract（公共协议）。
- [x] 黑话归一化已在 ADR005 中作为 knowledge 统一能力完成 MVP。

Evidence（证据）:

- `docs/architecture-decision-005-slang-normalization-for-knowledge-query.md`
- `tft/knowledge/contracts/query_nlu.go`
- `tft/agent/prompt.go`

## Context

试玩后还发现两个暂不适合塞进本次 MVP 的问题：

- 玩家手动输入文字比较慢，后续可能需要 OCR（Optical Character Recognition，图片文字识别）。
- 黑话、昵称、玩家口语还不完整，例如英雄外号、装备简称、阵容俗称。

另外，希望引入玩家评论帮助 AI 理解阵容核心，但不能要求手工长期维护在代码里，需要支持灵活注入。

## Decision

将这些能力作为 knowledge 输入层的可扩展路线，而不是直接写死在 agent prompt 中。

后续采用三个扩展点：

1. `Input Adapter`（输入适配器）：接收文字、OCR 结果、未来可能的截图识别结果，并统一转成用户问题。
2. `Alias Normalizer`（别名/黑话归一化器）：在 `QueryNLU` 前把“炸弹人/女枪/羊刀”等口语映射到知识库名称，延续 ADR 005。
3. `Community Insight Provider`（玩家评论洞察提供器）：从可替换的数据源读取玩家评论、阵容理解、注意事项，并以附加知识片段注入查询结果。

玩家评论不进入代码逻辑，不要求用户手写 Go 代码。它应该来自可替换来源：

- 本地 JSON/Markdown 文件。
- 后续脚本抓取或导入的社区评论。
- 未来可能的数据库或远程知识源。

## Consequences

好处：

- 不会把社区口味、黑话和临时版本理解绑死在 prompt 中。
- 后续更新版本时，可以替换评论数据源，而不是改 agent 代码。
- OCR、黑话、评论注入都能接在 knowledge 前后，不破坏当前 contract 边界。

风险：

- 玩家评论质量不稳定，需要标注来源、时间和置信度。
- 评论只能作为解释辅助，不能覆盖统计数据中的事实。
- OCR 识别可能出错，需要保留原始识别文本，方便排查。

## Proposed Interface

后续可引入类似结构：

```go
type CommunityInsight struct {
    Source     string
    Version    string
    TargetType string
    TargetID   string
    Content    string
    Confidence float64
}
```

`Source` 表示来源，`Version` 表示适用版本，`TargetType` 可以是 `comp`、`champion`、`trait`，`TargetID` 指向阵容、英雄或羁绊。

## Learning Note

要记住：OCR、黑话和玩家评论都属于“输入/理解增强”，不要让它们直接替代知识库事实。真正的回答仍然应该以统计知识库为主，评论只做解释补充。
