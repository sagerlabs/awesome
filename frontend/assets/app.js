// TFT Copilot 浮窗前端逻辑。
// 经典 script 加载（非 module），顶层函数即全局，供 HTML 内联 onclick 使用。

// ── 运行环境检测 ──────────────────────────────────────────────────
// Wails 环境：window.go 存在，通过 Go binding 直接调用
// 浏览器/Electron 环境：通过 SSE HTTP 接口
const IS_WAILS = typeof window !== 'undefined' && !!window.go
if (IS_WAILS) document.body.classList.add('wails')

// ── 配置加载 ──────────────────────────────────────────────────────
// Electron 环境：通过 IPC 读取用户数据目录的 config.json（可持久修改）
// 浏览器环境：fetch ./config.json，或使用 localStorage 缓存
let API_BASE   = 'http://localhost:8080'
let STREAM_URL = `${API_BASE}/v1/tft/nlu/stream`

function applyConfig(cfg) {
    if (cfg && cfg.api_base) {
        API_BASE   = cfg.api_base
        STREAM_URL = `${API_BASE}/v1/tft/nlu/stream`
    }
}

async function loadConfig() {
    if (window.electronAPI) {
        // Electron：从用户数据目录读（main.js 负责初始化和持久化）
        const cfg = await window.electronAPI.getConfig()
        applyConfig(cfg)
    } else {
        // 浏览器：优先 localStorage，其次 fetch config.json
        const saved = localStorage.getItem('tft_api_base')
        if (saved) { applyConfig({ api_base: saved }); return }
        try {
            const res = await fetch('./config.json')
            if (res.ok) applyConfig(await res.json())
        } catch (e) {
            console.log('config.json 未找到，使用默认地址')
        }
    }
}
loadConfig()

// ── 会话 ID ───────────────────────────────────────────────────────
// 后端的教练反馈闭环（识别"不对/继续追问/补充局面"）按 session 隔离；
// 不带 session_id 的请求是无状态的，多轮追问会失去上一轮上下文。
// 这里每个标签页一个会话（sessionStorage），换服务器时重置。
let SESSION_ID = ''

function newSessionID() {
    if (window.crypto && crypto.randomUUID) return crypto.randomUUID()
    return 'sess-' + Date.now() + '-' + Math.random().toString(36).slice(2, 10)
}

function initSession(reset) {
    try {
        if (!reset) {
            const saved = sessionStorage.getItem('tft_session_id')
            if (saved) { SESSION_ID = saved; return }
        }
        SESSION_ID = newSessionID()
        sessionStorage.setItem('tft_session_id', SESSION_ID)
    } catch (e) {
        // sessionStorage 不可用（隐私模式等）：退化为内存里的本页会话
        SESSION_ID = SESSION_ID || newSessionID()
    }
}
initSession(false)

// ── DOM 引用 ──────────────────────────────────────────────────────
const bubble   = document.getElementById('bubble')
const panel    = document.getElementById('panel')
const messages = document.getElementById('messages')
const input    = document.getElementById('chat-input')
const sendBtn  = document.getElementById('send-btn')
const closeBtn = document.getElementById('close-btn')
const badge    = document.getElementById('badge')
const welcomeMsg = document.getElementById('welcome-msg')

welcomeMsg.innerHTML = renderMd(
    "我是你的 **TFT Copilot**（云顶教练浮窗）。\n\n" +
    "你可以直接问阵容、装备、羁绊、几费卡强度和打工过渡。\n\n" +
    "> 例如：`剑魔打工强吗？`、`四费卡谁能C？`"
)

// ── 面板开关 ──────────────────────────────────────────────────────
let panelOpen = IS_WAILS
if (IS_WAILS) {
    panel.classList.add('open')
    bubble.classList.add('panel-open')
    closeBtn.textContent = '—'
    closeBtn.title = '最小化'
}

function togglePanel() {
    if (IS_WAILS) return
    panelOpen = !panelOpen
    panel.classList.toggle('open', panelOpen)
    bubble.classList.toggle('panel-open', panelOpen)
    badge.classList.remove('show')
    if (panelOpen) {
        positionPanel()
        setTimeout(() => input.focus(), 300)
    }
}

