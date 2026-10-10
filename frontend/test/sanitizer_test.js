/**
 * FIX-04 前端安全渲染测试（Node + jsdom）
 * ========================================
 *
 * 验证内容：
 *   A. 实际发布的 index.html 脚本接线：marked / marked_fallback / sanitizer
 *      均为本地 vendor 资源且成功加载（strict CSP 下不依赖任何内联脚本）。
 *   B. renderMd 全链路：marked.parse 输出经 TFTSanitize 后，
 *      script / 事件属性 / javascript: 链接 / 危险协议全部不可达。
 *   C. 合法 Markdown（标题/表格/代码块/链接/中文）正常渲染。
 *   D. 流式分段：恶意文本在任意切分点分段渲染，同样不可达。
 *   E. 降级路径：marked 缺失时 _markedFallback 先转义再补标签，
 *      原始 HTML 不会穿透。
 *
 * 运行（需 jsdom，见 Makefile test-frontend）：
 *   NODE_PATH=<jsom所在node_modules> node --test frontend/test/
 */

'use strict'

const test = require('node:test')
const assert = require('node:assert')
const fs = require('node:fs')
const path = require('node:path')
const { JSDOM, VirtualConsole } = require('jsdom')

const ROOT = path.resolve(__dirname, '..', '..')
const INDEX_HTML = path.join(ROOT, 'frontend', 'index.html')

// ── 测试环境搭建 ───────────────────────────────────────────────────────────

