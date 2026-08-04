// TFT Copilot 前端逻辑（全页 Web 版）
// 经典 script 加载（非 module），兼容三种运行环境：
//   - 浏览器（Go 内嵌静态资源 / 直接打开）
//   - Electron 覆盖层（window.electronAPI）
//   - Wails 桌面窗（window.go.main.App）
//
// 布局模式（body[data-layout]）：
//   web     — 默认。侧边栏 + 主聊天区（Claude 风格全页）
//   compact — Wails 自动启用，或 ?layout=compact。无侧边栏单列聊天
//   pet     — 【桌宠预留】?layout=pet 或 window.TFTSetLayout('pet')。
//             透明背景 + 紧凑对话；宠物形象层未来挂接到 #main 之前，
//             对话 / SSE / 局面快填逻辑全部复用，无需改动。

'use strict'

// ── 运行环境检测 ──────────────────────────────────────────────────────────
const IS_WAILS = typeof window !== 'undefined' && !!window.go
if (IS_WAILS) document.body.classList.add('wails')

// ── 布局模式 ──────────────────────────────────────────────────────────────
const LAYOUTS = ['web', 'compact', 'pet']
let LAYOUT = 'web'

function setLayout(mode) {
    if (!LAYOUTS.includes(mode)) mode = 'web'
    LAYOUT = mode
    document.body.dataset.layout = mode
}
// 桌宠形态的外部挂接点（未来桌面壳调用）
window.TFTSetLayout = setLayout

function initLayout() {
    const param = new URLSearchParams(location.search).get('layout')
    if (param) { setLayout(param); return }
    // Wails / Electron 覆盖层默认紧凑小窗
    if (IS_WAILS || window.electronAPI) { setLayout('compact'); return }
    setLayout('web')
}
initLayout()

// ── 配置加载 ──────────────────────────────────────────────────────────────
// Electron：经 IPC 读写用户数据目录的 config.json
// 浏览器：localStorage 优先，其次 fetch ./config.json；
//         都没有时自动探测（当前页同源 → localhost:8080）
let API_BASE = 'http://localhost:8080'

const nluStreamURL = () => `${API_BASE}/v1/tft/nlu/stream`
const healthURL    = () => `${API_BASE}/v1/tft/health`

function applyConfig(cfg) {
    if (cfg && cfg.api_base) API_BASE = String(cfg.api_base).replace(/\/+$/, '')
}

// 返回 true 表示读到了用户保存的配置（此时跳过自动探测）
async function loadConfig() {
    if (window.electronAPI) {
        const cfg = await window.electronAPI.getConfig()
        applyConfig(cfg)
        return !!(cfg && cfg.api_base)
    }
    const saved = localStorage.getItem('tft_api_base')
    if (saved) { applyConfig({ api_base: saved }); return true }
    try {
        const res = await fetch('./config.json')
        if (res.ok) {
            const cfg = await res.json()
            applyConfig(cfg)
            return !!(cfg && cfg.api_base)
        }
    } catch (e) {
        console.log('config.json 未找到，使用自动探测')
    }
    return false
}

// 自动探测后端：优先当前页同源（Go 内嵌部署），其次 localhost:8080（本地开发）。
// 503 也算命中：服务存在，只是数据未加载（degraded）。
async function detectBase() {
    const candidates = []
    if (/^https?:$/.test(location.protocol)) candidates.push(location.origin)
    if (!candidates.includes('http://localhost:8080')) candidates.push('http://localhost:8080')
    for (const base of candidates) {
        try {
            const res = await fetch(`${base}/v1/tft/health`, { signal: AbortSignal.timeout(3000) })
            if (res.ok || res.status === 503) { API_BASE = base; return }
        } catch (e) { /* 尝试下一个候选 */ }
    }
    if (candidates.length) API_BASE = candidates[0]
}

// ── 多会话存储 ────────────────────────────────────────────────────────────
// 每个会话一个独立 session_id（后端教练反馈闭环按 session 隔离）。
// 结构：{ id, title, sessionId, created, updated, messages: [{role, content, ts}] }
const STORE_KEY = 'tft_conversations_v1'

