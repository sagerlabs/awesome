# 自动回放评测设计（ADR-010 Phase 4 Demo）

## 状态

Demo 已实现（`cmd/tft-replay-eval`），未接入 CI。本文档说明设计取舍和
什么时候值得把它升级为正式评测。

## 目标

教练反馈闭环（ADR-010）已经能把用户否定的回答落盘到
`data/feedback_cases.jsonl`。回放评测回答的问题是：

> 当我们修了 Prompt / 知识库 / 查询逻辑之后，那些曾经被否定的对话，
> 现在 Agent 能不能纠偏成功？

也就是把 rejected 样本变成回归测试，而不只是人工复盘清单。

## 回放协议

每条 rejected 样本包含两个关键字段：`previous_user_input`（触发坏回答的
原问题）和 `user_input`（用户的否定）。回放对每条样本开一个全新 session
跑两轮：

```text
第 1 轮  重放 previous_user_input   → 建立会话状态（FeedbackMemory）
第 2 轮  重放 user_input（否定）     → Agent 收到 advice_rejected 信号
                                      评测的是这一轮的纠偏回答
```

两轮共用 session ID，是因为 feedback detector 只在同一会话内才会把
"不对/没用"识别为对上一轮的否定——这正是被评测的链路。

## 判定维度

来自 ADR-010 Phase 4 定义。规则只判机械可验证的部分：

| 维度 | 判定方式 | 通过线 |
|---|---|---|
| answered | 回放无错且回答非空 | 必须 |
| not_repeated | 与当时坏回答（`advice_summary`）字符 bigram 重叠率 < 0.6 | 必须 |
| acknowledged | 回答含纠偏开场词（"没命中/重新/上一轮"等） | 观察项 |
| actionable | 回答含行动措辞（"建议/优先/下一步"等） | 观察项 |

通过 = answered 且 not_repeated。acknowledged/actionable 只统计不卡分：
措辞类指标用关键词判定误报率高，硬卡会逼 Prompt 写八股。

**有意不做的判定**：「新回答是否真的更好」。这需要 ground truth 或
LLM-as-judge，在样本量小、评判标准未定的现阶段做出来就是噪声。
报告里未通过样本会列出新回答摘要，质量判断留给人工抽查。

## 运行模式

```bash
# 离线演示：内置 demoRunner 模拟纠偏后的 Agent，验证评测器本身
make replay-eval-demo

# 真实回放：打运行中的服务（顺序请求，避开限流和 LLM 配额）
go run ./cmd/tft-replay-eval -mode http -addr http://localhost:8080 -max 20
```

http 模式走 `/v1/tft/nlu/stream`（与前端同一条 SSE 链路），拼接 token
块得到完整回答。`-max` 默认 50，取最新样本，控制 LLM 成本。

## 已知局限 / 升级条件

1. **LLM 非确定性**：同一样本两次回放结果可能不同，所以现在只适合
   人工触发的趋势观察，不适合 CI 红绿门禁。要进 CI 需要先固定
   temperature、对模型版本做 pin，或改用多次回放取通过率。
2. **样本量**：bigram 重叠这类规则在几十条样本上看趋势可以，做统计
   结论不行。等 rejected 样本到几百条再谈通过率目标。
3. **质量判定缺位**：升级路径是加一个 `Judge` 实现（接口已留好），
   用强模型按 ADR-010 的四个维度打分，规则判定降级为快速预筛。

## 文件

- `cmd/tft-replay-eval/main.go` — 入口、两轮回放编排
- `cmd/tft-replay-eval/replay.go` — 样本加载、Runner 接口、HTTP/demo 实现
- `cmd/tft-replay-eval/judge.go` — 规则判定与报告
- `cmd/tft-replay-eval/testdata/demo_cases.jsonl` — 演示样本