// 零网络加载：解析真实 index.html，按 script[src] 顺序把本地文件内容
// 以经典 <script> 方式注入执行（与浏览器加载语义一致：顶层 'use strict'
// 下函数声明仍进全局）。
// 注意：url 必须用 http origin——file:// 在 jsdom 里是 opaque origin，
// localStorage 不可用会让 app.js 顶层初始化抛未处理 rejection。
async function loadPage() {
    const html = fs.readFileSync(INDEX_HTML, 'utf8')
    const vc = new VirtualConsole() // 吞掉 jsdom 不支持 API 的报错噪音
    const dom = new JSDOM(html, {
        runScripts: 'dangerously',
        url: 'http://localhost/',
        virtualConsole: vc,
        pretendToBeVisual: true,
    })
    // 离线桩：测试不发任何真实网络请求（也不受沙箱代理干扰），
    // app.js 里 loadConfig/detectBase/checkHealth 均有 try/catch 兜底。
    dom.window.fetch = () => Promise.reject(new Error('offline: test env'))
    const doc = dom.window.document
    const scripts = Array.from(doc.querySelectorAll('script[src]'))
    for (const s of scripts) {
        const rel = s.getAttribute('src').split('?')[0].replace(/^\//, '')
        const file = path.join(ROOT, 'frontend', rel)
        const el = doc.createElement('script')
        el.textContent = fs.readFileSync(file, 'utf8')
        doc.body.appendChild(el)
    }
    // 等异步初始化（loadConfig/checkHealth 等）跑完一拍
    await new Promise((r) => setTimeout(r, 100))
    return dom
}

// 统计可执行注入面的检测：处理后的 HTML 里不应再出现这些模式。
// 注意模式只匹配"真实标签内"的属性（<tag ... onxxx=），
// 已转义的文本内容（&lt;img src=x onerror=a）不可执行，不算危险。
const DANGEROUS_PATTERNS = [
    /<script/i,
    /<[a-z][^>]*\son[a-z]+\s*=/i, // <tag ... on*= 事件属性
    /<[a-z][^>]*\ssrcdoc\s*=/i,
    /(?:href|src)\s*=\s*["']?\s*javascript:/i,
    /(?:href|src)\s*=\s*["']?\s*vbscript:/i,
]

function assertNoDanger(html, label) {
    for (const re of DANGEROUS_PATTERNS) {
        assert.ok(
            !re.test(html),
            `${label}: 仍含危险模式 ${re} -> ${JSON.stringify(html.slice(0, 200))}`
        )
    }
}

// ── A. 页面脚本接线 ───────────────────────────────────────────────────────

test('页面加载：marked / fallback / sanitizer / renderMd 全部就位（无内联脚本）', async (t) => {
    const dom = await loadPage()
    t.after(() => dom.window.close())

    const w = dom.window
    assert.equal(typeof w.marked, 'object', '本地 vendor marked 未加载')
    assert.equal(typeof w.marked.parse, 'function', 'marked.parse 不可用')
    assert.equal(typeof w._markedFallback, 'function', 'marked_fallback.js 未加载')
    assert.equal(typeof w.TFTSanitize, 'function', 'sanitizer.js 未加载')
    assert.equal(typeof w.renderMd, 'function', 'app.js renderMd 未定义')

    // strict CSP 前置条件：页面自身没有内联脚本与内联事件属性
    // （先剥掉 HTML 注释，注释里提到 "<script>" 不算违规）
    const html = fs.readFileSync(INDEX_HTML, 'utf8').replace(/<!--[\s\S]*?-->/g, '')
    assert.ok(!/<script(?![^>]*\bsrc=)[^>]*>[\s\S]*?<\//i.test(html), 'index.html 存在内联脚本块')
    assert.ok(!/\son[a-z]+\s*="/.test(html), 'index.html 存在 on*= 内联事件属性')
})

// ── B. 恶意输入全链路 ─────────────────────────────────────────────────────

const MALICIOUS_INPUTS = [
    '<script>alert(1)</script>',
    '<img src=x onerror=alert(1)>',
    '<a href="javascript:alert(1)">点我</a>',
    '<a href="JaVaScRiPt:alert(1)">大小写混合</a>',
    '<a href="jav\tascript:alert(1)">控制字符绕过</a>',
    '<a href="data:text/html,<script>alert(1)</script>">data 协议</a>',
    '<iframe src="javascript:alert(1)"></iframe>',
    '<svg onload=alert(1)><circle r="10"/></svg>',
    '<details open ontoggle=alert(1)>x</details>',
    '<form action="javascript:alert(1)"><button>提交</button></form>',
    '<style>body{background:url(javascript:alert(1))}</style>',
    '<div style="background:url(javascript:alert(1))">styled</div>',
    '<video><source onerror="alert(1)"></video>',
    '<meta http-equiv="refresh" content="0;url=javascript:alert(1)">',
    '<link rel="stylesheet" href="http://evil.example/x.css">',
    '<p>正常文本中间夹 <script>alert(1)</script> 尾巴</p>',
]

test('renderMd：恶意 HTML 全链路清洗（含事件属性/危险协议/整体移除标签）', async (t) => {
    const dom = await loadPage()
    t.after(() => dom.window.close())
    const { renderMd } = dom.window

    for (const input of MALICIOUS_INPUTS) {
        const out = renderMd(input)
        assert.equal(typeof out, 'string', `renderMd(${input.slice(0, 30)}) 应返回字符串`)
        assertNoDanger(out, `renderMd(${JSON.stringify(input.slice(0, 40))})`)
    }
})

test('TFTSanitize：script/iframe/style 连同内容整体移除，不留空壳', async (t) => {
    const dom = await loadPage()
    t.after(() => dom.window.close())
    const { TFTSanitize } = dom.window

    const out = TFTSanitize('<p>前</p><script>alert(1)</script><p>后</p>')
    assert.ok(!out.includes('script'), 'script 标签残留')
    assert.ok(out.includes('前') && out.includes('后'), '安全内容被误删')

    const out2 = TFTSanitize('<iframe src="https://evil.example"></iframe>')
    assert.ok(!out2.includes('iframe'), 'iframe 标签残留')
})

test('TFTSanitize：安全链接保留并强制 target=_blank + noopener', async (t) => {
    const dom = await loadPage()
    t.after(() => dom.window.close())
    const { TFTSanitize } = dom.window

    const out = TFTSanitize('<a href="https://metatft.com/tft-comps">MetaTFT</a>')
    assert.ok(out.includes('href="https://metatft.com/tft-comps"'), '安全 href 被误删')
    assert.ok(out.includes('target="_blank"'), '缺少 target=_blank')
    assert.ok(out.includes('rel="noopener noreferrer"'), '缺少 rel=noopener')
})

// ── C. 合法 Markdown 不误杀 ───────────────────────────────────────────────

test('renderMd：合法 Markdown 正常渲染（中文/表格/代码块/粗体/列表）', async (t) => {
    const dom = await loadPage()
    t.after(() => dom.window.close())
    const { renderMd } = dom.window

    const md = [
        '## 魔女法杖运营',
        '',
        '主 C **阿狸**，副 C `霞`，装备优先：',
        '',
        '| 位置 | 英雄 | 装备 |',
        '| --- | --- | --- |',
        '| 主C | 阿狸 | 蓝霸符 |',
        '| 副C | 霞 | 巨杀 |',
        '',
        '- 前期拿体系牌',
        '- 3-2 升 6 级',
        '',
        '> 注意：4 魔女羁绊是核心',
        '',
        '[MetaTFT 参考](https://metatft.com/tft-comps)',
    ].join('\n')

    const out = renderMd(md)
    assert.ok(/<h2[^>]*>/.test(out) || /<strong>/.test(out), '标题丢失')
    assert.ok(out.includes('魔女法杖运营'), '中文内容丢失')
    assert.ok(/<table>/.test(out), '表格丢失')
    assert.ok(/<code>/.test(out), '行内代码丢失')
    assert.ok(/<strong>/.test(out) || /<b>/.test(out), '粗体丢失')
    assert.ok(/阿狸/.test(out) && /巨杀/.test(out), '表格内容丢失')
    assert.ok(out.includes('metatft.com'), '安全外链丢失')
    assertNoDanger(out, '合法 Markdown')
})

// ── D. 流式分段渲染 ──────────────────────────────────────────────────────

test('renderMd：恶意文本按任意切分点分段渲染，危险模式不可达', async (t) => {
    const dom = await loadPage()
    t.after(() => dom.window.close())
    const { renderMd } = dom.window

    const full = [
        '## 标题开头',
        '',
        '先来一段正常说明，然后注入 <img src=x onerror=alert(1)> 图片，',
        '再来一个 [危险链接](javascript:alert(1))，',
        '还有 <script>alert(1)</script> 脚本和 <iframe src="https://evil.example"></iframe>。',
        '',
        '收尾：阵容推荐 **阿狸主 C**。',
    ].join('\n')

    // 模拟 SSE 分片：从第 1 个字符开始，每个切分点都渲染一次
    for (let cut = 1; cut <= full.length; cut += 7) {
        const partial = full.slice(0, cut)
        const out = renderMd(partial)
        assertNoDanger(out, `分段渲染 cut=${cut}`)
    }
})

// ── E. 降级路径 ──────────────────────────────────────────────────────────

test('_markedFallback：先转义再补标签，原始 HTML 不穿透', async (t) => {
    const dom = await loadPage()
    t.after(() => dom.window.close())
    const { _markedFallback, TFTSanitize } = dom.window

    const raw = '<script>alert(1)</script> **加粗** `code`'
    const out = _markedFallback(raw)
    // 转义后的 script 不可执行
    assert.ok(!/<script/i.test(out), 'fallback 输出存在未转义的 script')
    assert.ok(out.includes('&lt;script&gt;'), '原始 HTML 未转义')
    assert.ok(out.includes('<strong>加粗</strong>'), '粗体丢失')
    assert.ok(out.includes('<code>code</code>'), '行内代码丢失')

    // fallback 输出再过 sanitizer 双保险
    assertNoDanger(TFTSanitize(out), 'fallback+sanitize 链路')
})

test('兜底：sanitizer 缺失时 renderMd 退化为纯转义（不注入）', async (t) => {
    const dom = await loadPage()
    t.after(() => dom.window.close())

    // 模拟 sanitizer 加载失败：删掉全局函数后调 renderMd
    const w = dom.window
    const saved = w.TFTSanitize
    delete w.TFTSanitize
    try {
        const out = w.renderMd('<script>alert(1)</script>**加粗**')
        assert.ok(!/<script/i.test(out), '兜底路径存在未转义 script')
        assert.ok(out.includes('&lt;script&gt;'), '兜底路径未转义')
    } finally {
        w.TFTSanitize = saved
    }
})
