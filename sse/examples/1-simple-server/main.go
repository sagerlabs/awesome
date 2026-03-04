// Package main is a simple SSE server example that demonstrates a typewriter effect.
package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	// 模拟一段长文本，用于演示打字机效果
	const longText = `你好！这是一个 SSE (Server-Sent Events) 的打字机效果演示。
SSE 是一种允许服务器向客户端推送数据的技术。
与 WebSocket 不同，SSE 是单向的（服务器 -> 客户端），基于标准的 HTTP 协议。
它非常适合用于：
1. 实时日志流
2. 股票行情更新
3. AI 生成内容的逐字输出（就像我现在这样！）
...
演示结束。`

	// 处理打字机效果的 SSE 路由
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		// 设置 SSE 必须的 HTTP 头
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// 发送连接成功消息
		fmt.Fprintf(w, "data: [START]\n\n")
		flusher.Flush()

		// 模拟逐字输出
		runes := []rune(longText)
		for _, char := range runes {
			// 检查客户端是否断开连接
			select {
			case <-r.Context().Done():
				return
			default:
				// 构造 SSE 消息格式: data: <content>\n\n
				// 注意：实际生产中可能需要处理换行符等转义
				content := string(char)
				if content == "\n" {
					content = "<br>" // 简单处理换行，以便在 HTML 中显示
				}
				
				fmt.Fprintf(w, "data: %s\n\n", content)
				flusher.Flush()
				
				// 模拟打字延迟 (50ms - 150ms 之间会更自然，这里固定 100ms)
				time.Sleep(100 * time.Millisecond)
			}
		}
		
		// 发送结束消息
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	})

	// 前端页面
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<head>
    <title>SSE Typewriter Effect</title>
    <style>
        body {
            font-family: 'Courier New', Courier, monospace;
            background-color: #f4f4f4;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        #content {
            white-space: pre-wrap; /* 保留空白符和换行 */
            min-height: 200px;
        }
        .cursor {
            display: inline-block;
            width: 8px;
            height: 18px;
            background-color: #333;
            animation: blink 1s step-end infinite;
            vertical-align: middle;
            margin-left: 2px;
        }
        @keyframes blink {
            0%, 100% { opacity: 1; }
            50% { opacity: 0; }
        }
        button {
            padding: 10px 20px;
            background-color: #007bff;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 16px;
            margin-bottom: 20px;
        }
        button:hover {
            background-color: #0056b3;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>SSE 打字机效果演示</h1>
        <button onclick="startTypewriter()">重新开始</button>
        <div id="output">
            <span id="content"></span><span class="cursor"></span>
        </div>
    </div>

    <script>
        let eventSource = null;
        const contentDiv = document.getElementById('content');

        function startTypewriter() {
            // 如果已有连接，先关闭
            if (eventSource) {
                eventSource.close();
            }
            
            // 清空内容
            contentDiv.innerHTML = '';
            
            // 建立新连接
            eventSource = new EventSource('/events');
            
            eventSource.onmessage = function(event) {
                if (event.data === '[START]') {
                    console.log('开始接收...');
                    return;
                }
                if (event.data === '[DONE]') {
                    console.log('接收完成');
                    eventSource.close();
                    return;
                }
                
                // 追加内容
                // 注意：这里简单处理了 <br>，实际应用中可能需要更复杂的 Markdown 解析
                if (event.data === '<br>') {
                    contentDiv.innerHTML += '<br>';
                } else {
                    contentDiv.textContent += event.data;
                }
                
                // 自动滚动到底部
                window.scrollTo(0, document.body.scrollHeight);
            };
            
            eventSource.onerror = function(err) {
                console.error('EventSource failed:', err);
                eventSource.close();
            };
        }

        // 页面加载时自动开始
        startTypewriter();
    </script>
</body>
</html>
`)
	})

	fmt.Println("Server starting on :8080...")
	fmt.Println("Visit http://localhost:8080 to see the typewriter effect")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