let conversations = []
let activeId = null

function uid() {
    if (window.crypto && crypto.randomUUID) return crypto.randomUUID()
    return 'id-' + Date.now() + '-' + Math.random().toString(36).slice(2, 10)
}

function loadConvs() {
    try {
        const raw = localStorage.getItem(STORE_KEY)
        if (raw) {
            const data = JSON.parse(raw)
            if (Array.isArray(data)) conversations = data
        }
    } catch (e) { conversations = [] }
    conversations.sort((a, b) => b.updated - a.updated)
}

function saveConvs() {
    try { localStorage.setItem(STORE_KEY, JSON.stringify(conversations)) } catch (e) { /* 存储满 */ }
}

function getConv(id) { return conversations.find((c) => c.id === id) }

function createConv() {
    const conv = {
        id: uid(),
        title: '新对话',
        sessionId: uid(),
        created: Date.now(),
        updated: Date.now(),
        messages: [],
    }
    conversations.unshift(conv)
    activeId = conv.id
    saveConvs()
    return conv
}

function ensureActiveConv() {
    return getConv(activeId) || createConv()
}

function pushMsg(role, content) {
    const conv = ensureActiveConv()
    conv.messages.push({ role, content, ts: Date.now() })
    conv.updated = Date.now()
    // 首条用户消息自动生成标题
    if (role === 'user' && conv.title === '新对话') {
        conv.title = content.length > 18 ? content.slice(0, 18) + '…' : content
    }
    saveConvs()
    return conv
}

function deleteConv(id) {
    const idx = conversations.findIndex((c) => c.id === id)
    if (idx === -1) return
    conversations.splice(idx, 1)
    if (activeId === id) activeId = conversations.length ? conversations[0].id : null
    saveConvs()
}

// ── 静态文案与数据 ────────────────────────────────────────────────────────
// 能力卡片（对应 README 能力范围；点击填入输入框）
const CAP_CARDS = [
    { icon: '🏆', name: '版本阵容', desc: '当前版本强势阵容与环境解读', q: '当前版本最强的三套阵容是什么？' },
    { icon: '🗡', name: '装备规划', desc: '有羊刀/珠光护手，能玩什么', q: '我有羊刀和珠光护手，可以玩什么？' },
    { icon: '⭐', name: '英雄评估', desc: '几费卡谁能 C、打工强度', q: '四费卡谁能C？' },
    { icon: '🔗', name: '羁绊搭配', desc: '海魔人、未来战士怎么搭', q: '海魔人能玩吗？' },
    { icon: '⚡', name: '局内决策', desc: '告诉我局面，给你老玩家建议', q: '3-2，6级，40血，有剑魔羊刀，能不能冲海魔人？', coach: true },
]

// 局面快填（对应后端 NLU 的 game_stage / level / gold / hp 结构化解析）
const COACH_GROUPS = [
    { name: '阶段', tokens: ['2-1', '2-7', '3-2', '4-1', '4-7', '5-1'] },
    { name: '等级', tokens: ['5级', '6级', '7级', '8级', '9级'] },
    { name: '经济', tokens: ['10金', '20金', '30金', '40金', '50金'] },
    { name: '血量', tokens: ['80血', '60血', '40血', '20血', '个位数'] },
]

// ── DOM 引用 ──────────────────────────────────────────────────────────────
const $ = (id) => document.getElementById(id)
const appEl         = $('app')
const sidebar       = $('sidebar')
const sideMask      = $('side-mask')
const sideToggle    = $('side-toggle')
const topbarTitle   = $('topbar-title')
const topbarDot     = $('topbar-dot')
const minimizeBtn   = $('minimize-btn')
const scrollEl      = $('scroll')
const welcomeEl     = $('welcome')
const capCardsEl    = $('cap-cards')
const welcomeMeta   = $('welcome-meta')
const messages      = $('messages')
const input         = $('chat-input')
const sendBtn       = $('send-btn')
const composer      = $('composer')
const coachToggle   = $('coach-toggle')
const coachGroups   = $('coach-groups')
const newchatBtn    = $('newchat-btn')
const convListEl    = $('conv-list')
const settingsBtn   = $('settings-btn')
const settingsModal = $('settings-modal')
const settingsClose = $('settings-close')
const apiInput      = $('api-input')
const settingsSave  = $('settings-save')
const connText      = $('conn-text')
const connSub       = $('conn-sub')
const statusDot     = $('status-dot')
const healthStatus  = $('health-status')
const healthVersion = $('health-version')
const healthComps   = $('health-comps')

