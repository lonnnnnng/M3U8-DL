const api = () => window.go?.main?.App;

const themeStorageKey = "m3u8dl-theme";
const themeModes = ["light", "dark", "auto"];
const themeLabels = {
  light: "浅色",
  dark: "深色",
  auto: "自动"
};
const systemThemeMedia = window.matchMedia?.("(prefers-color-scheme: dark)") || null;

const el = {
  url: document.querySelector("#url"),
  urlCount: document.querySelector("#url-count"),
  baseURL: document.querySelector("#baseURL"),
  saveDir: document.querySelector("#saveDir"),
  defaultSaveDir: document.querySelector("#defaultSaveDir"),
  tmpDir: document.querySelector("#tmpDir"),
  saveName: document.querySelector("#saveName"),
  savePattern: document.querySelector("#savePattern"),
  linkNameSeparator: document.querySelector("#linkNameSeparator"),
  customRange: document.querySelector("#customRange"),
  taskHttpRequestTimeout: document.querySelector("#taskHttpRequestTimeout"),
  taskRetryCount: document.querySelector("#taskRetryCount"),
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
  outputFormat: document.querySelector("#outputFormat"),
  useFFmpegMerge: document.querySelector("#useFFmpegMerge"),
  noDateInfo: document.querySelector("#noDateInfo"),
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
  resetForm: document.querySelector("#reset-form"),
  newTaskFab: document.querySelector("#new-task-fab"),
  taskForm: document.querySelector("#task-form"),
  createDialog: document.querySelector("#create-dialog"),
  closeCreateDialog: document.querySelector("#close-create-dialog"),
  cancelCreateDialog: document.querySelector("#cancel-create-dialog"),
  chooseDefaultDir: document.querySelector("#choose-default-dir"),
  chooseTmpDir: document.querySelector("#choose-tmp-dir"),
  chooseKeyFile: document.querySelector("#choose-key-file"),
  chooseDecryptionBinary: document.querySelector("#choose-decryption-binary"),
  chooseMuxImportFile: document.querySelector("#choose-mux-import-file"),
  addMuxImport: document.querySelector("#add-mux-import"),
  refreshTasks: document.querySelector("#refresh-tasks"),
  themeToggle: document.querySelector("#theme-toggle"),
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
  versionLabel: document.querySelector("#running-count"),
  activeSpeed: document.querySelector("#active-speed"),
  viewTitle: document.querySelector("#view-title"),
  viewKicker: document.querySelector("#view-kicker"),
  detailTitle: document.querySelector("#detail-title"),
  detailActions: document.querySelector("#detail-actions"),
  taskMoreActions: document.querySelector("#task-more-actions"),
  taskSummary: document.querySelector("#task-summary"),
  fileList: document.querySelector("#file-list"),
  log: document.querySelector("#log"),
  toastHost: document.querySelector("#toast-host"),
  confirmDialog: document.querySelector("#confirm-dialog"),
  confirmTitle: document.querySelector("#confirm-dialog-title"),
  confirmMessage: document.querySelector("#confirm-dialog-message"),
  confirmCancel: document.querySelector("#confirm-dialog-cancel"),
  confirmOK: document.querySelector("#confirm-dialog-ok"),
  taskContextMenu: document.querySelector("#task-context-menu"),
  settingsGrid: document.querySelector(".settings-grid"),
  settingsPanel: document.querySelector(".settings-panel"),
  saveSettings: document.querySelector("#save-settings"),
  settingsSaveState: document.querySelector("#settings-save-state"),
  settingsTabs: Array.from(document.querySelectorAll("[data-settings-tab]")),
  settingsPanels: Array.from(document.querySelectorAll("[data-settings-panel]")),
  navItems: Array.from(document.querySelectorAll("[data-view]")),
  views: Array.from(document.querySelectorAll(".view"))
};

let settings = {};
let tasks = [];
let selectedTaskId = "";
let createOnly = false;
let createActionButton = null;
let activeView = "dashboard";
let activeSettingsTab = "basic";
let taskSearchText = "";
let taskStatusFilter = "all";
let taskSortMode = "newest";
let confirmResolver = null;
let confirmReadyAt = 0;
let createDialogReturnFocus = null;
let createDialogBaseline = "";
let confirmDialogReturnFocus = null;
let taskContextMenuReturnTaskId = "";
let settingsDirty = false;
let applyingSettings = false;

const defaultTools = [
  { name: "ffmpeg", label: "FFmpeg" },
  { name: "ffprobe", label: "FFprobe" },
  { name: "mkvmerge", label: "mkvmerge" },
  { name: "mp4decrypt", label: "mp4decrypt" },
  { name: "shaka-packager", label: "Shaka Packager" }
];

const taskContextActions = [
  { action: "start", label: "开始", icon: "play" },
  { action: "stop", label: "停止", icon: "stop" },
  { action: "retry", label: "重试", icon: "retry" },
  { separator: true },
  { action: "copy-url", label: "复制地址", icon: "link" },
  { action: "copy-command", label: "复制命令", icon: "copy" },
  { action: "copy-log", label: "复制日志", icon: "copy" },
  { action: "clone", label: "复制为新任务", icon: "plus" },
  { separator: true },
  { action: "remove", label: "移除", icon: "trash", danger: true }
];

const viewMeta = {
  dashboard: ["任务监控", "下载任务"],
  settings: ["参数设置", "下载参数"],
  output: ["输出文件", "完成文件"]
};