function positionPanel() {
    // 根据气泡当前位置计算面板应该出现在哪里
    const bRect  = bubble.getBoundingClientRect()
    const pw     = parseInt(getComputedStyle(document.documentElement).getPropertyValue('--panel-w'))
    const ph     = parseInt(getComputedStyle(document.documentElement).getPropertyValue('--panel-h'))
    const margin = 12
    const vw     = window.innerWidth
    const vh     = window.innerHeight

    // 水平：优先显示在气泡左侧，不够则右侧
    let left = bRect.left - pw - margin
    if (left < margin) left = bRect.right + margin
    if (left + pw > vw - margin) left = vw - pw - margin

    // 垂直：从气泡上方往上展开，不够则往下
    let top = bRect.bottom - ph
    if (top < margin) top = bRect.top
    if (top + ph > vh - margin) top = vh - ph - margin

    panel.style.left   = `${Math.max(margin, left)}px`
    panel.style.top    = `${Math.max(margin, top)}px`
    panel.style.right  = 'auto'
    panel.style.bottom = 'auto'
}

bubble.addEventListener('click', (e) => {
    if (!isDragged) togglePanel()
})
closeBtn.addEventListener('click', togglePanel)
closeBtn.addEventListener('click', async (e) => {
    if (!IS_WAILS) return
    e.stopPropagation()
    if (window.go?.main?.App?.Minimize) await window.go.main.App.Minimize()
})

// ── 气泡拖拽 ──────────────────────────────────────────────────────
let isDragging = false
let isDragged  = false   // 区分点击和拖拽
let dragOffX = 0, dragOffY = 0

bubble.addEventListener('mousedown', (e) => {
    isDragging = true
    isDragged  = false
    dragOffX = e.clientX - bubble.getBoundingClientRect().left
    dragOffY = e.clientY - bubble.getBoundingClientRect().top
    bubble.classList.add('dragging')
    e.preventDefault()
})

document.addEventListener('mousemove', (e) => {
    if (!isDragging) return
    isDragged = true

    const x = e.clientX - dragOffX
    const y = e.clientY - dragOffY
    const size = parseInt(getComputedStyle(document.documentElement).getPropertyValue('--bubble-size'))

    const clampedX = Math.max(0, Math.min(window.innerWidth  - size, x))
    const clampedY = Math.max(0, Math.min(window.innerHeight - size, y))

    bubble.style.left   = clampedX + 'px'
    bubble.style.top    = clampedY + 'px'
    bubble.style.right  = 'auto'
    bubble.style.bottom = 'auto'

    if (panelOpen) positionPanel()
})

document.addEventListener('mouseup', () => {
    if (isDragging) {
        isDragging = false
        bubble.classList.remove('dragging')
        snapToEdge()
    }
})

// 松手后吸附到最近的边
function snapToEdge() {
    const bRect = bubble.getBoundingClientRect()
    const size  = parseInt(getComputedStyle(document.documentElement).getPropertyValue('--bubble-size'))
    const cx    = bRect.left + size / 2
    const margin = 16

    // 吸附到左边或右边
    const snapLeft = cx < window.innerWidth / 2
    const targetX  = snapLeft ? margin : window.innerWidth - size - margin

    bubble.style.transition = 'left 0.3s cubic-bezier(0.34,1.56,0.64,1), top 0.3s ease'
    bubble.style.left = targetX + 'px'

    setTimeout(() => {
        bubble.style.transition = ''
        if (panelOpen) positionPanel()
    }, 350)
}

// 触摸支持
bubble.addEventListener('touchstart', (e) => {
    const t = e.touches[0]
    isDragging = true; isDragged = false
    dragOffX = t.clientX - bubble.getBoundingClientRect().left
    dragOffY = t.clientY - bubble.getBoundingClientRect().top
    bubble.classList.add('dragging')
}, { passive: true })

document.addEventListener('touchmove', (e) => {
    if (!isDragging) return
    isDragged = true
    const t = e.touches[0]
    const size = parseInt(getComputedStyle(document.documentElement).getPropertyValue('--bubble-size'))
    const x = Math.max(0, Math.min(window.innerWidth  - size, t.clientX - dragOffX))
    const y = Math.max(0, Math.min(window.innerHeight - size, t.clientY - dragOffY))
    bubble.style.left = x + 'px'; bubble.style.top = y + 'px'
    bubble.style.right = 'auto';  bubble.style.bottom = 'auto'
    if (panelOpen) positionPanel()
}, { passive: true })

