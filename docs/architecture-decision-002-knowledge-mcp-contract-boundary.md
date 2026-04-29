# ADR-002: 将 Knowledge 设计为 MCP 时的 Contract 边界

## 状态

Accepted

## 背景

当前项目中：

- `agent` 依赖 `knowledge` 适配层
- `knowledge` 依赖 `data`
- `knowledge` 的接口虽然使用 `[]byte`，但语义上仍然围绕 `agent.Context` 和 `agent.NluEnrichedContext`

这带来几个问题：

1. `knowledge` 与 `agent` 在语义层高度耦合，只是没有形成 Go import cycle。
2. `knowledge` 为了避免直接引用 `agent`，复制了一套内部 context/result 类型，存在双份 schema 演进风险。
3. 如果未来将 `knowledge` 作为 MCP tool 独立暴露，继续直接围绕 `agent` 类型设计，会让边界混乱，难以独立部署、独立测试和跨语言接入。

## 目标

- 让 `knowledge` 能作为独立 MCP 能力暴露
- 保持 MCP 边界可序列化、可跨语言、可独立部署
- 避免 `agent` 与 `knowledge` 围绕同一语义模型各自复制类型
- 保持 `knowledge` 内部实现仍然是强类型，避免全仓库退化为“满地字节流”

## 决策

采用“传输层泛化，领域层强类型”的设计：

1. MCP 边界对外使用 JSON 文本
2. `knowledge` 内部 service 保持强类型 request/response struct
3. 新增独立 contract/schema 层，承载 `knowledge` 对外协议
4. `agent` 与 `knowledge` 都依赖 contract 层，而不是互相依赖
5. 不使用泛型作为 MCP 主接口方案

## 为什么不优先选择泛型

泛型适合：

- 复用容器逻辑
- 复用算法逻辑
- 在同一语言、同一进程内抽象公共代码

泛型不适合直接解决 MCP 边界问题，因为 MCP 关注的是：

- 序列化格式
- 跨进程通信
- 跨语言兼容
- 协议稳定性

在这类问题上，明确的 JSON schema 比泛型更重要。

因此，MCP 边界应优先选择：

- `string`
- `[]byte`
- 或明确 request/response struct 经过 JSON 编解码

而不是把泛型暴露为外部协议核心。

## 推荐分层

建议后续按下面的方向重构：

### 1. Contract 层

新增类似目录：

```text
tft/knowledge/contracts/
```

用于定义稳定协议，例如：

- `QueryNLURequest`
- `QueryNLUResponse`
- `GetCompByIDRequest`
- `GetCompByIDResponse`
- `GetMetaCompByIDRequest`
- `GetMetaCompByIDResponse`

这一层只放：

- request/response schema
- 枚举
- 可序列化结构

不放：

- 业务逻辑
- store 查询逻辑
- agent 专属行为

### 2. Knowledge Service 层

`knowledge` 内部继续使用强类型 service：

- 输入为 contract struct
- 输出为 contract struct
- 内部可依赖 `data.Store`

例如：

```go
type Service interface {
    QueryNLU(ctx context.Context, req contracts.QueryNLURequest) (contracts.QueryNLUResponse, error)
    GetCompByID(ctx context.Context, req contracts.GetCompByIDRequest) (contracts.GetCompByIDResponse, error)
}
```

### 3. MCP Adapter 层

MCP 适配层只做：

- JSON 解析
- 参数校验
- 调用 service
- JSON 序列化

也就是说，MCP adapter 是 transport adapter，不是业务实现本身。

## 对现有 `knowledge/tool.go` 的建议

当前接口：

- 传输形式是对的，适合 MCP
- 语义边界还不够干净

建议逐步改造成：

1. 注释不再直接引用 `agent.Context`
2. 使用 `contracts.QueryNLURequest` / `contracts.QueryNLUResponse`
3. `QueryRequest` / `QueryResponse` 可以保留为 transport alias

例如：

```go
type QueryRequest []byte
type QueryResponse []byte
```

可以保留，但它们只代表传输格式，不代表领域模型本身。

## 对 `agent` 的建议

`agent` 不应继续通过“猜测 knowledge JSON 结构”的方式协作。

建议：

- `agent` 依赖 `contracts`
- `knowledge` 依赖 `contracts`
- `agent` 中的 adapter 只负责 transport 与 contract 的转换

这样可以消除现在这种“没有 import cycle，但有语义循环”的状态。

## 备选方案

### 方案 A：继续保持现状

优点：

- 改动最小

缺点：

- schema 双份维护
- 语义耦合持续增长
- MCP 独立化成本越来越高

### 方案 B：让 `knowledge` 直接依赖 `agent`

优点：

- 短期开发快

缺点：

- 很容易形成真实循环依赖
- `knowledge` 无法独立成为 MCP

不采用。

### 方案 C：把所有接口都改成泛型

优点：

- 在 Go 内部看起来更抽象

缺点：

- 对跨进程协议无帮助
- 增加理解成本
- 不利于 MCP 场景的稳定 schema 管理

不采用。

## 后续演进建议

第一阶段：

- 抽 `contracts` 目录
- 将 `internalContext`、`internalNluEnrichedContext` 收敛到 contract schema

第二阶段：

- 将 `knowledge/tool.go` 改为面向 contract 的 transport 接口
- `agent` adapter 改为基于 contract 反序列化

第三阶段：

- 增加真正的 MCP adapter
- 将 `knowledge` 独立暴露为 tool/server

## 结论

当 `knowledge` 需要作为 MCP 时，最合适的方案不是“全用泛型”，而是：

- 对外：JSON 文本/字节流
- 对内：强类型 contract + service
- 结构上：抽独立 contract 层，切断 `agent` 与 `knowledge` 的语义缠绕

这是一个架构决策，因此本文采用 ADR 命名。
