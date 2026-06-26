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
  maxActiveTasks: document.querySelector("#maxActiveTasks"),
  threadCount: document.querySelector("#threadCount"),
  retryCount: document.querySelector("#retryCount"),
  maxSpeed: document.querySelector("#maxSpeed"),
  customProxy: document.querySelector("#customProxy"),
  useSystemProxy: document.querySelector("#useSystemProxy"),
  coreStatus: document.querySelector("#core-status"),
  coreVersion: document.querySelector("#core-version"),
  corePath: document.querySelector("#core-path"),
  refreshCore: document.querySelector("#refresh-core"),
  refreshTasks: document.querySelector("#refresh-tasks"),
  startPending: document.querySelector("#start-pending"),
  stopRunning: document.querySelector("#stop-running"),
  retryFailed: document.querySelector("#retry-failed"),
  clearFinished: document.querySelector("#clear-finished"),
  copyCommand: document.querySelector("#copy-command"),
  copyLog: document.querySelector("#copy-log"),
  exportLog: document.querySelector("#export-log"),
  startSelected: document.querySelector("#start-selected"),
  stopSelected: document.querySelector("#stop-selected"),
  retrySelected: document.querySelector("#retry-selected"),
  removeSelected: document.querySelector("#remove-selected"),
  refreshFiles: document.querySelector("#refresh-files"),
  openFolder: document.querySelector("#open-folder"),
  clearLog: document.querySelector("#clear-log"),
  taskSearch: document.querySelector("#task-search"),
  taskStatusFilter: document.querySelector("#task-status-filter"),
  taskSort: document.querySelector("#task-sort"),
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
  toastHost: document.querySelector("#toast-host"),
  navItems: Array.from(document.querySelectorAll("[data-view]")),
  views: Array.from(document.querySelectorAll(".view"))
};

let settings = {};
let tasks = [];
let selectedTaskId = "";
let createOnly = false;
let createActionButton = null;
let activeView = "dashboard";
let taskSearchText = "";
let taskStatusFilter = "all";
let taskSortMode = "newest";

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
    queued: Boolean(field(task, "queued", "Queued", false)),
    progress: Number(field(task, "progress", "Progress", 0)) || 0,
    progressText: field(task, "progressText", "ProgressText"),
    speedText: field(task, "speedText", "SpeedText"),
    elapsedText: field(task, "elapsedText", "ElapsedText"),
    remainingText: field(task, "remainingText", "RemainingText"),
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

function taskDisplayStatusText(task) {
  if (task?.status === "pending" && task?.queued) {
    return "排队中";
  }
  return taskStatusText(task?.status);
}

function taskTimingText(task) {
  const parts = [];
  if (task?.elapsedText) {
    parts.push(`耗时 ${task.elapsedText}`);
  }
  if (task?.remainingText) {
    parts.push(`剩余 ${task.remainingText}`);
  }
  return parts.join(" · ");
}

function taskTimeValue(value) {
  if (!value) return 0;
  const time = new Date(value).getTime();
  return Number.isNaN(time) ? 0 : time;
}

function taskSortRank(task) {
  if (task?.status === "running") return 0;
  if (task?.status === "pending" && task?.queued) return 1;
  if (task?.status === "pending") return 2;
  if (task?.status === "failed") return 3;
  if (task?.status === "stopped") return 4;
  if (task?.status === "completed") return 5;
  return 6;
}