// ── 连接健康检查 ──────────────────────────────────────────────────────────
// 后端 GET /v1/tft/health 返回 {status, version, git_commit, build_time, comp_count}；
// 数据未加载时返回 503 + status=degraded。
let lastHealth = null

function setConn(state, data) {
    const cls = 'status-dot dot-' + (state === 'connecting' ? 'connecting' : state)
    statusDot.className = cls
    if (topbarDot) topbarDot.className = 'status-dot topbar-dot ' + cls.replace('status-dot ', '')

    let main = '连接中…', sub = ''
    switch (state) {
        case 'online':
            main = '已连接'
            sub = data && data.comp_count != null ? `${data.comp_count} 套阵容` : '运行中'
            break
        case 'local':
            main = '本地引擎'; sub = '桌面模式'
            break
        case 'degraded':
            main = '服务降级'; sub = '数据未加载'
            break
        case 'offline':
            main = '未连接'; sub = '点 ⚙ 检查服务器'
            break
    }
    connText.textContent = main
    connSub.textContent = sub
    renderSettingsHealth()
    renderWelcomeMeta(state, data)
}

function renderWelcomeMeta(state, data) {
    if (!welcomeMeta) return
    if (state === 'online' && data) {
        const ver = [data.version, data.git_commit].filter(Boolean).join(' · ')
        welcomeMeta.textContent = `知识库已就绪：${data.comp_count} 套阵容${ver ? ' · ' + ver : ''}`
    } else if (state === 'local') {
        welcomeMeta.textContent = '本地引擎运行中'
    } else if (state === 'offline') {
        welcomeMeta.textContent = `未连接到后端（${API_BASE}），点右下角 ⚙ 检查服务器`
    } else {
        welcomeMeta.textContent = ''
    }
}

function renderSettingsHealth() {
    if (!healthStatus) return
    if (IS_WAILS) {
        healthStatus.textContent = '本地引擎'
        healthStatus.className = 'ok'
        healthVersion.textContent = '—'
        healthComps.textContent = '—'
        return
    }
    if (!lastHealth) {
        healthStatus.textContent = '未连接'
        healthStatus.className = 'bad'
        healthVersion.textContent = '—'
        healthComps.textContent = '—'
        return
    }
    const ok = lastHealth.status === 'ok'
    healthStatus.textContent = ok ? '正常' : '降级'
    healthStatus.className = ok ? 'ok' : 'warn'
    healthVersion.textContent = [lastHealth.version, lastHealth.git_commit].filter(Boolean).join(' · ') || '—'
    healthComps.textContent = lastHealth.comp_count != null ? String(lastHealth.comp_count) : '—'
}

async function checkHealth() {
    if (IS_WAILS) { setConn('local'); return }
    try {
        const res = await fetch(healthURL(), { signal: AbortSignal.timeout(5000) })
        let data = null
        try { data = await res.json() } catch (e) { /* 非 JSON 响应 */ }
        if (data && data.status === 'ok') {
            lastHealth = data
            setConn('online', data)
        } else if (data && data.status === 'degraded') {
            lastHealth = data
            setConn('degraded', data)
        } else {
            lastHealth = null
            setConn('offline')
        }
    } catch (e) {
        lastHealth = null
        setConn('offline')
    }
}

