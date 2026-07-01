const api = () => window.go?.main?.App;

const el = {
  url: document.querySelector("#url"),
  baseURL: document.querySelector("#baseURL"),
  saveDir: document.querySelector("#saveDir"),
  tmpDir: document.querySelector("#tmpDir"),
  saveName: document.querySelector("#saveName"),
  savePattern: document.querySelector("#savePattern"),
  linkNameSeparator: document.querySelector("#linkNameSeparator"),
  customRange: document.querySelector("#customRange"),
  headers: document.querySelector("#headers"),
  selectVideo: document.querySelector("#selectVideo"),
  selectAudio: document.querySelector("#selectAudio"),
  selectSubtitle: document.querySelector("#selectSubtitle"),
  dropVideo: document.querySelector("#dropVideo"),
  dropAudio: document.querySelector("#dropAudio"),
  dropSubtitle: document.querySelector("#dropSubtitle"),
  subFormat: document.querySelector("#subFormat"),
  keys: document.querySelector("#keys"),
  keyTextFile: document.querySelector("#keyTextFile"),
  decryptionEngine: document.querySelector("#decryptionEngine"),
  decryptionBinaryPath: document.querySelector("#decryptionBinaryPath"),
  customHLSMethod: document.querySelector("#customHLSMethod"),
  customHLSKey: document.querySelector("#customHLSKey"),
  customHLSIV: document.querySelector("#customHLSIV"),
  adKeywords: document.querySelector("#adKeywords"),
  taskStartAt: document.querySelector("#taskStartAt"),
  liveRecordLimit: document.querySelector("#liveRecordLimit"),
  liveWaitTime: document.querySelector("#liveWaitTime"),
  liveTakeCount: document.querySelector("#liveTakeCount"),
  muxAfterDone: document.querySelector("#muxAfterDone"),
  muxImportPath: document.querySelector("#muxImportPath"),
  muxImportLang: document.querySelector("#muxImportLang"),
  muxImportName: document.querySelector("#muxImportName"),
  muxImports: document.querySelector("#muxImports"),
  autoSelect: document.querySelector("#autoSelect"),
  subOnly: document.querySelector("#subOnly"),
  autoSubtitleFix: document.querySelector("#autoSubtitleFix"),
  mp4RealTimeDecryption: document.querySelector("#mp4RealTimeDecryption"),
  livePerformAsVOD: document.querySelector("#livePerformAsVOD"),
  liveRealTimeMerge: document.querySelector("#liveRealTimeMerge"),
  discardLiveSegments: document.querySelector("#discardLiveSegments"),
  livePipeMux: document.querySelector("#livePipeMux"),
  liveFixVTTByAudio: document.querySelector("#liveFixVTTByAudio"),
  muxMP4: document.querySelector("#muxMP4"),
  noDateInfo: document.querySelector("#noDateInfo"),
  binaryMerge: document.querySelector("#binaryMerge"),
  appendURLParams: document.querySelector("#appendURLParams"),
  skipDownload: document.querySelector("#skipDownload"),
  skipMerge: document.querySelector("#skipMerge"),
  checkSegmentsCount: document.querySelector("#checkSegmentsCount"),
  writeMetaJSON: document.querySelector("#writeMetaJSON"),
  keepSegments: document.querySelector("#keepSegments"),
  noLog: document.querySelector("#noLog"),
  concurrentDownload: document.querySelector("#concurrentDownload"),
  ffmpegPath: document.querySelector("#ffmpegPath"),
  chooseFFmpeg: document.querySelector("#choose-ffmpeg"),
  checkFFmpeg: document.querySelector("#check-ffmpeg"),
  ffmpegCheck: document.querySelector("#ffmpeg-check"),
  checkTools: document.querySelector("#check-tools"),
  toolList: document.querySelector("#tool-list"),
  maxActiveTasks: document.querySelector("#maxActiveTasks"),
  threadCount: document.querySelector("#threadCount"),
  retryCount: document.querySelector("#retryCount"),
  httpRequestTimeout: document.querySelector("#httpRequestTimeout"),
  maxSpeed: document.querySelector("#maxSpeed"),
  customProxy: document.querySelector("#customProxy"),
  useSystemProxy: document.querySelector("#useSystemProxy"),
  coreStatus: document.querySelector("#core-status"),
  coreVersion: document.querySelector("#core-version"),
  corePath: document.querySelector("#core-path"),
  coreCapabilities: document.querySelector("#core-capabilities"),
  refreshCore: document.querySelector("#refresh-core"),
  saveFormSettings: document.querySelector("#save-form-settings"),
  resetForm: document.querySelector("#reset-form"),
  chooseTmpDir: document.querySelector("#choose-tmp-dir"),
  chooseKeyFile: document.querySelector("#choose-key-file"),
  chooseDecryptionBinary: document.querySelector("#choose-decryption-binary"),
  chooseMuxImportFile: document.querySelector("#choose-mux-import-file"),
  addMuxImport: document.querySelector("#add-mux-import"),
  refreshTasks: document.querySelector("#refresh-tasks"),
  startPending: document.querySelector("#start-pending"),
  stopRunning: document.querySelector("#stop-running"),
  retryFailed: document.querySelector("#retry-failed"),
  clearFinished: document.querySelector("#clear-finished"),
  copyCommand: document.querySelector("#copy-command"),
  copyURL: document.querySelector("#copy-url"),
  copyLog: document.querySelector("#copy-log"),
  exportLog: document.querySelector("#export-log"),
  cloneSelected: document.querySelector("#clone-selected"),
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
    baseURL: field(request, "baseURL", "BaseURL"),
    saveDir: field(request, "saveDir", "SaveDir"),
    tmpDir: field(request, "tmpDir", "TmpDir"),
    saveName: field(request, "saveName", "SaveName"),
    savePattern: field(request, "savePattern", "SavePattern"),
    linkNameSeparator: field(request, "linkNameSeparator", "LinkNameSeparator"),
    headers: field(request, "headers", "Headers", []),
    customRange: field(request, "customRange", "CustomRange"),
    ffmpegPath: field(request, "ffmpegPath", "FFmpegPath"),
    httpRequestTimeout: Number(field(request, "httpRequestTimeout", "HTTPRequestTimeout", 0)) || 0,
    threadCount: field(request, "threadCount", "ThreadCount", 0),
    retryCount: field(request, "retryCount", "RetryCount", 0),
    maxSpeed: field(request, "maxSpeed", "MaxSpeed"),
    subFormat: field(request, "subFormat", "SubFormat"),
    selectVideo: field(request, "selectVideo", "SelectVideo"),
    selectAudio: field(request, "selectAudio", "SelectAudio"),
    selectSubtitle: field(request, "selectSubtitle", "SelectSubtitle"),
    dropVideo: field(request, "dropVideo", "DropVideo"),
    dropAudio: field(request, "dropAudio", "DropAudio"),
    dropSubtitle: field(request, "dropSubtitle", "DropSubtitle"),
    keys: field(request, "keys", "Keys", []),
    keyTextFile: field(request, "keyTextFile", "KeyTextFile"),
    decryptionEngine: field(request, "decryptionEngine", "DecryptionEngine"),
    decryptionBinaryPath: field(request, "decryptionBinaryPath", "DecryptionBinaryPath"),
    mp4RealTimeDecryption: Boolean(field(request, "mp4RealTimeDecryption", "MP4RealTimeDecryption", false)),
    customHLSMethod: field(request, "customHLSMethod", "CustomHLSMethod"),
    customHLSKey: field(request, "customHLSKey", "CustomHLSKey"),
    customHLSIV: field(request, "customHLSIV", "CustomHLSIV"),
    adKeywords: field(request, "adKeywords", "AdKeywords", []),
    taskStartAt: field(request, "taskStartAt", "TaskStartAt"),
    liveRecordLimit: field(request, "liveRecordLimit", "LiveRecordLimit"),
    liveWaitTime: Number(field(request, "liveWaitTime", "LiveWaitTime", 0)) || 0,
    liveTakeCount: Number(field(request, "liveTakeCount", "LiveTakeCount", 0)) || 0,
    muxAfterDone: field(request, "muxAfterDone", "MuxAfterDone"),
    muxImports: field(request, "muxImports", "MuxImports", []),
    autoSelect: Boolean(field(request, "autoSelect", "AutoSelect", false)),
    subOnly: Boolean(field(request, "subOnly", "SubOnly", false)),
    disableSubtitleFix: Boolean(field(request, "disableSubtitleFix", "DisableSubtitleFix", false)),
    livePerformAsVOD: Boolean(field(request, "livePerformAsVOD", "LivePerformAsVOD", false)),
    liveRealTimeMerge: Boolean(field(request, "liveRealTimeMerge", "LiveRealTimeMerge", false)),
    disableLiveKeepSegments: Boolean(field(request, "disableLiveKeepSegments", "DisableLiveKeepSegments", false)),
    livePipeMux: Boolean(field(request, "livePipeMux", "LivePipeMux", false)),
    liveFixVTTByAudio: Boolean(field(request, "liveFixVTTByAudio", "LiveFixVTTByAudio", false)),
    muxMP4: Boolean(field(request, "muxMP4", "MuxMP4", false)),
    noDateInfo: Boolean(field(request, "noDateInfo", "NoDateInfo", false)),
    binaryMerge: Boolean(field(request, "binaryMerge", "BinaryMerge", false)),
    appendURLParams: Boolean(field(request, "appendURLParams", "AppendURLParams", false)),
    skipDownload: Boolean(field(request, "skipDownload", "SkipDownload", false)),
    skipMerge: Boolean(field(request, "skipMerge", "SkipMerge", false)),
    keepSegments: Boolean(field(request, "keepSegments", "KeepSegments", false)),
    disableMetaJSON: Boolean(field(request, "disableMetaJSON", "DisableMetaJSON", false)),
    disableSegmentCheck: Boolean(field(request, "disableSegmentCheck", "DisableSegmentCheck", false)),
    noLog: Boolean(field(request, "noLog", "NoLog", false)),
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
  setActionState(el.copyURL, hasTask && Boolean(task.request?.url), hasTask ? "当前任务没有地址" : "请先选择一个任务");
  setActionState(el.copyLog, hasLogs, hasTask ? "当前任务没有日志" : "请先选择一个任务");
  setActionState(el.cloneSelected, hasTask, "请先选择一个任务");
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
  const advancedSummary = taskAdvancedSummary(task.request);
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
    <div><span>高级参数</span><strong title="${escapeHTML(advancedSummary)}">${escapeHTML(advancedSummary)}</strong></div>
    <div><span>开始时间</span><strong>${formatDate(task.startedAt || task.createdAt)}</strong></div>
    <div><span>结束时间</span><strong>${formatDate(task.finishedAt)}</strong></div>
    <div class="summary-command"><span>命令</span><code title="${escapeHTML(commandLine)}">${escapeHTML(commandLine)}</code></div>
  `;
  renderFiles(task.files);
  el.log.textContent = (task.logs || []).join("\n");
  el.log.scrollTop = el.log.scrollHeight;
  updateSelectedTaskActions();
}

function taskAdvancedSummary(request = {}) {
  const parts = [];
  if (request.baseURL) parts.push("BaseURL");
  if (request.tmpDir) parts.push("临时目录");
  if (request.savePattern) parts.push(`模板 ${request.savePattern}`);
  if (request.httpRequestTimeout) parts.push(`超时 ${request.httpRequestTimeout}s`);
  if (request.subOnly) parts.push("只下载字幕");
  if (request.subFormat) parts.push(`字幕 ${request.subFormat}`);
  if (request.disableSubtitleFix) parts.push("不修复字幕");
  if (request.selectVideo) parts.push("选视频");
  if (request.selectAudio) parts.push("选音频");
  if (request.selectSubtitle) parts.push("选字幕");
  if (request.dropVideo || request.dropAudio || request.dropSubtitle) parts.push("丢弃过滤");
  if (Array.isArray(request.keys) && request.keys.length) parts.push(`Key ${request.keys.length}条`);
  if (request.keyTextFile) parts.push("Key 文件");
  if (request.decryptionEngine) parts.push(`解密 ${request.decryptionEngine}`);
  if (request.decryptionBinaryPath) parts.push("解密工具");
  if (request.mp4RealTimeDecryption) parts.push("MP4 实时解密");
  if (request.customHLSMethod) parts.push(`HLS ${request.customHLSMethod}`);
  if (request.customHLSKey) parts.push("自定义 HLS Key");
  if (request.customHLSIV) parts.push("自定义 HLS IV");
  if (Array.isArray(request.adKeywords) && request.adKeywords.length) parts.push(`广告过滤 ${request.adKeywords.length}条`);
  if (request.taskStartAt) parts.push(`定时 ${request.taskStartAt}`);
  if (request.livePerformAsVOD) parts.push("直播当点播");
  if (request.liveRealTimeMerge) parts.push("直播实时合并");
  if (request.disableLiveKeepSegments) parts.push("直播丢弃分片");
  if (request.livePipeMux) parts.push("PipeMux");
  if (request.liveRecordLimit) parts.push(`录制 ${request.liveRecordLimit}`);
  if (request.liveWaitTime) parts.push(`刷新 ${request.liveWaitTime}s`);
  if (request.liveTakeCount) parts.push(`首取 ${request.liveTakeCount}`);
  if (request.liveFixVTTByAudio) parts.push("直播字幕对齐");
  if (request.muxAfterDone) parts.push("最终混流");
  if (Array.isArray(request.muxImports) && request.muxImports.length) parts.push(`导入轨道 ${request.muxImports.length}条`);
  if (request.noDateInfo) parts.push("无日期 metadata");
  if (request.appendURLParams) parts.push("追加 URL 参数");
  if (request.skipDownload) parts.push("只解析资源");
  if (request.skipMerge) parts.push("跳过合并");
  if (request.keepSegments) parts.push("保留临时文件");
  if (request.disableMetaJSON) parts.push("不写 meta");
  if (request.disableSegmentCheck) parts.push("不校验分片数");
  if (request.noLog) parts.push("关闭日志文件");
  if (request.binaryMerge) parts.push("二进制直拼");
  if (request.concurrentDownload) parts.push("多轨并发");
  if (request.maxSpeed) parts.push(`限速 ${request.maxSpeed}`);
  return parts.length ? parts.join(" / ") : "默认";
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
        <button type="button" data-copy-file-path="${escapeHTML(file.path)}">复制路径</button>
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
  el.fileList.querySelectorAll("[data-copy-file-path]").forEach((button) => {
    button.addEventListener("click", async () => {
      await withBusy(button, "复制中", async () => {
        try {
          await copyText(button.dataset.copyFilePath, "文件路径已复制。");
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
    baseURL: el.baseURL.value.trim(),
    saveDir: el.saveDir.value.trim(),
    tmpDir: el.tmpDir.value.trim(),
    saveName: el.saveName.value.trim(),
    savePattern: el.savePattern.value.trim(),
    linkNameSeparator: el.linkNameSeparator.value.trim(),
    customRange: el.customRange.value.trim(),
    ffmpegPath: el.ffmpegPath.value.trim(),
    headers: el.headers.value.split(/\r?\n/).map((line) => line.trim()).filter(Boolean),
    threadCount: Number(el.threadCount.value) || 0,
    retryCount: Number(el.retryCount.value) || 0,
    httpRequestTimeout: Number(el.httpRequestTimeout.value) || 0,
    maxSpeed: el.maxSpeed.value.trim(),
    subFormat: el.subFormat.value || "SRT",
    selectVideo: el.selectVideo.value.trim(),
    selectAudio: el.selectAudio.value.trim(),
    selectSubtitle: el.selectSubtitle.value.trim(),
    dropVideo: el.dropVideo.value.trim(),
    dropAudio: el.dropAudio.value.trim(),
    dropSubtitle: el.dropSubtitle.value.trim(),
    keys: el.keys.value.split(/\r?\n/).map((line) => line.trim()).filter(Boolean),
    keyTextFile: el.keyTextFile.value.trim(),
    decryptionEngine: el.decryptionEngine.value || "MP4DECRYPT",
    decryptionBinaryPath: el.decryptionBinaryPath.value.trim(),
    mp4RealTimeDecryption: el.mp4RealTimeDecryption.checked,
    customHLSMethod: el.customHLSMethod.value,
    customHLSKey: el.customHLSKey.value.trim(),
    customHLSIV: el.customHLSIV.value.trim(),
    adKeywords: el.adKeywords.value.split(/\r?\n/).map((line) => line.trim()).filter(Boolean),
    taskStartAt: el.taskStartAt.value.trim(),
    liveRecordLimit: el.liveRecordLimit.value.trim(),
    liveWaitTime: Number(el.liveWaitTime.value) || 0,
    liveTakeCount: Number(el.liveTakeCount.value) || 0,
    muxAfterDone: el.muxAfterDone.value.trim(),
    muxImports: el.muxImports.value.split(/\r?\n/).map((line) => line.trim()).filter(Boolean),
    customProxy: el.customProxy.value.trim(),
    autoSelect: el.autoSelect.checked,
    subOnly: el.subOnly.checked,
    disableSubtitleFix: !el.autoSubtitleFix.checked,
    livePerformAsVOD: el.livePerformAsVOD.checked,
    liveRealTimeMerge: el.liveRealTimeMerge.checked,
    disableLiveKeepSegments: el.discardLiveSegments.checked,
    livePipeMux: el.livePipeMux.checked,
    liveFixVTTByAudio: el.liveFixVTTByAudio.checked,
    muxMP4: el.muxMP4.checked,
    noDateInfo: el.noDateInfo.checked,
    binaryMerge: el.binaryMerge.checked,
    appendURLParams: el.appendURLParams.checked,
    skipDownload: el.skipDownload.checked,
    skipMerge: el.skipMerge.checked,
    keepSegments: el.keepSegments.checked,
    disableMetaJSON: !el.writeMetaJSON.checked,
    disableSegmentCheck: !el.checkSegmentsCount.checked,
    noLog: el.noLog.checked,
    concurrentDownload: el.concurrentDownload.checked,
    useSystemProxy: el.useSystemProxy.checked
  };
}

function fillFormFromRequest(request = {}) {
  const normalized = normalizeRequest(request);
  el.url.value = normalized.url || "";
  el.baseURL.value = normalized.baseURL || "";
  el.saveDir.value = normalized.saveDir || "";
  el.tmpDir.value = normalized.tmpDir || "";
  el.saveName.value = normalized.saveName || "";
  el.savePattern.value = normalized.savePattern || "";
  el.linkNameSeparator.value = normalized.linkNameSeparator || "|";
  el.customRange.value = normalized.customRange || "";
  el.ffmpegPath.value = normalized.ffmpegPath || "";
  el.headers.value = safeCloneHeaders(normalized.headers).join("\n");
  el.threadCount.value = normalized.threadCount || "";
  el.retryCount.value = normalized.retryCount || "";
  el.httpRequestTimeout.value = normalized.httpRequestTimeout || "";
  el.maxSpeed.value = normalized.maxSpeed || "";
  el.subFormat.value = normalized.subFormat || "SRT";
  el.selectVideo.value = normalized.selectVideo || "";
  el.selectAudio.value = normalized.selectAudio || "";
  el.selectSubtitle.value = normalized.selectSubtitle || "";
  el.dropVideo.value = normalized.dropVideo || "";
  el.dropAudio.value = normalized.dropAudio || "";
  el.dropSubtitle.value = normalized.dropSubtitle || "";
  el.keys.value = "";
  el.keyTextFile.value = normalized.keyTextFile || "";
  el.decryptionEngine.value = normalized.decryptionEngine || "MP4DECRYPT";
  el.decryptionBinaryPath.value = normalized.decryptionBinaryPath || "";
  el.mp4RealTimeDecryption.checked = normalized.mp4RealTimeDecryption;
  el.customHLSMethod.value = normalized.customHLSMethod || "";
  el.customHLSKey.value = "";
  el.customHLSIV.value = "";
  el.adKeywords.value = Array.isArray(normalized.adKeywords) ? normalized.adKeywords.join("\n") : "";
  el.taskStartAt.value = "";
  el.liveRecordLimit.value = normalized.liveRecordLimit || "";
  el.liveWaitTime.value = normalized.liveWaitTime || "";
  el.liveTakeCount.value = normalized.liveTakeCount || "";
  el.muxAfterDone.value = normalized.muxAfterDone || "";
  el.muxImportPath.value = "";
  el.muxImportLang.value = "";
  el.muxImportName.value = "";
  el.muxImports.value = Array.isArray(normalized.muxImports) ? normalized.muxImports.join("\n") : "";
  el.customProxy.value = cloneProxyValue(normalized.customProxy);
  el.autoSelect.checked = normalized.autoSelect;
  el.subOnly.checked = normalized.subOnly;
  el.autoSubtitleFix.checked = !normalized.disableSubtitleFix;
  el.livePerformAsVOD.checked = normalized.livePerformAsVOD;
  el.liveRealTimeMerge.checked = normalized.liveRealTimeMerge;
  el.discardLiveSegments.checked = normalized.disableLiveKeepSegments;
  el.livePipeMux.checked = normalized.livePipeMux;
  el.liveFixVTTByAudio.checked = normalized.liveFixVTTByAudio;
  el.muxMP4.checked = normalized.muxMP4;
  el.noDateInfo.checked = normalized.noDateInfo;
  el.binaryMerge.checked = normalized.binaryMerge;
  el.appendURLParams.checked = normalized.appendURLParams;
  el.skipDownload.checked = normalized.skipDownload;
  el.skipMerge.checked = normalized.skipMerge;
  el.keepSegments.checked = normalized.keepSegments;
  el.writeMetaJSON.checked = !normalized.disableMetaJSON;
  el.checkSegmentsCount.checked = !normalized.disableSegmentCheck;
  el.noLog.checked = normalized.noLog;
  el.concurrentDownload.checked = normalized.concurrentDownload;
  el.useSystemProxy.checked = normalized.useSystemProxy;
}

function safeCloneHeaders(headers = []) {
  if (!Array.isArray(headers)) {
    return [];
  }
  return headers.filter((header) => {
    const name = String(header || "").split(":", 1)[0].trim().toLowerCase();
    return !["cookie", "authorization", "proxy-authorization", "x-api-key", "x-auth-token"].includes(name);
  });
}

function cloneProxyValue(value) {
  const proxy = String(value || "").trim();
  if (!proxy || proxy.includes(":redacted@")) {
    return "";
  }
  const at = proxy.indexOf("@");
  if (at < 0) {
    return proxy;
  }
  const beforeAt = proxy.slice(0, at);
  const userPart = beforeAt.includes("://") ? beforeAt.slice(beforeAt.indexOf("://") + 3) : beforeAt;
  return userPart.includes(":") ? "" : proxy;
}

function resetPreflightResult() {
  el.preflightResult.className = "preflight-card field-wide empty";
  el.preflightResult.textContent = "尚未预检查。";
}

function resetCreateForm() {
  applySettings(settings);
  el.url.value = "";
  el.saveName.value = "";
  el.linkNameSeparator.value = "|";
  el.customRange.value = "";
  el.headers.value = "";
  el.keys.value = "";
  el.customHLSKey.value = "";
  el.customHLSIV.value = "";
  el.taskStartAt.value = "";
  el.muxImportPath.value = "";
  el.muxImportLang.value = "";
  el.muxImportName.value = "";
  resetPreflightResult();
}

function collectSettings() {
  return {
    defaultSaveDir: el.saveDir.value.trim(),
    ffmpegPath: el.ffmpegPath.value.trim(),
    baseURL: el.baseURL.value.trim(),
    tmpDir: el.tmpDir.value.trim(),
    savePattern: el.savePattern.value.trim(),
    maxActiveTasks: Number(el.maxActiveTasks.value) || 2,
    threadCount: Number(el.threadCount.value) || 8,
    retryCount: Number(el.retryCount.value) || 3,
    httpRequestTimeout: Number(el.httpRequestTimeout.value) || 100,
    maxSpeed: el.maxSpeed.value.trim(),
    subFormat: el.subFormat.value || "SRT",
    selectVideo: el.selectVideo.value.trim(),
    selectAudio: el.selectAudio.value.trim(),
    selectSubtitle: el.selectSubtitle.value.trim(),
    dropVideo: el.dropVideo.value.trim(),
    dropAudio: el.dropAudio.value.trim(),
    dropSubtitle: el.dropSubtitle.value.trim(),
    keyTextFile: el.keyTextFile.value.trim(),
    decryptionEngine: el.decryptionEngine.value || "MP4DECRYPT",
    decryptionBinaryPath: el.decryptionBinaryPath.value.trim(),
    mp4RealTimeDecryption: el.mp4RealTimeDecryption.checked,
    customHLSMethod: el.customHLSMethod.value,
    adKeywords: el.adKeywords.value.split(/\r?\n/).map((line) => line.trim()).filter(Boolean),
    liveRecordLimit: el.liveRecordLimit.value.trim(),
    liveWaitTime: Number(el.liveWaitTime.value) || 0,
    liveTakeCount: Number(el.liveTakeCount.value) || 0,
    muxAfterDone: el.muxAfterDone.value.trim(),
    muxImports: el.muxImports.value.split(/\r?\n/).map((line) => line.trim()).filter(Boolean),
    autoSelect: el.autoSelect.checked,
    subOnly: el.subOnly.checked,
    disableSubtitleFix: !el.autoSubtitleFix.checked,
    livePerformAsVOD: el.livePerformAsVOD.checked,
    liveRealTimeMerge: el.liveRealTimeMerge.checked,
    disableLiveKeepSegments: el.discardLiveSegments.checked,
    livePipeMux: el.livePipeMux.checked,
    liveFixVTTByAudio: el.liveFixVTTByAudio.checked,
    muxMP4: el.muxMP4.checked,
    noDateInfo: el.noDateInfo.checked,
    binaryMerge: el.binaryMerge.checked,
    appendURLParams: el.appendURLParams.checked,
    skipDownload: el.skipDownload.checked,
    skipMerge: el.skipMerge.checked,
    keepSegments: el.keepSegments.checked,
    disableMetaJSON: !el.writeMetaJSON.checked,
    disableSegmentCheck: !el.checkSegmentsCount.checked,
    noLog: el.noLog.checked,
    concurrentDownload: el.concurrentDownload.checked,
    useSystemProxy: el.useSystemProxy.checked,
    customProxy: el.customProxy.value.trim()
  };
}

function applySettings(nextSettings) {
  settings = nextSettings || {};
  el.saveDir.value = field(settings, "defaultSaveDir", "DefaultSaveDir", "");
  el.ffmpegPath.value = field(settings, "ffmpegPath", "FFmpegPath", "");
  el.baseURL.value = field(settings, "baseURL", "BaseURL", "");
  el.tmpDir.value = field(settings, "tmpDir", "TmpDir", "");
  el.savePattern.value = field(settings, "savePattern", "SavePattern", "");
  el.maxActiveTasks.value = field(settings, "maxActiveTasks", "MaxActiveTasks", 2);
  el.threadCount.value = field(settings, "threadCount", "ThreadCount", 8);
  el.retryCount.value = field(settings, "retryCount", "RetryCount", 3);
  el.httpRequestTimeout.value = field(settings, "httpRequestTimeout", "HTTPRequestTimeout", 100);
  el.maxSpeed.value = field(settings, "maxSpeed", "MaxSpeed", "");
  el.subFormat.value = field(settings, "subFormat", "SubFormat", "SRT") || "SRT";
  el.selectVideo.value = field(settings, "selectVideo", "SelectVideo", "");
  el.selectAudio.value = field(settings, "selectAudio", "SelectAudio", "");
  el.selectSubtitle.value = field(settings, "selectSubtitle", "SelectSubtitle", "");
  el.dropVideo.value = field(settings, "dropVideo", "DropVideo", "");
  el.dropAudio.value = field(settings, "dropAudio", "DropAudio", "");
  el.dropSubtitle.value = field(settings, "dropSubtitle", "DropSubtitle", "");
  el.keys.value = "";
  el.keyTextFile.value = field(settings, "keyTextFile", "KeyTextFile", "");
  el.decryptionEngine.value = field(settings, "decryptionEngine", "DecryptionEngine", "MP4DECRYPT") || "MP4DECRYPT";
  el.decryptionBinaryPath.value = field(settings, "decryptionBinaryPath", "DecryptionBinaryPath", "");
  el.mp4RealTimeDecryption.checked = Boolean(field(settings, "mp4RealTimeDecryption", "MP4RealTimeDecryption", false));
  el.customHLSMethod.value = field(settings, "customHLSMethod", "CustomHLSMethod", "");
  el.customHLSKey.value = "";
  el.customHLSIV.value = "";
  el.adKeywords.value = field(settings, "adKeywords", "AdKeywords", []).join("\n");
  el.taskStartAt.value = "";
  el.liveRecordLimit.value = field(settings, "liveRecordLimit", "LiveRecordLimit", "");
  el.liveWaitTime.value = field(settings, "liveWaitTime", "LiveWaitTime", "");
  el.liveTakeCount.value = field(settings, "liveTakeCount", "LiveTakeCount", "");
  el.muxAfterDone.value = field(settings, "muxAfterDone", "MuxAfterDone", "");
  el.muxImports.value = field(settings, "muxImports", "MuxImports", []).join("\n");
  el.autoSelect.checked = Boolean(field(settings, "autoSelect", "AutoSelect", true));
  el.subOnly.checked = Boolean(field(settings, "subOnly", "SubOnly", false));
  el.autoSubtitleFix.checked = !Boolean(field(settings, "disableSubtitleFix", "DisableSubtitleFix", false));
  el.livePerformAsVOD.checked = Boolean(field(settings, "livePerformAsVOD", "LivePerformAsVOD", false));
  el.liveRealTimeMerge.checked = Boolean(field(settings, "liveRealTimeMerge", "LiveRealTimeMerge", false));
  el.discardLiveSegments.checked = Boolean(field(settings, "disableLiveKeepSegments", "DisableLiveKeepSegments", false));
  el.livePipeMux.checked = Boolean(field(settings, "livePipeMux", "LivePipeMux", false));
  el.liveFixVTTByAudio.checked = Boolean(field(settings, "liveFixVTTByAudio", "LiveFixVTTByAudio", false));
  el.muxMP4.checked = Boolean(field(settings, "muxMP4", "MuxMP4", false));
  el.noDateInfo.checked = Boolean(field(settings, "noDateInfo", "NoDateInfo", false));
  el.binaryMerge.checked = Boolean(field(settings, "binaryMerge", "BinaryMerge", true));
  el.appendURLParams.checked = Boolean(field(settings, "appendURLParams", "AppendURLParams", false));
  el.skipDownload.checked = Boolean(field(settings, "skipDownload", "SkipDownload", false));
  el.skipMerge.checked = Boolean(field(settings, "skipMerge", "SkipMerge", false));
  el.keepSegments.checked = Boolean(field(settings, "keepSegments", "KeepSegments", false));
  el.writeMetaJSON.checked = !Boolean(field(settings, "disableMetaJSON", "DisableMetaJSON", false));
  el.checkSegmentsCount.checked = !Boolean(field(settings, "disableSegmentCheck", "DisableSegmentCheck", false));
  el.noLog.checked = Boolean(field(settings, "noLog", "NoLog", false));
  el.concurrentDownload.checked = Boolean(field(settings, "concurrentDownload", "ConcurrentDownload", false));
  el.useSystemProxy.checked = Boolean(field(settings, "useSystemProxy", "UseSystemProxy", true));
  el.customProxy.value = field(settings, "customProxy", "CustomProxy", "");
  syncMergeModeFromFFmpegPath();
}

function syncMergeModeFromFFmpegPath() {
  if (el.muxAfterDone.value.trim()) {
    return;
  }
  const hasFFmpegPath = el.ffmpegPath.value.trim() !== "";
  el.muxMP4.checked = hasFFmpegPath;
  el.binaryMerge.checked = !hasFFmpegPath;
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
  renderCoreCapabilities(info);
}

function renderCoreCapabilities(info = {}) {
  const groups = field(info, "capabilities", "Capabilities", []);
  const unsupported = field(info, "unsupported", "Unsupported", []);
  const capabilityError = field(info, "capabilityError", "CapabilityError", "");
  if (!Array.isArray(groups) || groups.length === 0) {
    el.coreCapabilities.innerHTML = `<div class="capability-empty">${escapeHTML(capabilityError || "当前核心未返回能力清单。")}</div>`;
    return;
  }

  const rows = groups.map((group) => {
    const label = field(group, "label", "Label", field(group, "name", "Name", "能力"));
    const items = field(group, "items", "Items", []);
    return `
      <div class="capability-row">
        <b>${escapeHTML(label)}</b>
        <div class="capability-tags">
          ${items.map((item) => `<span title="${escapeHTML(item)}">${escapeHTML(item)}</span>`).join("")}
        </div>
      </div>
    `;
  });
  if (Array.isArray(unsupported) && unsupported.length > 0) {
    rows.push(`
      <div class="capability-row">
        <b>未支持</b>
        <div class="capability-tags">
          ${unsupported.map((item) => `<span class="unsupported" title="${escapeHTML(item)}">${escapeHTML(item)}</span>`).join("")}
        </div>
      </div>
    `);
  }
  el.coreCapabilities.innerHTML = rows.join("");
}

function applyToolCheck(node, info = {}) {
  const status = field(info, "status", "Status", "error");
  const path = field(info, "path", "Path", "");
  const version = field(info, "version", "Version", "");
  const error = field(info, "error", "Error", "");
  node.classList.toggle("ok", status === "ready");
  node.classList.toggle("error", status !== "ready");
  node.textContent = status === "ready" ? `${version || "可用"} | ${path}` : (error || "检测失败");
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
      ${checks.map(renderPreflightCheck).join("")}
    </div>
    ${commandLines.length ? `<code title="${escapeHTML(commandLines.join("\n"))}">${escapeHTML(commandLines.join("\n"))}</code>` : ""}
  `;
}

