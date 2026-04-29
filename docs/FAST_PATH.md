# Fast Path 设计记录

## 背景

当前 NLU 流式接口的完整链路通常会调用两次 LLM（大模型）：

```text
用户输入
  -> LLM NLU 提取结构化字段
  -> knowledge 查询
  -> LLM formatter 生成自然语言回答
```

这会导致常见短问题响应偏慢，例如：

- `剑魔打工强吗？`
- `羊刀给谁？`
- `四费卡谁能C？`
- `海魔人能玩吗？`

这些问题结构稳定，不一定每次都需要先调用 LLM 做 NLU。

## 当前方案

新增 `FastNLUExtractor`（快速自然语言理解提取器）。

它在 NLU Graph（自然语言理解执行图）的第一个节点执行：

```text
先尝试规则解析
  -> 命中：直接生成 QueryNLURequest
  -> 未命中：回退到原来的 LLM NLU
```

也就是说：

```text
简单高频问题：1 次 LLM
复杂模糊问题：仍然 2 次 LLM
```

## 当前支持的快速识别

### 英雄问题

例如：

```text
剑魔打工强吗？
```

会解析成：

```json
{
  "intent": "champion_query",
  "champions": {"剑魔": 1},
  "role_query": "work"
}
```

### 装备问题

例如：

```text
羊刀给谁？
```

会解析成：

```json
{
  "intent": "item_query",
  "items": ["羊刀"]
}
```

### 垂直查询

例如：

```text
四费卡谁能C？
```

会解析成：

```json
{
  "intent": "vertical_query",
  "unit_cost": 4,
  "role_query": "carry"
}
```

### 羁绊查询

例如：

```text
海魔人能玩吗？
```

会解析成：

```json
{
  "intent": "trait_query",
  "traits": ["海魔人"]
}
```

## 为什么不直接零 LLM

当前只跳过第一段 LLM NLU，不跳过最后的 formatter LLM。

原因：

- formatter 负责把结构化证据说成人话。
- 现在回答还需要自然表达和边界提醒。
- 直接模板化会快，但体验可能突然下降。

后续如果仍然慢，可以继续做第二阶段：

```text
高频问题模板回答
  -> 命中后直接返回
  -> 零次 LLM
```

适合模板化的问题：

- 装备给谁。
- 某英雄能不能打工。
- 某羁绊能不能玩。
- 当前版本前三阵容。

## 代码入口

- `tft/agent/fast_nlu.go`
- `tft/agent/graph.go`
- `tft/agent/fast_nlu_test.go`

## 我应该记住什么

Fast path（快速通道）的核心不是让系统变聪明，而是：

> 对高置信度、高频、结构稳定的问题，避免不必要的大模型调用。

复杂问题仍然交给 LLM NLU，简单问题用规则先接住。