const previewTasks = [
  {
    id: "preview-live",
    title: "直播频道录制",
    status: "running",
    progress: 0.64,
    progressText: "64%",
    speedText: "8.03 MiB/s",
    elapsedText: "01:26",
    remainingText: "00:48",
    lastMessage: "正在下载新分片",
    createdAt: "2026-07-27T12:00:00+08:00",
    startedAt: "2026-07-27T12:00:03+08:00",
    request: {
      url: "https://example.com/live/index.m3u8",
      saveDir: "~/Downloads/M3U8-DL",
      saveName: "直播频道录制",
      autoSelect: true,
      useFFmpegMerge: true,
      outputFormat: "mp4"
    },
    commandLine: "M3U8-DL https://example.com/live/index.m3u8 -M format=mp4:muxer=ffmpeg",
    logs: [
      "$ M3U8-DL https://example.com/live/index.m3u8",
      "读取播放列表成功，检测到直播流",
      "已选择视频 1920x1080 AVC / 音频 AAC",
      "开始下载分片，线程数 16",
      "已下载 652.80 MiB",
      "当前速度 8.03 MiB/s"
    ]
  },
  {
    id: "preview-done",
    title: "示例课程 - 第一集",
    status: "completed",
    progress: 1,
    progressText: "100%",
    elapsedText: "03:18",
    lastMessage: "下载与混流完成",
    createdAt: "2026-07-27T11:40:00+08:00",
    startedAt: "2026-07-27T11:40:02+08:00",
    finishedAt: "2026-07-27T11:43:20+08:00",
    request: {
      url: "https://media.example.com/course/01/master.m3u8",
      saveDir: "~/Downloads/M3U8-DL",
      saveName: "示例课程-第一集",
      autoSelect: true,
      useFFmpegMerge: true,
      outputFormat: "mp4"
    },
    commandLine: "M3U8-DL https://media.example.com/course/01/master.m3u8 -M format=mp4:muxer=ffmpeg",
    logs: ["任务完成，输出示例课程-第一集.mp4"]
  },
  {
    id: "preview-failed",
    title: "加密媒体测试",
    status: "failed",
    progress: 0.18,
    progressText: "18%",
    elapsedText: "00:22",
    lastMessage: "密钥请求返回 403",
    createdAt: "2026-07-27T11:30:00+08:00",
    startedAt: "2026-07-27T11:30:01+08:00",
    finishedAt: "2026-07-27T11:30:23+08:00",
    request: {
      url: "https://media.example.com/encrypted/master.m3u8",
      saveDir: "~/Downloads/M3U8-DL",
      saveName: "加密媒体测试",
      autoSelect: true,
      useFFmpegMerge: true,
      outputFormat: "mp4"
    },
    commandLine: "M3U8-DL https://media.example.com/encrypted/master.m3u8",
    logs: ["读取播放列表成功", "下载密钥失败: HTTP 403"]
  }
];

const defaultLinkNameSeparator = "|";
const defaultOutputFormat = "mp4";

function readThemeMode() {
  try {
    const storedTheme = window.localStorage.getItem(themeStorageKey);
    return themeModes.includes(storedTheme) ? storedTheme : "auto";
  } catch (_) {
    return "auto";
  }
}

function applyThemeMode(themeMode, { persist = false } = {}) {
  const normalizedMode = themeModes.includes(themeMode) ? themeMode : "auto";
  const systemDark = systemThemeMedia?.matches === true;
  const effectiveTheme = normalizedMode === "dark" || (normalizedMode === "auto" && systemDark) ? "dark" : "light";

  document.documentElement.dataset.themeMode = normalizedMode;
  document.documentElement.dataset.theme = effectiveTheme;

  if (persist) {
    try {
      window.localStorage.setItem(themeStorageKey, normalizedMode);
    } catch (_) {
      // long: WebView 的持久化不可用时仍保留当前会话主题，避免主题按钮失去即时反馈。
    }
  }

  if (!el.themeToggle) return;
  const currentIndex = themeModes.indexOf(normalizedMode);
  const nextMode = themeModes[(currentIndex + 1) % themeModes.length];
  const description = `主题：${themeLabels[normalizedMode]}；点击切换为${themeLabels[nextMode]}`;
  el.themeToggle.setAttribute("aria-label", description);
  el.themeToggle.dataset.tooltip = description;
  el.themeToggle.title = description;
  el.themeToggle.querySelector("use")?.setAttribute("href", `#icon-theme-${normalizedMode}`);
}

function cycleThemeMode() {
  const currentMode = themeModes.includes(document.documentElement.dataset.themeMode)
    ? document.documentElement.dataset.themeMode
    : readThemeMode();
  const nextMode = themeModes[(themeModes.indexOf(currentMode) + 1) % themeModes.length];
  applyThemeMode(nextMode, { persist: true });
}

function field(source, camel, pascal, fallback = "") {
  return source?.[camel] ?? source?.[pascal] ?? fallback;
}

function normalizeOutputFormat(value) {
  const format = String(value || "").trim().toLowerCase();
  return format === "ts" ? "ts" : defaultOutputFormat;
}

function currentOutputFormat() {
  return normalizeOutputFormat(el.outputFormat?.value || defaultOutputFormat);
}

function syncMergeControls() {
  if (!el.outputFormat || !el.useFFmpegMerge) return;
  const format = currentOutputFormat();
  el.outputFormat.value = format;
  if (format === "mp4") {
    // long: MP4 需要重新封装，界面直接把 FFmpeg 合并设为开启，避免保存出看似 MP4 但实际不可播放的直拼文件。
    el.useFFmpegMerge.checked = true;
  } else if (!el.useFFmpegMerge.checked) {
    el.outputFormat.value = "ts";
  }
}

function linkNameSeparatorValue() {
  return el.linkNameSeparator?.value.trim() || defaultLinkNameSeparator;
}

function buildURLPlaceholder(separator = defaultLinkNameSeparator) {
  return [
    "单个任务：",
    "https://example.com/index.m3u8",
    "",
    "批量任务，每行一个地址：",
    "https://example.com/ep01/index.m3u8",
    "https://example.com/ep02/index.m3u8",
    "",
    `带保存名，当前分隔符为 ${separator}：`,
    `第一集${separator}https://example.com/ep01/index.m3u8`,
    `第二集${separator}https://example.com/ep02/index.m3u8`
  ].join("\n");
}

function updateURLPlaceholder() {
  if (!el.url) return;
  // long: 批量任务格式最容易输错，placeholder 跟随保存名分隔符变化，让用户在输入地址时直接看到可复制的格式。
  el.url.placeholder = buildURLPlaceholder(linkNameSeparatorValue());
}

function updateURLCount() {
  if (!el.urlCount || !el.url) return;
  const count = el.url.value.split(/\r?\n/).map((line) => line.trim()).filter(Boolean).length;
  el.urlCount.textContent = `${count} 个地址`;
}

function optionalNumberInput(input, fallback = 0) {
  if (!input || String(input.value).trim() === "") return fallback;
  const value = Number(input.value);
  return Number.isFinite(value) ? value : fallback;
}

function updateTaskOverridePlaceholders() {
  if (el.taskHttpRequestTimeout) {
    el.taskHttpRequestTimeout.placeholder = `默认 ${el.httpRequestTimeout?.value || 100}`;
  }
  if (el.taskRetryCount) {
    el.taskRetryCount.placeholder = `默认 ${el.retryCount?.value || 3}`;
  }
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
    outputFormat: normalizeOutputFormat(field(request, "outputFormat", "OutputFormat", field(request, "muxMP4", "MuxMP4", false) ? "mp4" : "ts")),
    useFFmpegMerge: Boolean(field(request, "useFFmpegMerge", "UseFFmpegMerge", field(request, "muxMP4", "MuxMP4", false))),
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

function iconMarkup(name) {
  return `<svg class="button-icon" aria-hidden="true"><use href="#icon-${escapeHTML(name)}"></use></svg>`;
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

function dialogFocusableElements(dialog) {
  if (!dialog) return [];
  return Array.from(dialog.querySelectorAll("button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), summary, [tabindex]:not([tabindex='-1'])"))
    .filter((node) => !node.hidden && node.getAttribute("aria-hidden") !== "true" && node.getClientRects().length > 0);
}

function trapDialogFocus(event, dialog) {
  if (event.key !== "Tab" || !dialog || dialog.hidden) return false;
  const focusable = dialogFocusableElements(dialog);
  if (!focusable.length) return false;
  const first = focusable[0];
  const last = focusable[focusable.length - 1];
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault();
    last.focus();
    return true;
  }
  if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault();
    first.focus();
    return true;
  }
  return false;
}

