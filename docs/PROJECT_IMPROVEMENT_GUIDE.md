# 项目收口改进指南

## 这次改进解决什么

这次不是继续横向加功能，而是把项目从“能跑的 MVP”收口成“能维护、能讲清楚、能面试展示”的 Agent（智能体）项目。

核心目标：

- Primary API（主接口）明确为 `/v1/tft/nlu` 和 `/v1/tft/nlu/stream`。
- Legacy API（历史兼容接口）保留 `/v1/tft/analyze`，但不再作为主要演示入口。
- Alias（别名/黑话）统一从 `tft/knowledge/data/aliases.json` 读取。
- Work Query（打工查询）有结构化评分，不再完全靠 Prompt（提示词）猜。
- Observability（可观测性）能看到 Fast NLU（快速自然语言理解）、LLM（大模型）调用和 knowledge（知识库）命中情况。
- README 和 Makefile 对齐当前真实项目结构。

## 你应该怎么理解现在的链路

```text
玩家问题
  -> Fast NLU / LLM NLU
  -> QueryNLURequest
  -> knowledge 查询
  -> QueryNLUResponse
  -> Format LLM
  -> 玩家答案
```

你要记住一个边界：

```text
事实由 knowledge 给
表达由 LLM 给
流程由 agent 控制
```

这就是这个项目作为 AI Agent 面试项目最值得讲的地方。

## 如何判断 ADR 是否落地

不要只看文档里的复选框，要按“代码证据”判断：

- Contract（公共协议）是否有对应结构体。
- Knowledge（知识库）是否真的消费这个字段。
- Agent（智能体）是否真的生成或传递这个字段。
- Prompt（提示词）是否使用查询结果，而不是自由编。
- Test（测试）是否覆盖关键路径。
- README / Makefile 是否能让别人复现。

## 以后改功能的顺序

推荐顺序：

1. 先补 `docs/TEST_CASES.md`，写清楚失败样例。
2. 再判断失败属于 Alias / Query / Data / Prompt / UX 哪类。
3. 如果是确定映射，改 `aliases.json`。
4. 如果是事实缺失，改 knowledge 数据或查询。
5. 如果是表达别扭，最后再改 Prompt。
6. 改完跑 `go test ./...`。

不要一上来就改 Prompt。Prompt 是最后一层，不是万能胶。

## 现在最适合演示的问题

```text
当前版本最强的三套阵容是什么？
我有珠光护手，可以玩什么阵容？
剑魔打工强吗？
二阶段几费卡打工强？
四费卡谁能C？
海魔人能玩吗？
九五能玩吗？
```

这些问题能展示：阵容、装备、英雄垂直查询、阶段打工、羁绊和黑话识别。
