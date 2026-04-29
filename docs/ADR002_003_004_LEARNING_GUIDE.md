# ADR002/003/004 学习指南

## 这次完成了什么

这次不是改业务回答文案，而是在补 Agent 的工程化骨架。

对应关系：

- ADR002：把 knowledge（知识库）边界做成稳定 contract（公共协议），并补齐 MCP（模型上下文协议）暴露。
- ADR003：把“工具怎么被组织和执行”显式化，补出轻量 tool registry（工具注册表）、planner（规划器）和 executor（执行器）。
- ADR004：保持启动时构建静态 Graph（执行图），但把可变依赖放进 runtime（运行时配置）里注入。

一句话理解：

> Graph 的路线不变，但 Graph 里用什么工具、怎么执行、怎么测试，开始从硬编码走向可替换。

## 入口在哪

主要入口仍然是：

- `tft/agent/agent.go`
- `tft/agent/graph.go`
- `cmd/tft-knowledge-mcp/main.go`

具体看代码时建议按这个顺序：

1. `NewAgentWithConfig`
2. `NewDefaultGraphRuntime`
3. `BuildGraphWithRuntime`
4. `RecommendationExecutor.Execute`
5. `cmd/tft-knowledge-mcp/main.go`

这样看最顺，因为它对应真实启动顺序。

## 核心结构是什么

### GraphRuntime（执行图运行时配置）

位置：

- `tft/agent/runtime.go`

它负责装配 Graph 运行时可变的东西：

- `Parser`：输入解析器。
- `Tools`：工具注册表。
- `PromptBuilder`：提示词构造器。
- `Planner`：规划器。
- `Executor`：执行器。

这里的关键点是：Graph 拓扑仍然是静态的，但 Graph 节点依赖不再散落在节点内部。

### ToolRegistry（工具注册表）

位置：

- `tft/agent/runtime.go`

当前登记了四个能力：

- `hero_comps`：英雄查阵容。
- `item_fit`：装备适配阵容。
- `comp_tier`：版本强度基准。
- `intersection`：多路结果合并。

这就是 ADR003 里 tool registry 的 MVP（最小可用版本）。它还不是复杂插件系统，但已经把工具目录显式化了。

### StaticToolPlanner（静态规划器）

位置：

- `tft/agent/runtime.go`

它现在永远返回固定计划：

```text
hero_comps + item_fit + comp_tier
```

为什么还要有这个结构？

因为现在虽然不让 LLM 自由选工具，但我们已经把“工具选择”这个概念留出来了。后续要做动态 planner（规划器）时，不需要推翻执行器。

### RecommendationExecutor（推荐执行器）

位置：

- `tft/agent/runtime.go`

它负责执行一次推荐流程：

```text
用户输入
  -> Parser
  -> Planner
  -> ToolRegistry 里的工具
  -> Intersection
  -> Recommendations
```

以前 `Agent.computeRecommendations` 里会自己 new parser、new tool、new intersection。现在它统一调用 executor，减少重复逻辑。

## 数据流怎么走

### 普通 Graph 路径

```text
NewAgentWithConfig
  -> NewDefaultGraphRuntime
  -> BuildGraphWithRuntime
  -> parser node
  -> hero_comps / item_fit / comp_tier 并行节点
  -> intersection node
  -> llm_input node
  -> llm node
```

注意：Graph 还是启动时编译，不是每个请求动态构建。

### 推荐计算路径

```text
Analyze
  -> computeRecommendations
  -> RecommendationExecutor.Execute
  -> Parser
  -> StaticToolPlanner
  -> ToolRegistry
  -> Intersection
```

这条路径用于先算结构化推荐，再给最终结果补充 `Recommendations`。

### MCP 路径

```text
cmd/tft-knowledge-mcp
  -> data.Store
  -> knowledge.Store
  -> knowledge.UnifiedStore
  -> mcp.Adapter
  -> StdioServer
```

`query_nlu` 的 MCP schema（输入结构说明）已经同步了 `unit_cost`、`role_query`，所以外部 MCP 调用方能看到垂直查询字段。

## 为什么这么设计

### 为什么不每个请求动态构建 Graph

因为当前项目的主流程是稳定的：

```text
解析 -> 工具查询 -> 汇合 -> LLM 生成
```

这类受控 workflow（工作流）适合启动时构建。每个请求动态拼 Graph 会增加调试成本，而且现在收益不明显。

### 为什么要加 Runtime

Runtime（运行时配置）解决的是“静态 Graph 会不会太死”的问题。

我们保留静态 Graph，但把可变部分抽出来：

- 工具可以换。
- PromptBuilder 可以换。
- Planner 可以换。
- 测试可以注入 fake tool（假的工具）。

这比“每次请求重新拼图”轻得多，也更适合当前阶段。

### 为什么 Planner 现在还是静态的

因为现在工具协议刚稳定，不适合立刻让 LLM 自由选择工具。

当前的 `StaticToolPlanner` 只做一件事：把固定工具计划显式表达出来。它像一个占位骨架，后面可以替换成更聪明的 planner。

## 如果换一种设计会有什么问题

### 如果继续在 Graph 节点里 new tool

问题：

- 测试很难替换工具。
- tool registry 没有统一位置。
- 后续要做不同运行模式时，每个节点都要改。
- Graph 既负责流程，又负责依赖装配，职责会变重。

### 如果现在就做完全动态 Planner

问题：

- LLM 可能乱选工具。
- 观测和调试还不够成熟。
- 当前业务问题还没复杂到必须自治。
- 面试展示时容易显得“架构很大，但稳定性不足”。

### 如果把 MCP 当成第一入口

问题：

- 本地调用链会被远程边界复杂化。
- 开发和测试速度会变慢。
- contract（公共协议）没稳定时，MCP 只是把混乱暴露出去。

所以现在更合理的是：

```text
本地强类型实现稳定
  -> MCP adapter 暴露边界
  -> 未来再独立部署
```

## 怎么判断 ADR002/003/004 已完成

### ADR002

看三件事：

- agent 和 knowledge 是否都依赖 `contracts`。
- knowledge 是否有 MCP adapter 和 stdio server。
- MCP schema 是否跟 contract 字段同步。

### ADR003

看三件事：

- 工具是否有统一 registry。
- 工具选择是否有 planner 概念。
- 工具执行是否有 executor，而不是散落在 agent 里。

### ADR004

看三件事：

- Graph 是否仍是启动时构建。
- Graph 节点是否不再直接 new tool。
- 可变依赖是否通过 runtime 注入。

## 我应该记住什么

最重要的不是记住类名，而是记住这条架构线：

```text
contract 解决边界
registry 解决工具目录
planner 解决选什么
executor 解决怎么跑
runtime 解决依赖注入
graph 解决稳定流程
```

中文理解：

- contract（公共协议）：两个模块之间说好传什么。
- registry（注册表）：系统有哪些工具。
- planner（规划器）：这次要用哪些工具。
- executor（执行器）：按计划真正执行。
- runtime（运行时配置）：把可变依赖集中注入。
- graph（执行图）：稳定的主流程骨架。

当前项目最适合记住一句话：

> 先让系统边界稳定，再逐步增加智能选择能力。

这也是面试里比较好讲的点：你不是一上来堆大词，而是能解释为什么这个阶段只做轻量 planner 和 runtime。