function restoreDialogFocus(target, fallback) {
  const next = target?.isConnected && !target.hidden && target.getClientRects().length > 0 ? target : fallback;
  window.setTimeout(() => next?.focus(), 0);
}

function updateSettingsSaveState(message = "") {
  const backendReady = Boolean(api());
  const status = message || (settingsDirty ? "有未保存更改" : (backendReady ? "已保存" : "预览模式"));
  if (el.settingsSaveState) {
    el.settingsSaveState.textContent = status;
    el.settingsSaveState.classList.toggle("dirty", settingsDirty);
    el.settingsSaveState.classList.toggle("saved", backendReady && !settingsDirty);
  }
  if (el.saveSettings && el.saveSettings.dataset.busy !== "true") {
    el.saveSettings.disabled = !backendReady || !settingsDirty;
    el.saveSettings.setAttribute("aria-disabled", el.saveSettings.disabled ? "true" : "false");
    el.saveSettings.title = !backendReady ? "请在桌面客户端中保存设置" : (settingsDirty ? "" : "设置没有变化");
  }
  const settingsNav = el.navItems.find((button) => button.dataset.view === "settings");
  if (settingsNav) {
    settingsNav.classList.toggle("dirty", settingsDirty);
    settingsNav.setAttribute("aria-label", settingsDirty ? "设置，有未保存更改" : "设置");
    settingsNav.title = settingsDirty ? "有未保存的设置更改" : "";
  }
}

function markSettingsDirty() {
  if (applyingSettings) return;
  settingsDirty = true;
  updateSettingsSaveState();
}

function closeConfirmDialog(result = false) {
  if (!el.confirmDialog) return;
  const wasOpen = !el.confirmDialog.hidden;
  const returnFocus = confirmDialogReturnFocus;
  el.confirmDialog.hidden = true;
  el.confirmDialog.setAttribute("aria-hidden", "true");
  confirmReadyAt = 0;
  confirmDialogReturnFocus = null;
  const resolve = confirmResolver;
  confirmResolver = null;
  resolve?.(result);
  if (wasOpen) {
    restoreDialogFocus(returnFocus, el.taskMoreActions?.querySelector("summary"));
  }
}

function confirmAction(options = {}) {
  if (!el.confirmDialog) {
    notify("确认弹窗初始化失败，已取消操作。", "error");
    return Promise.resolve(false);
  }
  if (confirmResolver) {
    closeConfirmDialog(false);
  }
  el.confirmTitle.textContent = options.title || "确认操作";
  el.confirmMessage.textContent = options.message || "确认执行这个操作？";
  el.confirmOK.textContent = options.confirmText || "确定";
  el.confirmOK.classList.toggle("danger-primary", Boolean(options.danger));
  confirmDialogReturnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
  el.confirmDialog.hidden = false;
  el.confirmDialog.setAttribute("aria-hidden", "false");
  // long: 确认框来自菜单点击时，短暂屏蔽确认按钮，防止同一次鼠标动作意外落到危险操作上。
  confirmReadyAt = Date.now() + 350;
  window.setTimeout(() => el.confirmCancel?.focus(), 0);
  return new Promise((resolve) => {
    confirmResolver = resolve;
  });
}

async function withBusy(button, busyText, action) {
  if (!button) {
    return action();
  }
  if (button.disabled) {
    return undefined;
  }
  const previousHTML = button.innerHTML;
  const previousTitle = button.title;
  const previousAriaLabel = button.getAttribute("aria-label");
  button.disabled = true;
  button.setAttribute("aria-disabled", "true");
  button.dataset.busy = "true";
  if (busyText) {
    const label = button.querySelector(".button-label");
    if (label) {
      label.textContent = busyText;
    } else if (button.querySelector(".button-icon")) {
      // long: 纯图标按钮忙碌时保留图标并旋转，通过辅助文本说明状态，避免窄按钮被临时文案撑坏。
      button.title = busyText;
      button.setAttribute("aria-label", busyText);
    } else {
      button.textContent = busyText;
    }
  }
  try {
    return await action();
  } finally {
    button.disabled = false;
    button.setAttribute("aria-disabled", "false");
    delete button.dataset.busy;
    button.innerHTML = previousHTML;
    button.title = previousTitle;
    if (previousAriaLabel === null) {
      button.removeAttribute("aria-label");
    } else {
      button.setAttribute("aria-label", previousAriaLabel);
    }
    updateToolbarActionStates();
    updateSelectedTaskActions();
  }
}

function setActionState(button, enabled, reason = "") {
  if (!button || button.dataset.busy === "true") return;
  button.disabled = !enabled;
  button.setAttribute("aria-disabled", enabled ? "false" : "true");
  button.title = enabled ? (button.dataset.tooltip || "") : reason;
}

function taskActionAvailability(task) {
  const hasTask = Boolean(task);
  const isRunning = task?.status === "running";
  const isQueued = task?.status === "pending" && task?.queued;
  const canStart = hasTask && !isRunning && !isQueued && task.status !== "failed";
  const canStop = hasTask && (isRunning || isQueued);
  const canRetry = hasTask && task.status === "failed";
  const canRemove = hasTask && !isRunning;
  const hasLogs = hasTask && Array.isArray(task.logs) && task.logs.length > 0;
  const startDisabledReason = hasTask && task.status === "failed" ? "失败任务请使用重试" : "运行中或排队中的任务不能重复开始";

  return {
    hasTask,
    hasLogs,
    canStart,
    canStop,
    canRetry,
    canRemove,
    startDisabledReason
  };
}