function renderPreflightCheck(check) {
  const detail = check.detail && check.detail !== check.message ? check.detail : "";
  return `
    <div class="preflight-row ${escapeHTML(check.status)} ${escapeHTML(check.name || "")}">
      <b>${escapeHTML(check.label || check.name)}</b>
      <span title="${escapeHTML(check.detail || check.message)}">${escapeHTML(check.message || "-")}</span>
      ${detail ? `<small>${escapeHTML(detail)}</small>` : ""}
    </div>
  `;
}

async function refreshCoreInfo() {
  el.coreStatus.textContent = "检查中";
  el.coreStatus.classList.remove("error");
  el.coreVersion.textContent = "-";
  el.corePath.textContent = "-";
  el.coreCapabilities.innerHTML = `<div class="capability-empty">正在读取核心能力...</div>`;
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
  resetPreflightResult();
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
      throw new Error("复制失败，请手动选择内容。");
    }
  }
  if (successMessage) {
    notify(successMessage, "success");
  }
  return true;
}

function collectSecretInputs() {
  return {
    keys: el.keys.value,
    customHLSKey: el.customHLSKey.value,
    customHLSIV: el.customHLSIV.value,
    taskStartAt: el.taskStartAt.value
  };
}

function restoreSecretInputs(secret = {}) {
  el.keys.value = secret.keys || "";
  el.customHLSKey.value = secret.customHLSKey || "";
  el.customHLSIV.value = secret.customHLSIV || "";
  el.taskStartAt.value = secret.taskStartAt || "";
}

