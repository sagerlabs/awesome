const { app, BrowserWindow, screen, ipcMain } = require('electron')
const path = require('path')

let win

function createWindow() {
    const { width, height } = screen.getPrimaryDisplay().workAreaSize

    win = new BrowserWindow({
        // 全屏透明覆盖层，让气泡悬浮在所有窗口之上
        width,
        height,
        x: 0,
        y: 0,

        transparent: true,        // 背景透明
        frame: false,             // 无边框（去掉标题栏）
        alwaysOnTop: true,        // 始终置顶
        skipTaskbar: true,        // 不在任务栏显示
        resizable: false,
        hasShadow: false,

        // 允许点击穿透：透明区域的鼠标事件穿透到底层窗口
        // 气泡和面板区域通过 setIgnoreMouseEvents 动态控制
        webPreferences: {
            nodeIntegration: false,
            contextIsolation: true,
            preload: path.join(__dirname, 'preload.js'),
        },
    })

    // 默认让鼠标事件穿透（透明区域不拦截点击）
    win.setIgnoreMouseEvents(true, { forward: true })

    win.loadFile(path.join(__dirname, '../frontend/index.html'))

    // 开发时打开 DevTools
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

// ── IPC：前端通知 Electron 哪些区域需要响应鼠标 ─────────────────
// 气泡/面板出现时：停止穿透
ipcMain.on('set-interactive', () => {
    win.setIgnoreMouseEvents(false)
})

// 鼠标离开所有交互区域时：恢复穿透
ipcMain.on('set-passthrough', () => {
    win.setIgnoreMouseEvents(true, { forward: true })
})