function updateSelectedTaskActions() {
  const task = selectedTask();
  const state = taskActionAvailability(task);

  if (el.detailActions) {
    el.detailActions.hidden = !state.hasTask;
  }
  if (el.taskMoreActions) {
    el.taskMoreActions.hidden = !state.hasTask;
    if (!state.hasTask) {
      el.taskMoreActions.open = false;
    }
  }

  // long: 任务详情一次只展示当前状态真正可执行的主动作，避免开始、停止、重试三个互斥按钮同时占用标题栏。
  if (el.startSelected) el.startSelected.hidden = !state.canStart;
  if (el.stopSelected) el.stopSelected.hidden = !state.canStop;
  if (el.retrySelected) el.retrySelected.hidden = !state.canRetry;

  setActionState(el.copyCommand, state.hasTask && Boolean(task.commandLine), state.hasTask ? "当前任务还没有生成命令" : "请先选择一个任务");
  setActionState(el.copyURL, state.hasTask && Boolean(task.request?.url), state.hasTask ? "当前任务没有地址" : "请先选择一个任务");
  setActionState(el.copyLog, state.hasLogs, state.hasTask ? "当前任务没有日志" : "请先选择一个任务");
  setActionState(el.cloneSelected, state.hasTask, "请先选择一个任务");
  setActionState(el.startSelected, state.canStart, state.hasTask ? state.startDisabledReason : "请先选择一个任务");
  setActionState(el.stopSelected, state.canStop, state.hasTask ? "只有运行中或排队中的任务可以停止" : "请先选择一个任务");
  setActionState(el.retrySelected, state.canRetry, state.hasTask ? "只有失败任务可以重试" : "请先选择一个任务");
  setActionState(el.removeSelected, state.canRemove, state.hasTask ? "运行中的任务不能移除" : "请先选择一个任务");
  setActionState(el.exportLog, state.hasLogs, state.hasTask ? "当前任务没有日志可导出" : "请先选择一个任务");
  setActionState(el.clearLog, state.hasLogs, state.hasTask ? "当前任务没有日志可清空" : "请先选择一个任务");
  setActionState(el.refreshFiles, state.hasTask, "请先选择一个任务");
  setActionState(el.openFolder, state.hasTask, "请先选择一个任务");
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
  if (activeView === "settings" && el.settingsGrid) {
    el.settingsGrid.scrollTop = 0;
    switchSettingsTab(activeSettingsTab, { resetScroll: true });
  }
}

function switchSettingsTab(name, options = {}) {
  const nextTab = el.settingsPanels.some((panel) => panel.dataset.settingsPanel === name) ? name : "basic";
  activeSettingsTab = nextTab;
  el.settingsTabs.forEach((button) => {
    const active = button.dataset.settingsTab === nextTab;
    button.classList.toggle("active", active);
    button.setAttribute("aria-selected", active ? "true" : "false");
    button.tabIndex = active ? 0 : -1;
  });
  el.settingsPanels.forEach((panel) => {
    const active = panel.dataset.settingsPanel === nextTab;
    panel.classList.toggle("active", active);
    panel.hidden = !active;
    if (active && options.resetScroll) {
      panel.scrollTop = 0;
    }
  });
}

function focusSettingsTabByOffset(offset) {
  if (!el.settingsTabs.length) return;
  const currentIndex = Math.max(0, el.settingsTabs.findIndex((button) => button.dataset.settingsTab === activeSettingsTab));
  const nextIndex = (currentIndex + offset + el.settingsTabs.length) % el.settingsTabs.length;
  const nextButton = el.settingsTabs[nextIndex];
  switchSettingsTab(nextButton.dataset.settingsTab, { resetScroll: true });
  nextButton.focus();
}

function openCreateDialog(options = {}) {
  if (!el.createDialog) return;
  if (options.reset) {
    resetCreateForm();
  }
  createDialogReturnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
  el.createDialog.hidden = false;
  el.createDialog.setAttribute("aria-hidden", "false");
  el.createDialog.classList.add("open");
  createDialogBaseline = createDraftSnapshot();
  window.setTimeout(() => {
    el.url?.focus();
  }, 0);
}

function createPanelIsInline() {
  return el.createDialog?.classList.contains("inline-create-panel") === true;
}

function closeCreateDialog() {
  if (!el.createDialog) return;
  if (createPanelIsInline()) {
    // long: 创建表单现在是任务工作台的固定左栏，完成任务后只更新草稿基线，不能像旧弹窗一样从布局中移除。
    el.createDialog.hidden = false;
    el.createDialog.setAttribute("aria-hidden", "false");
    el.createDialog.classList.remove("open");
    createDialogBaseline = createDraftSnapshot();
    return;
  }
  const wasOpen = !el.createDialog.hidden;
  const returnFocus = createDialogReturnFocus;
  el.createDialog.classList.remove("open");
  el.createDialog.hidden = true;
  el.createDialog.setAttribute("aria-hidden", "true");
  createDialogReturnFocus = null;
  if (wasOpen) {
    restoreDialogFocus(returnFocus, el.newTaskFab);
  }
}

function createDraftSnapshot() {
  if (!el.taskForm) return "";
  const values = Array.from(el.taskForm.querySelectorAll("input, select, textarea")).map((input) => ({
    id: input.id,
    value: input.type === "checkbox" ? input.checked : input.value
  }));
  return JSON.stringify(values);
}

function createDraftIsDirty() {
  return Boolean(createDialogBaseline) && createDraftSnapshot() !== createDialogBaseline;
}

async function requestCloseCreateDialog() {
  if (createPanelIsInline()) return;
  if (!el.createDialog || el.createDialog.hidden) return;
  if (!createDraftIsDirty()) {
    closeCreateDialog();
    return;
  }
  const confirmed = await confirmAction({
    title: "放弃当前任务草稿",
    message: "表单里有尚未创建的内容。确认关闭并放弃这些更改？",
    confirmText: "放弃更改",
    danger: true
  });
  if (!confirmed) return;
  resetCreateForm();
  createDialogBaseline = createDraftSnapshot();
  closeCreateDialog();
}

function taskContextActionState(task, action) {
  const state = taskActionAvailability(task);
  switch (action) {
    case "start":
      return { enabled: state.canStart, reason: state.hasTask ? state.startDisabledReason : "请先选择一个任务" };
    case "stop":
      return { enabled: state.canStop, reason: state.hasTask ? "只有运行中或排队中的任务可以停止" : "请先选择一个任务" };
    case "retry":
      return { enabled: state.canRetry, reason: state.hasTask ? "只有失败任务可以重试" : "请先选择一个任务" };
    case "copy-url":
      return { enabled: state.hasTask && Boolean(task.request?.url), reason: state.hasTask ? "当前任务没有地址" : "请先选择一个任务" };
    case "copy-command":
      return { enabled: state.hasTask && Boolean(task.commandLine), reason: state.hasTask ? "当前任务还没有生成命令" : "请先选择一个任务" };
    case "copy-log":
      return { enabled: state.hasLogs, reason: state.hasTask ? "当前任务没有日志" : "请先选择一个任务" };
    case "clone":
      return { enabled: state.hasTask, reason: "请先选择一个任务" };
    case "remove":
      return { enabled: state.canRemove, reason: state.hasTask ? "运行中的任务不能移除" : "请先选择一个任务" };
    default:
      return { enabled: false, reason: "" };
  }
}