function muxImportValue(value) {
  return String(value || "").trim().replace(/\r?\n/g, " ").replaceAll(":", "\\:");
}

function buildMuxImportLine() {
  const path = muxImportValue(el.muxImportPath.value).replace(/^path=/i, "");
  if (!path) {
    return "";
  }
  const parts = [`path=${path}`];
  const lang = muxImportValue(el.muxImportLang.value);
  const name = muxImportValue(el.muxImportName.value);
  if (lang) {
    parts.push(`lang=${lang.replace(/^lang=/i, "")}`);
  }
  if (name) {
    parts.push(`name=${name.replace(/^name=/i, "")}`);
  }
  return parts.join(":");
}

function appendMuxImportLine(line) {
  const existing = el.muxImports.value.trim();
  el.muxImports.value = existing ? `${existing}\n${line}` : line;
  el.muxImportPath.value = "";
  el.muxImportLang.value = "";
  el.muxImportName.value = "";
}

async function saveCurrentSettings(button) {
  await withBusy(button, "保存中", async () => {
    const secret = collectSecretInputs();
    try {
      applySettings(await api().SaveSettings(collectSettings()));
      restoreSecretInputs(secret);
      notify("设置已保存。", "success");
    } catch (error) {
      restoreSecretInputs(secret);
      reportError(error);
    }
  });
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

el.chooseTmpDir.addEventListener("click", async () => {
  await withBusy(el.chooseTmpDir, "选择中", async () => {
    try {
      const selected = await api().ChooseDirectory(el.tmpDir.value.trim());
      if (selected) {
        el.tmpDir.value = selected;
      }
    } catch (error) {
      reportError(error);
    }
  });
});

el.chooseKeyFile.addEventListener("click", async () => {
  await withBusy(el.chooseKeyFile, "选择中", async () => {
    try {
      const selected = await api().ChooseFile(el.keyTextFile.value.trim());
      if (selected) {
        el.keyTextFile.value = selected;
      }
    } catch (error) {
      reportError(error);
    }
  });
});

el.chooseDecryptionBinary.addEventListener("click", async () => {
  await withBusy(el.chooseDecryptionBinary, "选择中", async () => {
    try {
      const selected = await api().ChooseFile(el.decryptionBinaryPath.value.trim());
      if (selected) {
        el.decryptionBinaryPath.value = selected;
      }
    } catch (error) {
      reportError(error);
    }
  });
});

el.chooseMuxImportFile.addEventListener("click", async () => {
  await withBusy(el.chooseMuxImportFile, "选择中", async () => {
    try {
      const selected = await api().ChooseFile(el.muxImportPath.value.trim());
      if (selected) {
        el.muxImportPath.value = selected;
      }
    } catch (error) {
      reportError(error);
    }
  });
});

el.addMuxImport.addEventListener("click", () => {
  const line = buildMuxImportLine();
  if (!line) {
    notify("请选择或填写外部轨道文件。", "warning");
    return;
  }
  appendMuxImportLine(line);
  notify("外部轨道已添加。", "success");
});

document.querySelector("#save-settings").addEventListener("click", async () => {
  await saveCurrentSettings(document.querySelector("#save-settings"));
});
el.saveFormSettings.addEventListener("click", async () => {
  await saveCurrentSettings(el.saveFormSettings);
});
el.resetForm.addEventListener("click", () => {
  resetCreateForm();
  notify("表单已恢复为默认参数。", "success");
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
el.chooseFFmpeg.addEventListener("click", async () => {
  await withBusy(el.chooseFFmpeg, "选择中", async () => {
    try {
      const selected = await api().ChooseFile(el.ffmpegPath.value.trim());
      if (selected) {
        el.ffmpegPath.value = selected;
        syncMergeModeFromFFmpegPath();
      }
    } catch (error) {
      reportError(error);
    }
  });
});
el.ffmpegPath.addEventListener("input", syncMergeModeFromFFmpegPath);
el.muxAfterDone.addEventListener("input", () => {
  if (!el.muxAfterDone.value.trim()) {
    syncMergeModeFromFFmpegPath();
  }
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
el.copyURL.addEventListener("click", async () => {
  const task = selectedTask();
  if (!task) {
    notify("请先选择一个任务。", "warning");
    return;
  }
  await withBusy(el.copyURL, "复制中", async () => {
    try {
      await copyText(task.request?.url, "任务地址已复制。");
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
el.cloneSelected.addEventListener("click", () => {
  const task = selectedTask();
  if (!task) {
    notify("请先选择一个任务。", "warning");
    return;
  }
  fillFormFromRequest(task.request);
  switchView("create");
  notify("任务参数已复制到新建表单，敏感请求头、密钥、代理密码和定时开始未回填。", "success");
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
