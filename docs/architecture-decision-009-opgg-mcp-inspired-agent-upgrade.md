# ADR009: 借鉴 OP.GG MCP 提升 TFT Agent 的数据工具能力

## Status

Accepted and partially implemented

## Context

我们实际测试了 OP.GG 公开 MCP（Model Context Protocol，模型上下文协议）服务的 TFT 工具。

它的 `tft_list_meta_decks` 能返回当前版本阵容数据，并包含：

- 阵容名称、多语言名称。
- `opTier`、平均名次、前四率、吃鸡率、登场率、样本量。
- 核心英雄、核心装备、站位。
- 羁绊档位。
- `early`（前期过渡阵容）。
- `middle`（中期过渡阵容）。
- `metadata`（数据统计量和更新时间）。

测试时 OP.GG 返回的数据时间为 `2026-05-09T06:04:17Z`，样本统计量约 `6,500,384`。

这说明 OP.GG MCP 的优势不是“会像教练一样回答”，而是它把游戏数据包装成了稳定、可调用、只读的工具层。

当前项目已经具备：

- 中文 NLU（Natural Language Understanding，自然语言理解）。
- Fast NLU（快速意图识别）。
- Alias（黑话/别名映射）。
- Knowledge（知识库）查询。
- Contract（公共协议）。
- Prompt（提示词）事实边界。
- 中文老玩家风格回答。
- 简易 MCP stdio（标准输入输出）服务。

但当前项目与 OP.GG MCP 相比还有差距：

- 阵容数据缺少显式 `early/middle` 过渡结构。
- MCP 工具粒度偏基础，仍以 `get_all_meta_comps` / `query_nlu` 为主。
- 工具返回暂不支持 `desired_output_fields`（期望字段裁剪），容易返回过大 JSON。
- 返回结果缺少统一 `metadata`，例如版本、更新时间、样本总量、来源。
- 工具命名还没有完全面向外部 AI 客户端的语义化设计。

## Decision

我们不照搬 OP.GG MCP，而是采用“OP.GG 数据工具能力 + 本项目中文教练 Agent”的组合路线。

本项目继续保留当前主链路：

```text
用户输入
  -> Fast NLU / LLM NLU
  -> knowledge query
  -> evidence
  -> LLM formatter
  -> 中文教练式回答
```

同时新增一层面向 MCP 和内部 Agent 共用的数据工具设计：

```text
tft_list_meta_comps
tft_get_comp_plan
tft_get_champion_builds
tft_get_item_fits
tft_get_trait_insight
tft_query_nlu
```

其中：

- `tft_list_meta_comps` 返回当前版本强势阵容，可支持 `limit`、`tier`、`desired_output_fields`。
- `tft_get_comp_plan` 返回某套阵容的 `final`、`early`、`middle` 三段计划。
- `tft_get_champion_builds` 返回英雄装备组合统计。
- `tft_get_item_fits` 返回某件装备适配英雄和阵容。
- `tft_get_trait_insight` 返回羁绊强度、代表阵容、常见单位。
- `tft_query_nlu` 保留中文自然语言入口。

所有 MCP 工具必须只返回结构化事实，不直接生成长篇自然语言。

LLM 仍然只负责把工具结果整理成中文教练回答。

## Comparison

### 当前 Agent 的优势

- 中文输入体验更好，能处理“剑魔、羊刀、九五、打工、能C”等玩家说法。
- 回答风格更贴近国服玩家，而不是只返回 JSON。
- 已经有 Prompt 事实边界，能降低旧版本幻觉。
- 已经能围绕装备、英雄、羁绊、版本公告做综合回答。
- 更适合面试展示 Agent 架构：Agent、Knowledge、Contract、Prompt、Eval 都能讲清楚。

### OP.GG MCP 的优势

- 数据结构更完整。
- 阵容带 `early/middle/final` 节奏。
- 样本量和更新时间明确。
- 工具是标准 MCP 形态，外部 AI 客户端容易调用。
- 工具粒度清晰，偏只读数据服务。

### 本项目要吸收的部分

- 增加 `early/middle` 过渡数据。
- 增加统一 `metadata`。
- 增加字段裁剪能力。
- 增加更语义化的 MCP 工具。
- 让 MCP 工具与 Agent 内部 knowledge 查询共用同一套 service，不维护两份逻辑。

## Implementation Plan

### Phase 1: Metadata 统一

在 knowledge contract 中增加统一 metadata：

```go
type KnowledgeMetadata struct {
    Version      string `json:"version"`
    Source       string `json:"source"`
    UpdatedAt    string `json:"updated_at"`
    SampleCount  int    `json:"sample_count,omitempty"`
}
```

所有主要返回结构都应能携带 metadata。

### Phase 2: 阵容计划结构

新增 CompPlan（阵容计划）：

```go
type CompPlan struct {
    Final  BoardSnapshot `json:"final"`
    Early  BoardSnapshot `json:"early,omitempty"`
    Middle BoardSnapshot `json:"middle,omitempty"`
}

type BoardSnapshot struct {
    Level  string        `json:"level,omitempty"`
    Units  []BoardUnit   `json:"units"`
    Traits []TraitMarker `json:"traits,omitempty"`
}
```

