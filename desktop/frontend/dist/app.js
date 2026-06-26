const api = () => window.go?.main?.App;

const el = {
  url: document.querySelector("#url"),
  saveDir: document.querySelector("#saveDir"),
  saveName: document.querySelector("#saveName"),
  linkNameSeparator: document.querySelector("#linkNameSeparator"),
  customRange: document.querySelector("#customRange"),
  headers: document.querySelector("#headers"),
  autoSelect: document.querySelector("#autoSelect"),
  muxMP4: document.querySelector("#muxMP4"),
  binaryMerge: document.querySelector("#binaryMerge"),
  concurrentDownload: document.querySelector("#concurrentDownload"),
  ffmpegPath: document.querySelector("#ffmpegPath"),
  checkFFmpeg: document.querySelector("#check-ffmpeg"),
  ffmpegCheck: document.querySelector("#ffmpeg-check"),
  checkTools: document.querySelector("#check-tools"),
  toolList: document.querySelector("#tool-list"),
  threadCount: document.querySelector("#threadCount"),
  retryCount: document.querySelector("#retryCount"),
  maxSpeed: document.querySelector("#maxSpeed"),
  customProxy: document.querySelector("#customProxy"),
  useSystemProxy: document.querySelector("#useSystemProxy"),
  coreStatus: document.querySelector("#core-status"),
  coreVersion: document.querySelector("#core-version"),
  corePath: document.querySelector("#core-path"),
  refreshCore: document.querySelector("#refresh-core"),
  copyCommand: document.querySelector("#copy-command"),
  copyLog: document.querySelector("#copy-log"),
  preflightResult: document.querySelector("#preflight-result"),
  taskList: document.querySelector("#task-list"),
  taskCount: document.querySelector("#task-count"),
  runningCount: document.querySelector("#running-count"),
  activeSpeed: document.querySelector("#active-speed"),
  viewTitle: document.querySelector("#view-title"),
  viewKicker: document.querySelector("#view-kicker"),
  detailTitle: document.querySelector("#detail-title"),
  taskSummary: document.querySelector("#task-summary"),
  fileList: document.querySelector("#file-list"),
  log: document.querySelector("#log"),
  navItems: Array.from(document.querySelectorAll("[data-view]")),
  views: Array.from(document.querySelectorAll(".view"))
};

let settings = {};
let tasks = [];
let selectedTaskId = "";
let createOnly = false;
let activeView = "dashboard";

const defaultTools = [
  { name: "ffmpeg", label: "FFmpeg" },
  { name: "ffprobe", label: "FFprobe" },
  { name: "mkvmerge", label: "mkvmerge" },
  { name: "mp4decrypt", label: "mp4decrypt" },
  { name: "shaka-packager", label: "Shaka Packager" }
];

const viewMeta = {
  dashboard: ["任务监控", "下载任务"],
  create: ["新建任务", "创建下载"],
  settings: ["参数设置", "下载参数"],
  output: ["输出与日志", "文件日志"]
};

function field(source, camel, pascal, fallback = "") {
  return source?.[camel] ?? source?.[pascal] ?? fallback;
}

function normalizeRequest(request = {}) {
  return {
    url: field(request, "url", "URL"),
    saveDir: field(request, "saveDir", "SaveDir"),
    saveName: field(request, "saveName", "SaveName"),
    linkNameSeparator: field(request, "linkNameSeparator", "LinkNameSeparator"),
    headers: field(request, "headers", "Headers", []),
    customRange: field(request, "customRange", "CustomRange"),
    ffmpegPath: field(request, "ffmpegPath", "FFmpegPath"),
    threadCount: field(request, "threadCount", "ThreadCount", 0),
    retryCount: field(request, "retryCount", "RetryCount", 0),
    maxSpeed: field(request, "maxSpeed", "MaxSpeed"),
    autoSelect: Boolean(field(request, "autoSelect", "AutoSelect", false)),
    muxMP4: Boolean(field(request, "muxMP4", "MuxMP4", false)),
    binaryMerge: Boolean(field(request, "binaryMerge", "BinaryMerge", false)),
    concurrentDownload: Boolean(field(request, "concurrentDownload", "ConcurrentDownload", false)),
    useSystemProxy: Boolean(field(request, "useSystemProxy", "UseSystemProxy", false)),
    customProxy: field(request, "customProxy", "CustomProxy")
  };
}