// ── 会话列表渲染与交互 ────────────────────────────────────────────────────
function renderConvList() {
    convListEl.innerHTML = ''
    if (!conversations.length) {
        const empty = document.createElement('div')
        empty.className = 'conv-empty'
        empty.textContent = '暂无历史对话'
        convListEl.appendChild(empty)
        return
    }
    conversations
        .slice()
        .sort((a, b) => b.updated - a.updated)
        .forEach((conv) => {
            const item = document.createElement('button')
            item.type = 'button'
            item.className = 'conv-item' + (conv.id === activeId ? ' active' : '')
            item.dataset.id = conv.id
            item.innerHTML = `
                <span class="conv-title">${escHtml(conv.title)}</span>
                <span class="conv-del" data-del="${conv.id}" title="删除对话">✕</span>`
            convListEl.appendChild(item)
        })
}

convListEl.addEventListener('click', (e) => {
    const del = e.target.closest('[data-del]')
    if (del) {
        e.stopPropagation()
        if (isStreaming && del.dataset.del === activeId) stopStreaming()
        deleteConv(del.dataset.del)
        renderConvList()
        renderActiveConv()
        return
    }
    const item = e.target.closest('.conv-item')
    if (item && item.dataset.id !== activeId) {
        if (isStreaming) stopStreaming()
        activeId = item.dataset.id
        renderConvList()
        renderActiveConv()
        closeSidebarOnMobile()
        if (!IS_WAILS) input.focus()
    }
})

// ── 当前会话渲染 ──────────────────────────────────────────────────────────
function renderActiveConv() {
    const conv = getConv(activeId)
    messages.innerHTML = ''
    stickBottom = true

    if (!conv || !conv.messages.length) {
        welcomeEl.classList.remove('hide')
        topbarTitle.textContent = 'TFT Copilot'
        return
    }
    welcomeEl.classList.add('hide')
    topbarTitle.textContent = conv.title
    conv.messages.forEach((m) => {
        if (m.role === 'ai') addMessageDOM('ai', m.content, { md: true, ts: m.ts })
        else addMessageDOM('user', m.content, { ts: m.ts })
    })
    scrollToBottom(true)
}

function newChat() {
    if (isStreaming) stopStreaming()
    createConv()
    renderConvList()
    renderActiveConv()
    input.value = ''
    autoResize()
    if (!IS_WAILS) input.focus()
}

newchatBtn.addEventListener('click', newChat)

// ── 欢迎页能力卡 ──────────────────────────────────────────────────────────
function buildCapCards() {
    capCardsEl.innerHTML = ''
    CAP_CARDS.forEach((c) => {
        const card = document.createElement('button')
        card.type = 'button'
        card.className = 'cap-card'
        card.innerHTML = `
            <div class="cap-head"><span class="cap-icon">${c.icon}</span><span class="cap-name">${c.name}</span></div>
            <div class="cap-desc">${c.desc}</div>`
        card.addEventListener('click', () => {
            fillInput(c.q)
            if (c.coach && !composer.classList.contains('coach-open')) toggleCoach()
        })
        capCardsEl.appendChild(card)
    })
}

// ── 消息渲染 ──────────────────────────────────────────────────────────────
let stickBottom = true   // 用户上翻浏览历史时，不再强制滚动到底部

scrollEl.addEventListener('scroll', () => {
    stickBottom = scrollEl.scrollHeight - scrollEl.scrollTop - scrollEl.clientHeight < 80
}, { passive: true })

function scrollToBottom(force) {
    if (force || stickBottom) scrollEl.scrollTop = scrollEl.scrollHeight
}

function fmtTime(ts) {
    const d = ts ? new Date(ts) : new Date()
    return String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0')
}