数据来源优先从脚本自动生成；如果当前 MetaTFT 数据无法直接稳定产出，则先允许为空，不由 LLM 编造。

### Phase 3: 字段裁剪

为列表型工具增加：

```json
{
  "limit": 5,
  "desired_output_fields": [
    "name",
    "tier",
    "avg_placement",
    "top4_rate",
    "win_rate",
    "best_build"
  ]
}
```

字段裁剪必须是白名单，不允许任意路径读取，避免接口不可控。

### Phase 4: MCP 工具升级

保留旧工具兼容，同时新增语义化工具：

```text
tft_list_meta_comps
tft_get_comp_plan
tft_get_champion_builds
tft_get_item_fits
tft_get_trait_insight
tft_query_nlu
```

旧工具如 `get_all_meta_comps` 可继续保留，但 README 和 MCP Quick Start 应推荐新工具。

### Phase 5: Agent 回答升级

当返回中存在 `early/middle` 时，Prompt 应优先输出：

```text
前期怎么过渡
中期怎么补质量
最终阵容怎么成型
装备优先级
什么时候别硬冲
```

如果没有 `early/middle`，必须明确“当前知识库没有前中期过渡数据”，不能编。

## Consequences

### Positive

- Agent 会更像“决策教练”，不是单纯知识查询。
- MCP 对外展示更专业，更接近 OP.GG 这类数据站能力。
- 字段裁剪能降低 LLM 上下文压力和响应延迟。
- metadata 能提升可信度，方便排查版本过期问题。
- 面试时可以清楚解释：我们借鉴了成熟游戏数据 MCP，但保留了自己的中文 Agent 优势。

### Negative

- Contract 会变复杂。
- 数据脚本需要继续维护。
- 如果 early/middle 数据来源不稳定，不能强行上线为硬事实。
- MCP 工具和 Agent 内部查询必须避免重复实现。

## Non-Goals

- 不做 OCR（图像识别）。
- 不做自动抓取玩家评论。
- 不把 OP.GG MCP 作为生产依赖。
- 不让 LLM 凭记忆补齐前中期过渡。
- 不追求一次实现所有 OP.GG 工具。

## Success Criteria

- 能通过 MCP 查询当前版本前 5 套阵容，并返回精简字段。已实现。
- 每个阵容结果包含 metadata。已实现，来源于 MetaTFT、TFTSet、趋势日期和样本量聚合。
- 至少部分阵容能返回 early/middle/final 三段结构。已支持结构；当前可从 MetaTFT 阵容数据派生成型 `final`，`early/middle` 仍等待数据流水线补充。
- 用户问“二阶段怎么过渡”“这套中期怎么补”时，Agent 能引用结构化数据回答。接口和 Prompt 已支持；回答质量取决于后续是否注入 early/middle 数据。
- `go test ./...`、MCP adapter tests、关键 TEST_CASES 均通过。已通过本次改动相关测试。

## Implementation Notes

本次落地了以下代码能力：

- 在 `contracts` 中新增 `KnowledgeMetadata`、`CompPlan`、`BoardSnapshot`、`BoardUnit`、`TraitMarker`。
- 在 `models.MetaComp` 中保留同名结构，允许后续数据脚本直接写入 `metadata` 和 `plan`。
- `UnifiedStore.QueryNLU` 会把聚合 metadata 放入 `QueryNLUResponse`。
- `CompSummary` 会携带单个阵容 metadata 和 plan。
- 如果原始数据没有 `plan`，knowledge 会从阵容单位、羁绊、核心装备中派生 `final` 成型棋盘。
- MCP adapter 保留旧工具，同时新增：

```text
tft_query_nlu
tft_list_meta_comps
tft_get_comp_plan
tft_get_champion_builds
tft_get_item_fits
tft_get_trait_insight
```

- `tft_list_meta_comps` 支持 `tier`、`tiers`、`limit`、`offset` 和 `desired_output_fields` 字段裁剪。
- Agent Prompt 会把知识库元信息和阵容计划作为证据提供给 LLM。
- 新增 `scripts/update_opgg_mcp_mvp.py`，可从 OP.GG MCP 的 `tft_list_meta_decks` 构建最小 knowledge 数据集。
- 新增 `make data-opgg-mvp` 和 `make data-opgg-mvp-check`，用于更新或 dry-run 校验 OP.GG MCP MVP 数据管线。
- 新增 `docs/OPGG_MCP_MVP_PIPELINE.md` 记录使用方式和限制。

仍未完成的部分：

- MetaTFT 数据源没有稳定 early/middle 棋盘，不能让 LLM 编造。
- OP.GG MCP 当前实际只返回 10 套 meta decks；脚本支持 top 20，但不会补假数据。
- 如果后续要商业化，需要确认 OP.GG 数据使用条款。

## Learning Summary

OP.GG MCP 证明了一件事：

```text
强 Agent 不一定要让模型更聪明，而是要让工具返回更稳定、更细、更适合模型使用的数据。
```

本项目下一步应继续坚持：

```text
数据决定事实
工具暴露能力
Agent 组织流程
LLM 负责表达
```
