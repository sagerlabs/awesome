const { app, BrowserWindow, screen, ipcMain } = require('electron')
const path = require('path')
const fs   = require('fs')

let win

// ── config.json 管理 ──────────────────────────────────────────────
// 打包进 asar 的默认配置（只读）
const defaultConfigPath = path.join(__dirname, '../frontend/config.json')

// 用户数据目录的配置（可读写）
// Mac:   ~/Library/Application Support/tft-copilot/config.json
// Win:   %APPDATA%/tft-copilot/config.json
// Linux: ~/.config/tft-copilot/config.json
function getUserConfigPath() {
    return path.join(app.getPath('userData'), 'config.json')
}

function loadConfig() {
    const userConfigPath = getUserConfigPath()

    // ── 第一优先：用户数据目录（用户手动改或 UI 保存的）──────────
    if (fs.existsSync(userConfigPath)) {
        try {
            return JSON.parse(fs.readFileSync(userConfigPath, 'utf-8'))
        } catch (e) {
            console.error('用户 config.json 解析失败，继续查找', e)
        }
    }

    // ── 第二优先：二进制同级目录（CI 没注入 IP 时用户手动放置）────
    // app.isPackaged = true 时，process.execPath 是打包后的二进制路径
    // 例：/Applications/TFT Copilot.app/Contents/MacOS/TFT Copilot
    // 同级目录：/Applications/TFT Copilot.app/Contents/MacOS/config.json
    if (app.isPackaged) {
        const execDirConfig = path.join(path.dirname(process.execPath), 'config.json')
        if (fs.existsSync(execDirConfig)) {
            try {
                const cfg = JSON.parse(fs.readFileSync(execDirConfig, 'utf-8'))
                // 找到了就复制一份到用户目录，统一后续读写入口
                saveConfig(cfg)
                console.log('从二进制同级目录加载配置:', execDirConfig)
                return cfg
            } catch (e) {
                console.error('二进制同级 config.json 解析失败', e)
            }
        }
    }

    // ── 第三优先：asar 内的默认配置（打包时 CI 注入的）─────────────
    if (fs.existsSync(defaultConfigPath)) {
        try {
            const cfg = JSON.parse(fs.readFileSync(defaultConfigPath, 'utf-8'))
            // 复制到用户目录，方便后续持久化修改
            saveConfig(cfg)
            console.log('从 asar 默认配置初始化:', defaultConfigPath)
            return cfg
        } catch (e) {
            console.error('asar 内 config.json 解析失败', e)
        }
    }

    // ── 兜底：硬编码默认值 ─────────────────────────────────────────
    console.warn('未找到任何 config.json，使用默认地址 localhost:8080')
    return { api_base: 'http://localhost:8080' }
}

function saveConfig(cfg) {
    const userConfigPath = getUserConfigPath()
    fs.mkdirSync(path.dirname(userConfigPath), { recursive: true })
    fs.writeFileSync(userConfigPath, JSON.stringify(cfg, null, 2), 'utf-8')
    console.log('用户配置已保存:', userConfigPath)
}

// ── 窗口创建 ──────────────────────────────────────────────────────
function createWindow() {
    const { width, height } = screen.getPrimaryDisplay().workAreaSize

    win = new BrowserWindow({
        width,
        height,
        x: 0,
        y: 0,
        transparent: true,
        frame: false,
        alwaysOnTop: true,
        skipTaskbar: true,
        resizable: false,
        hasShadow: false,
        webPreferences: {
            nodeIntegration: false,
            contextIsolation: true,
            preload: path.join(__dirname, 'preload.js'),
        },
    })

    win.setIgnoreMouseEvents(true, { forward: true })
    win.loadFile(path.join(__dirname, '../frontend/index.html'))

    if (process.env.NODE_ENV === 'development') {
        win.webContents.openDevTools({ mode: 'detach' })
    }
}

app.whenReady().then(createWindow)

app.on('window-all-closed', () => {
    if (process.platform !== 'darwin') app.quit()
})

app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow()
})

// ── IPC 处理 ──────────────────────────────────────────────────────

// 鼠标穿透控制
ipcMain.on('set-interactive', () => win.setIgnoreMouseEvents(false))
ipcMain.on('set-passthrough', () => win.setIgnoreMouseEvents(true, { forward: true }))

// 前端请求读取配置
ipcMain.handle('get-config', () => loadConfig())

// 前端保存新配置（用户在设置面板修改 IP 后触发）
ipcMain.handle('save-config', (_, cfg) => {
    saveConfig(cfg)
    return { ok: true, path: getUserConfigPath() }
})

// 前端请求获取配置文件路径（显示给用户，方便手动编辑）
ipcMain.handle('get-config-path', () => getUserConfigPath())