document.addEventListener('touchend', () => {
    if (isDragging) { isDragging = false; bubble.classList.remove('dragging'); snapToEdge() }
})

// ── 消息渲染 ──────────────────────────────────────────────────────
function addMessage(role, text) {
    const div = document.createElement('div')
    div.className = `msg ${role}`
    div.innerHTML = `
    <div class="msg-avatar">${role === 'ai' ? '⚔' : '你'}</div>
    <div class="msg-bubble">${escHtml(text)}</div>
  `
    messages.appendChild(div)
    scrollToBottom()
    return div.querySelector('.msg-bubble')
}

// 打字指示器
function showTyping() {
    const div = document.createElement('div')
    div.className = 'msg ai'; div.id = 'typing'
    div.innerHTML = `
    <div class="msg-avatar">⚔</div>
    <div class="msg-bubble">
      <div class="typing-indicator">
        <div class="typing-dot"></div>
        <div class="typing-dot"></div>
        <div class="typing-dot"></div>
      </div>
    </div>
  `
    messages.appendChild(div)
    scrollToBottom()
    return div
}

function hideTyping() {
    const el = document.getElementById('typing')
    if (el) el.remove()
}

function scrollToBottom() {
    messages.scrollTop = messages.scrollHeight
}

function escHtml(str) {
    return str.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
}

// ── SSE 行解析 ────────────────────────────────────────────────────
// 只把 JSON.parse 包进 try/catch：解析失败跳过该行即可，
// 但解析成功后的业务错误（type=error）必须抛给上层展示，不能被吞掉。
function parseSSEChunk(line) {
    if (!line.startsWith('data:')) return null
    const raw = line.slice(5).trim()
    if (!raw) return null
    try {
        return JSON.parse(raw)
    } catch (e) {
        return null // 非 JSON 行（注释/心跳），跳过
    }
}

// ── 发送消息 ──────────────────────────────────────────────────────
let isStreaming = false

async function sendMessage() {
    const text = input.value.trim()
    if (!text || isStreaming) return

    // 显示用户消息
    addMessage('user', text)
    input.value = ''
    autoResize()

    // 隐藏快捷标签（首次发送后）
    document.getElementById('suggestions').style.display = 'none'

    isStreaming = true
    sendBtn.disabled = true
    sendBtn.textContent = '⏸'

    // 显示打字动画
    const typingEl = showTyping()

    // 创建 AI 消息气泡（先隐藏）
    let aiBubble = null
    let fullText = ''

    try {
        if (IS_WAILS) {
            // ── Wails 模式：直接调 Go binding，一次性返回 ──────────────
            // Wails 暂不支持流式，Go 端 Analyze 返回完整结果
            typingEl.remove()
            aiBubble = addMessage('ai', '')

            const result = await window.go.main.App.Analyze(text)
            fullText = result
            aiBubble.classList.add('md')
            aiBubble.innerHTML = renderMd(fullText.replace(/\n\s*\n/g, '\n\n'))
        } else {
            // ── HTTP SSE 模式：流式逐 chunk 显示 ────────────────────────
            const response = await fetch(STREAM_URL, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ input: text, session_id: SESSION_ID }),
                signal: AbortSignal.timeout(90000),
            })

            if (!response.ok) throw new Error(`HTTP ${response.status}`)

            const reader  = response.body.getReader()
            const decoder = new TextDecoder()
            let   buffer  = ''

            let streamDone = false
            while (!streamDone) {
                const { done, value } = await reader.read()
                if (done) break

                buffer += decoder.decode(value, { stream: true })
                const lines = buffer.split('\n')
                buffer = lines.pop()

                for (const line of lines) {
                    const chunk = parseSSEChunk(line)
                    if (!chunk) continue

                    if (chunk.type === 'token' && chunk.content) {
                        if (!aiBubble) { typingEl.remove(); aiBubble = addMessage('ai', '') }
                        fullText += chunk.content
                        aiBubble.textContent = fullText
                        scrollToBottom()
                    } else if (chunk.type === 'done') {
                        if (aiBubble && fullText) {
                            aiBubble.classList.add('md')
                            aiBubble.innerHTML = renderMd(fullText)
                            scrollToBottom()
                        }
                        streamDone = true  // 退出 while
                        break
                    } else if (chunk.type === 'error') {
                        throw new Error(chunk.error || '推理出错')
                    }
                }
            }
        }

        // 如果没收到任何内容
        if (!aiBubble) {
            hideTyping()
            addMessage('ai', '暂时没有找到合适的推荐，请尝试提供更多英雄或装备信息。')
        }

    } catch (err) {
        hideTyping()
        if (!aiBubble) {
            const errBubble = addMessage('ai', '')
            errBubble.classList.add('md')
            errBubble.innerHTML = renderMd(`**连接失败**：${err.message}\n\n请检查服务器是否启动：\`${API_BASE}\``)
        }
    } finally {
        isStreaming = false
        sendBtn.disabled = false
        sendBtn.textContent = '➤'
        // 未开着面板时显示角标
        if (!panelOpen) badge.classList.add('show')
    }
}

