# ADR-004: 采用静态 Graph 骨架与动态 Runtime 配置的混合方案

## 状态

Accepted

## Implementation Status（实现状态）

Status: Done for MVP（最小可用版本已完成）

Checklist:

- [x] `BuildGraph`（构建执行图）仍在服务初始化时构建并复用，静态 Graph 骨架已满足本 ADR。
- [x] `knowledge` 适配器已有独立构造入口，可从 knowledge data（知识库数据）加载后注入 agent。
- [x] Graph 节点不再直接 `new` tool（创建工具），改为通过 `GraphRuntime` 注入。
- [x] 已新增 `ToolRegistry`（工具注册表）和 `PromptBuilder`（提示词构造器）注入点。
- [x] `AgentConfig` 支持传入自定义 `GraphRuntime`，方便测试和后续模式切换。
- [ ] 多个预编译 Graph 模板尚未出现真实需求，不纳入当前 MVP 完成标准。

Evidence（证据）:

- `tft/agent/graph.go`
- `tft/agent/agent.go`
- `tft/agent/knowledge_runtime.go`
- `tft/agent/knowledge_adapter.go`
- `tft/agent/runtime.go`
- `tft/agent/runtime_test.go`

## 日期

2026-03-29

## 背景

当前项目中的 `agent`（智能体）在初始化时就会调用 `BuildGraph`（构建执行图）并完成编译，后续请求直接复用已经编译好的 Graph（执行图）。

当前实现的特点是：

- Graph 的节点和边在启动时确定
- 请求进入接口后，不会重新拼装 Graph
- 节点内部直接依赖具体实现，例如直接创建 tool（工具）实例、直接使用固定 store（存储）和固定 prompt（提示词）构造方式

这带来一个新的设计问题：

1. 继续保持“启动时构图”是否会让后续扩展变得僵硬
2. 是否应该在接口请求里按需动态构建 Graph
3. 后续引入 `tool calling`（工具调用）、`knowledge`（知识库）独立化、`MCP`（模型上下文协议）后，哪些部分应该动态，哪些部分应该固定

## 目标

- 保持当前系统的稳定性和可调试性
- 避免每次请求重复构建 Graph 带来的复杂度和额外开销
- 为后续 `tool registry`（工具注册表）、`planner`（规划器）、`knowledge provider`（知识提供者）、多模型切换预留扩展点
- 让当前项目的演进方式符合“先稳定边界，再增加自治”的整体路线

## 决策

采用“静态 Graph 骨架 + 动态 Runtime 配置”的混合方案：

1. Graph 的整体骨架保持静态，并在服务启动时预编译
2. 每次请求不重新构建完整 Graph
3. 真正需要变化的部分通过 Runtime 配置注入，而不是通过重建 Graph 注入
4. 后续如出现少量明显不同的工作流，可以新增少量预编译 Graph 模板，而不是在接口层无限动态拼图

## 为什么这样决策

### 1. 当前项目更像受控工作流，而不是完全开放的 Runtime

目前仓库中的主流程仍然是：

- parser（解析）
- tool fan-out（工具并发分发）
- intersection（结果汇合）
- llm（大模型生成）

这是一条相对稳定的 `workflow`（工作流），更适合预编译骨架，而不是每次请求重新定义节点和边。

### 2. 启动时构图更利于稳定性和调试

预编译 Graph 有明显优势：

- 更稳定的延迟表现
- 更清晰的节点边界
- 更容易做 tracing（链路追踪）和 observability（可观测性）
- 更容易定位问题到底发生在哪个节点

对于学习型项目，这种确定性很重要。

### 3. 每次请求动态构图会把复杂度提前引入

如果把“灵活性”放在接口层按请求重建 Graph，常见问题会变成：

- Graph 构建逻辑和业务逻辑混在一起
- 不同模式之间难以复用
- 调试时很难区分是 Graph 结构问题还是节点内部行为问题
- 后续 contract（共享协议）尚未稳定时，动态构图只会把边界问题放大

因此，当前阶段不采用“每次请求动态构图”作为主方案。

### 4. OpenAI 和 Claude 的实践更接近“请求级配置”，而不是“请求级重建系统骨架”

从 OpenAI 和 Anthropic 的官方接口设计看：

- tools（工具列表）通常是请求级传入
- tool choice（工具选择策略）通常是请求级传入
- MCP 连接通常是请求级启用

但这并不意味着每次请求都要重建整个运行骨架。更接近的理解是：

- 骨架可以稳定存在
- 本次允许什么能力暴露给模型，是请求级动态决定的

本项目采用的混合方案与这一思路一致。

## 静态部分与动态部分的边界

### 保持静态的部分

- Graph 的节点拓扑结构
- 节点命名和主流程顺序
- fan-out / fan-in（分发 / 汇合）关系
- 少量主流程模板，例如 baseline graph（基础图）、nlu graph（NLU 图）、stream graph（流式图）

### 逐步动态化的部分

- tool instances（工具实例）
- tool registry（工具注册表）
- knowledge provider（知识提供者）
- prompt builder（提示词构造器）
- model selection（模型选择）
- feature flags（功能开关）
- per-request options（单次请求参数）

## 推荐的演进方式

### 第一阶段

继续保持当前做法：

- 启动时构建并编译 Graph
- 请求时直接复用已编译 Graph

但要开始把节点里的“硬编码依赖”抽出来。

### 第二阶段

引入 Runtime 配置对象，例如：

- `GraphRuntime`
- `ToolRegistry`
- `KnowledgeProvider`
- `PromptBuilder`

让节点不再直接 `new` 具体 tool，而是从 Runtime 中获取依赖。

### 第三阶段

当工作流出现明显分化时，按模式维护少量预编译模板，例如：

- baseline graph（基础图）
- tool-calling graph（工具调用图）
- mcp-backed graph（基于 MCP 的图）

仍然避免在每次请求里完全动态构图。

## 不采用的方案

### 方案 A：继续把所有依赖写死在 Graph 节点中

优点：

- 改动最小

缺点：

- 后续替换 `knowledge` 或工具实现时需要频繁改 Graph
- Graph 会逐渐承担过多依赖装配责任
- 不利于后续做模式切换和实验

不作为长期方案。

### 方案 B：每次请求动态构建完整 Graph

优点：

- 表面上看最灵活

缺点：

- 增加构建和调试复杂度
- 让接口层承担过多运行时编排责任
- 过早引入框架化复杂度
- 当前项目阶段并不需要这种灵活度

不采用。

### 方案 C：只保留一个大而全的 Graph，通过条件分支覆盖所有模式

优点：

- 只有一个 Graph 模板

缺点：

- 节点职责容易膨胀
- 分支过多后可读性会明显下降
- 很难看清某一模式真正走了哪条链路

不采用。

## 对当前代码的直接指导

结合当前仓库，后续推荐按下面方向调整：

1. 保留 `BuildGraph` 在初始化阶段构建主 Graph
2. 将 Graph 节点中直接创建工具实例的逻辑逐步改为从 Runtime 获取
3. 将 store / knowledge / prompt / model 等依赖从“图内硬编码”改为“图外注入”
4. 在确实存在明显不同工作流时，再新增独立 Graph 模板

## 一句话结论

当前项目不应在每次接口请求里动态重建整个 Graph，而应：

- 保持 Graph 骨架静态
- 将可变能力做成 Runtime 配置
- 在需要时增加少量预编译模板

这样最符合当前项目的复杂度阶段，也最利于后续演进到 tools、planner 和 MCP。