function normalizeTask(task = {}) {
  return {
    id: field(task, "id", "ID"),
    title: field(task, "title", "Title", "未命名任务"),
    status: field(task, "status", "Status", "pending"),
    progress: Number(field(task, "progress", "Progress", 0)) || 0,
    progressText: field(task, "progressText", "ProgressText"),
    speedText: field(task, "speedText", "SpeedText"),
    lastMessage: field(task, "lastMessage", "LastMessage"),
    exitCode: Number(field(task, "exitCode", "ExitCode", 0)) || 0,
    createdAt: field(task, "createdAt", "CreatedAt"),
    startedAt: field(task, "startedAt", "StartedAt"),
    finishedAt: field(task, "finishedAt", "FinishedAt"),
    request: normalizeRequest(field(task, "request", "Request", {})),
    args: field(task, "args", "Args", []),
    command: field(task, "command", "Command"),
    commandLine: field(task, "commandLine", "CommandLine"),
    logs: field(task, "logs", "Logs", []),
    files: normalizeFiles(field(task, "files", "Files", []))
  };
}

function normalizeFiles(files = []) {
  return files.map((file) => ({
    path: field(file, "path", "Path"),
    name: field(file, "name", "Name"),
    size: Number(field(file, "size", "Size", 0)) || 0,
    modified: field(file, "modified", "Modified"),
    extension: field(file, "extension", "Extension")
  }));
}

function normalizeToolInfo(tool = {}) {
  return {
    name: field(tool, "name", "Name"),
    label: field(tool, "label", "Label") || field(tool, "name", "Name"),
    command: field(tool, "command", "Command"),
    status: field(tool, "status", "Status", "pending"),
    path: field(tool, "path", "Path"),
    version: field(tool, "version", "Version"),
    error: field(tool, "error", "Error")
  };
}

function normalizePreflightCheck(check = {}) {
  return {
    name: field(check, "name", "Name"),
    label: field(check, "label", "Label"),
    status: field(check, "status", "Status", "error"),
    message: field(check, "message", "Message"),
    detail: field(check, "detail", "Detail")
  };
}

function taskStatusText(status) {
  return {
    pending: "等待",
    running: "下载中",
    completed: "完成",
    failed: "失败",
    stopped: "已停止"
  }[status] || status;
}

function formatSize(bytes) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let value = bytes;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value.toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

function formatDate(value) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return date.toLocaleString("zh-CN", { hour12: false });
}

function switchView(name) {
  activeView = viewMeta[name] ? name : "dashboard";
  const [kicker, title] = viewMeta[activeView];
  el.viewKicker.textContent = kicker;
  el.viewTitle.textContent = title;
  el.navItems.forEach((button) => {
    button.classList.toggle("active", button.dataset.view === activeView);
  });
  el.views.forEach((view) => {
    view.classList.toggle("active", view.id === `view-${activeView}`);
  });
}

function renderTasks() {
  const running = tasks.filter((task) => task.status === "running");
  el.taskCount.textContent = `${tasks.length}`;
  el.runningCount.textContent = `${running.length} 个运行中`;
  el.activeSpeed.textContent = running.find((task) => task.speedText)?.speedText || "0 B/s";
  if (!tasks.length) {
    el.taskList.innerHTML = `<div class="empty">暂无任务。</div>`;
    renderDetail();
    return;
  }

  el.taskList.innerHTML = tasks.map((task) => `
    <button class="task-card ${task.id === selectedTaskId ? "selected" : ""}" data-task-id="${task.id}" type="button">
      <div class="task-row">
        <strong>${escapeHTML(task.title)}</strong>
        <span class="badge ${task.status}">${taskStatusText(task.status)}</span>
      </div>
      <div class="task-url">${escapeHTML(task.request.url)}</div>
      <div class="progress">
        <span style="width:${Math.round(task.progress * 100)}%"></span>
      </div>
      <div class="task-meta">
        <span>${task.progressText || `${Math.round(task.progress * 100)}%`}</span>
        <span>${escapeHTML(task.speedText || "")}</span>
      </div>
      <div class="task-message">${escapeHTML(task.lastMessage || "")}</div>
    </button>
  `).join("");

  el.taskList.querySelectorAll("[data-task-id]").forEach((button) => {
    button.addEventListener("click", () => {
      selectedTaskId = button.dataset.taskId;
      renderTasks();
      renderDetail();
    });
  });

  renderDetail();
}

