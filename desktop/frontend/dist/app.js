const api = () => window.go?.main?.App;

const el = {
  url: document.querySelector("#url"),
  saveDir: document.querySelector("#saveDir"),
  saveName: document.querySelector("#saveName"),
  customRange: document.querySelector("#customRange"),
  headers: document.querySelector("#headers"),
  autoSelect: document.querySelector("#autoSelect"),
  muxMP4: document.querySelector("#muxMP4"),
  binaryMerge: document.querySelector("#binaryMerge"),
  concurrentDownload: document.querySelector("#concurrentDownload"),
  ffmpegPath: document.querySelector("#ffmpegPath"),
  threadCount: document.querySelector("#threadCount"),
  retryCount: document.querySelector("#retryCount"),
  maxSpeed: document.querySelector("#maxSpeed"),
  customProxy: document.querySelector("#customProxy"),
  useSystemProxy: document.querySelector("#useSystemProxy"),
  taskList: document.querySelector("#task-list"),
  taskCount: document.querySelector("#task-count"),
  detailTitle: document.querySelector("#detail-title"),
  taskSummary: document.querySelector("#task-summary"),
  fileList: document.querySelector("#file-list"),
  log: document.querySelector("#log")
};

let settings = {};
let tasks = [];
let selectedTaskId = "";
let createOnly = false;

function field(source, camel, pascal, fallback = "") {
  return source?.[camel] ?? source?.[pascal] ?? fallback;
}

function normalizeRequest(request = {}) {
  return {
    url: field(request, "url", "URL"),
    saveDir: field(request, "saveDir", "SaveDir"),
    saveName: field(request, "saveName", "SaveName"),
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
    lastMessage: field(task, "lastMessage", "LastMessage"),
    exitCode: Number(field(task, "exitCode", "ExitCode", 0)) || 0,
    createdAt: field(task, "createdAt", "CreatedAt"),
    startedAt: field(task, "startedAt", "StartedAt"),
    finishedAt: field(task, "finishedAt", "FinishedAt"),
    request: normalizeRequest(field(task, "request", "Request", {})),
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

function renderTasks() {
  el.taskCount.textContent = `${tasks.length} 个任务`;
  if (!tasks.length) {
    el.taskList.innerHTML = `<div class="empty">还没有任务。输入 m3u8 地址创建第一个下载任务。</div>`;
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
        <span>${escapeHTML(task.lastMessage || "")}</span>
      </div>
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
  el.taskSummary.className = "summary";
  el.taskSummary.innerHTML = `
    <div><span>状态</span><strong>${taskStatusText(task.status)}</strong></div>
    <div><span>输出目录</span><strong>${escapeHTML(task.request.saveDir)}</strong></div>
    <div><span>保存名</span><strong>${escapeHTML(task.request.saveName || "自动")}</strong></div>
    <div><span>参数</span><strong>${task.request.autoSelect ? "自动选轨" : "手动选轨"} / ${task.request.muxMP4 ? "MP4" : "原始输出"}</strong></div>
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

async function loadInitialState() {
  applySettings(await api().GetSettings());
  tasks = (await api().ListTasks()).map(normalizeTask);
  if (tasks.length && !selectedTaskId) {
    selectedTaskId = tasks[0].id;
  }
  renderTasks();
}

async function createTask(startNow) {
  const task = normalizeTask(await api().CreateTask(collectRequest()));
  upsertTask(task);
  selectedTaskId = task.id;
  if (startNow) {
    await api().StartTask(task.id);
  }
  el.url.value = "";
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

window.runtime?.EventsOn("task:update", (rawTask) => upsertTask(rawTask));
window.runtime?.EventsOn("task:log", (event) => {
  const taskId = field(event, "taskId", "TaskID");
  const line = field(event, "line", "Line");
  const task = tasks.find((item) => item.id === taskId);
  if (task) {
    task.logs = [...(task.logs || []), line].slice(-500);
    task.lastMessage = line;
  }
  if (taskId === selectedTaskId) {
    el.log.textContent += `${line}\n`;
    el.log.scrollTop = el.log.scrollHeight;
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
  await createTask(!createOnly);
  createOnly = false;
});

document.querySelector("#create-only").addEventListener("click", () => {
  createOnly = true;
  document.querySelector("#task-form").requestSubmit();
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

loadInitialState().catch((error) => {
  el.taskList.innerHTML = `<div class="empty">初始化失败：${escapeHTML(error)}</div>`;
});
