# ADR-003: 面向 Harness 思路的渐进式迁移路线

## 状态

Accepted

## 背景

当前项目 `awesome` 的目标，不只是完成一个单点的 TFT AI 助手，而是逐步演进为一个可持续扩展的 AI Agent 学习项目。

你已经明确了几个方向：

- 当前项目先保持可用、可学习、可迭代
- 后期希望支持 `tool calling`（工具调用：让模型自主选择调用哪个工具）
- 后期希望把 `knowledge` 逐步演进为 MCP 能力
- 希望参考 `harness`（Agent 运行/编排框架思路），但不希望一开始就照搬重型架构

因此需要一份架构决策文档，回答三个问题：

1. 当前项目分别对应 `harness` 的哪些概念
2. 未来应该如何分阶段迁移
3. 哪些先做，哪些后做

## 先解释 Harness 是什么

本文里说的 `harness`，不是某一个特定产品名，而是一类通用思路：

- `planner`（规划器：决定下一步做什么）
- `executor`（执行器：真正执行调用、动作、工具）
- `tool interface`（工具接口：为模型提供稳定可调用能力）
- `runtime`（运行时：负责状态、控制、生命周期）
- `observation`（观测信息：每一步执行后得到的结果）
- `memory`（记忆：跨步骤或跨轮次保留的信息）
- `orchestration`（编排：把多步流程组织起来）

换句话说，`harness` 更像是一套“Agent 系统骨架”，而不是某个单一功能模块。

## 当前项目与 Harness 概念的映射

当前仓库已经具备很多 `harness` 雏形，只是还没有完全抽象成统一运行框架。

### 1. `agent` 对应什么

当前的 [tft/agent](/Users/cokecola/GolandProjects/awesome/tft/agent) 更接近：

- `orchestrator`（编排器：组织整个流程）
- 局部 `planner`（局部规划器：决定这次分析走哪些步骤）

例如：

- `graph.go` 负责把解析、工具查询、交集计算、LLM 输出串起来
- `agent.go` 负责对外暴露统一入口

因此，当前 `agent` 已经承担了小型 `runtime + orchestration` 的职责。

### 2. `tool` 对应什么

当前的 [tft/tool](/Users/cokecola/GolandProjects/awesome/tft/tool) 对应：

- `tools`（工具：可被上层调用的能力单元）

例如：

- 英雄查阵容
- 装备适配
- 阵容强度
- 交集计算

这些已经很接近未来 `tool calling` 里的工具形态，只是目前主要还是代码固定编排，不是模型自主选择。

### 3. `knowledge` 对应什么

当前的 [tft/knowledge](/Users/cokecola/GolandProjects/awesome/tft/knowledge) 对应：

- `knowledge provider`（知识提供者）
- 未来可演进为 `MCP tool server`（MCP 工具服务）

它的价值是：

- 承载相对稳定的知识与查询能力
- 作为独立能力对外提供数据支持

### 4. `data` 对应什么

当前的 [tft/data](/Users/cokecola/GolandProjects/awesome/tft/data) 对应：

- `domain data layer`（领域数据层：事实数据、索引和基础查询）

这一层更接近底座，不直接等于 harness 概念，但它是 tool/knowledge 的支撑层。

### 5. `handler` / `sse` 对应什么

当前的 [tft/handler.go](/Users/cokecola/GolandProjects/awesome/tft/handler.go) 和 [tft/sse](/Users/cokecola/GolandProjects/awesome/tft/sse) 对应：

- `transport layer`（传输层：HTTP / SSE 对外暴露）

这不是 agent 思考逻辑本身，而是把 runtime 的结果发送给用户。

### 6. `trace` / `logger` 对应什么

当前的 `trace`、`logger`、可观测性文档，对应：

- `observation`（观测）
- `telemetry`（遥测：日志、链路、指标）

这是未来真正做 `harness runtime` 时非常关键的一层，因为 Agent 系统如果没有观测，很快就会变成“看不懂为什么它这样做”。

## 当前项目的真实阶段判断

当前项目更像：

**单领域 Agent MVP，带有小型 harness 雏形**

而不是：

- 完整通用 Agent Framework
- 完整 MCP Native 系统
- 完整多工具自主规划系统

这个判断很重要，因为它决定了迁移策略必须是“渐进式”，而不是“大重构后一步到位”。

## 决策

采用三阶段迁移，而不是一次性重写。

之所以这样决策，是因为：

1. 当前项目已经可用，不能为了学习框架把已有可运行链路推翻
2. 你当前的目标是“边做边学”，不是“直接做最大最通用的架构”
3. `harness` 的很多能力要在系统复杂度足够高时才真正值得引入

## 三阶段迁移路线

## 阶段一：边界收敛期

### 目标

把当前项目从“能跑的功能系统”整理成“边界清晰的 Agent 系统”。

### 重点

1. 抽清 `agent internal model`（Agent 内部模型）
2. 抽清 `contracts`（共享边界协议）
3. 让 `knowledge` 更像独立查询能力，而不是 `agent` 中间态的镜像
4. 继续完善测试、可观测性和基础稳定性

### 这阶段对应的关键词

