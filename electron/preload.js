const { contextBridge, ipcRenderer } = require('electron')

contextBridge.exposeInMainWorld('electronAPI', {
    // 鼠标穿透
    setInteractive: () => ipcRenderer.send('set-interactive'),
    setPassthrough: () => ipcRenderer.send('set-passthrough'),

    // 配置读写
    getConfig:     () => ipcRenderer.invoke('get-config'),
    saveConfig:    (cfg) => ipcRenderer.invoke('save-config', cfg),
    getConfigPath: () => ipcRenderer.invoke('get-config-path'),
})