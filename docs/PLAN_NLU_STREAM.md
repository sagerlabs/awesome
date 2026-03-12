# NLU Stream 接口实现计划

## 目标
1. 基于现有的nlu analyse创建一个支持stream的接口
2. 在BuildNluGraph中新加一个node，把查出来的数据打包发给LLM润色

## 实现步骤

### 1. 修改 `tft/agent/graph.go`
- 在BuildNluGraph中添加新的node `llm_refine`
- 这个node接收`NluEnrichedContext`，将数据打包成prompt发给LLM润色
- 修改Graph的输出类型，使其支持流式输出

### 2. 修改 `tft/agent/agent.go`
- 添加 `NluAnalyzeStream` 方法，支持流式输出
- 类似现有的 `AnalyzeStream` 方法

### 3. 修改 `tft/handler.go`
- 添加新的路由 `/v1/tft/nlu/stream`
- 添加 `NluAnalyzeStream` 处理函数
- 复用现有的流式推送逻辑

### 4. 添加prompt模板
- 在 `tft/prompt/` 目录下添加NLU润色的prompt模板

## 文件变更列表
- `tft/agent/graph.go` - 添加新node，修改BuildNluGraph
- `tft/agent/agent.go` - 添加NluAnalyzeStream方法
- `tft/handler.go` - 添加新路由和处理函数
- `tft/prompt/nlu_refine.tmpl` - 添加润色prompt模板（如果需要）