// ── 快捷标签填充 ──────────────────────────────────────────────────
function fillInput(text) {
    input.value = text
    input.focus()
    autoResize()
}

// Markdown 渲染（marked 由 index.html 头部加载，CDN 失败时有降级实现）
function renderMd(text) {
    const cleaned = text
        .replace(/\n{3,}/g, '\n\n')   // 3个以上空行压缩成2个
        .replace(/^\s+|\s+$/g, '')     // 去首尾空白
    return marked.parse(cleaned)
}

// ── 输入框自动高度 ────────────────────────────────────────────────
function autoResize() {
    input.style.height = 'auto'
    input.style.height = Math.min(input.scrollHeight, 100) + 'px'
}
input.addEventListener('input', autoResize)

// Enter 发送，Shift+Enter 换行
input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        sendMessage()
    }
})

// ── 设置弹窗 ──────────────────────────────────────────────────────
const settingsBtn   = document.getElementById('settings-btn')
const settingsModal = document.getElementById('settings-modal')
const apiInput      = document.getElementById('api-input')
const settingsSave  = document.getElementById('settings-save')

settingsBtn.addEventListener('click', (e) => {
    e.stopPropagation()
    const isOpen = settingsModal.classList.toggle('show')
    if (isOpen) apiInput.value = API_BASE
})

// 点击面板外关闭设置弹窗
document.addEventListener('click', (e) => {
    if (!settingsModal.contains(e.target) && e.target !== settingsBtn) {
        settingsModal.classList.remove('show')
    }
})

settingsSave.addEventListener('click', async () => {
    const val = apiInput.value.trim().replace(/\/+$/, '')
    if (!val) return

    applyConfig({ api_base: val })

    if (window.electronAPI) {
        // Electron：持久化到用户数据目录的 config.json
        const result = await window.electronAPI.saveConfig({ api_base: val })
        console.log('配置已保存到:', result.path)
    } else {
        // 浏览器：存 localStorage
        localStorage.setItem('tft_api_base', val)
    }

    settingsModal.classList.remove('show')

    // 清空对话并开新会话：换了后端，旧的多轮上下文不再有意义
    initSession(true)
    messages.innerHTML = ''
    const b = addMessage('ai', '')
    b.classList.add('md')
    b.innerHTML = renderMd('✅ 已连接到 `' + val + '`')
    document.getElementById('suggestions').style.display = 'flex'
})

sendBtn.addEventListener('click', sendMessage)

// ── Electron 鼠标穿透控制 ─────────────────────────────────────────
// 浏览器直接打开时 window.electronAPI 不存在，降级为空函数
const electron = window.electronAPI || { setInteractive: ()=>{}, setPassthrough: ()=>{} }

// 鼠标进入气泡/面板区域：停止穿透，让 Electron 窗口响应点击
bubble.addEventListener('mouseenter', () => electron.setInteractive())
panel.addEventListener('mouseenter',  () => electron.setInteractive())

// 鼠标离开所有交互区域：恢复穿透，点击事件透传到底层窗口
bubble.addEventListener('mouseleave', () => {
    if (!panelOpen) electron.setPassthrough()
})
panel.addEventListener('mouseleave', () => {
    if (!panelOpen) electron.setPassthrough()
})

// 面板关闭时恢复穿透
const _origToggle = togglePanel
window.togglePanel = function() {
    _origToggle()
    if (!panelOpen) electron.setPassthrough()
    else electron.setInteractive()
}