function renderDetail() {
  const task = selectedTask();
  if (!task) {
    el.detailTitle.textContent = "任务详情";
    el.taskSummary.className = "summary empty";
    el.taskSummary.textContent = "请选择一个任务。";
    el.fileList.className = "file-list empty";
    el.fileList.textContent = "暂无文件。";
    el.log.textContent = "";
    return;
  }

  el.detailTitle.textContent = task.title;
  const commandLine = task.commandLine || "任务开始后生成";
  el.taskSummary.className = "summary";
  el.taskSummary.innerHTML = `
    <div><span>状态</span><strong>${taskStatusText(task.status)}</strong></div>
    <div><span>进度</span><strong>${task.progressText || `${Math.round(task.progress * 100)}%`}</strong></div>
    <div><span>速度</span><strong>${escapeHTML(task.speedText || "0 B/s")}</strong></div>
    <div><span>输出目录</span><strong>${escapeHTML(task.request.saveDir)}</strong></div>
    <div><span>保存名</span><strong>${escapeHTML(task.request.saveName || "自动")}</strong></div>
    <div><span>参数</span><strong>${task.request.autoSelect ? "自动选轨" : "手动选轨"} / ${task.request.muxMP4 ? "MP4" : "原始输出"}</strong></div>
    <div><span>开始时间</span><strong>${formatDate(task.startedAt || task.createdAt)}</strong></div>
    <div><span>结束时间</span><strong>${formatDate(task.finishedAt)}</strong></div>
    <div class="summary-command"><span>命令</span><code title="${escapeHTML(commandLine)}">${escapeHTML(commandLine)}</code></div>
  `;
  renderFiles(task.files);
  el.log.textContent = (task.logs || []).join("\n");
  el.log.scrollTop = el.log.scrollHeight;
}

function renderFiles(files) {
  if (!files || !files.length) {
    el.fileList.className = "file-list empty";
    el.fileList.textContent = "暂无文件。下载完成后会显示输出文件、meta 和字幕。";
    return;
  }
  el.fileList.className = "file-list";
  el.fileList.innerHTML = files.map((file) => `
    <div class="file-row">
      <div>
        <strong>${escapeHTML(file.name)}</strong>
        <span>${formatSize(file.size)} · ${escapeHTML(file.extension || "")}</span>
      </div>
      <div class="file-actions">
        <button type="button" data-open-file="${escapeHTML(file.path)}">打开</button>
        <button type="button" data-delete-file="${escapeHTML(file.path)}">删除</button>
      </div>
    </div>
  `).join("");

  el.fileList.querySelectorAll("[data-open-file]").forEach((button) => {
    button.addEventListener("click", () => api().RevealPath(button.dataset.openFile));
  });
  el.fileList.querySelectorAll("[data-delete-file]").forEach((button) => {
    button.addEventListener("click", async () => {
      if (!window.confirm("确认删除这个输出文件？")) return;
      await api().DeleteTaskFile(selectedTaskId, button.dataset.deleteFile);
      await refreshFiles();
    });
  });
}

function selectedTask() {
  return tasks.find((task) => task.id === selectedTaskId) || null;
}

function upsertTask(rawTask) {
  const task = normalizeTask(rawTask);
  const index = tasks.findIndex((item) => item.id === task.id);
  if (index >= 0) {
    tasks[index] = task;
  } else {
    tasks.unshift(task);
  }
  if (!selectedTaskId) {
    selectedTaskId = task.id;
  }
  renderTasks();
}

function collectRequest() {
  return {
    url: el.url.value.trim(),
    saveDir: el.saveDir.value.trim(),
    saveName: el.saveName.value.trim(),
    linkNameSeparator: el.linkNameSeparator.value.trim(),
    customRange: el.customRange.value.trim(),
    ffmpegPath: el.ffmpegPath.value.trim(),
    headers: el.headers.value.split(/\r?\n/).map((line) => line.trim()).filter(Boolean),
    threadCount: Number(el.threadCount.value) || 0,
    retryCount: Number(el.retryCount.value) || 0,
    maxSpeed: el.maxSpeed.value.trim(),
    customProxy: el.customProxy.value.trim(),
    autoSelect: el.autoSelect.checked,
    muxMP4: el.muxMP4.checked,
    binaryMerge: el.binaryMerge.checked,
    concurrentDownload: el.concurrentDownload.checked,
    useSystemProxy: el.useSystemProxy.checked
  };
}

