# ADR010: Coach Feedback Loop for Unsatisfactory Advice

## Status

Accepted and partially implemented

## Context

当前 TFT Agent 已经能基于 knowledge（知识库）、metadata（元信息）、CompPlan（阵容计划）和 Prompt（提示词）给出建议。

但一个游戏教练类 Agent 不能只看“有没有回答”，还要看“玩家是否接受了建议”。

如果用户在一次回答后继续追问、否定、换问法，或者没有沿着建议行动，这通常说明：

- 答案没有解决真实决策问题。
- 回答缺少关键上下文，例如阶段、血量、等级、经济、牌面。
- 推荐过于泛化，没有给出可执行下一步。
- 知识库命中不足，LLM 仍然说得太像“查资料”。
- 回答风格没有贴近玩家当下压力。

这类信号应该进入 feedback loop（反馈闭环），而不是只当作普通下一轮对话。

## Decision

引入 Coach Feedback Loop（教练反馈闭环）：

```text
Agent 给建议
  -> 用户接受 / 继续补信息 / 反问 / 否定
  -> 判断上一轮建议是否有效
  -> 必要时触发澄清、补查、改写或记录评测样本
```

这个 loop 不要求现在立刻做复杂记忆系统。

MVP 阶段先采用轻量规则：

- 如果用户直接追问“为什么”“你确定吗”“不对”“没用”“答非所问”，标记上一轮为 `advice_rejected`。
- 如果用户补充新局面，例如“我现在 3-2 七级 40 血”，标记为 `advice_needs_context`。
- 如果用户继续问推荐的下一步问题，例如“那这套几级启动”，标记为 `advice_accepted_or_continued`。
- 如果用户换问完全无关问题，不做负反馈。

当出现 `advice_rejected` 时，下一轮回答策略应该改变：

- 先承认上一轮可能没有命中问题。
- 不重复原答案。
- 优先要求或利用缺失上下文。
- 如果知识库没有证据，明确说没有，而不是换一种说法硬答。
- 把该样本记录到 Eval（评测集）候选。

## Design

### Runtime Signal

在会话层维护一个轻量状态：

```go
type AdviceFeedbackState struct {
    LastUserInput string
    LastAdvice    string
    LastIntent    string
    Feedback      string // accepted, rejected, needs_context, unrelated
}
```

MVP 不一定持久化，可以先只在单次会话或前端状态里保存。

### Detection

通过规则优先识别，不先上 LLM：

```text
rejected:
  不对、不是、答非所问、没用、你确定、幻觉、当前版本不是、这不对

needs_context:
  我现在、如果我有、我场上、我血量、我等级、我经济、我装备

continued:
  那、继续、这套、几级、怎么过渡、装备怎么给
```

如果规则无法判断，默认 `unrelated`，不要过度解读用户。

### Response Strategy

当 feedback 是 `rejected`：

```text
上一轮可能没命中你的问题，我重新按当前知识库证据看。
```

然后重新给结论，不要长篇道歉。

当 feedback 是 `needs_context`：

```text
这个补充信息很关键，我按你现在的阶段/血量/装备重新判断。
```

当 feedback 是 `continued`：

```text
沿着上一套建议继续推进，直接回答下一步操作。
```

### Eval Logging

被判定为 `advice_rejected` 的对话，应追加到一个人工可读文件，例如：

```text
docs/FEEDBACK_CASES.md
```

记录字段：

```text
用户原问题
Agent 原回答摘要
用户反馈
判定类型
可能原因
后续修复方向
```

这不是为了监控用户，而是为了形成项目的 Eval（评测）资产。

## Consequences

### Positive

- Agent 会更像真实教练，而不是一次性问答机器人。
- 用户不接受建议时，系统能主动修正路线。
- 可以把失败样本沉淀成 Eval，提高项目面试含金量。
- 有助于发现 Prompt、Data（数据）、Query（查询逻辑）和 Alias（黑话映射）问题。

### Negative

- 需要维护一点会话状态。
- 如果误判用户意图，可能显得啰嗦。
- 如果每轮都输出“你可以继续问”，会增加 token（词元）成本。

## Non-Goals

- 不做长期用户画像。
- 不做复杂强化学习。
- 不把用户不追问简单等同于满意。
- 不把所有下一轮问题都视为上一轮失败。

## Implementation Plan

### Phase 1: 文档和人工记录

- 新增 ADR010。
- 新增 `docs/FEEDBACK_CASES.md` 模板。
- 人工把明显失败回答记录进去。

### Phase 2: 轻量规则识别

- 在前端或 handler 层保存上一轮问题和回答摘要。
- 新增 feedback detector（反馈识别器）。
- 把 feedback 类型放入 prompt evidence。

### Phase 3: 回答策略调整

- Prompt 根据 `advice_rejected / needs_context / continued` 改变开头策略。
- rejected 时禁止重复上一轮原话。

### Phase 4: Eval 集成

- 把 rejected 样本整理成自动化 Eval。
- 评测维度包含：

```text
是否承认上一轮没命中
是否使用新上下文
是否避免重复错误答案
是否给出更可执行操作
```

## Success Criteria

- 用户指出“不对/答非所问”后，下一轮不会机械重复旧答案。
- 用户补充局面后，Agent 会重新基于新局面判断。
- 至少 20 条 rejected case 被整理成 Eval 候选。
- Prompt 没有因为 feedback loop 变得啰嗦。

## Implementation Notes

已实现 MVP：

- 新增 `AdviceFeedback` contract，用于在 Agent Prompt 中携带上一轮反馈信号。
- 新增 `FeedbackMemory`，在 Agent 进程内保存上一轮用户问题和回答摘要。
- 新增规则型 feedback detector，不额外调用 LLM。
- 主 NLU 链路会把 feedback 注入 `NluContext -> QueryNLUResponse -> BuildNluFormatPrompt`。
- 流式回答结束后会记录上一轮回答，供下一轮判断。
- `advice_rejected` 会追加到 `docs/FEEDBACK_CASES.md`，作为后续 Eval 候选。

暂未实现：

- 多用户 session 隔离。当前 MVP 是进程内单 Agent 轻量状态。
- 自动化 Eval 消费 `FEEDBACK_CASES.md`。
- 长期用户画像或跨设备反馈记忆。

## Learning Summary

一个教练型 Agent 的关键不是每次都自信回答，而是知道什么时候上一轮建议没有被用户接受。

```text
好的 Agent 不只生成答案，也观察答案是否推动了用户决策。
```

对本项目来说，feedback loop 是从“知识查询助手”升级到“决策教练”的关键一步。
