# ADR005 学习指南：黑话归一化

## 这次完成了什么

这次实现的是 alias normalization（别名归一化），也就是把玩家黑话、简称、外号转成知识库能稳定识别的标准名。

例子：

```text
羊刀 -> 鬼索的狂暴之刃
炸弹人 -> 吉格斯
龙王 -> 奥瑞利安·索尔
```

核心原则：

```text
LLM 负责提取原话
knowledge 负责确定性映射
agent 负责解释给玩家听
```

## 入口在哪

主要入口有三个：

- `tft/knowledge/data/aliases.json`
- `tft/knowledge/unified_store.go`
- `tft/agent/prompt.go`

看代码建议按这个顺序：

1. `aliases.json`
2. `Loader.loadAliases`
3. `Store.ResolveAlias`
4. `UnifiedStore.normalizeQueryContext`
5. `QueryNLUResponse.NormalizedTerms`
6. `BuildNluFormatPrompt`

## 核心数据结构

### aliases.json

位置：

- `tft/knowledge/data/aliases.json`

它是以后你边测试边补黑话的地方。

结构：

```json
{
  "heroes": {
    "炸弹人": "吉格斯"
  },
  "items": {
    "羊刀": "鬼索的狂暴之刃"
  },
  "traits": {
    "机甲": "霸天机甲"
  }
}
```

不要把黑话写死在 Go 代码里，后续版本更新、玩家说法变化时，只改 JSON 就够。

### NormalizedTerm（归一化记录）

位置：

- `tft/knowledge/contracts/query_nlu.go`

结构：

```go
type NormalizedTerm struct {
    Type       string `json:"type"`
    Raw        string `json:"raw"`
    Normalized string `json:"normalized"`
}
```

它用来告诉 agent：

```text
玩家说的是：羊刀
知识库实际查的是：鬼索的狂暴之刃
```

## 数据流怎么走

完整链路：

```text
用户输入：羊刀给谁？
  -> NLU 提取 items=["羊刀"]
  -> knowledge 读取 aliases.json
  -> normalizeQueryContext: 羊刀 -> 鬼索的狂暴之刃
  -> QueryNLU 使用标准名查询装备数据
  -> response 返回 normalized_terms
  -> prompt 展示“已识别黑话：羊刀 => 鬼索的狂暴之刃”
  -> LLM 回答“我按羊刀=鬼索的狂暴之刃理解”
```

关键点：

- 查询必须用标准名。
- 回答可以保留玩家原话。
- 不要把最终答案强行转回黑话。

## 为什么不用 LLM 猜

不用 LLM 作为主方案，是因为黑话需要稳定、可测试、可维护。

如果让 LLM 猜：

- 版本更新后容易混旧赛季。
- 同一个外号可能在不同语境下变化。
- 外部 MCP（模型上下文协议）调用方不一定经过你的 NLU prompt。
- 回归测试很难稳定。

用 JSON 映射的好处：

- 行为确定。
- 容易测试。
- 你试玩时可以随手补。
- knowledge 独立成工具后仍然可用。

## 如果映射错了怎么办

因为 response 会返回 `normalized_terms`，最终 prompt 会展示映射关系。

例如：

```text
已识别黑话：炸弹人 => 吉格斯
```

如果映射错了，用户一眼能看出来，后续只需要改 `aliases.json`。

## 这次剑魔问题说明了什么

`剑魔打工强吗` 的问题不是赛季没有剑魔，而是查询链路之前只在 vertical_query（垂直查询）里返回英雄洞察。

修正后的逻辑：

```text
剑魔
  -> aliases.json 映射为 亚托克斯
  -> champion_query 触发 MatchedChampions
  -> 只返回亚托克斯自己的英雄洞察
  -> agent 可以基于命中的阵容和装备回答
```

以后如果出现“明明有这张卡，但回答没有命中”，优先检查三件事：

- `aliases.json` 是否有黑话映射。
- `normalized_terms` 是否记录了映射结果。
- `MatchedChampions` 或 `MatchedComps` 是否真的返回了数据。

## 我应该记住什么

最重要的一句话：

> 黑话是输入理解问题，不是大模型自由发挥问题。

工程上应该这样分工：

- NLU（自然语言理解）：只提取玩家原话。
- aliases.json：保存黑话到标准名的映射。
- knowledge（知识库）：查询前统一归一化。
- contract（公共协议）：返回 `normalized_terms`。
- agent：告诉玩家“我按 A=B 理解”。

这也是面试里很好讲的点：你没有把不稳定问题丢给 prompt，而是把它沉到了可测试的数据层。