function collectSettings() {
  return {
    defaultSaveDir: el.saveDir.value.trim(),
    ffmpegPath: el.ffmpegPath.value.trim(),
    threadCount: Number(el.threadCount.value) || 8,
    retryCount: Number(el.retryCount.value) || 3,
    maxSpeed: el.maxSpeed.value.trim(),
    autoSelect: el.autoSelect.checked,
    muxMP4: el.muxMP4.checked,
    binaryMerge: el.binaryMerge.checked,
    concurrentDownload: el.concurrentDownload.checked,
    useSystemProxy: el.useSystemProxy.checked,
    customProxy: el.customProxy.value.trim()
  };
}

function applySettings(nextSettings) {
  settings = nextSettings || {};
  el.saveDir.value = field(settings, "defaultSaveDir", "DefaultSaveDir", "");
  el.ffmpegPath.value = field(settings, "ffmpegPath", "FFmpegPath", "");
  el.threadCount.value = field(settings, "threadCount", "ThreadCount", 8);
  el.retryCount.value = field(settings, "retryCount", "RetryCount", 3);
  el.maxSpeed.value = field(settings, "maxSpeed", "MaxSpeed", "");
  el.autoSelect.checked = Boolean(field(settings, "autoSelect", "AutoSelect", true));
  el.muxMP4.checked = Boolean(field(settings, "muxMP4", "MuxMP4", true));
  el.binaryMerge.checked = Boolean(field(settings, "binaryMerge", "BinaryMerge", false));
  el.concurrentDownload.checked = Boolean(field(settings, "concurrentDownload", "ConcurrentDownload", false));
  el.useSystemProxy.checked = Boolean(field(settings, "useSystemProxy", "UseSystemProxy", true));
  el.customProxy.value = field(settings, "customProxy", "CustomProxy", "");
}

function applyCoreInfo(info = {}) {
  const status = field(info, "status", "Status", "error");
  const error = field(info, "error", "Error", "");
  const fullVersion = field(info, "fullVersion", "FullVersion", "");
  el.coreStatus.textContent = status === "ready" ? "可用" : "异常";
  el.coreStatus.classList.toggle("error", status !== "ready");
  el.coreVersion.textContent = fullVersion || "-";
  el.corePath.textContent = field(info, "cliPath", "CLIPath", "") || error || "-";
  el.corePath.title = el.corePath.textContent;
}

function applyToolCheck(node, info = {}) {
  const status = field(info, "status", "Status", "error");
  const path = field(info, "path", "Path", "");
  const version = field(info, "version", "Version", "");
  const error = field(info, "error", "Error", "");
  node.classList.toggle("ok", status === "ready");
  node.classList.toggle("error", status !== "ready");
  node.textContent = status === "ready" ? `${version || "可用"} · ${path}` : (error || "检测失败");
  node.title = node.textContent;
}

function renderToolList(rawTools = defaultTools, pendingText = "尚未检测") {
  const tools = rawTools.map(normalizeToolInfo);
  el.toolList.innerHTML = tools.map((tool) => {
    const ready = tool.status === "ready";
    const failed = tool.status === "error";
    const detail = ready ? [tool.version, tool.path].filter(Boolean).join(" · ") : (tool.error || pendingText);
    const statusText = ready ? "可用" : (failed ? "不可用" : pendingText);
    return `
      <div class="tool-row ${ready ? "ready" : ""} ${failed ? "failed" : ""}">
        <div>
          <strong>${escapeHTML(tool.label)}</strong>
          <span title="${escapeHTML(detail)}">${escapeHTML(detail)}</span>
        </div>
        <b>${escapeHTML(statusText)}</b>
      </div>
    `;
  }).join("");
}