- `boundary`（边界）
- `contracts`（共享协议）
- `tool normalization`（工具标准化）

### 当前仓库里优先要做什么

先做：

- 把 `knowledge` 与 `agent` 的共享语义抽到独立 `contracts`
- 让 `knowledge` 接受更明确的查询参数，而不是完整 `agent.Context`
- 统一 tool 输入输出结构

后做：

- 暂时不要上多轮 planner
- 暂时不要上复杂 memory
- 暂时不要引入过重的 runtime 状态机

### 为什么先做这些

之所以先做边界，是因为：

- 以后不管是 MCP、tool calling、planner，都会依赖稳定边界
- 如果边界不稳，后面每增加一个能力都要重复返工

## 阶段二：工具自治期

### 目标

从“代码固定编排调用工具”，迁移到“模型可参与选择工具”。

### 重点

1. 让 tool 成为真正稳定、可枚举、可描述的能力单元
2. 引入轻量 `planner`（规划器）
3. 引入统一 `executor`（执行器）
4. 增加 tool schema、tool description、tool result normalization

### 这阶段对应的关键词

- `planner`（规划器）
- `executor`（执行器）
- `tool registry`（工具注册表）
- `tool calling`（工具调用）

### 目标形态

用户请求来了以后：

1. planner 判断问题类型
2. planner 决定调用哪些 tool
3. executor 执行 tool
4. agent 汇总结果再生成回复

### 当前仓库里适合如何演进

先做：

- 把 `tft/tool` 里的能力做成统一注册形式
- 每个 tool 明确定义输入/输出 schema
- 加一个轻量 tool registry

后做：

- 暂时不要做通用多 agent
- 暂时不要做复杂反思链路
- 暂时不要做太重的自治循环

### 为什么不是一开始就让 LLM 自由选择所有东西

之所以不建议一开始就完全放开，是因为：

- 没有稳定工具协议时，模型很容易乱调
- 没有好的观测时，很难调试 planner 为什么这么选
- 对学习项目来说，先做受控自治更容易理解

## 阶段三：运行时与 MCP 化期

### 目标

把当前系统从“单体内的 agent + tool”逐步演进为“带 MCP 工具边界的 Agent Runtime”。

### 重点

1. 把 `knowledge` 独立成 MCP 能力
2. 逐步抽出真正的 `runtime`（运行时控制层）
3. 增加更明确的 observation / checkpoint / memory 能力
4. 为未来的多 domain agent 打底

### 这阶段对应的关键词

- `MCP server`（MCP 服务）
- `runtime`（运行时）
- `checkpoint`（检查点）
- `memory`（记忆）
- `replay`（回放）

### 先做什么

先做：

- 先把 `knowledge` MCP 化
- 先保留本地调用与 MCP 调用双通路
- 先把 transport 和 service 分开

后做：

- 暂时不要过早做分布式复杂部署
- 暂时不要为了“像框架”而引入大量抽象层

### 为什么 MCP 放到第三阶段

之所以不是第一阶段就强上 MCP，是因为：

- MCP 是边界暴露形式，不是边界设计本身
- 如果 contract 还没稳定，先做 MCP 只是把混乱远程化
- 先把本地模块边界理顺，再 MCP 化，迁移成本最低

## 当前项目中哪些概念要先落地

### 第一优先级

- `contracts`（共享协议）
- `tool schema`（工具输入输出结构）
- `tool registry`（工具注册表）
- `observation`（执行观测）

### 第二优先级

- `planner`（轻量规划器）
- `executor`（统一执行器）
- `knowledge` 的 MCP adapter（MCP 适配层）

### 第三优先级

- `memory`（记忆）
- `checkpoint`（检查点）
- `replay`（执行回放）
- 更通用的 runtime 状态管理

## 当前项目中哪些东西暂时不要过度设计

1. 不要过早做通用多 Agent 框架
2. 不要过早把所有模块都 MCP 化
3. 不要为了抽象而抽象出过多 interface
4. 不要在 contract 还未稳定前引入复杂 planner

之所以不建议这些先做，是因为它们会抬高复杂度，却不一定立即提高学习收益。

## 一句话总结三阶段

第一阶段：

**先把边界讲清楚**

第二阶段：

**再让模型开始更自由地使用工具**

第三阶段：

**最后把这些能力升级成真正的 runtime 和 MCP 体系**

## 对学习顺序的建议

如果你的目标是“边做边学 AI Agent 开发”，推荐按下面顺序理解：

1. `contracts`（共享协议）
2. `tools`（工具）
3. `planner`（规划器）
4. `executor`（执行器）
5. `runtime`（运行时）
6. `memory/checkpoint`（记忆/检查点）
7. `MCP`（跨边界工具化）

之所以这样排序，是因为前面的概念是后面的基础。

## 结论

当前项目已经具备 `harness` 化的很多基础模块，但它现在更适合走“渐进式迁移”而不是“整体重写”。

本 ADR 的核心结论是：

- 当前项目先做边界收敛
- 中期做工具自治
- 后期做 runtime 与 MCP 化

这样既能保证项目持续可用，也最适合学习和吸收 Agent 工程化能力。
