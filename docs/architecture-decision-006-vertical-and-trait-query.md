# ADR 006: 垂直英雄查询与羁绊查询

## Status

Accepted

## Implementation Status（实现状态）

Status: Done for MVP（最小可用版本已完成）

Checklist:

- [x] `contracts.QueryNLURequest` 已新增 `unit_cost` 和 `role_query`。
- [x] `contracts.QueryNLUResponse` 已新增 `matched_champions` 和 `matched_traits`。
- [x] knowledge 层已实现英雄垂直查询。
- [x] knowledge 层已实现羁绊查询。
- [x] Agent prompt（提示词）已展示垂直英雄和羁绊命中结果。
- [x] 已补充单元测试。

Evidence（证据）:

- `tft/knowledge/contracts/query_nlu.go`
- `tft/knowledge/vertical_trait_query.go`
- `tft/knowledge/models/champion_profile.go`
- `tft/knowledge/data/champion_profiles.json`
- `tft/agent/prompt.go`
- `tft/knowledge/internal_query_test.go`
- `tft/agent/prompt_test.go`

## Context

当前 MVP 已经能回答“我有这些英雄/装备，能玩什么阵容”，但试玩后暴露出两个缺口：

- 缺少垂直英雄查询，例如“四费卡谁最厉害”“谁能 C”“谁能抗”。
- 缺少羁绊查询，例如“法师能玩吗”“这个羁绊怎么开”“哪些阵容依赖某羁绊”。

这两类问题不应该只交给 LLM prompt（大模型提示词）自由发挥，因为它们依赖知识库里的版本数据、阵容统计、英雄出装和羁绊激活信息。若只靠模型记忆，容易回答旧版本内容。

## Decision

在 `QueryNLU` 边界内新增垂直查询与羁绊查询结果结构，由 `knowledge` 负责产生事实数据，`agent` 只负责整理表达。

本阶段采用以下策略：

1. `contracts.QueryNLURequest` 新增 `unit_cost` 与 `role_query`。
2. `contracts.QueryNLUResponse` 新增 `matched_champions` 与 `matched_traits`。
3. 英雄垂直查询基于英雄在强阵容中的出现、出装、平均排名、样本量和可选英雄画像进行排序。
4. 羁绊查询基于 `MetaComp.Traits` 扫描当前知识库，返回该羁绊相关的强阵容、核心单位和激活档位。
5. 英雄费用与定位等稳定元信息放入可替换的 profile（画像）文件，不写死在 prompt 中。

## Consequences

好处：

- agent 不需要知道知识库内部文件结构，只消费统一 contract。
- 垂直查询和羁绊查询可以被测试覆盖，减少旧版本幻觉。
- 后续知识库更新时，只要 profile 和阵容数据同步，回答会跟随更新。

代价：

- 英雄费用数据不是当前 `team_comps/champions/items` 统计文件天然提供的，需要维护一层轻量 profile。
- “谁能 C/谁能抗”是基于装备与阵容统计的启发式判断，不等于完整战斗模拟。

## Implementation Notes

垂直查询的 MVP（最小可用版本）回答范围：

- 按费用过滤，例如 `unit_cost=4` 表示四费卡。
- 按定位过滤，例如 `role_query=carry` 表示能 C，`role_query=tank` 表示能抗，`role_query=all` 表示综合比较。
- 返回推荐英雄、费用、定位标签、最佳阵容、推荐装备、平均排名、前四率、吃鸡率。

羁绊查询的 MVP 回答范围：

- 识别玩家提到的羁绊名。
- 返回命中该羁绊的高质量阵容。
- 展示阵容中的羁绊档位、核心单位、运营节奏和装备方向。

暂不在本 ADR 内解决：

- OCR（图片文字识别）输入。
- 玩家评论/社区理解注入。
- 大规模黑话归一化，详见 ADR 005。

## Learning Note

要记住：垂直查询和羁绊查询是“知识检索能力”，不是“文案能力”。事实必须从 `knowledge` 出来，`agent` 只负责把事实讲清楚。
