# Agent 开发学习指南

## 先给结论

这个项目不是“一个 prompt 调大模型”。

它更像一个小型 Agent 系统：

```text
用户问题
  -> NLU（自然语言理解）
  -> Alias Normalizer（黑话归一化）
  -> Knowledge Query（知识查询）
  -> Structured Evidence（结构化证据）
  -> LLM Formatter（大模型表达）
  -> 用户答案
```

你现在要理解的不是“怎么写一句更好的 prompt”，而是：

> 如何把不稳定的大模型，放进一个有边界、有工具、有数据、有测试的系统里。

## 1. 什么是 Agent

在这个项目里，Agent 不是 LLM 本身。

Agent 是一个“会组织流程的系统”：

- 接收用户输入。
- 判断用户想问什么。
- 调用知识库和工具。
- 拿到结构化结果。
- 让 LLM 把结果说成人话。
- 控制 LLM 不要胡编。

对应代码：

- `tft/agent/agent.go`
- `tft/agent/graph.go`
- `tft/agent/runtime.go`

你可以这样理解：

```text
LLM 是脑子的一部分
Agent 是整个工作流
```

## 2. 什么是 Tool

Tool（工具）是 Agent 能调用的能力单元。

在这个项目里，工具不是让 LLM 自由乱调，而是代码受控调用：

- 英雄查阵容。
- 装备适配阵容。
- 阵容强度查询。
- 交集计算。
- knowledge 查询。

对应代码：

- `tft/tool`
- `tft/knowledge`
- `tft/agent/runtime.go`

你现在有一个轻量 `ToolRegistry`（工具注册表）：

```text
ToolRegistry
  -> hero_comps
  -> item_fit
  -> comp_tier
  -> intersection
```

它的意义不是炫技，而是把“系统有哪些能力”显式列出来。

## 3. 什么是 Knowledge

Knowledge（知识库）是事实来源。

它负责回答：

- 当前版本有哪些阵容？
- 阵容平均排名、前四率、吃鸡率是多少？
- 某件装备适合哪些阵容？
- 某个英雄出现在哪些阵容里？
- 官方版本公告说了什么？
- 玩家黑话应该映射成什么标准名？

对应代码：

- `tft/knowledge/unified_store.go`
- `tft/knowledge/internal_query.go`
- `tft/knowledge/vertical_trait_query.go`
- `tft/knowledge/patch_note_query.go`
- `tft/knowledge/data`

最关键的理解：

```text
强度结论来自 knowledge
表达方式来自 LLM
```

如果让 LLM 自己凭记忆回答，就会出现旧版本阵容、假数据、假强度。

## 4. 什么是 Contract

Contract（公共协议）是 Agent 和 Knowledge 之间说好的数据格式。

比如：

- 请求里有什么字段。
- 返回里有什么字段。
- 英雄洞察怎么表示。
- 羁绊洞察怎么表示。
- 黑话归一化记录怎么表示。

对应代码：

- `tft/knowledge/contracts/query_nlu.go`

为什么它重要？

因为没有 contract，Agent 和 Knowledge 就会靠“猜 JSON 结构”协作，后续字段一多就会乱。

你可以把 contract 理解成：

```text
两个模块之间的合同
```

例如：

```go
type QueryNLURequest struct {
    Intent    string
    Champions map[string]int8
    Items     []string
    RoleQuery string
}
```

这表示：Agent 不应该把一整段自然语言直接丢给 Knowledge 让它猜，而是要给结构化请求。

## 5. 什么是 NLU

NLU（Natural Language Understanding，自然语言理解）负责把玩家原话转成结构化字段。

对应文件：

- `tft/prompt/prompt_nlu.tmpl`

例子：

```text
用户：剑魔打工强吗？
NLU 输出：
{
  "intent": "champion_query",
  "champions": {"剑魔": 1},
  "role_query": "work"
}
```

注意：NLU 不负责把“剑魔”翻译成“亚托克斯”。

这是我们刻意设计的：

```text
NLU 提取原话
Knowledge 做归一化
```

这样更稳定，也更容易测试。

## 6. 什么是 Alias Normalizer

Alias Normalizer（别名归一化器）负责把玩家黑话转成标准名。

对应文件：

- `tft/knowledge/data/aliases.json`
- `tft/knowledge/unified_store.go`
- `tft/knowledge/contracts/query_nlu.go`

例子：

```text
剑魔 -> 亚托克斯
羊刀 -> 鬼索的狂暴之刃
法爆 -> 珠光护手
```

为什么不让 LLM 猜？

- LLM 可能混旧版本。
- 同一个黑话可能会变。
- knowledge 以后作为 MCP 工具时，不一定经过当前 prompt。
- JSON 映射更稳定、更可测试。

所以这里的原则是：

```text
模糊语言交给 LLM 提取
确定映射交给代码和数据
```

## 7. 什么是 Graph

Graph（执行图）是 Agent 的主流程骨架。

对应文件：

- `tft/agent/graph.go`

