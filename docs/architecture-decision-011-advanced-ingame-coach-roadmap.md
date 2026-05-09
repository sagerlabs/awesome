# ADR011: Advanced In-Game Coach Roadmap

## Status

Proposed

## Context

当前项目已经具备：

- NLU（Natural Language Understanding，自然语言理解）提取。
- Knowledge（知识库）查询。
- OP.GG MCP MVP data pipeline（最小数据管线）。
- `early/middle/final`（前期/中期/成型）阵容计划。
- Feedback loop（反馈闭环）。
- 轻量 GameState（局内状态）文本解析，例如阶段、等级、血量、金币。
- 轻量 DecisionPolicy（决策策略）Prompt 提示。

但距离“任何人靠它上王者几百分”的局内教练，还缺更复杂的能力。

## Decision

以下能力暂不在当前阶段直接实现，作为后续高级教练路线：

```text
OCR board recognition（棋盘图像识别）
Opponent scouting（对手侦察）
Exact roll odds / EV（精确搜牌概率和期望）
Long session memory（长期会话记忆）
Automated feedback eval（自动反馈评测）
Action-level planner（动作级规划器）
```

当前阶段只做低风险、低延迟、可测试的能力：

- 从文本解析 `GameState`。
- 在 Prompt 中注入轻量 `DecisionPolicy`。
- 让回答明确“升不升、D不D、稳血还是贪经济”的方向。

## Why Not Now

### OCR board recognition

OCR（图像识别）需要截图权限、图像识别模型、棋盘单位定位和 UI 兼容。
这会显著增加工程复杂度，不适合作为当前最小闭环。

### Opponent scouting

对手侦察需要识别其他玩家阵容、同行数量、卡池压力。
目前没有稳定输入来源，不能让 LLM 凭空判断“有没有同行”。

### Exact roll odds / EV

精确 D 牌建议需要当前等级、金币、已有张数、外面已拿张数、卡池剩余量。
当前只知道用户文本里零散信息，不足以计算可靠 EV（Expected Value，期望收益）。

### Long session memory

长期记忆需要用户隔离、隐私边界、存储策略和过期策略。
当前反馈闭环是进程内轻量状态，足够 MVP。

### Action-level planner

动作级规划器需要稳定的 `GameState` schema（结构）、多步模拟和风险评分。
现在先用 Prompt 策略提示，不上复杂 planner。

## Next Steps

后续如果继续推进，建议顺序是：

1. 完整 `GameState` schema：阶段、等级、金币、血量、连胜/连败、场上单位、备战席、装备、目标阵容。
2. `DecisionPolicy` 独立模块：输入 GameState + Knowledge，输出升人口/D牌/合装备/转阵容建议。
3. `Eval` 自动化：把 feedback cases 变成回归测试。
4. OCR 只作为输入层增强，不改变核心决策层。

## Success Criteria

- 用户给出“3-2、6级、40血、50金币、已有牌和装备”时，Agent 能明确给出下一回合操作。
- 用户指出建议无效后，该样本能进入 Eval。
- 引入 OCR 前，纯文本输入已经能覆盖主要局内决策问题。

## Learning Summary

真正难的不是让 LLM 多说几句运营建议，而是把“局内状态”变成可计算、可测试、可复盘的决策输入。

```text
先把文字局面做成稳定决策，再考虑 OCR 和高级模拟。
```