function escHtml(str) {
    return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

// Markdown 渲染（marked 由 index.html 头部加载，CDN 失败时有降级实现）
function renderMd(text) {
    const cleaned = String(text)
        .replace(/\n{3,}/g, '\n\n')
        .replace(/^\s+|\s+$/g, '')
    return marked.parse(cleaned)
}

// 纯 DOM 消息（不写存储）；返回气泡元素
function addMessageDOM(role, text, opts = {}) {
    const div = document.createElement('div')
    div.className = `msg ${role}`

    const copyBtn = role === 'ai' ? '<button class="copy-btn" title="复制回答">复制</button>' : ''
    div.innerHTML = `
        <div class="msg-avatar">${role === 'ai' ? '⚔' : '你'}</div>
        <div class="msg-body">
            <div class="msg-bubble"></div>
            <div class="msg-meta"><span class="msg-time">${fmtTime(opts.ts)}</span>${copyBtn}</div>
        </div>`

    const bubbleEl = div.querySelector('.msg-bubble')
    if (opts.md) {
        bubbleEl.classList.add('md')
        bubbleEl.innerHTML = renderMd(text)
    } else {
        bubbleEl.textContent = text
    }

    messages.appendChild(div)
    scrollToBottom(role === 'user')
    return bubbleEl
}

// 复制 / 重试按钮（事件委托）
messages.addEventListener('click', (e) => {
    if (e.target.classList.contains('copy-btn')) {
        const body = e.target.closest('.msg-body')
        copyText(body.querySelector('.msg-bubble').innerText, e.target)
    } else if (e.target.classList.contains('retry-btn')) {
        const q = e.target.dataset.q
        if (q && !isStreaming) sendMessage(q)
    }
})

function copyText(text, btn) {
    const done = () => { btn.textContent = '已复制'; setTimeout(() => { btn.textContent = '复制' }, 1200) }
    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(done).catch(() => {})
    } else {
        const ta = document.createElement('textarea')
        ta.value = text
        ta.style.position = 'fixed'
        ta.style.opacity = '0'
        document.body.appendChild(ta)
        ta.select()
        try { document.execCommand('copy'); done() } catch (e) { /* 忽略 */ }
        ta.remove()
    }
}

// 打字指示器
function showTyping() {
    const div = document.createElement('div')
    div.className = 'msg ai'
    div.id = 'typing'
    div.innerHTML = `
        <div class="msg-avatar">⚔</div>
        <div class="msg-body">
            <div class="msg-bubble">
                <div class="typing-indicator">
                    <div class="typing-dot"></div>
                    <div class="typing-dot"></div>
                    <div class="typing-dot"></div>
                </div>
            </div>
        </div>`
    messages.appendChild(div)
    scrollToBottom(true)
    return div
}

function hideTyping() {
    const el = $('typing')
    if (el) el.remove()
}

// ── SSE 行解析 ────────────────────────────────────────────────────────────
// 只把 JSON.parse 包进 try/catch：解析失败跳过该行；
// 解析成功后的业务错误（type=error）必须抛给上层展示，不能被吞掉。
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

// ── 发送 / 停止 ───────────────────────────────────────────────────────────
let isStreaming = false
let abortCtl    = null
let lastQuery   = ''

function setStreamingUI(streaming) {
    isStreaming = streaming
    if (IS_WAILS) {
        // Wails 一次性返回，无法中止，流式期间禁用发送按钮
        sendBtn.disabled = streaming
        return
    }
    sendBtn.classList.toggle('stop', streaming)
    sendBtn.textContent = streaming ? '■' : '➤'
    const label = streaming ? '停止生成' : '发送'
    sendBtn.title = label
    sendBtn.setAttribute('aria-label', label)
}

function stopStreaming() {
    if (abortCtl) abortCtl.abort()
}