function compareTasks(left, right) {
  const newest = taskTimeValue(right.createdAt) - taskTimeValue(left.createdAt);
  switch (taskSortMode) {
    case "oldest":
      return taskTimeValue(left.createdAt) - taskTimeValue(right.createdAt);
    case "status": {
      const rank = taskSortRank(left) - taskSortRank(right);
      return rank || newest;
    }
    case "progress": {
      const progress = Number(right.progress || 0) - Number(left.progress || 0);
      return progress || newest;
    }
    case "title": {
      const title = String(left.title || "").localeCompare(String(right.title || ""), "zh-CN");
      return title || newest;
    }
    case "newest":
    default:
      return newest;
  }
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

function errorMessage(error) {
  return String(error?.message || error || "操作失败");
}

function notify(message, type = "info") {
  const text = String(message || "").trim();
  if (!text || !el.toastHost) return;
  const toast = document.createElement("div");
  toast.className = `toast ${type}`;
  toast.textContent = text;
  el.toastHost.appendChild(toast);
  window.setTimeout(() => {
    toast.remove();
  }, type === "error" ? 5200 : 3200);
}

function reportError(error) {
  notify(errorMessage(error), "error");
}

async function withBusy(button, busyText, action) {
  if (!button) {
    return action();
  }
  if (button.disabled) {
    return undefined;
  }
  const previousText = button.textContent;
  button.disabled = true;
  button.setAttribute("aria-disabled", "true");
  button.dataset.busy = "true";
  if (busyText) {
    button.textContent = busyText;
  }
  try {
    return await action();
  } finally {
    button.disabled = false;
    button.setAttribute("aria-disabled", "false");
    delete button.dataset.busy;
    button.textContent = previousText;
    updateToolbarActionStates();
    updateSelectedTaskActions();
  }
}

function setActionState(button, enabled, reason = "") {
  if (!button || button.dataset.busy === "true") return;
  button.disabled = !enabled;
  button.setAttribute("aria-disabled", enabled ? "false" : "true");
  button.title = enabled ? "" : reason;
}

function updateSelectedTaskActions() {
  const task = selectedTask();
  const hasTask = Boolean(task);
  const isRunning = task?.status === "running";
  const isQueued = task?.status === "pending" && task?.queued;
  const canStart = hasTask && !isRunning && !isQueued && task.status !== "failed";
  const canStop = hasTask && (isRunning || isQueued);
  const canRetry = hasTask && task.status === "failed";
  const canRemove = hasTask && !isRunning;
  const hasLogs = hasTask && Array.isArray(task.logs) && task.logs.length > 0;
  const startDisabledReason = hasTask && task.status === "failed" ? "失败任务请使用重试" : "运行中或排队中的任务不能重复开始";

  setActionState(el.copyCommand, hasTask && Boolean(task.commandLine), hasTask ? "当前任务还没有生成命令" : "请先选择一个任务");
  setActionState(el.copyLog, hasLogs, hasTask ? "当前任务没有日志" : "请先选择一个任务");
  setActionState(el.startSelected, canStart, hasTask ? startDisabledReason : "请先选择一个任务");
  setActionState(el.stopSelected, canStop, hasTask ? "只有运行中或排队中的任务可以停止" : "请先选择一个任务");
  setActionState(el.retrySelected, canRetry, hasTask ? "只有失败任务可以重试" : "请先选择一个任务");
  setActionState(el.removeSelected, canRemove, hasTask ? "运行中的任务不能移除" : "请先选择一个任务");
  setActionState(el.exportLog, hasLogs, hasTask ? "当前任务没有日志可导出" : "请先选择一个任务");
  setActionState(el.clearLog, hasLogs, hasTask ? "当前任务没有日志可清空" : "请先选择一个任务");
  setActionState(el.refreshFiles, hasTask, "请先选择一个任务");
  setActionState(el.openFolder, hasTask, "请先选择一个任务");
}

function updateToolbarActionStates() {
  const hasPending = tasks.some((task) => task.status === "pending");
  const hasRunning = tasks.some((task) => task.status === "running");
  const hasFailed = tasks.some((task) => task.status === "failed");
  const hasFinished = tasks.some((task) => ["completed", "failed", "stopped"].includes(task.status));

  setActionState(el.refreshTasks, true);
  setActionState(el.startPending, hasPending, "没有等待中的任务");
  setActionState(el.stopRunning, hasRunning, "没有运行中的任务");
  setActionState(el.retryFailed, hasFailed, "没有失败任务");
  setActionState(el.clearFinished, hasFinished, "没有可清理的完成、失败或停止任务");
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
  const visibleTasks = tasks.filter(taskMatchesCurrentFilter).slice().sort(compareTasks);
  el.taskCount.textContent = visibleTasks.length === tasks.length ? `${tasks.length}` : `${visibleTasks.length}/${tasks.length}`;
  el.runningCount.textContent = `${running.length} 个运行中`;
  el.activeSpeed.textContent = running.find((task) => task.speedText)?.speedText || "0 B/s";
  updateToolbarActionStates();
  if (!tasks.length) {
    el.taskList.innerHTML = `<div class="empty">暂无任务。</div>`;
    renderDetail();
    return;
  }
  if (!visibleTasks.length) {
    el.taskList.innerHTML = `<div class="empty">没有匹配的任务。</div>`;
    renderDetail();
    return;
  }

  el.taskList.innerHTML = visibleTasks.map((task) => `
    <button class="task-card ${task.id === selectedTaskId ? "selected" : ""}" data-task-id="${task.id}" type="button">
      <div class="task-row">
        <strong>${escapeHTML(task.title)}</strong>
        <span class="badge ${task.status} ${task.queued ? "queued" : ""}">${taskDisplayStatusText(task)}</span>
      </div>
      <div class="task-url">${escapeHTML(task.request.url)}</div>
      <div class="progress">
        <span style="width:${Math.round(task.progress * 100)}%"></span>
      </div>
      <div class="task-meta">
        <span>${task.progressText || `${Math.round(task.progress * 100)}%`}</span>
        <span>${escapeHTML(task.speedText || "")}</span>
      </div>
      ${taskTimingText(task) ? `<div class="task-time">${escapeHTML(taskTimingText(task))}</div>` : ""}
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

function taskMatchesCurrentFilter(task) {
  if (taskStatusFilter !== "all" && task.status !== taskStatusFilter) {
    return false;
  }
  if (!taskSearchText) {
    return true;
  }
  const request = task.request || {};
  const text = [
    task.title,
    taskDisplayStatusText(task),
    task.lastMessage,
    task.progressText,
    task.speedText,
    task.elapsedText,
    task.remainingText,
    request.url,
    request.saveName,
    request.saveDir
  ].join("\n").toLowerCase();
  return text.includes(taskSearchText);
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
    updateSelectedTaskActions();
    return;
  }

  el.detailTitle.textContent = task.title;
  const commandLine = task.commandLine || "任务开始后生成";
  el.taskSummary.className = "summary";
  el.taskSummary.innerHTML = `
    <div><span>状态</span><strong>${taskDisplayStatusText(task)}</strong></div>
    <div><span>进度</span><strong>${task.progressText || `${Math.round(task.progress * 100)}%`}</strong></div>
    <div><span>速度</span><strong>${escapeHTML(task.speedText || "0 B/s")}</strong></div>
    <div><span>耗时</span><strong>${escapeHTML(task.elapsedText || "-")}</strong></div>
    <div><span>预计剩余</span><strong>${escapeHTML(task.remainingText || "-")}</strong></div>
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
  updateSelectedTaskActions();
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
    button.addEventListener("click", () => {
      withBusy(button, "打开中", async () => {
        try {
          await api().RevealPath(button.dataset.openFile);
        } catch (error) {
          reportError(error);
        }
      });
    });
  });
  el.fileList.querySelectorAll("[data-delete-file]").forEach((button) => {
    button.addEventListener("click", async () => {
      if (!window.confirm("确认删除这个输出文件？")) return;
      await withBusy(button, "删除中", async () => {
        try {
          await api().DeleteTaskFile(selectedTaskId, button.dataset.deleteFile);
          await refreshFiles();
          notify("文件已删除。", "success");
        } catch (error) {
          reportError(error);
        }
      });
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
    maxActiveTasks: Number(el.maxActiveTasks.value) || 2,
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
  el.maxActiveTasks.value = field(settings, "maxActiveTasks", "MaxActiveTasks", 2);
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
    notify("没有可复制的内容。", "warning");
    return false;
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
    notify(successMessage, "success");
  }
  return true;
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
  const startNow = !createOnly;
  const button = createActionButton || event.submitter || document.querySelector("#create-start");
  try {
    await withBusy(button, "创建中", async () => {
      await createTask(startNow);
      notify(startNow ? "任务已创建并开始下载。" : "任务已创建。", "success");
    });
  } catch (error) {
    reportError(error);
  } finally {
    createOnly = false;
    createActionButton = null;
  }
});

document.querySelector("#create-only").addEventListener("click", () => {
  createOnly = true;
  createActionButton = document.querySelector("#create-only");
  document.querySelector("#task-form").requestSubmit();
});
document.querySelector("#preflight-task").addEventListener("click", async () => {
  await withBusy(document.querySelector("#preflight-task"), "检查中", async () => {
    el.preflightResult.className = "preflight-card field-wide";
    el.preflightResult.textContent = "正在预检查";
    try {
      renderPreflight(await api().PreflightDownload(collectRequest()));
      notify("预检查完成。", "success");
    } catch (error) {
      renderPreflight({
        status: "error",
        message: "预检查失败",
        checks: [{ name: "preflight", label: "预检查", status: "error", message: errorMessage(error) }]
      });
      reportError(error);
    }
  });
});
document.querySelector("#copy-form-command").addEventListener("click", async () => {
  await withBusy(document.querySelector("#copy-form-command"), "复制中", async () => {
    try {
      const commandText = await api().PreviewCommands(collectRequest());
      await copyText(commandText, "当前任务命令已复制。");
    } catch (error) {
      reportError(error);
    }
  });
});

document.querySelector("#choose-dir").addEventListener("click", async () => {
  await withBusy(document.querySelector("#choose-dir"), "选择中", async () => {
    try {
      const selected = await api().ChooseDirectory(el.saveDir.value.trim());
      if (selected) {
        el.saveDir.value = selected;
      }
    } catch (error) {
      reportError(error);
    }
  });
});

document.querySelector("#save-settings").addEventListener("click", async () => {
  await withBusy(document.querySelector("#save-settings"), "保存中", async () => {
    try {
      applySettings(await api().SaveSettings(collectSettings()));
      notify("设置已保存。", "success");
    } catch (error) {
      reportError(error);
    }
  });
});
el.checkFFmpeg.addEventListener("click", async () => {
  await withBusy(el.checkFFmpeg, "检测中", async () => {
    el.ffmpegCheck.classList.remove("ok", "error");
    el.ffmpegCheck.textContent = "检测中";
    try {
      applyToolCheck(el.ffmpegCheck, await api().CheckFFmpeg(el.ffmpegPath.value.trim()));
      notify("FFmpeg 检测完成。", "success");
    } catch (error) {
      applyToolCheck(el.ffmpegCheck, { status: "error", error: errorMessage(error) });
      reportError(error);
    }
  });
});
el.checkTools.addEventListener("click", async () => {
  await withBusy(el.checkTools, "检测中", async () => {
    renderToolList(defaultTools, "检测中");
    try {
      renderToolList(await api().CheckTools(el.ffmpegPath.value.trim()));
      notify("外部工具检测完成。", "success");
    } catch (error) {
      renderToolList(defaultTools.map((tool) => ({
        ...tool,
        status: "error",
        error: errorMessage(error)
      })));
      reportError(error);
    }
  });
});
el.refreshCore.addEventListener("click", () => {
  withBusy(el.refreshCore, "刷新中", async () => {
    try {
      await refreshCoreInfo();
      notify("下载核心状态已刷新。", "success");
    } catch (error) {
      applyCoreInfo({ status: "error", error: errorMessage(error) });
      reportError(error);
    }
  });
});
el.taskSearch.addEventListener("input", () => {
  taskSearchText = el.taskSearch.value.trim().toLowerCase();
  renderTasks();
});
el.taskStatusFilter.addEventListener("change", () => {
  taskStatusFilter = el.taskStatusFilter.value || "all";
  renderTasks();
});
el.taskSort.addEventListener("change", () => {
  taskSortMode = el.taskSort.value || "newest";
  renderTasks();
});

el.refreshTasks.addEventListener("click", async () => {
  await withBusy(el.refreshTasks, "刷新中", async () => {
    try {
      await refreshTasks();
      notify("任务列表已刷新。", "success");
    } catch (error) {
      reportError(error);
    }
  });
});
el.startPending.addEventListener("click", async () => {
  await withBusy(el.startPending, "开始中", async () => {
    try {
      const count = await api().StartPendingTasks();
      await refreshTasks();
      notify(count ? `已开始 ${count} 个等待任务。` : "没有等待中的任务。", count ? "success" : "warning");
    } catch (error) {
      reportError(error);
    }
  });
});
el.stopRunning.addEventListener("click", async () => {
  if (!window.confirm("确认停止所有运行中的任务？")) return;
  await withBusy(el.stopRunning, "停止中", async () => {
    try {
      const count = await api().StopRunningTasks();
      await refreshTasks();
      notify(count ? `已请求停止 ${count} 个运行中任务。` : "没有运行中的任务。", count ? "success" : "warning");
    } catch (error) {
      reportError(error);
    }
  });
});
el.retryFailed.addEventListener("click", async () => {
  await withBusy(el.retryFailed, "重试中", async () => {
    try {
      const count = await api().RetryFailedTasks();
      await refreshTasks();
      notify(count ? `已重试 ${count} 个失败任务。` : "没有失败任务。", count ? "success" : "warning");
    } catch (error) {
      reportError(error);
    }
  });
});
el.clearFinished.addEventListener("click", async () => {
  await withBusy(el.clearFinished, "清理中", async () => {
    try {
      await api().ClearFinishedTasks();
      await refreshTasks();
      notify("已清理完成、失败和停止的任务。", "success");
    } catch (error) {
      reportError(error);
    }
  });
});
el.startSelected.addEventListener("click", async () => {
  if (!selectedTaskId) {
    notify("请先选择一个任务。", "warning");
    return;
  }
  await withBusy(el.startSelected, "开始中", async () => {
    try {
      await api().StartTask(selectedTaskId);
      await refreshTasks();
      notify("任务已开始。", "success");
    } catch (error) {
      reportError(error);
    }
  });
});
el.stopSelected.addEventListener("click", async () => {
  if (!selectedTaskId) {
    notify("请先选择一个任务。", "warning");
    return;
  }
  await withBusy(el.stopSelected, "停止中", async () => {
    try {
      await api().StopTask(selectedTaskId);
      await refreshTasks();
      notify("已请求停止任务。", "success");
    } catch (error) {
      reportError(error);
    }
  });
});
el.retrySelected.addEventListener("click", async () => {
  if (!selectedTaskId) {
    notify("请先选择一个任务。", "warning");
    return;
  }
  await withBusy(el.retrySelected, "重试中", async () => {
    try {
      await api().RetryTask(selectedTaskId);
      await refreshTasks();
      notify("任务已重新排队。", "success");
    } catch (error) {
      reportError(error);
    }
  });
});
el.copyCommand.addEventListener("click", async () => {
  const task = selectedTask();
  if (!task) {
    notify("请先选择一个任务。", "warning");
    return;
  }
  await withBusy(el.copyCommand, "复制中", async () => {
    try {
      await copyText(task.commandLine, "任务命令已复制。");
    } catch (error) {
      reportError(error);
    }
  });
});
el.copyLog.addEventListener("click", async () => {
  const task = selectedTask();
  if (!task) {
    notify("请先选择一个任务。", "warning");
    return;
  }
  await withBusy(el.copyLog, "复制中", async () => {
    try {
      await copyText((task.logs || []).join("\n"), "任务日志已复制。");
    } catch (error) {
      reportError(error);
    }
  });
});
el.exportLog.addEventListener("click", async () => {
  if (!selectedTaskId) {
    notify("请先选择一个任务。", "warning");
    return;
  }
  await withBusy(el.exportLog, "导出中", async () => {
    try {
      const file = normalizeFiles([await api().ExportTaskLog(selectedTaskId)])[0];
      await refreshFiles();
      notify(`日志已导出：${file?.name || "完成"}`, "success");
    } catch (error) {
      reportError(error);
    }
  });
});
el.removeSelected.addEventListener("click", async () => {
  if (!selectedTaskId || !window.confirm("确认移除这个任务？输出文件不会删除。")) return;
  await withBusy(el.removeSelected, "移除中", async () => {
    try {
      await api().RemoveTask(selectedTaskId);
      selectedTaskId = "";
      await refreshTasks();
      notify("任务已移除。", "success");
    } catch (error) {
      reportError(error);
    }
  });
});
el.refreshFiles.addEventListener("click", async () => {
  await withBusy(el.refreshFiles, "刷新中", async () => {
    try {
      await refreshFiles();
      notify("文件列表已刷新。", "success");
    } catch (error) {
      reportError(error);
    }
  });
});
el.openFolder.addEventListener("click", async () => {
  if (!selectedTaskId) {
    notify("请先选择一个任务。", "warning");
    return;
  }
  await withBusy(el.openFolder, "打开中", async () => {
    try {
      await api().OpenTaskFolder(selectedTaskId);
    } catch (error) {
      reportError(error);
    }
  });
});
el.clearLog.addEventListener("click", async () => {
  if (!selectedTaskId) {
    notify("请先选择一个任务。", "warning");
    return;
  }
  await withBusy(el.clearLog, "清空中", async () => {
    try {
      const task = normalizeTask(await api().ClearTaskLog(selectedTaskId));
      upsertTask(task);
      el.log.textContent = "";
      notify("任务日志已清空。", "success");
    } catch (error) {
      reportError(error);
    }
  });
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
