// Package main demonstrates a ChatGPT-style SSE interaction.
// It simulates an AI chat interface where the server streams the response token by token.
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// ChatRequest represents the user's message
type ChatRequest struct {
	Message string `json:"message"`
}

// ChatResponse represents a chunk of the AI's response
type ChatResponse struct {
	Content string `json:"content"`
	Done    bool   `json:"done"`
}

func main() {
	// Serve static files (HTML/JS/CSS)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		serveChatInterface(w)
	})

	// Handle chat completions (SSE endpoint)
	http.HandleFunc("/api/chat", handleChatCompletion)

	fmt.Println("ChatGPT-style Server starting on :8081...")
	fmt.Println("Visit http://localhost:8081 to start chatting")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}

func handleChatCompletion(w http.ResponseWriter, r *http.Request) {
	// Ensure it's a POST request
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body (in a real app, you'd read the JSON body)
	// For SSE, browsers often use EventSource which only supports GET.
	// However, modern AI chat apps use fetch() with a custom reader to handle POST SSE.
	// Here, to keep it simple and compatible with standard EventSource, we'll use a query param
	// or just simulate a response regardless of input for this demo.

	// NOTE: Standard EventSource API does NOT support POST.
	// ChatGPT uses fetch() with a ReadableStream, not EventSource.
	// But to strictly follow "SSE architecture" requested, we can use GET with query params,
	// OR we can implement the fetch-based stream reader in the frontend.
	// Let's implement the modern fetch-based approach as it's what ChatGPT actually uses.

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Simulate AI "thinking"
	time.Sleep(time.Duration(500+rand.Intn(1000)) * time.Millisecond)

	// The mock response text
	responseText := "这是一个模拟的 AI 回答。我正在使用 SSE (Server-Sent Events) 技术流式传输我的思考过程。\n\n" +
		"与传统的请求-响应模式不同，你不需要等待我完全生成完所有内容才能看到第一个字。\n" +
		"这提供了更好的用户体验，让你感觉我在实时与你对话。\n\n" +
		"技术细节：\n" +
		"1. 前端使用 fetch() 发起 POST 请求\n" +
		"2. 后端设置 Content-Type: text/event-stream\n" +
		"3. 后端分块写入数据并 Flush\n" +
		"4. 前端使用 ReadableStream 读取数据块"

	// Stream the response token by token
	runes := []rune(responseText)
	for _, char := range runes {
		select {
		case <-r.Context().Done():
			// Client disconnected
			return
		default:
			// Construct the data chunk
			// In OpenAI's format, it's usually: data: {"choices": [{"delta": {"content": "..."}}]}
			// We'll use a simplified JSON format.
			chunk := ChatResponse{
				Content: string(char),
				Done:    false,
			}

			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

			// Simulate typing delay (randomized for realism)
			time.Sleep(time.Duration(20+rand.Intn(50)) * time.Millisecond)
		}
	}

	// Send the [DONE] signal
	endChunk := ChatResponse{Done: true}
	data, _ := json.Marshal(endChunk)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func serveChatInterface(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<head>
    <title>ChatGPT-style SSE Demo</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; background: #f7f7f8; }
        .chat-container { background: white; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); overflow: hidden; display: flex; flex-direction: column; height: 80vh; }
        .messages { flex: 1; overflow-y: auto; padding: 20px; }
        .message { margin-bottom: 20px; padding: 10px 15px; border-radius: 8px; max-width: 80%; line-height: 1.5; }
        .message.user { background: #e3f2fd; align-self: flex-end; margin-left: auto; }
        .message.ai { background: #f0f0f0; align-self: flex-start; margin-right: auto; white-space: pre-wrap; }
        .input-area { padding: 20px; border-top: 1px solid #eee; display: flex; gap: 10px; background: white; }
        input { flex: 1; padding: 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 16px; }
        button { padding: 12px 24px; background: #10a37f; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 16px; font-weight: 500; }
        button:disabled { background: #ccc; cursor: not-allowed; }
        .cursor { display: inline-block; width: 6px; height: 16px; background: #333; animation: blink 1s infinite; vertical-align: middle; }
        @keyframes blink { 50% { opacity: 0; } }
    </style>
</head>
<body>
    <div class="chat-container">
        <div class="messages" id="messages">
            <div class="message ai">你好！我是模拟的 AI 助手。请问有什么我可以帮你的吗？</div>
        </div>
        <div class="input-area">
            <input type="text" id="userInput" placeholder="输入你的问题..." onkeypress="if(event.key === 'Enter') sendMessage()">
            <button onclick="sendMessage()" id="sendBtn">发送</button>
        </div>
    </div>

    <script>
        const messagesDiv = document.getElementById('messages');
        const userInput = document.getElementById('userInput');
        const sendBtn = document.getElementById('sendBtn');

        async function sendMessage() {
            const text = userInput.value.trim();
            if (!text) return;

            // Add user message
            appendMessage('user', text);
            userInput.value = '';
            userInput.disabled = true;
            sendBtn.disabled = true;

            // Create AI message container
            const aiMessageDiv = appendMessage('ai', '');
            const cursor = document.createElement('span');
            cursor.className = 'cursor';
            aiMessageDiv.appendChild(cursor);

            try {
                // Use fetch for POST request with streaming response
                const response = await fetch('/api/chat', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ message: text })
                });

                if (!response.ok) throw new Error(response.statusText);

                // Read the stream
                const reader = response.body.getReader();
                const decoder = new TextDecoder();
                let buffer = '';

                while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;

                    const chunk = decoder.decode(value, { stream: true });
                    buffer += chunk;
                    
                    // Process SSE format (data: ...)
                    const lines = buffer.split('\n\n');
                    buffer = lines.pop(); // Keep the last incomplete chunk

                    for (const line of lines) {
                        if (line.startsWith('data: ')) {
                            const jsonStr = line.slice(6);
                            try {
                                const data = JSON.parse(jsonStr);
                                if (data.done) {
                                    cursor.remove();
                                    return;
                                }
                                if (data.content) {
                                    // Insert text before cursor
                                    aiMessageDiv.insertBefore(document.createTextNode(data.content), cursor);
                                    messagesDiv.scrollTop = messagesDiv.scrollHeight;
                                }
                            } catch (e) {
                                console.error('Error parsing JSON:', e);
                            }
                        }
                    }
                }
            } catch (err) {
                aiMessageDiv.textContent += '\n[Error: ' + err.message + ']';
                cursor.remove();
            } finally {
                userInput.disabled = false;
                sendBtn.disabled = false;
                userInput.focus();
            }
        }

        function appendMessage(role, text) {
            const div = document.createElement('div');
            div.className = 'message ' + role;
            div.textContent = text;
            messagesDiv.appendChild(div);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
            return div;
        }
    </script>
</body>
</html>
`)
}