它定义了流程：

```text
START
  -> parser
  -> hero_comps / item_fit / comp_tier
  -> intersection
  -> llm_input
  -> llm
  -> END
```

这个项目采用的是静态 Graph：

```text
服务启动时构建 Graph
请求进来时复用 Graph
```

为什么不每次请求动态构建？

- 当前流程稳定。
- 静态 Graph 更容易调试。
- 每次请求拼图会增加复杂度。
- 真正变化的不是流程，而是工具、数据和参数。

## 8. 什么是 Runtime

Runtime（运行时配置）是 Graph 里的可变依赖。

对应文件：

- `tft/agent/runtime.go`

它负责注入：

- Parser（解析器）
- ToolRegistry（工具注册表）
- PromptBuilder（提示词构造器）
- Planner（规划器）
- Executor（执行器）

为什么需要 Runtime？

因为我们不想在 Graph 节点里到处写：

```go
tool.NewHeroCompsTool(store)
```

那样会导致：

- 测试难替换。
- 工具目录不清晰。
- 后续扩展要改很多节点。

现在变成：

```text
Graph 保持静态
Runtime 注入可变能力
```

这就是 ADR004 的核心。

## 9. 什么是 Planner 和 Executor

Planner（规划器）决定用哪些工具。

Executor（执行器）负责真正执行。

现在的 planner 还是静态的：

```text
每次都用 hero_comps + item_fit + comp_tier
```

对应代码：

- `StaticToolPlanner`
- `RecommendationExecutor`

为什么现在不做复杂 planner？

因为项目还没到需要 LLM 自由选工具的阶段。

现在先把结构留出来：

```text
以后可以换 planner
但现在不引入不稳定性
```

这是一个很重要的工程判断。

## 10. 什么是 Prompt

Prompt（提示词）在这个项目里有两个作用。

### NLU Prompt

负责提取结构化字段。

对应文件：

- `tft/prompt/prompt_nlu.tmpl`

它回答：

```text
用户到底提到了哪些英雄、装备、阶段、玩法？
```

### Format Prompt

负责把 knowledge 返回的结构化证据说成人话。

对应文件：

- `tft/agent/prompt.go`

它回答：

```text
根据这些证据，怎么给玩家一个自然建议？
```

重要原则：

```text
Prompt 不应该承担所有业务逻辑
```

如果是黑话映射、打工评分、阶段过滤、版本数据，这些应该进入数据结构或查询逻辑，而不是无限堆 prompt。

## 11. 为什么测试很重要

Agent 测试不是只测代码能不能编译。

它要测：

- 有没有命中知识。
- 有没有编造数据。
- 黑话有没有映射。
- 问打工时有没有跑去讲主C。
- 问装备时有没有把装备携带者误说成阵容核心。
- 没数据时能不能克制。

对应文件：

- `tft/knowledge/internal_query_test.go`
- `tft/agent/prompt_test.go`
- `docs/TEST_CASES.md`

你现在要养成一个习惯：

```text
不要凭感觉改 Agent
先记录失败案例
再判断失败类型
最后决定改 prompt、改查询、补数据还是补 alias
```

## 12. 这个项目的 Agent 开发闭环

完整闭环是：

```text
1. 用户提出自然语言问题
2. NLU prompt 抽取结构化字段
3. aliases.json 做黑话归一化
4. knowledge 查询阵容/装备/英雄/羁绊/版本公告
5. contract 返回结构化证据
6. format prompt 约束 LLM 表达
7. 用户测试发现失败
8. TEST_CASES 记录失败类型
9. 修数据、修查询、修 prompt 或修 alias
```

这就是你真正要理解的 Agent 开发。

## 13. 面试时怎么讲

你可以这样讲：

> 我做的不是一个单 prompt 应用，而是一个面向云顶之弈的版本知识 Agent。  
> 它通过 NLU 把用户问题结构化，通过 aliases.json 解决玩家黑话，通过 knowledge 层查询版本数据，通过 contract 保持 agent 和 knowledge 的边界稳定，最后再让 LLM 基于结构化证据生成回答。  
> 我还用 ADR 记录架构取舍，用测试用例追踪幻觉、黑话、阶段查询和数据缺口。

这段话很重要，因为它把项目从“调用大模型”提升到了“Agent 工程化”。

## 14. 我应该记住什么

最重要的 6 句话：

- Agent 不是 LLM，Agent 是组织 LLM、工具、知识和流程的系统。
- Tool 是能力单元，Knowledge 是事实来源，Contract 是模块协议。
- Prompt 负责表达和轻量解析，不应该吞掉所有业务逻辑。
- 黑话映射、版本数据、阶段逻辑要尽量数据化和可测试。
- Graph 保持主流程稳定，Runtime 注入可变依赖。
- 真正的 Agent 开发是“测试失败 -> 归因 -> 修正确层级”的循环。

如果你能把这 6 句话讲清楚，你对这个项目的理解就已经超过“会调 API”的层次了。