function renderTaskContextMenu(task) {
  if (!el.taskContextMenu) return;
  el.taskContextMenu.innerHTML = taskContextActions.map((item) => {
    if (item.separator) {
      return `<div class="task-context-separator" role="separator"></div>`;
    }
    const state = taskContextActionState(task, item.action);
    return `
      <button class="task-context-item ${item.danger ? "danger" : ""} ${state.enabled ? "" : "is-disabled"}" type="button" role="menuitem" data-task-context-action="${item.action}" aria-disabled="${state.enabled ? "false" : "true"}" aria-label="${escapeHTML(state.enabled ? item.label : `${item.label}，${state.reason}`)}" data-disabled-reason="${escapeHTML(state.enabled ? "" : state.reason)}" title="${escapeHTML(state.enabled ? "" : state.reason)}">
        ${iconMarkup(item.icon)}<span>${escapeHTML(item.label)}</span>
      </button>
    `;
  }).join("");
}

function openTaskContextMenu(event, taskId, options = {}) {
  if (!el.taskContextMenu) return;
  event.preventDefault();
  event.stopPropagation();
  // long: 右键操作也代表用户切换当前任务，先同步选中项再渲染菜单，避免菜单动作和右侧详情指向不同任务。
  selectedTaskId = taskId;
  taskContextMenuReturnTaskId = taskId;
  const task = selectedTask();
  renderTasks();
  renderTaskContextMenu(task);
  el.taskContextMenu.hidden = false;
  el.taskContextMenu.style.left = "0px";
  el.taskContextMenu.style.top = "0px";

  window.requestAnimationFrame(() => {
    const rect = el.taskContextMenu.getBoundingClientRect();
    const source = el.taskList.querySelector(`[data-task-id="${CSS.escape(taskId)}"]`);
    const sourceRect = source?.getBoundingClientRect();
    const margin = 8;
    const requestedX = options.keyboard ? (sourceRect?.left || margin) + 18 : event.clientX;
    const requestedY = options.keyboard ? (sourceRect?.top || margin) + 18 : event.clientY;
    const x = Math.max(margin, Math.min(requestedX, window.innerWidth - rect.width - margin));
    const y = Math.max(margin, Math.min(requestedY, window.innerHeight - rect.height - margin));
    el.taskContextMenu.style.left = `${x}px`;
    el.taskContextMenu.style.top = `${y}px`;
    // long: 菜单打开后直接落到当前任务可执行的第一项，避免键盘用户先经过一串灰置动作才能操作。
    el.taskContextMenu.querySelector('[role="menuitem"][aria-disabled="false"]')?.focus();
  });
}

function closeTaskContextMenu(options = {}) {
  if (!el.taskContextMenu) return;
  const wasOpen = !el.taskContextMenu.hidden;
  el.taskContextMenu.hidden = true;
  if (wasOpen && options.restoreFocus && taskContextMenuReturnTaskId) {
    const taskId = taskContextMenuReturnTaskId;
    window.setTimeout(() => el.taskList.querySelector(`[data-task-id="${CSS.escape(taskId)}"]`)?.focus(), 0);
  }
  taskContextMenuReturnTaskId = "";
}

async function runTaskContextAction(action) {
  const taskId = selectedTaskId;
  closeTaskContextMenu({ restoreFocus: true });
  if (action === "remove") {
    await removeTask(taskId, null);
    return;
  }
  const actionButton = {
    start: el.startSelected,
    stop: el.stopSelected,
    retry: el.retrySelected,
    "copy-url": el.copyURL,
    "copy-command": el.copyCommand,
    "copy-log": el.copyLog,
    clone: el.cloneSelected
  }[action];
  actionButton?.click();
}