async function sendMessage(rawText) {
    const text = (typeof rawText === 'string' ? rawText : input.value).trim()
    if (!text || isStreaming) return
    lastQuery = text

    const conv = ensureActiveConv()
    welcomeEl.classList.add('hide')

    // 用户消息：DOM + 存储
    addMessageDOM('user', text)
    pushMsg('user', text)
    topbarTitle.textContent = conv.title
    renderConvList()

    input.value = ''
    autoResize()
    if (!IS_WAILS) input.focus()

    setStreamingUI(true)
    const typingEl = showTyping()

    let aiBubble = null
    let fullText = ''

    try {
        if (IS_WAILS) {
            // ── Wails 模式：Go binding 一次性返回 ──────────────────────
            typingEl.remove()
            aiBubble = addMessageDOM('ai', '')

            const result = await window.go.main.App.Analyze(text)
            fullText = result
            aiBubble.classList.add('md')
            aiBubble.innerHTML = renderMd(fullText.replace(/\n\s*\n/g, '\n\n'))
            scrollToBottom(true)
        } else {
            // ── HTTP SSE 模式：流式逐 chunk 显示 ────────────────────────
            abortCtl = new AbortController()
            const signal = (typeof AbortSignal.any === 'function')
                ? AbortSignal.any([abortCtl.signal, AbortSignal.timeout(90000)])
                : abortCtl.signal

            const response = await fetch(nluStreamURL(), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ input: text, session_id: conv.sessionId }),
                signal,
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
                        if (!aiBubble) {
                            typingEl.remove()
                            aiBubble = addMessageDOM('ai', '')
                            aiBubble.classList.add('streaming')
                        }
                        fullText += chunk.content
                        aiBubble.textContent = fullText
                        scrollToBottom()
                    } else if (chunk.type === 'done') {
                        streamDone = true
                        break
                    } else if (chunk.type === 'error') {
                        throw new Error(chunk.error || '推理出错')
                    }
                }
            }

            if (aiBubble && fullText) {
                aiBubble.classList.remove('streaming')
                aiBubble.classList.add('md')
                aiBubble.innerHTML = renderMd(fullText)
                scrollToBottom()
            }
        }

        if (aiBubble && fullText) {
            pushMsg('ai', fullText)
            saveConvs()
        } else if (!aiBubble) {
            hideTyping()
            const fallback = '暂时没有找到合适的推荐，请尝试提供更多英雄或装备信息。'
            addMessageDOM('ai', fallback)
            pushMsg('ai', fallback)
        }
        renderConvList()
    } catch (err) {
        hideTyping()
        if (aiBubble && fullText) {
            // 已输出部分内容：保留并标注中断
            aiBubble.classList.remove('streaming')
            aiBubble.classList.add('md')
            aiBubble.innerHTML = renderMd(fullText + '\n\n*(连接中断)*')
            pushMsg('ai', fullText + '\n\n*(连接中断)*')
        } else if (err.name === 'AbortError') {
            addMessageDOM('ai', '已停止生成。')
            pushMsg('ai', '已停止生成。')
        } else {
            showErrorBubble(err)
            pushMsg('ai', `**连接失败**：${err.message}`)
        }
        renderConvList()
    } finally {
        setStreamingUI(false)
        abortCtl = null
        // 一次请求后刷新一次健康状态（失败时及时变红）
        checkHealth()
    }
}

// 错误气泡：Markdown 说明 + 重试按钮
function showErrorBubble(err) {
    const msg = err.name === 'TimeoutError' ? '请求超时（90s）' : err.message
    const bubbleEl = addMessageDOM('ai',
        `**连接失败**：${escHtml(msg)}\n\n请检查服务器是否启动：\`${API_BASE}\``,
        { md: true })
    const body = bubbleEl.closest('.msg-body')
    const btn = document.createElement('button')
    btn.className = 'retry-btn'
    btn.textContent = '↻ 重试'
    btn.dataset.q = lastQuery
    body.querySelector('.msg-meta').appendChild(btn)
}

sendBtn.addEventListener('click', () => {
    if (isStreaming) { stopStreaming(); return }
    sendMessage()
})

// ── 局面快填 ──────────────────────────────────────────────────────────────
function buildCoach() {
    COACH_GROUPS.forEach((g) => {
        const group = document.createElement('div')
        group.className = 'coach-group'

        const label = document.createElement('span')
        label.className = 'coach-label'
        label.textContent = g.name

        const chips = document.createElement('div')
        chips.className = 'coach-chips'
        g.tokens.forEach((tok) => {
            const c = document.createElement('button')
            c.type = 'button'
            c.className = 'coach-chip'
            c.textContent = tok
            c.dataset.tok = tok
            chips.appendChild(c)
        })

        group.append(label, chips)
        coachGroups.appendChild(group)
    })
}

function toggleCoach() {
    const open = composer.classList.toggle('coach-open')
    coachToggle.classList.toggle('active', open)
    coachToggle.setAttribute('aria-expanded', String(open))
}