function renderPreflight(report = {}) {
  const status = field(report, "status", "Status", "error");
  const message = field(report, "message", "Message", "预检查失败");
  const taskCount = Number(field(report, "taskCount", "TaskCount", 0)) || 0;
  const checks = field(report, "checks", "Checks", []).map(normalizePreflightCheck);
  const commandLines = field(report, "commandLines", "CommandLines", []);
  el.preflightResult.className = `preflight-card field-wide ${status}`;
  el.preflightResult.innerHTML = `
    <div class="preflight-head">
      <strong>${escapeHTML(message)}</strong>
      <span>${taskCount ? `${taskCount} 个任务` : ""}</span>
    </div>
    <div class="preflight-list">
      ${checks.map((check) => `
        <div class="preflight-row ${escapeHTML(check.status)}">
          <b>${escapeHTML(check.label || check.name)}</b>
          <span title="${escapeHTML(check.detail || check.message)}">${escapeHTML(check.message || "-")}</span>
        </div>
      `).join("")}
    </div>
    ${commandLines.length ? `<code title="${escapeHTML(commandLines.join("\n"))}">${escapeHTML(commandLines.join("\n"))}</code>` : ""}
  `;
}

async function refreshCoreInfo() {
  el.coreStatus.textContent = "检查中";
  el.coreStatus.classList.remove("error");
  el.coreVersion.textContent = "-";
  el.corePath.textContent = "-";
  applyCoreInfo(await api().GetCoreInfo());
}

async function loadInitialState() {
  applySettings(await api().GetSettings());
  renderToolList();
  await refreshCoreInfo();
  tasks = (await api().ListTasks()).map(normalizeTask);
  if (tasks.length && !selectedTaskId) {
    selectedTaskId = tasks[0].id;
  }
  renderTasks();
}

async function createTask(startNow) {
  const request = collectRequest();
  const rawTasks = startNow ? await api().StartDownloads(request) : await api().CreateTasks(request);
  const createdTasks = (Array.isArray(rawTasks) ? rawTasks : [rawTasks]).map(normalizeTask);
  createdTasks.forEach((task) => upsertTask(task));
  selectedTaskId = createdTasks[0]?.id || selectedTaskId;
  el.url.value = "";
  switchView("dashboard");
}

async function refreshTasks() {
  tasks = (await api().ListTasks()).map(normalizeTask);
  if (selectedTaskId && !tasks.some((task) => task.id === selectedTaskId)) {
    selectedTaskId = tasks[0]?.id || "";
  }
  renderTasks();
}

async function refreshFiles() {
  if (!selectedTaskId) return;
  const files = normalizeFiles(await api().ListTaskFiles(selectedTaskId));
  const task = selectedTask();
  if (task) {
    task.files = files;
    renderDetail();
  }
}

function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

async function copyText(text, successMessage) {
  const value = String(text || "").trim();
  if (!value) {
    window.alert("没有可复制的内容。");
    return;
  }
  let copied = false;
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      copied = true;
    } catch {
      copied = false;
    }
  }
  if (!copied) {
    const textarea = document.createElement("textarea");
    textarea.value = value;
    textarea.setAttribute("readonly", "");
    textarea.style.position = "fixed";
    textarea.style.left = "-9999px";
    document.body.appendChild(textarea);
    textarea.select();
    copied = document.execCommand("copy");
    textarea.remove();
    if (!copied) {
      throw new Error("复制失败，请手动选择日志或命令。");
    }
  }
  if (successMessage) {
    window.alert(successMessage);
  }
}

window.runtime?.EventsOn("task:update", (rawTask) => upsertTask(rawTask));
window.runtime?.EventsOn("task:log", (event) => {
  const taskId = field(event, "taskId", "TaskID");
  const line = field(event, "line", "Line");
  const task = tasks.find((item) => item.id === taskId);
  if (task) {
    const logs = task.logs || [];
    task.logs = logs[logs.length - 1] === line ? logs : [...logs, line].slice(-500);
    task.lastMessage = line;
  }
  if (taskId === selectedTaskId) {
    const existing = el.log.textContent.trimEnd();
    if (!existing.endsWith(line)) {
      el.log.textContent += `${line}\n`;
      el.log.scrollTop = el.log.scrollHeight;
    }
  }
});
window.runtime?.EventsOn("task:files", (event) => {
  const taskId = field(event, "taskId", "TaskID");
  const files = normalizeFiles(field(event, "files", "Files", []));
  const task = tasks.find((item) => item.id === taskId);
  if (task) {
    task.files = files;
  }
  if (taskId === selectedTaskId) {
    renderDetail();
  }
});

document.querySelector("#task-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  try {
    await createTask(!createOnly);
  } catch (error) {
    window.alert(String(error?.message || error));
  } finally {
    createOnly = false;
  }
});

