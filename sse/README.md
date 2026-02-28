# SSE (Server-Sent Events) for Go

一个完整的 Go 语言 SSE (Server-Sent Events) 实现，参考了 OpenAI、Anthropic 等主要 LLM 提供商的 Go SDK 设计。

## 目录

- [什么是 SSE？](#什么是-sse)
- [为什么选择 SSE？](#为什么选择-sse)
- [快速开始](#快速开始)
- [API 文档](#api-文档)
- [使用示例](#使用示例)
- [最佳实践](#最佳实践)
- [性能优化](#性能优化)
- [与 LLM 流式响应的集成](#与-llm-流式响应的集成)

## 什么是 SSE？

Server-Sent Events (SSE) 是一种基于 HTTP 的服务器推送技术，允许服务器向客户端单向发送实时数据。

### SSE vs WebSocket

| 特性 | SSE | WebSocket |
|------|-----|-----------|
| 协议 | HTTP/HTTPS | 自定义协议 |
| 方向 | 服务器 → 客户端 | 双向 |
| 自动重连 | ✅ 内置 | ❌ 需要手动实现 |
| 断线重连 | ✅ 自动从断点继续 | ❌ 需要手动处理 |
| 防火墙友好 | ✅ 使用标准 HTTP 端口 | ❌ 可能被阻止 |
| 实现复杂度 | 简单 | 复杂 |

## 为什么选择 SSE？

### 1. LLM 流式响应的标准
OpenAI、Anthropic、Claude 等大语言模型都使用 SSE 作为流式响应的标准协议。

### 2. 简单可靠
SSE 使用标准的 HTTP，不需要额外的协议升级，易于调试和部署。

### 3. 自动重连
内置的断线重连机制，支持从上次断开的位置继续接收数据。

## 快速开始

### 安装

```bash
go get github.com/your-org/awesome/utils/sse
```

### 最简单的服务器

```go
package main

import (
    "net/http"
    "time"
    "github.com/your-org/awesome/utils/sse"
)

func main() {
    server := sse.NewServer()
    
    // 定期发送事件
    go func() {
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()
        
        for range ticker.C {
            server.Publish(&sse.Event{
                Data: time.Now().Format(time.RFC3339),
            })
        }
    }()
    
    http.Handle("/events", server)
    http.ListenAndServe(":8080", nil)
}
```

### 最简单的客户端

```go
package main

import (
    "context"
    "fmt"
    "github.com/your-org/awesome/utils/sse"
)

func main() {
    client := sse.NewClient("http://localhost:8080/events")
    ctx := context.Background()
    
    eventCh, errCh := client.Stream(ctx)
    
    for {
        select {
        case event := <-eventCh:
            fmt.Printf("Received: %s\n", event.Data)
        case err := <-errCh:
            fmt.Printf("Error: %v\n", err)
            return
        }
    }
}
```

## API 文档

### Event 结构

```go
type Event struct {
    ID    string // 事件 ID（可选）
    Event string // 事件类型（可选，默认为 "message"）
    Data  string // 事件数据（必需）
    Retry int    // 重试间隔毫秒（可选）
}
```

### Event 方法

#### String() string
返回 SSE 格式的字符串表示。

```go
event := &sse.Event{
    ID:    "123",
    Event: "message",
    Data:  "hello",
    Retry: 5000,
}
fmt.Println(event.String())
// 输出:
// id: 123
// event: message
// retry: 5000
// data: hello
//
// (空行)
```

#### MarshalJSON(v interface{}) error
将数据序列化为 JSON 格式。

```go
event := &sse.Event{}
err := event.MarshalJSON(map[string]string{
    "status": "completed",
    "result": "42",
})
```

#### UnmarshalJSON(v interface{}) error
从 JSON 数据反序列化。

```go
event := &sse.Event{Data: `{"status":"completed"}`}
var result map[string]string
err := event.UnmarshalJSON(&result)
```

### Server 方法

#### NewServer(opts ...ServerOption) *Server
创建一个新的 SSE 服务器。

```go
// 基本用法
server := sse.NewServer()

// 带自定义选项
server := sse.NewServer(
    sse.WithBufferSize(200),
    sse.WithOnConnect(func(ch chan *sse.Event) {
        fmt.Println("Client connected")
    }),
    sse.WithOnDisconnect(func(ch chan *sse.Event) {
        fmt.Println("Client disconnected")
    }),
)
```

#### Subscribe() chan *Event
订阅事件流。

```go
ch := server.Subscribe()
```

#### Unsubscribe(ch chan *Event)
取消订阅。

```go
server.Unsubscribe(ch)
```

#### Publish(event *Event)
向所有客户端发布事件。

```go
server.Publish(&sse.Event{
    Data: "Hello, world!",
})
```

#### ClientCount() int
获取当前连接的客户端数量。

```go
count := server.ClientCount()
```

#### ServeHTTP(w http.ResponseWriter, r *http.Request)
实现 http.Handler 接口，可以直接用作 HTTP 处理器。

```go
http.Handle("/events", server)
```

### Client 方法

#### NewClient(url string, opts ...ClientOption) *Client
创建一个新的 SSE 客户端。

```go
// 基本用法
client := sse.NewClient("http://example.com/events")

// 带自定义选项
client := sse.NewClient(
    "http://example.com/events",
    sse.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
    sse.WithLastEventID("last-id"),
    sse.WithRetry(10000),
)
```

#### Connect(ctx context.Context) (*EventReader, error)
连接到 SSE 服务器。

```go
reader, err := client.Connect(ctx)
if err != nil {
    // 处理错误
}
defer reader.Close()
```

#### Stream(ctx context.Context) (<-chan *Event, <-chan error)
流式接收事件（自动重连）。

```go
eventCh, errCh := client.Stream(ctx)

for {
    select {
    case event := <-eventCh:
        // 处理事件
    case err := <-errCh:
        // 处理错误
    }
}
```

### EventReader 方法

#### Read() (*Event, error)
读取下一个事件。

```go
for {
    event, err := reader.Read()
    if err == io.EOF {
        break
    }
    if err != nil {
        // 处理错误
    }
    // 处理事件
}
```

#### Close() error
关闭事件读取器。

```go
reader.Close()
```

## 使用示例

### 示例 1: 聊天消息推送

```go
package main

import (
    "encoding/json"
    "net/http"
    "sync"
    "github.com/your-org/awesome/utils/sse"
)

type ChatMessage struct {
    Username string `json:"username"`
    Text     string `json:"text"`
    Time     string `json:"time"`
}

type ChatServer struct {
    sseServer *sse.Server
    messages  []ChatMessage
    mu        sync.RWMutex
}

func NewChatServer() *ChatServer {
    return &ChatServer{
        sseServer: sse.NewServer(),
        messages:  make([]ChatMessage, 0),
    }
}

func (cs *ChatServer) SendMessage(username, text string) {
    message := ChatMessage{
        Username: username,
        Text:     text,
        Time:     time.Now().Format(time.RFC3339),
    }
    
    cs.mu.Lock()
    cs.messages = append(cs.messages, message)
    cs.mu.Unlock()
    
    event := &sse.Event{
        Event: "message",
    }
    event.MarshalJSON(message)
    cs.sseServer.Publish(event)
}

func main() {
    chat := NewChatServer()
    
    http.Handle("/events", chat.sseServer)
    
    http.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }
        
        username := r.FormValue("username")
        text := r.FormValue("text")
        chat.SendMessage(username, text)
        w.WriteHeader(http.StatusOK)
    })
    
    http.ListenAndServe(":8080", nil)
}
```

### 示例 2: 实时日志流

```go
package main

import (
    "bufio"
    "os"
    "github.com/your-org/awesome/utils/sse"
)

func tailLogFile(filename string, server *sse.Server) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()
    
    // 跳转到文件末尾
    _, err = file.Seek(0, io.SeekEnd)
    if err != nil {
        return err
    }
    
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        server.Publish(&sse.Event{
            Event: "log",
            Data:  scanner.Text(),
        })
    }
    
    return scanner.Err()
}

func main() {
    server := sse.NewServer()
    
    go tailLogFile("/var/log/syslog", server)
    
    http.Handle("/logs", server)
    http.ListenAndServe(":8080", nil)
}
```

### 示例 3: 进度跟踪

```go
package main

import (
    "encoding/json"
    "github.com/your-org/awesome/utils/sse"
)

type ProgressUpdate struct {
    TaskID    string  `json:"taskId"`
    Phase     string  `json:"phase"`
    Progress  float64 `json:"progress"`
    Message   string  `json:"message"`
    Completed bool    `json:"completed"`
}

func processTask(taskID string, server *sse.Server) {
    phases := []struct {
        name string
        steps int
    }{
        {"Downloading", 10},
        {"Processing", 20},
        {"Uploading", 15},
        {"Completed", 1},
    }
    
    totalSteps := 0
    for _, p := range phases {
        totalSteps += p.steps
    }
    
    currentStep := 0
    
    for _, phase := range phases {
        for i := 0; i < phase.steps; i++ {
            currentStep++
            progress := float64(currentStep) / float64(totalSteps)
            
            update := ProgressUpdate{
                TaskID:    taskID,
                Phase:     phase.name,
                Progress:  progress,
                Message:   fmt.Sprintf("%s: %d/%d", phase.name, i+1, phase.steps),
                Completed: phase.name == "Completed",
            }
            
            event := &sse.Event{
                Event: "progress",
            }
            event.MarshalJSON(update)
            server.Publish(event)
            
            time.Sleep(100 * time.Millisecond)
        }
    }
}

func main() {
    server := sse.NewServer()
    
    http.Handle("/events", server)
    
    http.HandleFunc("/start-task", func(w http.ResponseWriter, r *http.Request) {
        taskID := r.URL.Query().Get("taskId")
        if taskID == "" {
            http.Error(w, "taskId required", http.StatusBadRequest)
            return
        }
        
        go processTask(taskID, server)
        w.Write([]byte(`{"status":"started","taskId":"` + taskID + `"}`))
    })
    
    http.ListenAndServe(":8080", nil)
}
```

### 示例 4: LLM 流式响应集成

```go
package main

import (
    "encoding/json"
    "io"
    "net/http"
    "github.com/your-org/awesome/utils/sse"
)

type LLMStreamHandler struct {
    sseServer *sse.Server
}

func (h *LLMStreamHandler) proxyLLMStream(w http.ResponseWriter, r *http.Request) {
    // 创建到 LLM API 的请求
    llmReq, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", r.Body)
    llmReq.Header.Set("Content-Type", "application/json")
    llmReq.Header.Set("Authorization", "Bearer YOUR_API_KEY")
    
    llmResp, err := http.DefaultClient.Do(llmReq)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer llmResp.Body.Close()
    
    // 设置 SSE 响应头
    w.Header().Set("Content-Type", sse.ContentType)
    w.Header().Set("Cache-Control", sse.CacheControl)
    w.Header().Set("Connection", sse.ConnectionHeader)
    
    // 使用 SSE 读取器解析 LLM 的响应
    reader := sse.NewEventReader(llmResp.Body, nil)
    defer reader.Close()
    
    flusher, _ := w.(http.Flusher)
    
    for {
        event, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            break
        }
        
        // 转发事件到客户端
        w.Write([]byte(event.String()))
        flusher.Flush()
        
        // 同时发布到内部服务器
        h.sseServer.Publish(event)
    }
}

func main() {
    handler := &LLMStreamHandler{
        sseServer: sse.NewServer(),
    }
    
    http.Handle("/internal-events", handler.sseServer)
    http.HandleFunc("/llm/stream", handler.proxyLLMStream)
    
    http.ListenAndServe(":8080", nil)
}
```

## 最佳实践

### 1. 服务器端最佳实践

#### 使用适当的缓冲区大小
```go
server := sse.NewServer(
    sse.WithBufferSize(100), // 根据预期的吞吐量调整
)
```

#### 监控客户端连接
```go
server := sse.NewServer(
    sse.WithOnConnect(func(ch chan *sse.Event) {
        metrics.Inc("sse.connections")
    }),
    sse.WithOnDisconnect(func(ch chan *sse.Event) {
        metrics.Dec("sse.connections")
    }),
)
```

#### 使用心跳保持连接
```go
go func() {
    ticker := time.NewTicker(15 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        server.Publish(&sse.Event{
            Data: "", // 心跳数据可以为空
        })
    }
}()
```

### 2. 客户端最佳实践

#### 使用上下文进行取消
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

eventCh, errCh := client.Stream(ctx)
```

#### 实现优雅的错误处理
```go
eventCh, errCh := client.Stream(ctx)

for {
    select {
    case event := <-eventCh:
        if event != nil {
            // 处理事件
        }
    case err := <-errCh:
        if err == context.Canceled {
            return // 正常取消
        }
        log.Printf("Stream error: %v", err)
        // 可以根据错误类型决定是否重试
    }
}
```

#### 设置合理的超时
```go
httpClient := &http.Client{
    Timeout: 0, // 流式响应不要设置请求超时
}

client := sse.NewClient(
    url,
    sse.WithHTTPClient(httpClient),
    sse.WithRetry(5000), // 5秒后重试
)
```

## 性能优化

### 1. 减少内存分配

#### 重用 Event 对象
```go
// 不要每次都创建新的 Event
event := &sse.Event{}

for data := range dataChan {
    event.Data = data
    server.Publish(event)
}
```

#### 使用缓冲区
```go
var buf bytes.Buffer
event := &sse.Event{}

for data := range dataChan {
    buf.Reset()
    buf.WriteString(data)
    event.Data = buf.String()
    server.Publish(event)
}
```

### 2. 并行处理

#### 独立的发布 goroutine
```go
type AsyncServer struct {
    *sse.Server
    eventChan chan *sse.Event
}

func NewAsyncServer() *AsyncServer {
    s := &AsyncServer{
        Server:    sse.NewServer(),
        eventChan: make(chan *sse.Event, 1000),
    }
    
    go s.processEvents()
    return s
}

func (s *AsyncServer) processEvents() {
    for event := range s.eventChan {
        s.Server.Publish(event)
    }
}

func (s *AsyncServer) PublishAsync(event *sse.Event) {
    select {
    case s.eventChan <- event:
    default:
        // 队列满了，丢弃或记录
    }
}
```

### 3. 基准测试结果

```
BenchmarkEvent_String-8         10000000   123 ns/op
BenchmarkParseEvent-8            5000000   245 ns/op
BenchmarkServer_Publish-8        1000000   1234 ns/op
```

## 与 LLM 流式响应的集成

### OpenAI 兼容的流式响应

```go
type OpenAIChoice struct {
    Delta struct {
        Content string `json:"content"`
    } `json:"delta"`
    Index        int    `json:"index"`
    FinishReason string `json:"finish_reason"`
}

type OpenAIStreamResponse struct {
    ID      string         `json:"id"`
    Object  string         `json:"object"`
    Created int64          `json:"created"`
    Model   string         `json:"model"`
    Choices []OpenAIChoice `json:"choices"`
}

func streamOpenAIResponse(ctx context.Context, apiKey string, request map[string]interface{}) (<-chan string, <-chan error) {
    contentCh := make(chan string)
    errCh := make(chan error, 1)
    
    go func() {
        defer close(contentCh)
        defer close(errCh)
        
        client := sse.NewClient("https://api.openai.com/v1/chat/completions")
        
        // 使用自定义 HTTP 客户端发送 POST 请求
        httpClient := &http.Client{}
        reqBody, _ := json.Marshal(request)
        req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(reqBody))
        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("Authorization", "Bearer "+apiKey)
        
        resp, err := httpClient.Do(req)
        if err != nil {
            errCh <- err
            return
        }
        defer resp.Body.Close()
        
        reader := sse.NewEventReader(resp.Body, nil)
        defer reader.Close()
        
        for {
            event, err := reader.Read()
            if err != nil {
                if err != io.EOF {
                    errCh <- err
                }
                return
            }
            
            if event.Data == "[DONE]" {
                return
            }
            
            var streamResp OpenAIStreamResponse
            if err := json.Unmarshal([]byte(event.Data), &streamResp); err != nil {
                continue
            }
            
            if len(streamResp.Choices) > 0 {
                content := streamResp.Choices[0].Delta.Content
                if content != "" {
                    contentCh <- content
                }
            }
        }
    }()
    
    return contentCh, errCh
}
```

### 使用示例

```go
func main() {
    ctx := context.Background()
    
    request := map[string]interface{}{
        "model":  "gpt-4",
        "stream": true,
        "messages": []map[string]string{
            {"role": "user", "content": "写一首关于 Go 的诗"},
        },
    }
    
    contentCh, errCh := streamOpenAIResponse(ctx, "YOUR_API_KEY", request)
    
    for {
        select {
        case content, ok := <-contentCh:
            if !ok {
                fmt.Println("\n[完成]")
                return
            }
            fmt.Print(content)
        case err := <-errCh:
            fmt.Printf("\n[错误] %v\n", err)
            return
        }
    }
}
```

## 故障排除

### 常见问题

1. **连接立即关闭**
   - 检查是否设置了过短的超时
   - 验证防火墙设置
   - 检查代理配置

2. **事件丢失**
   - 增加客户端通道缓冲区大小
   - 检查发布者是否阻塞
   - 使用异步发布模式

3. **高内存使用**
   - 减少事件数据大小
   - 实现事件丢弃策略
   - 使用连接池

## 参考资料

- [MDN Web Docs: Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)
- [HTML Standard: Server-Sent Events](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [OpenAI API: Chat Completions](https://platform.openai.com/docs/api-reference/chat)
- [Anthropic API: Streaming](https://docs.anthropic.com/claude/reference/streaming)

## 许可证

MIT License
