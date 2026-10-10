/**
 * marked 本地降级实现（FIX-04）
 *
 * 背景：index.html 通过 <script src="assets/vendor/marked.min.js"> 加载本地 marked，
 * 以支持严格 CSP（script-src 'self'，禁止内联脚本）。若 vendor 文件缺失/损坏，
 * 页面不能白屏也不能退回"直接把原文当 HTML"的危险路径，因此提供本降级实现。
 *
 * 关键约束：
 *  1. 本文件必须是外链脚本（不能内联），否则会被 strict CSP 拦截；
 *  2. 输出仍会被 app.js 的 renderMd 再过一遍 window.TFTSanitize，双重防线；
 *  3. 先整体转义再补少量标签，保证任何输入都不会产生可执行 HTML。
 */
(function () {
    'use strict';

    function escapeHtml(text) {
        return String(text)
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#39;')
    }

    function markedFallback(text) {
        if (text === null || text === undefined) return ''
        return escapeHtml(text)
            // 标题：# / ## / ###
            .replace(/^(#{1,3})\s+(.+)$/gm, '<strong>$2</strong>')
            // 粗体 / 行内代码
            .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
            .replace(/`([^`\n]+?)`/g, '<code>$1</code>')
            // 无序列表
            .replace(/^[-*]\s+(.+)$/gm, '<li>$1</li>')
            // 引用（原文已被转义成 &gt;）
            .replace(/^&gt;\s?(.+)$/gm, '<em>$1</em>')
            // 段落内换行
            .replace(/\n/g, '<br>')
    }

    window._markedFallback = markedFallback

    // vendor 未加载成功时补位，让 app.js 的 `marked.parse(...)` 分支依然可用
    if (typeof window.marked === 'undefined' || !window.marked || typeof window.marked.parse !== 'function') {
        window.marked = { parse: markedFallback }
    }
})()
