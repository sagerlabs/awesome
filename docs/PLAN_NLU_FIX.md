# NLU流式推理问题修复计划

## 问题描述
- NLU流式推理耗时55秒，只输出了46个token
- 输出内容似乎被截断了

## 发现的问题

### 1. MaxTokens配置问题（model.go）
```go
// 问题代码
maxTokens := 60 // 默认值太低
// ...
if maxTokens > 150 {
    maxTokens = 3333  // 这个逻辑有问题，注释说"防止误配置"但实际设为3333
}
```

### 2. Token计数逻辑错误（handler.go）
```go
// 问题代码：tokenCount++ 在 flush() 函数里，每次flush才+1，不是每次收到token+1
flush := func() {
    s := buf.String()
    if s == "" {
        return
    }
    tokenCount++  // 这里计数错误！
    srv.Publish(buildEvent("message", StreamChunk{Type: "token", Content: s}, false))
    buf.Reset()
}
```

### 3. 流式输出缓冲逻辑
- `flushThreshold = 2` 可能太小
- 只在特定标点时才flush，可能导致内容累积

## 修复步骤

1. **修复 model.go** - 提高默认MaxTokens，修复逻辑错误
2. **修复 handler.go** - 修正token计数逻辑
3. **测试验证** - 验证修复后的效果