function renderTasks() {
  const running = tasks.filter((task) => task.status === "running");
  const visibleTasks = tasks.filter(taskMatchesCurrentFilter).slice().sort(compareTasks);
  if (visibleTasks.length && !visibleTasks.some((task) => task.id === selectedTaskId)) {
    // long: 筛选或搜索隐藏当前任务时，详情区要跟随切到第一条可见任务，避免用户对不可见任务执行复制、重试或移除。
    selectedTaskId = visibleTasks[0].id;
  }
  el.taskCount.textContent = visibleTasks.length === tasks.length ? `${tasks.length}` : `${visibleTasks.length}/${tasks.length}`;
  el.activeSpeed.textContent = running.find((task) => task.speedText)?.speedText || "0 B/s";
  updateToolbarActionStates();
  if (!tasks.length) {
    selectedTaskId = "";
    el.taskList.innerHTML = `<div class="empty">暂无任务。</div>`;
    renderDetail();
    return;
  }
  if (!visibleTasks.length) {
    // long: 筛选结果为空时详情必须同步清空，避免用户对列表中不可见的任务执行开始、重试或移除。
    selectedTaskId = "";
    el.taskList.innerHTML = `<div class="empty">没有匹配的任务。</div>`;
    renderDetail();
    return;
  }

  el.taskList.innerHTML = visibleTasks.map((task) => {
    const progressText = task.progressText || `${Math.round(task.progress * 100)}%`;
    const timingText = taskTimingText(task);
    const sideMeta = [task.speedText, timingText].filter(Boolean).join(" · ");
    return `
      <button class="task-card ${task.id === selectedTaskId ? "selected" : ""}" data-task-id="${task.id}" type="button" role="option" aria-selected="${task.id === selectedTaskId ? "true" : "false"}" tabindex="${task.id === selectedTaskId ? "0" : "-1"}">
        <div class="task-row">
          <strong>${escapeHTML(task.title)}</strong>
          <span class="badge ${task.status} ${task.queued ? "queued" : ""}">${taskDisplayStatusText(task)}</span>
        </div>
        <div class="task-url">${escapeHTML(task.request.url)}</div>
        <div class="progress">
          <span style="width:${Math.round(task.progress * 100)}%"></span>
        </div>
        <div class="task-meta">
          <span>${escapeHTML(progressText)}</span>
          <span>${escapeHTML(sideMeta)}</span>
        </div>
        ${task.lastMessage ? `<div class="task-message">${escapeHTML(task.lastMessage)}</div>` : ""}
      </button>
    `;
  }).join("");

  el.taskList.querySelectorAll("[data-task-id]").forEach((button) => {
    button.addEventListener("click", () => {
      closeTaskContextMenu();
      selectedTaskId = button.dataset.taskId;
      renderTasks();
      renderDetail();
    });
    button.addEventListener("contextmenu", (event) => openTaskContextMenu(event, button.dataset.taskId));
    button.addEventListener("keydown", (event) => {
      const cards = Array.from(el.taskList.querySelectorAll("[data-task-id]"));
      const currentIndex = cards.indexOf(button);
      let nextIndex = -1;
      if (event.key === "ArrowDown") nextIndex = Math.min(cards.length - 1, currentIndex + 1);
      if (event.key === "ArrowUp") nextIndex = Math.max(0, currentIndex - 1);
      if (event.key === "Home") nextIndex = 0;
      if (event.key === "End") nextIndex = cards.length - 1;
      if (nextIndex >= 0) {
        event.preventDefault();
        const nextTaskId = cards[nextIndex].dataset.taskId;
        selectedTaskId = nextTaskId;
        renderTasks();
        window.setTimeout(() => el.taskList.querySelector(`[data-task-id="${CSS.escape(nextTaskId)}"]`)?.focus(), 0);
        return;
      }
      if (event.key === "ContextMenu" || (event.shiftKey && event.key === "F10")) {
        openTaskContextMenu(event, button.dataset.taskId, { keyboard: true });
      }
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
  const progressText = task.progressText || `${Math.round(task.progress * 100)}%`;
  el.taskSummary.className = "summary";
  el.taskSummary.innerHTML = `
    <section class="summary-progress">
      <div class="summary-progress-heading"><strong>${escapeHTML(progressText)}</strong><span>${taskDisplayStatusText(task)}</span></div>
      <div class="progress"><span style="width:${Math.round(task.progress * 100)}%"></span></div>
    </section>
    <section class="summary-metrics">
      <div><span>当前速度</span><strong>${escapeHTML(task.speedText || "0 B/s")}</strong></div>
      <div><span>耗时</span><strong>${escapeHTML(task.elapsedText || "-")}</strong></div>
      <div><span>预计剩余</span><strong>${escapeHTML(task.remainingText || "-")}</strong></div>
      <div><span>输出格式</span><strong>${task.request.useFFmpegMerge ? `FFmpeg ${String(task.request.outputFormat || "mp4").toUpperCase()}` : "二进制 TS"}</strong></div>
    </section>
    <section class="summary-facts">
      <div><span>输出目录</span><strong>${escapeHTML(task.request.saveDir)}</strong></div>
      <div><span>保存名</span><strong>${escapeHTML(task.request.saveName || "自动")}</strong></div>
      <div><span>选轨</span><strong>${task.request.autoSelect ? "自动选轨" : "手动选轨"}</strong></div>
      <div><span>开始时间</span><strong>${formatDate(task.startedAt || task.createdAt)}</strong></div>
      <div><span>结束时间</span><strong>${formatDate(task.finishedAt)}</strong></div>
      <div><span>高级参数</span><strong title="${escapeHTML(advancedSummary)}">${escapeHTML(advancedSummary)}</strong></div>
    </section>
    <section class="summary-command"><span>命令</span><code title="${escapeHTML(commandLine)}">${escapeHTML(commandLine)}</code></section>
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
  if (request.retryCount || request.retryCount === 0) parts.push(`失败重试 ${request.retryCount}次`);
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
  if (request.useFFmpegMerge) parts.push(`FFmpeg ${String(request.outputFormat || "mp4").toUpperCase()} 合并`);
  if (!request.useFFmpegMerge) parts.push("二进制直拼 TS");
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
      const confirmed = await confirmAction({
        title: "删除输出文件",
        message: `确认删除文件「${button.dataset.deleteFile}」？这个操作会删除磁盘上的文件。`,
        confirmText: "删除",
        danger: true
      });
      if (!confirmed) return;
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
    retryCount: optionalNumberInput(el.taskRetryCount, optionalNumberInput(el.retryCount, 0)),
    httpRequestTimeout: optionalNumberInput(el.taskHttpRequestTimeout, optionalNumberInput(el.httpRequestTimeout, 0)),
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
    outputFormat: currentOutputFormat(),
    useFFmpegMerge: el.useFFmpegMerge.checked,
    muxMP4: el.useFFmpegMerge.checked && currentOutputFormat() === "mp4",
    noDateInfo: el.noDateInfo.checked,
    binaryMerge: !el.useFFmpegMerge.checked,
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
  el.linkNameSeparator.value = normalized.linkNameSeparator || linkNameSeparatorValue();
  updateURLPlaceholder();
  updateURLCount();
  el.customRange.value = normalized.customRange || "";
  el.taskRetryCount.value = (normalized.retryCount || normalized.retryCount === 0) ? normalized.retryCount : "";
  el.taskHttpRequestTimeout.value = normalized.httpRequestTimeout || "";
  el.ffmpegPath.value = normalized.ffmpegPath || "";
  el.headers.value = safeCloneHeaders(normalized.headers).join("\n");
  el.threadCount.value = normalized.threadCount || "";
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
  el.outputFormat.value = normalized.outputFormat || defaultOutputFormat;
  el.useFFmpegMerge.checked = normalized.useFFmpegMerge || normalized.outputFormat === "mp4";
  syncMergeControls();
  el.noDateInfo.checked = normalized.noDateInfo;
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
  el.saveDir.value = field(settings, "defaultSaveDir", "DefaultSaveDir", "");
  el.url.value = "";
  updateURLCount();
  el.saveName.value = "";
  el.customRange.value = "";
  el.taskRetryCount.value = "";
  el.taskHttpRequestTimeout.value = "";
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
    defaultSaveDir: el.defaultSaveDir.value.trim(),
    ffmpegPath: el.ffmpegPath.value.trim(),
    baseURL: el.baseURL.value.trim(),
    tmpDir: el.tmpDir.value.trim(),
    savePattern: el.savePattern.value.trim(),
    linkNameSeparator: el.linkNameSeparator.value.trim() || defaultLinkNameSeparator,
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
    outputFormat: currentOutputFormat(),
    useFFmpegMerge: el.useFFmpegMerge.checked,
    muxMP4: el.useFFmpegMerge.checked && currentOutputFormat() === "mp4",
    noDateInfo: el.noDateInfo.checked,
    binaryMerge: !el.useFFmpegMerge.checked,
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
  applyingSettings = true;
  settings = nextSettings || {};
  const defaultSaveDir = field(settings, "defaultSaveDir", "DefaultSaveDir", "");
  el.defaultSaveDir.value = defaultSaveDir;
  if (!el.saveDir.value.trim()) {
    el.saveDir.value = defaultSaveDir;
  }
  el.ffmpegPath.value = field(settings, "ffmpegPath", "FFmpegPath", "");
  el.baseURL.value = field(settings, "baseURL", "BaseURL", "");
  el.tmpDir.value = field(settings, "tmpDir", "TmpDir", "");
  el.savePattern.value = field(settings, "savePattern", "SavePattern", "");
  el.linkNameSeparator.value = field(settings, "linkNameSeparator", "LinkNameSeparator", defaultLinkNameSeparator) || defaultLinkNameSeparator;
  updateURLPlaceholder();
  el.maxActiveTasks.value = field(settings, "maxActiveTasks", "MaxActiveTasks", 2);
  el.threadCount.value = field(settings, "threadCount", "ThreadCount", 8);
  el.retryCount.value = field(settings, "retryCount", "RetryCount", 3);
  el.httpRequestTimeout.value = field(settings, "httpRequestTimeout", "HTTPRequestTimeout", 100);
  updateTaskOverridePlaceholders();
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
  el.outputFormat.value = normalizeOutputFormat(field(settings, "outputFormat", "OutputFormat", field(settings, "muxMP4", "MuxMP4", false) ? "mp4" : defaultOutputFormat));
  el.useFFmpegMerge.checked = Boolean(field(settings, "useFFmpegMerge", "UseFFmpegMerge", field(settings, "muxMP4", "MuxMP4", true)));
  syncMergeControls();
  el.noDateInfo.checked = Boolean(field(settings, "noDateInfo", "NoDateInfo", false));
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
  applyingSettings = false;
  settingsDirty = false;
  updateSettingsSaveState();
}

function applyCoreInfo(info = {}) {
  const status = field(info, "status", "Status", "error");
  const error = field(info, "error", "Error", "");
  const version = field(info, "version", "Version", "");
  const fullVersion = field(info, "fullVersion", "FullVersion", "");
  el.coreStatus.textContent = status === "ready" ? "可用" : "异常";
  el.coreStatus.classList.toggle("error", status !== "ready");
  el.coreVersion.textContent = fullVersion || "-";
  // long: 品牌区展示核心真实版本，应用升级后无需再维护一份独立的前端版本常量。
  if (version && el.versionLabel) {
    el.versionLabel.textContent = `v${String(version).replace(/^v/i, "")}`;
  }
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
  const backend = api();
  if (!backend) {
    // long: 静态预览使用隔离的演示任务还原真实工作台密度；Wails 后端存在时不会读取这些数据。
    applySettings({});
    renderToolList();
    tasks = previewTasks.map(normalizeTask);
    selectedTaskId = tasks[0]?.id || "";
    renderTasks();
    el.coreStatus.textContent = "未连接";
    el.coreVersion.textContent = "-";
    el.corePath.textContent = "请在桌面客户端中查看";
    el.coreCapabilities.innerHTML = `<div class="capability-empty">桌面运行时连接后显示核心能力。</div>`;
    updateSettingsSaveState("预览模式");
    return;
  }
  applySettings(await backend.GetSettings());
  renderToolList();
  await refreshCoreInfo();
  tasks = (await backend.ListTasks()).map(normalizeTask);
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
  updateURLCount();
  resetPreflightResult();
  closeCreateDialog();
  switchView("dashboard");
}

async function refreshTasks() {
  tasks = (await api().ListTasks()).map(normalizeTask);
  if (selectedTaskId && !tasks.some((task) => task.id === selectedTaskId)) {
    selectedTaskId = tasks[0]?.id || "";
  }
  renderTasks();
}

function nextTaskIdAfterRemoving(taskId) {
  const visibleTasks = tasks.filter(taskMatchesCurrentFilter).slice().sort(compareTasks);
  const index = visibleTasks.findIndex((task) => task.id === taskId);
  if (index < 0) return "";
  return visibleTasks[index + 1]?.id || visibleTasks[index - 1]?.id || "";
}

async function removeTask(taskId = selectedTaskId, button = el.removeSelected) {
  const task = tasks.find((item) => item.id === taskId);
  const state = taskActionAvailability(task);
  if (!state.hasTask) {
    notify("请先选择一个任务。", "warning");
    return;
  }
  if (!state.canRemove) {
    notify("运行中的任务不能移除。", "warning");
    return;
  }

  // long: 移除任务只删除桌面端队列记录，不处理下载完成的媒体文件，所以确认文案要把影响范围说清楚。
  const confirmed = await confirmAction({
    title: "移除任务",
    message: `确认移除任务「${task.title || task.request?.url || task.id}」？输出文件不会删除。`,
    confirmText: "移除",
    danger: true
  });
  if (!confirmed) return;

  const nextTaskId = nextTaskIdAfterRemoving(taskId);
  await withBusy(button, "移除中", async () => {
    try {
      await api().RemoveTask(taskId);
      selectedTaskId = nextTaskId;
      await refreshTasks();
      notify("任务已移除。", "success");
    } catch (error) {
      reportError(error);
    }
  });
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
  updateSettingsSaveState();
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

el.chooseDefaultDir.addEventListener("click", async () => {
  await withBusy(el.chooseDefaultDir, "选择中", async () => {
    try {
      const selected = await api().ChooseDirectory(el.defaultSaveDir.value.trim());
      if (selected) {
        el.defaultSaveDir.value = selected;
        markSettingsDirty();
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
        markSettingsDirty();
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
        markSettingsDirty();
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
        markSettingsDirty();
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
  markSettingsDirty();
  notify("外部轨道已添加。", "success");
});

el.saveSettings.addEventListener("click", async () => {
  await saveCurrentSettings(el.saveSettings);
});
el.resetForm.addEventListener("click", () => {
  resetCreateForm();
  createDialogBaseline = createDraftSnapshot();
  notify("表单已恢复为默认参数。", "success");
});
el.linkNameSeparator.addEventListener("input", updateURLPlaceholder);
el.url.addEventListener("input", updateURLCount);
el.retryCount.addEventListener("input", updateTaskOverridePlaceholders);
el.httpRequestTimeout.addEventListener("input", updateTaskOverridePlaceholders);
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
        markSettingsDirty();
      }
    } catch (error) {
      reportError(error);
    }
  });
});
el.outputFormat.addEventListener("change", syncMergeControls);
el.useFFmpegMerge.addEventListener("change", syncMergeControls);
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
el.settingsPanel?.addEventListener("input", (event) => {
  if (event.target.matches("input, select, textarea")) {
    markSettingsDirty();
  }
});
el.settingsPanel?.addEventListener("change", (event) => {
  if (event.target.matches("input, select, textarea")) {
    markSettingsDirty();
  }
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
  const confirmed = await confirmAction({
    title: "停止运行任务",
    message: "确认停止所有运行中的任务？已经下载的临时片段会按当前任务参数保留或清理。",
    confirmText: "停止",
    danger: false
  });
  if (!confirmed) return;
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
  openCreateDialog();
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
  await removeTask(selectedTaskId, el.removeSelected);
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
el.taskContextMenu?.addEventListener("click", (event) => {
  const button = event.target.closest("[data-task-context-action]");
  if (!button) return;
  if (button.getAttribute("aria-disabled") === "true") {
    notify(button.dataset.disabledReason || "当前状态不能执行这个操作。", "warning");
    return;
  }
  void runTaskContextAction(button.dataset.taskContextAction);
});
el.taskContextMenu?.addEventListener("keydown", (event) => {
  const items = Array.from(el.taskContextMenu.querySelectorAll('[role="menuitem"][aria-disabled="false"]'));
  if (!items.length) return;
  const currentIndex = Math.max(0, items.indexOf(document.activeElement));
  let nextIndex = -1;
  if (event.key === "ArrowDown") nextIndex = (currentIndex + 1) % items.length;
  if (event.key === "ArrowUp") nextIndex = (currentIndex - 1 + items.length) % items.length;
  if (event.key === "Home") nextIndex = 0;
  if (event.key === "End") nextIndex = items.length - 1;
  if (nextIndex >= 0) {
    event.preventDefault();
    items[nextIndex]?.focus();
    return;
  }
  if (event.key === "Tab") {
    event.preventDefault();
    closeTaskContextMenu({ restoreFocus: true });
  }
});
el.confirmCancel?.addEventListener("click", () => closeConfirmDialog(false));
el.confirmOK?.addEventListener("click", () => {
  if (Date.now() < confirmReadyAt) return;
  closeConfirmDialog(true);
});
el.confirmDialog?.addEventListener("click", (event) => {
  if (event.target === el.confirmDialog) {
    closeConfirmDialog(false);
  }
});
el.taskList?.addEventListener("scroll", closeTaskContextMenu);
el.settingsTabs.forEach((button) => {
  button.addEventListener("click", () => switchSettingsTab(button.dataset.settingsTab, { resetScroll: true }));
  button.addEventListener("keydown", (event) => {
    if (event.key === "ArrowRight") {
      event.preventDefault();
      focusSettingsTabByOffset(1);
      return;
    }
    if (event.key === "ArrowLeft") {
      event.preventDefault();
      focusSettingsTabByOffset(-1);
      return;
    }
    if (event.key === "Home") {
      event.preventDefault();
      switchSettingsTab(el.settingsTabs[0]?.dataset.settingsTab, { resetScroll: true });
      el.settingsTabs[0]?.focus();
      return;
    }
    if (event.key === "End") {
      event.preventDefault();
      const lastTab = el.settingsTabs[el.settingsTabs.length - 1];
      switchSettingsTab(lastTab?.dataset.settingsTab, { resetScroll: true });
      lastTab?.focus();
    }
  });
});
el.navItems.forEach((button) => {
  button.addEventListener("click", () => switchView(button.dataset.view));
});
el.themeToggle?.addEventListener("click", cycleThemeMode);
if (systemThemeMedia) {
  const handleSystemThemeChange = () => {
    // long: 只有自动模式跟随 macOS 变化，手动浅色/深色选择必须保持稳定。
    if (document.documentElement.dataset.themeMode === "auto") applyThemeMode("auto");
  };
  if (typeof systemThemeMedia.addEventListener === "function") {
    systemThemeMedia.addEventListener("change", handleSystemThemeChange);
  } else if (typeof systemThemeMedia.addListener === "function") {
    systemThemeMedia.addListener(handleSystemThemeChange);
  }
}
document.querySelectorAll("[data-view-shortcut]").forEach((button) => {
  button.addEventListener("click", () => {
    if (button.dataset.viewShortcut === "create") {
      openCreateDialog({ reset: true });
      return;
    }
    switchView(button.dataset.viewShortcut);
  });
});
el.newTaskFab?.addEventListener("click", () => openCreateDialog({ reset: true }));
el.closeCreateDialog?.addEventListener("click", () => void requestCloseCreateDialog());
el.cancelCreateDialog?.addEventListener("click", () => void requestCloseCreateDialog());
el.createDialog?.addEventListener("click", (event) => {
  if (!createPanelIsInline() && event.target === el.createDialog) {
    void requestCloseCreateDialog();
  }
});
el.taskMoreActions?.addEventListener("click", (event) => {
  if (event.target.closest("button")) {
    window.setTimeout(() => {
      el.taskMoreActions.open = false;
    }, 0);
  }
});
window.addEventListener("click", (event) => {
  if (!el.taskContextMenu || el.taskContextMenu.hidden || el.taskContextMenu.contains(event.target)) return;
  closeTaskContextMenu();
});
window.addEventListener("click", (event) => {
  if (el.taskMoreActions?.open && !el.taskMoreActions.contains(event.target)) {
    el.taskMoreActions.open = false;
  }
});
window.addEventListener("contextmenu", (event) => {
  if (event.target.closest?.(".task-card") || event.target.closest?.(".task-context-menu")) return;
  closeTaskContextMenu();
});
window.addEventListener("resize", closeTaskContextMenu);
window.addEventListener("keydown", (event) => {
  if (trapDialogFocus(event, el.confirmDialog)) return;
  if (!createPanelIsInline() && trapDialogFocus(event, el.createDialog)) return;

  const commandKey = event.metaKey || event.ctrlKey;
  const key = event.key.toLowerCase();
  if (commandKey && key === "n" && (!el.confirmDialog || el.confirmDialog.hidden) && (createPanelIsInline() || !el.createDialog || el.createDialog.hidden)) {
    event.preventDefault();
    openCreateDialog({ reset: true });
    return;
  }
  if (commandKey && key === "s" && activeView === "settings") {
    event.preventDefault();
    if (settingsDirty && !el.saveSettings.disabled) {
      el.saveSettings.click();
    }
    return;
  }
  if (commandKey && event.key === "Enter" && el.createDialog && !el.createDialog.hidden) {
    event.preventDefault();
    document.querySelector("#task-form")?.requestSubmit(document.querySelector("#create-start"));
    return;
  }
  if (event.key === "Escape") {
    if (el.taskContextMenu && !el.taskContextMenu.hidden) {
      event.preventDefault();
      closeTaskContextMenu({ restoreFocus: true });
      return;
    }
    if (el.taskMoreActions) {
      el.taskMoreActions.open = false;
    }
  }
  if (event.key === "Escape" && el.confirmDialog && !el.confirmDialog.hidden) {
    closeConfirmDialog(false);
    return;
  }
  if (event.key === "Escape" && !createPanelIsInline() && el.createDialog && !el.createDialog.hidden) {
    void requestCloseCreateDialog();
  }
});

// long: 初始化创建区的草稿基线；内嵌模式保持左栏常驻，旧弹窗模式仍按原行为收起。
applyThemeMode(readThemeMode());
closeCreateDialog();
switchView("dashboard");
loadInitialState().catch((error) => {
  el.taskList.innerHTML = `<div class="empty">初始化失败：${escapeHTML(error)}</div>`;
});
