// Electron main process: owns the window and the engine child process.
//
// The engine is the repo's headless Lua build (spawned as `luajit engine/host.lua`
// with cwd = <repo>/src) speaking line-delimited JSON over stdio; protocol lines
// are prefixed with @@RPC@@ so engine log output passes through harmlessly.
//
// The engine holds exactly one build per process (in-process reloads keep stale
// state), so every build load restarts the child.
const { app, BrowserWindow, ipcMain } = require("electron");
const { spawn, spawnSync } = require("child_process");
const readline = require("readline");
const path = require("path");
const fs = require("fs");

const repoRoot = path.resolve(__dirname, "..");
const srcDir = path.join(repoRoot, "src");
const buildsDir = path.join(srcDir, "Builds");
const RPC_PREFIX = "@@RPC@@";
const SMOKE = process.env.SMOKE === "1";

function findLuajit() {
  const candidates = [
    process.env.POB_LUAJIT,
    "luajit",
    "E:\\env\\lua\\luaJIT\\bin\\luajit.exe",
  ].filter(Boolean);
  for (const candidate of candidates) {
    const probe = spawnSync(candidate, ["-v"], { encoding: "utf8" });
    if (!probe.error) {
      return candidate;
    }
  }
  throw new Error("luajit not found; set POB_LUAJIT to the luajit executable");
}

class EngineProcess {
  constructor(luajitPath, onLog) {
    this.luajitPath = luajitPath;
    this.onLog = onLog;
    this.child = null;
    this.pending = new Map();
    this.nextId = 1;
    this.readyPromise = null;
  }

  start() {
    this.stop();
    const child = spawn(this.luajitPath, [path.join(__dirname, "engine", "host.lua")], {
      cwd: srcDir,
      stdio: ["pipe", "pipe", "pipe"],
    });
    this.child = child;

    let resolveReady, rejectReady;
    this.readyPromise = new Promise((resolve, reject) => {
      resolveReady = resolve;
      rejectReady = reject;
    });
    const readyTimer = setTimeout(() => rejectReady(new Error("engine boot timed out")), 120000);

    readline.createInterface({ input: child.stdout }).on("line", (line) => {
      if (!line.startsWith(RPC_PREFIX)) {
        if (line.trim()) this.onLog(line);
        return;
      }
      let message;
      try {
        message = JSON.parse(line.slice(RPC_PREFIX.length));
      } catch (err) {
        this.onLog(`bad rpc line: ${line}`);
        return;
      }
      if (message.ready) {
        clearTimeout(readyTimer);
        resolveReady();
        return;
      }
      const pending = this.pending.get(message.id);
      if (pending) {
        this.pending.delete(message.id);
        if (message.error !== undefined) {
          pending.reject(new Error(message.error));
        } else {
          pending.resolve(message.result);
        }
      }
    });
    readline.createInterface({ input: child.stderr }).on("line", (line) => {
      if (line.trim()) this.onLog(line);
    });
    child.on("exit", (code) => {
      clearTimeout(readyTimer);
      rejectReady(new Error(`engine exited during boot (code ${code})`));
      for (const pending of this.pending.values()) {
        pending.reject(new Error(`engine exited (code ${code})`));
      }
      this.pending.clear();
    });
    return this.readyPromise;
  }

  stop() {
    if (this.child) {
      this.child.removeAllListeners("exit");
      this.child.kill();
      this.child = null;
    }
    for (const pending of this.pending.values()) {
      pending.reject(new Error("engine restarted"));
    }
    this.pending.clear();
  }

  async call(method, params) {
    if (!this.child) {
      throw new Error("engine not running");
    }
    await this.readyPromise;
    const id = this.nextId++;
    const payload = JSON.stringify({ id, method, params });
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      this.child.stdin.write(payload + "\n");
    });
  }
}

let mainWindow = null;
let engine = null;

function sendToRenderer(channel, payload) {
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send(channel, payload);
  }
}

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1280,
    height: 840,
    backgroundColor: "#0d0d0f",
    show: false,
    webPreferences: {
      preload: path.join(__dirname, "preload.js"),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });
  mainWindow.setMenuBarVisibility(false);
  const loadOptions = {};
  if (process.env.SMOKE_SEARCH) {
    loadOptions.query = { search: process.env.SMOKE_SEARCH };
  }
  mainWindow.loadFile(path.join(__dirname, "renderer", "index.html"), loadOptions);
  mainWindow.once("ready-to-show", () => mainWindow.show());
}

app.whenReady().then(() => {
  engine = new EngineProcess(findLuajit(), (line) => sendToRenderer("engine:log", line));

  ipcMain.handle("builds:list", () => {
    if (!fs.existsSync(buildsDir)) {
      return [];
    }
    return fs
      .readdirSync(buildsDir)
      .filter((name) => name.toLowerCase().endsWith(".xml"))
      .sort();
  });

  // Restarts the engine and loads the named build from src/Builds
  ipcMain.handle("engine:loadBuildFile", async (event, fileName) => {
    const filePath = path.join(buildsDir, fileName);
    if (path.relative(buildsDir, filePath).startsWith("..")) {
      throw new Error("build path escapes Builds directory");
    }
    const xml = fs.readFileSync(filePath, "utf8");
    await engine.start();
    return engine.call("loadBuildXML", { xml, name: fileName.replace(/\.xml$/i, "") });
  });

  ipcMain.handle("engine:newBuild", async () => {
    await engine.start();
    return engine.call("newBuild");
  });

  ipcMain.handle("engine:call", (event, method, params) => engine.call(method, params));

  if (SMOKE) {
    ipcMain.once("renderer:stats-rendered", async () => {
      try {
        await new Promise((resolve) => setTimeout(resolve, 800));
        const image = await mainWindow.webContents.capturePage();
        fs.writeFileSync(path.join(__dirname, "smoke.png"), image.toPNG());
        console.log("SMOKE OK");
        app.exit(0);
      } catch (err) {
        console.error("SMOKE FAILED:", err);
        app.exit(1);
      }
    });
    setTimeout(() => {
      console.error("SMOKE FAILED: timed out waiting for stats render");
      app.exit(1);
    }, 180000);
  }

  createWindow();
});

app.on("window-all-closed", () => {
  if (engine) engine.stop();
  app.quit();
});