coachToggle.addEventListener('click', toggleCoach)

coachGroups.addEventListener('click', (e) => {
    const chip = e.target.closest('.coach-chip')
    if (chip) insertSituation(chip.dataset.tok)
})

// 以"，"连接局面片段，拼出 "3-2，6级，40血，" 这样的输入
function insertSituation(tok) {
    const cur = input.value
    const sep = cur && !/[，,？?\s]$/.test(cur) ? '，' : ''
    input.value = cur + sep + tok + '，'
    input.focus()
    autoResize()
}

function fillInput(text) {
    input.value = text
    input.focus()
    autoResize()
}

// ── 输入框 ────────────────────────────────────────────────────────────────
function autoResize() {
    input.style.height = 'auto'
    input.style.height = Math.min(input.scrollHeight, 160) + 'px'
}
input.addEventListener('input', autoResize)

// Enter 发送，Shift+Enter 换行
input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        sendMessage()
    }
})

// ── 侧边栏响应式 ──────────────────��───────────────────────────────────────
function closeSidebarOnMobile() {
    appEl.classList.remove('side-open')
}

sideToggle.addEventListener('click', () => {
    if (window.innerWidth <= 860) appEl.classList.toggle('side-open')
    else appEl.classList.toggle('side-closed')
})
sideMask.addEventListener('click', closeSidebarOnMobile)

// Escape：先关设置弹窗，再关移动端侧边栏
document.addEventListener('keydown', (e) => {
    if (e.key !== 'Escape') return
    if (settingsModal.classList.contains('show')) {
        settingsModal.classList.remove('show')
    } else {
        closeSidebarOnMobile()
    }
})

// ── 设置弹窗 ──────────────────────────────────────────────────────────────
settingsBtn.addEventListener('click', () => {
    apiInput.value = API_BASE
    renderSettingsHealth()
    settingsModal.classList.add('show')
})
settingsClose.addEventListener('click', () => settingsModal.classList.remove('show'))
settingsModal.addEventListener('click', (e) => {
    if (e.target === settingsModal) settingsModal.classList.remove('show')
})

settingsSave.addEventListener('click', async () => {
    const val = apiInput.value.trim().replace(/\/+$/, '')
    if (!val) return

    applyConfig({ api_base: val })

    if (window.electronAPI) {
        try {
            const result = await window.electronAPI.saveConfig({ api_base: val })
            console.log('配置已保存到:', result.path)
        } catch (e) { /* 忽略持久化失败 */ }
    } else {
        localStorage.setItem('tft_api_base', val)
    }

    settingsModal.classList.remove('show')

    // 换了后端，旧的多轮上下文不再有意义：全部会话重置 session_id
    conversations.forEach((c) => { c.sessionId = uid() })
    saveConvs()
    addMessageDOM('ai', `已切换到 \`${val}\`，正在检测连接…（历史对话的多轮上下文已重置）`, { md: true })
    welcomeEl.classList.add('hide')
    checkHealth()
})

// ── Electron 鼠标穿透控制（覆盖层场景；compact/pet 布局下生效） ───────────
const electron = window.electronAPI || { setInteractive: () => {}, setPassthrough: () => {} }
sidebar.addEventListener('mouseenter', () => electron.setInteractive())
document.getElementById('main').addEventListener('mouseenter', () => electron.setInteractive())

// ── Wails 最小化 ──────────────────��───────────────────────────────────────
if (minimizeBtn) {
    minimizeBtn.addEventListener('click', async () => {
        if (window.go?.main?.App?.Minimize) await window.go.main.App.Minimize()
    })
}

// ── 启动 ──────────────────────────────────────────────────────────────────
buildCapCards()
buildCoach()
loadConvs()
if (!conversations.length) createConv()
else activeId = conversations[0].id
renderConvList()
renderActiveConv()

loadConfig()
    .then((hasSaved) => ((hasSaved || IS_WAILS) ? null : detectBase()))
    .finally(() => { checkHealth() })
if (!IS_WAILS) setInterval(checkHealth, 30000)
