# ADR-002 Learning Guideline

> 临时学习笔记：等你理解完 ADR-002 这一阶段后，可以删除这个文件。

## 你只需要先记住的 4 句话

1. `contracts` 是边界协议，不是业务实现。
2. `agent` 不应该自己定义跨边界结构，只能引用或别名 `contracts`。
3. `knowledge/models` 是知识库内部模型，出边界前要转成 `contracts`。
4. 判断一个结构该不该进 `contracts`：看它是不是 `agent <-> knowledge` 两边都要理解的 JSON 数据。

## 当前 ADR-002 已经做完的部分

### 第一段：QueryNLU 边界收敛

入口文件：

- `tft/knowledge/contracts/query_nlu.go`
- `tft/agent/context.go`
- `tft/agent/nlu_data_query.go`
- `tft/knowledge/internal_query.go`
- `tft/knowledge/unified_store.go`
- `tft/agent/knowledge_adapter.go`

核心变化：

- `agent.Context` 现在是 `contracts.QueryNLURequest` 的别名。
- `agent.NluEnrichedContext` 现在是 `contracts.QueryNLUResponse` 的别名。
- `knowledge` 不再维护一份 `internalContext` 镜像结构。
- `QueryNLU` 的输入输出都统一走共享 contract。

你要理解的重点：

```text
agent.Context
    ↓ alias
contracts.QueryNLURequest
    ↓ JSON transport
knowledge.QueryNLU
    ↓
contracts.QueryNLUResponse
```

### 第二段：Meta 类型收敛

入口文件：

- `tft/knowledge/contracts/meta.go`
- `tft/agent/meta_types.go`
- `tft/knowledge/meta_contracts.go`
- `tft/knowledge/unified_store.go`

核心变化：

- `MetaComp / MetaChampion / MetaItem` 只在 `contracts` 里定义一份。
- `agent` 侧只保留类型别名。
- `knowledge/models` 继续作为内部加载和索引用模型。
- `knowledge` 对外返回前会把 `models` 转成 `contracts`。

你要理解的重点：

```text
knowledge/models.MetaComp
    ↓ convert
contracts.MetaComp
    ↓ JSON transport
agent.MetaComp
```

### 第三段：Tool 查询接口协议化

入口文件：

- `tft/knowledge/contracts/tool_queries.go`
- `tft/knowledge/tool.go`
- `tft/knowledge/unified_store.go`
- `tft/agent/knowledge_adapter.go`

核心变化：

- `TFTKnowledgeTool` 里的工具方法不再直接接收裸 `string`。
- 工具方法统一接收 `knowledge.Request`，返回 `knowledge.Response`。
- `agent` adapter 对外仍保留好用的类型安全方法，例如 `GetMetaCompByID(clusterID string)`。
- adapter 内部负责把类型安全方法转换成 `contracts.*Request / contracts.*Response`。

这一段覆盖：

- `GetCompByID(clusterID string)`
- `GetMetaCompByID(clusterID string)`
- `GetMetaCompByName(name string)`
- `SearchMetaComps(query string)`
- `GetMetaChampionByName(name string)`
- `GetMetaItemByName(name string)`

你要理解的重点：

```text
agent.GetMetaCompByID("xxx")
    ↓ adapter
contracts.GetMetaCompByIDRequest
    ↓ JSON transport
knowledge.GetMetaCompByID(req)
    ↓
contracts.GetMetaCompByIDResponse
```

这样以后接 MCP 时，工具入参和出参会更清晰。

### 第四段：MCP Adapter 与独立入口

入口文件：

- `tft/knowledge/mcp/adapter.go`
- `tft/knowledge/mcp/stdio.go`
- `cmd/tft-knowledge-mcp/main.go`

核心变化：

- `Adapter` 负责把 MCP 风格的 tool name 映射到 `TFTKnowledgeTool`。
- `StdioServer` 负责 JSON-RPC 的 `initialize`、`tools/list`、`tools/call`。
- `cmd/tft-knowledge-mcp` 提供独立启动入口。

你要理解的重点：

```text
MCP client
    ↓ tools/call
tft/knowledge/mcp.StdioServer
    ↓
tft/knowledge/mcp.Adapter
    ↓
knowledge.TFTKnowledgeTool
    ↓
contracts.*Response
```

这一层只做 transport adapter（传输适配），不写业务查询逻辑。

## 忙的时候怎么快速跟上

如果只有 10 分钟：

1. 看 `tft/knowledge/contracts/query_nlu.go`，知道 QueryNLU 的边界。
2. 看 `tft/knowledge/contracts/meta.go`，知道 Meta 的边界。
3. 看 `tft/knowledge/contracts/tool_queries.go`，知道工具查询的 request/response。
4. 看 `tft/knowledge/mcp/adapter.go`，知道 MCP tool 怎么映射到 knowledge。
5. 看 `tft/agent/meta_types.go`，确认 agent 只保留 alias。
6. 看 `tft/knowledge/meta_contracts.go`，理解 models 怎么转 contracts。

如果只有 3 分钟：

只记住这句话：

```text
agent 和 knowledge 不再各自维护一套共享结构，跨边界结构统一放 contracts。
```