document.querySelector("#create-only").addEventListener("click", () => {
  createOnly = true;
  document.querySelector("#task-form").requestSubmit();
});
document.querySelector("#preflight-task").addEventListener("click", async () => {
  el.preflightResult.className = "preflight-card field-wide";
  el.preflightResult.textContent = "正在预检查";
  try {
    renderPreflight(await api().PreflightDownload(collectRequest()));
  } catch (error) {
    renderPreflight({
      status: "error",
      message: "预检查失败",
      checks: [{ name: "preflight", label: "预检查", status: "error", message: String(error?.message || error) }]
    });
  }
});
document.querySelector("#copy-form-command").addEventListener("click", async () => {
  try {
    const commandText = await api().PreviewCommands(collectRequest());
    await copyText(commandText, "当前任务命令已复制。");
  } catch (error) {
    window.alert(String(error?.message || error));
  }
});

document.querySelector("#choose-dir").addEventListener("click", async () => {
  const selected = await api().ChooseDirectory(el.saveDir.value.trim());
  if (selected) {
    el.saveDir.value = selected;
  }
});

document.querySelector("#save-settings").addEventListener("click", async () => {
  applySettings(await api().SaveSettings(collectSettings()));
});
el.checkFFmpeg.addEventListener("click", async () => {
  el.ffmpegCheck.classList.remove("ok", "error");
  el.ffmpegCheck.textContent = "检测中";
  try {
    applyToolCheck(el.ffmpegCheck, await api().CheckFFmpeg(el.ffmpegPath.value.trim()));
  } catch (error) {
    applyToolCheck(el.ffmpegCheck, { status: "error", error: String(error?.message || error) });
  }
});
el.checkTools.addEventListener("click", async () => {
  renderToolList(defaultTools, "检测中");
  try {
    renderToolList(await api().CheckTools(el.ffmpegPath.value.trim()));
  } catch (error) {
    renderToolList(defaultTools.map((tool) => ({
      ...tool,
      status: "error",
      error: String(error?.message || error)
    })));
  }
});
el.refreshCore.addEventListener("click", () => {
  refreshCoreInfo().catch((error) => {
    applyCoreInfo({ status: "error", error: String(error?.message || error) });
  });
});

document.querySelector("#refresh-tasks").addEventListener("click", refreshTasks);
document.querySelector("#clear-finished").addEventListener("click", async () => {
  await api().ClearFinishedTasks();
  await refreshTasks();
});
document.querySelector("#start-selected").addEventListener("click", async () => {
  if (selectedTaskId) await api().StartTask(selectedTaskId);
});
document.querySelector("#stop-selected").addEventListener("click", async () => {
  if (selectedTaskId) await api().StopTask(selectedTaskId);
});
document.querySelector("#retry-selected").addEventListener("click", async () => {
  if (selectedTaskId) await api().RetryTask(selectedTaskId);
});
el.copyCommand.addEventListener("click", async () => {
  const task = selectedTask();
  if (!task) return;
  try {
    await copyText(task.commandLine, "任务命令已复制。");
  } catch (error) {
    window.alert(String(error?.message || error));
  }
});
el.copyLog.addEventListener("click", async () => {
  const task = selectedTask();
  if (!task) return;
  try {
    await copyText((task.logs || []).join("\n"), "任务日志已复制。");
  } catch (error) {
    window.alert(String(error?.message || error));
  }
});
document.querySelector("#remove-selected").addEventListener("click", async () => {
  if (!selectedTaskId || !window.confirm("确认移除这个任务？输出文件不会删除。")) return;
  await api().RemoveTask(selectedTaskId);
  selectedTaskId = "";
  await refreshTasks();
});
document.querySelector("#refresh-files").addEventListener("click", refreshFiles);
document.querySelector("#open-folder").addEventListener("click", async () => {
  if (selectedTaskId) await api().OpenTaskFolder(selectedTaskId);
});
document.querySelector("#clear-log").addEventListener("click", () => {
  el.log.textContent = "";
});
el.navItems.forEach((button) => {
  button.addEventListener("click", () => switchView(button.dataset.view));
});
document.querySelectorAll("[data-view-shortcut]").forEach((button) => {
  button.addEventListener("click", () => switchView(button.dataset.viewShortcut));
});

loadInitialState().catch((error) => {
  el.taskList.innerHTML = `<div class="empty">初始化失败：${escapeHTML(error)}</div>`;
});
