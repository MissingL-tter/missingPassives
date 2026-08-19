const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("pob", {
  listBuilds: () => ipcRenderer.invoke("builds:list"),
  loadBuildFile: (fileName) => ipcRenderer.invoke("engine:loadBuildFile", fileName),
  newBuild: () => ipcRenderer.invoke("engine:newBuild"),
  call: (method, params) => ipcRenderer.invoke("engine:call", method, params),
  onEngineLog: (handler) => ipcRenderer.on("engine:log", (event, line) => handler(line)),
  statsRendered: () => ipcRenderer.send("renderer:stats-rendered"),
});
