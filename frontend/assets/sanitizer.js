/**
 * TFT Copilot HTML 清洗器（FIX-04）
 * ================================
 *
 * 作用：对 marked.parse() 的输出做白名单过滤，用于所有 innerHTML 赋值前的
 * 最后一道防线（流式完成、Wails 一次性返回、历史重放、中断恢复共用）。
 *
 * 策略：
 *   1. 标签白名单：只保留 Markdown 正常产物 + 少量排版标签；
 *      危险标签（script/iframe/...）连同内容整体移除；
 *      其余未知标签解包（保留子内容，去标签本身）。
 *   2. 属性白名单：每个标签单独配置；on* 事件属性一律不在白名单内，
 *      style 内联样式默认剔除（防 CSS 注入/外链）。
 *   3. URL 协议白名单：仅 http/https/mailto 与相对路径；
 *      javascript:/data:/vbscript:/file: 一律移除属性。
 *   4. 注释节点删除（防条件注释残留）。
 *
 * 实现基于 DOM（template 元素解析，脚本不执行），非正则替换。
 *
 * 暴露：window.TFTSanitize(html) -> string
 */
(function () {
    'use strict';

    // 允许的标签（marked 输出 + 常见排版）
    var ALLOWED_TAGS = {
        p: 1, br: 1, hr: 1,
        strong: 1, em: 1, b: 1, i: 1, u: 1, s: 1, del: 1, ins: 1, mark: 1,
        code: 1, pre: 1, blockquote: 1, kbd: 1, samp: 1,
        ul: 1, ol: 1, li: 1, dl: 1, dt: 1, dd: 1,
        h1: 1, h2: 1, h3: 1, h4: 1, h5: 1, h6: 1,
        a: 1, span: 1, div: 1,
        table: 1, thead: 1, tbody: 1, tfoot: 1, tr: 1, th: 1, td: 1,
        caption: 1, col: 1, colgroup: 1,
        img: 1, input: 1, sup: 1, sub: 1, abbr: 1, details: 1, summary: 1
    };

    // 整体移除（含子节点）的危险标签
    var DROP_TAGS = {
        script: 1, style: 1, iframe: 1, frame: 1, frameset: 1,
        object: 1, embed: 1, applet: 1, link: 1, meta: 1, base: 1,
        form: 1, button: 1, select: 1, textarea: 1, option: 1,
        video: 1, audio: 1, source: 1, track: 1, canvas: 1,
        svg: 1, math: 1, noscript: 1, title: 1, head: 1
    };

    // 每个标签允许的属性（不在表内的标签不允许任何属性）
    var ALLOWED_ATTRS = {
        a: { href: 1, title: 1 },
        img: { src: 1, alt: 1, title: 1, width: 1, height: 1 },
        code: { class: 1 },
        span: { class: 1 },
        div: { class: 1 },
        th: { align: 1, colspan: 1, rowspan: 1 },
        td: { align: 1, colspan: 1, rowspan: 1 },
        input: { type: 1, checked: 1, disabled: 1 },
        abbr: { title: 1 },
        ol: { start: 1 },
        details: { open: 1 }
    };

    // URL 安全性：相对路径 / 白名单协议
    function isSafeUrl(url) {
        if (url == null) return false;
        var u = String(url).trim();
        if (u === '') return false;
        // 锚点 / 相对路径（./ ../ /）与协议相对（//example.com）
        if (u.charAt(0) === '#' || u.charAt(0) === '/') return true;
        if (u.slice(0, 2) === './' || u.slice(0, 3) === '../') return true;
        // 控制字符 / 空白混入（jav\tascript: 之类绕过）
        if (/[\x00-\x1f\x7f\s]/.test(u)) return false;
        // 显式协议（捕获组不含冒号，比较时用裸协议名）
        var m = /^([a-zA-Z][a-zA-Z0-9+.\-]*):/.exec(u);
        if (m) {
            var proto = m[1].toLowerCase();
            return proto === 'http' || proto === 'https' || proto === 'mailto';
        }
        // 无协议的裸字符串按相对路径处理
        return true;
    }

    function sanitizeNode(node) {
        var children = Array.prototype.slice.call(node.childNodes);
        for (var i = 0; i < children.length; i++) {
            var child = children[i];
            if (child.nodeType === 1 /* ELEMENT */) {
                var tag = child.nodeName.toLowerCase();

                if (DROP_TAGS[tag]) {
                    child.parentNode && child.parentNode.removeChild(child);
                    continue;
                }

                if (!ALLOWED_TAGS[tag]) {
                    // 先清理子树，再解包（保留安全的子内容）
                    sanitizeNode(child);
                    var frag = node.ownerDocument.createDocumentFragment();
                    while (child.firstChild) frag.appendChild(child.firstChild);
                    if (child.parentNode) {
                        child.parentNode.replaceChild(frag, child);
                    }
                    continue;
                }

                // 属性过滤
                var allowed = ALLOWED_ATTRS[tag] || {};
                var attrs = Array.prototype.slice.call(child.attributes);
                for (var j = 0; j < attrs.length; j++) {
                    var name = attrs[j].name.toLowerCase();
                    // on* 事件属性、style、srcdoc 一律剔除
                    if (name.lastIndexOf('on', 0) === 0 || name === 'style' || name === 'srcdoc') {
                        child.removeAttribute(attrs[j].name);
                        continue;
                    }
                    if (!allowed[name]) {
                        child.removeAttribute(attrs[j].name);
                    }
                }

                // URL 属性协议校验
                var urlAttrs = ['href', 'src'];
                for (var k = 0; k < urlAttrs.length; k++) {
                    var ua = urlAttrs[k];
                    if (child.hasAttribute && child.hasAttribute(ua)) {
                        var val = child.getAttribute(ua);
                        if (!isSafeUrl(val)) {
                            child.removeAttribute(ua);
                        }
                    }
                }

                // a 标签：强制新窗口 + noopener（防 tabnabbing）
                if (tag === 'a' && child.getAttribute('href')) {
                    child.setAttribute('target', '_blank');
                    child.setAttribute('rel', 'noopener noreferrer');
                }

                sanitizeNode(child);
            } else if (child.nodeType === 8 /* COMMENT */) {
                child.parentNode && child.parentNode.removeChild(child);
            }
            // TEXT / CDATA 不动
        }
    }

    function sanitize(html) {
        if (html == null) return '';
        var tpl = document.createElement('template');
        tpl.innerHTML = String(html);
        sanitizeNode(tpl.content);
        return tpl.innerHTML;
    }

    // 兼容 Wails（无 window）与测试环境
    if (typeof window !== 'undefined') {
        window.TFTSanitize = sanitize;
    }
    if (typeof module !== 'undefined' && module.exports) {
        module.exports = { sanitize: sanitize, isSafeUrl: isSafeUrl };
    }
})();
