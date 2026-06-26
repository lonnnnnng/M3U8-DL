package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusStopped   = "stopped"
)

const maxTaskLogs = 500

var (
	progressRegex      = regexp.MustCompile(`(?i)(?:下载进度|download progress)\s+(\d+)\s*/\s*(\d+)(?:[，,]\s*(?:速度|speed)\s+(.+))?`)
	taskLogTimestampRE = regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\]`)
)

type App struct {
	ctx       context.Context
	mu        sync.Mutex
	tasks     map[string]*Task
	order     []string
	running   map[string]*runningTask
	settings  Settings
	statePath string
}

type runningTask struct {
	cancel context.CancelFunc
	cmd    *exec.Cmd
}

type Settings struct {
	DefaultSaveDir     string `json:"defaultSaveDir"`
	FFmpegPath         string `json:"ffmpegPath"`
	MaxActiveTasks     int    `json:"maxActiveTasks"`
	ThreadCount        int    `json:"threadCount"`
	RetryCount         int    `json:"retryCount"`
	MaxSpeed           string `json:"maxSpeed"`
	AutoSelect         bool   `json:"autoSelect"`
	MuxMP4             bool   `json:"muxMP4"`
	BinaryMerge        bool   `json:"binaryMerge"`
	ConcurrentDownload bool   `json:"concurrentDownload"`
	UseSystemProxy     bool   `json:"useSystemProxy"`
	CustomProxy        string `json:"customProxy"`
}

type DownloadRequest struct {
	URL                string   `json:"url"`
	SaveDir            string   `json:"saveDir"`
	SaveName           string   `json:"saveName"`
	LinkNameSeparator  string   `json:"linkNameSeparator"`
	Headers            []string `json:"headers"`
	CustomRange        string   `json:"customRange"`
	FFmpegPath         string   `json:"ffmpegPath"`
	ThreadCount        int      `json:"threadCount"`
	RetryCount         int      `json:"retryCount"`
	MaxSpeed           string   `json:"maxSpeed"`
	AutoSelect         bool     `json:"autoSelect"`
	MuxMP4             bool     `json:"muxMP4"`
	BinaryMerge        bool     `json:"binaryMerge"`
	ConcurrentDownload bool     `json:"concurrentDownload"`
	UseSystemProxy     bool     `json:"useSystemProxy"`
	CustomProxy        string   `json:"customProxy"`
}

type Task struct {
	ID             string          `json:"id"`
	Title          string          `json:"title"`
	Status         string          `json:"status"`
	Queued         bool            `json:"queued"`
	Progress       float64         `json:"progress"`
	ProgressText   string          `json:"progressText"`
	SpeedText      string          `json:"speedText"`
	ElapsedText    string          `json:"elapsedText,omitempty"`
	RemainingText  string          `json:"remainingText,omitempty"`
	LastMessage    string          `json:"lastMessage"`
	FailureMessage string          `json:"failureMessage,omitempty"`
	ExitCode       int             `json:"exitCode"`
	CreatedAt      time.Time       `json:"createdAt"`
	StartedAt      *time.Time      `json:"startedAt,omitempty"`
	FinishedAt     *time.Time      `json:"finishedAt,omitempty"`
	Request        DownloadRequest `json:"request"`
	Args           []string        `json:"args"`
	Command        string          `json:"command"`
	CommandLine    string          `json:"commandLine,omitempty"`
	Logs           []string        `json:"logs"`
	Files          []TaskFile      `json:"files"`
}

type TaskFile struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Modified  time.Time `json:"modified"`
	Extension string    `json:"extension"`
}

type CoreInfo struct {
	Status      string `json:"status"`
	CLIPath     string `json:"cliPath"`
	Version     string `json:"version"`
	FullVersion string `json:"fullVersion"`
	Error       string `json:"error,omitempty"`
}

type ToolInfo struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Command string `json:"command"`
	Status  string `json:"status"`
	Path    string `json:"path"`
	Version string `json:"version"`
	Error   string `json:"error,omitempty"`
}

type PreflightReport struct {
	Status       string           `json:"status"`
	Message      string           `json:"message"`
	TaskCount    int              `json:"taskCount"`
	CommandLines []string         `json:"commandLines"`
	Checks       []PreflightCheck `json:"checks"`
}

type PreflightCheck struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

type toolCheckSpec struct {
	Name     string
	Label    string
	Commands []string
	Args     []string
}

type cliVersionInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	FullVersion string `json:"fullVersion"`
}

type cliProgressEvent struct {
	Type      string  `json:"type"`
	Timestamp string  `json:"timestamp"`
	Stream    string  `json:"stream"`
	Current   int     `json:"current"`
	Total     int     `json:"total"`
	Speed     string  `json:"speed"`
	Bytes     int64   `json:"bytes"`
	Percent   float64 `json:"percent"`
}

type cliSummaryEvent struct {
	Type      string             `json:"type"`
	Timestamp string             `json:"timestamp"`
	Status    string             `json:"status"`
	Outputs   []cliSummaryOutput `json:"outputs"`
}

type cliSummaryOutput struct {
	Path        string `json:"path"`
	MediaType   string `json:"mediaType,omitempty"`
	Language    string `json:"language,omitempty"`
	Name        string `json:"name,omitempty"`
	StreamCount int    `json:"streamCount,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

type cliErrorEvent struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

type taskLogEvent struct {
	TaskID string `json:"taskId"`
	Line   string `json:"line"`
}

type persistedState struct {
	Settings Settings `json:"settings"`
	Order    []string `json:"order"`
	Tasks    []*Task  `json:"tasks"`
}

func NewApp() *App {
	app := &App{
		tasks:    map[string]*Task{},
		running:  map[string]*runningTask{},
		settings: defaultSettings(),
	}
	app.statePath = app.defaultStatePath()
	_ = app.loadState()
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func defaultSettings() Settings {
	return Settings{
		DefaultSaveDir:     defaultSaveDir(),
		MaxActiveTasks:     2,
		ThreadCount:        8,
		RetryCount:         3,
		AutoSelect:         true,
		MuxMP4:             true,
		UseSystemProxy:     true,
		ConcurrentDownload: false,
	}
}

func defaultSaveDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	return filepath.Join(home, "Downloads", "m3u8dl-go")
}

func (a *App) defaultStatePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		configDir = filepath.Join(defaultSaveDir(), ".config")
	}
	return filepath.Join(configDir, "m3u8dl-go", "desktop-state.json")
}

func (a *App) DefaultSaveDir() string {
	return defaultSaveDir()
}

func (a *App) GetSettings() Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.settings = normalizeSettings(a.settings)
	return a.settings
}

func (a *App) GetCoreInfo() CoreInfo {
	cliPath, err := findCLIPath()
	if err != nil {
		return CoreInfo{
			Status: "error",
			Error:  err.Error(),
		}
	}
	info, err := readCLIVersion(cliPath)
	if err != nil {
		return CoreInfo{
			Status:  "error",
			CLIPath: cliPath,
			Error:   err.Error(),
		}
	}
	info.CLIPath = cliPath
	return info
}

func (a *App) CheckFFmpeg(path string) ToolInfo {
	info, err := probeToolVersion(strings.TrimSpace(path), "ffmpeg", []string{"-version"})
	info.Name = "ffmpeg"
	info.Label = "FFmpeg"
	if err != nil {
		info.Status = "error"
		info.Error = err.Error()
	}
	return info
}

func (a *App) CheckTools(ffmpegPath string) []ToolInfo {
	specs := desktopToolCheckSpecs(ffmpegPath)
	tools := make([]ToolInfo, 0, len(specs))
	for _, spec := range specs {
		info, err := probeToolCandidates(spec.Commands, spec.Args)
		info.Name = spec.Name
		info.Label = spec.Label
		if len(spec.Commands) > 0 {
			info.Command = spec.Commands[0]
		}
		if err != nil {
			info.Status = "error"
			info.Error = err.Error()
		}
		tools = append(tools, info)
	}
	return tools
}

func (a *App) SaveSettings(settings Settings) (Settings, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.settings = normalizeSettings(settings)
	if err := a.saveStateLocked(); err != nil {
		return a.settings, err
	}
	return a.settings, nil
}

func normalizeSettings(settings Settings) Settings {
	if strings.TrimSpace(settings.DefaultSaveDir) == "" {
		settings.DefaultSaveDir = defaultSaveDir()
	}
	settings.CustomProxy = clearRedactedProxyForRuntime(settings.CustomProxy)
	if settings.ThreadCount <= 0 {
		settings.ThreadCount = 8
	}
	if settings.MaxActiveTasks <= 0 {
		settings.MaxActiveTasks = 2
	}
	if settings.RetryCount < 0 {
		settings.RetryCount = 3
	}
	return settings
}

func (a *App) ChooseDirectory(current string) (string, error) {
	if a.ctx == nil {
		return "", errors.New("应用尚未初始化")
	}
	selected, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:            "选择输出目录",
		DefaultDirectory: current,
	})
	if err != nil {
		return "", err
	}
	return selected, nil
}

func (a *App) ListTasks() []Task {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.taskSnapshotsLocked()
}

func (a *App) CreateTask(req DownloadRequest) (Task, error) {
	requests, err := expandDownloadRequests(req)
	if err != nil {
		return Task{}, err
	}
	if len(requests) != 1 {
		return Task{}, errors.New("检测到多行地址，请使用批量创建任务")
	}
	return a.createTask(requests[0])
}

func (a *App) PreviewCommands(req DownloadRequest) (string, error) {
	requests, err := expandDownloadRequests(req)
	if err != nil {
		return "", err
	}
	lines := make([]string, 0, len(requests))
	for _, item := range requests {
		item = a.applyRequestDefaults(item)
		lines = append(lines, taskCommandLine(defaultCLICommandName(), buildCLIArgs(item)))
	}
	return strings.Join(lines, "\n"), nil
}

func (a *App) PreflightDownload(req DownloadRequest) PreflightReport {
	report := PreflightReport{Status: "ready", Message: "预检查通过"}
	requests, err := expandDownloadRequests(req)
	if err != nil {
		report.addCheck("input", "下载地址", "error", err.Error(), "")
		report.finalize()
		return report
	}
	report.TaskCount = len(requests)
	if len(requests) == 1 {
		report.addCheck("input", "下载地址", "ready", "已识别 1 个任务", requests[0].URL)
	} else {
		report.addCheck("input", "下载地址", "ready", fmt.Sprintf("已识别 %d 个批量任务", len(requests)), "")
	}

	cliPath := ""
	coreReady := false
	cliPath, coreErr := findCLIPath()
	if coreErr != nil {
		report.addCheck("core", "下载核心", "error", "找不到下载核心", coreErr.Error())
	} else if info, err := readCLIVersion(cliPath); err != nil {
		report.addCheck("core", "下载核心", "error", "下载核心不可用", err.Error())
	} else {
		coreReady = true
		report.addCheck("core", "下载核心", "ready", info.FullVersion, cliPath)
	}

	checkedDirs := map[string]bool{}
	checkedFFmpeg := map[string]bool{}
	var validationErrors []string
	for _, item := range requests {
		item = a.applyRequestDefaults(item)
		args := buildCLIArgs(item)
		report.CommandLines = append(report.CommandLines, taskCommandLine(defaultCLICommandName(), args))

		if coreReady {
			if err := validateCLIArguments(cliPath, args); err != nil {
				validationErrors = append(validationErrors, redactPreflightError(err.Error(), item))
			}
		}

		if !checkedDirs[item.SaveDir] {
			checkedDirs[item.SaveDir] = true
			if err := ensureWritableDirectory(item.SaveDir); err != nil {
				report.addCheck("output", "输出目录", "error", "输出目录不可写", err.Error())
			} else {
				report.addCheck("output", "输出目录", "ready", "可写", item.SaveDir)
			}
		}

		if item.MuxMP4 || strings.TrimSpace(item.FFmpegPath) != "" {
			key := strings.TrimSpace(item.FFmpegPath)
			if key == "" {
				key = "ffmpeg"
			}
			if checkedFFmpeg[key] {
				continue
			}
			checkedFFmpeg[key] = true
			info, err := probeToolVersion(item.FFmpegPath, "ffmpeg", []string{"-version"})
			if err != nil {
				report.addCheck("ffmpeg", "FFmpeg", "error", "MP4 混流需要 FFmpeg", err.Error())
			} else {
				report.addCheck("ffmpeg", "FFmpeg", "ready", info.Version, info.Path)
			}
		}
	}
	if coreReady {
		if len(validationErrors) == 0 {
			report.addCheck("args", "参数校验", "ready", "核心参数校验通过", fmt.Sprintf("%d 个任务", len(requests)))
		} else {
			report.addCheck("args", "参数校验", "error", "核心参数校验失败", strings.Join(validationErrors, "\n"))
		}
	}
	report.finalize()
	return report
}

func (a *App) CreateTasks(req DownloadRequest) ([]Task, error) {
	requests, err := expandDownloadRequests(req)
	if err != nil {
		return nil, err
	}
	tasks := make([]Task, len(requests))
	for i := len(requests) - 1; i >= 0; i-- {
		task, err := a.createTask(requests[i])
		if err != nil {
			return nil, err
		}
		tasks[i] = task
	}
	return tasks, nil
}

func (a *App) createTask(req DownloadRequest) (Task, error) {
	req = a.applyRequestDefaults(req)
	if strings.TrimSpace(req.URL) == "" {
		return Task{}, errors.New("请先填写 m3u8 地址")
	}
	if err := os.MkdirAll(req.SaveDir, 0755); err != nil {
		return Task{}, fmt.Errorf("创建输出目录失败: %w", err)
	}

	args := buildCLIArgs(req)
	task := &Task{
		ID:          newTaskID(),
		Title:       taskTitle(req),
		Status:      StatusPending,
		CreatedAt:   time.Now(),
		Request:     req,
		Args:        args,
		Command:     defaultCLICommandName(),
		CommandLine: taskCommandLine(defaultCLICommandName(), args),
	}

	a.mu.Lock()
	a.tasks[task.ID] = task
	a.order = append([]string{task.ID}, a.order...)
	_ = a.saveStateLocked()
	snapshot := taskSnapshot(task)
	a.mu.Unlock()

	a.emitTask(snapshot)
	return snapshot, nil
}

func (a *App) StartTask(id string) error {
	a.mu.Lock()
	task := a.tasks[id]
	if task == nil {
		a.mu.Unlock()
		return errors.New("任务不存在")
	}
	if _, ok := a.running[id]; ok {
		a.mu.Unlock()
		return nil
	}
	if task.Status == StatusRunning {
		a.mu.Unlock()
		return nil
	}
	if !a.hasTaskSlotLocked() {
		task.Status = StatusPending
		task.Queued = true
		task.ElapsedText = ""
		task.RemainingText = ""
		task.LastMessage = "已加入队列，等待空闲任务槽"
		task.FailureMessage = ""
		task.ExitCode = 0
		now := time.Now()
		task.Logs = appendLimited(task.Logs, timestampTaskLogLine("已加入队列，等待空闲任务槽。", now))
		task.FinishedAt = nil
		a.emitTaskLocked(task)
		_ = a.saveStateLocked()
		a.mu.Unlock()
		return nil
	}
	task.Status = StatusRunning
	task.Queued = false
	task.Progress = 0
	task.ProgressText = ""
	task.SpeedText = ""
	task.ElapsedText = "0秒"
	task.RemainingText = ""
	task.LastMessage = "正在启动下载核心"
	task.FailureMessage = ""
	task.ExitCode = 0
	task.Files = nil
	now := time.Now()
	task.Logs = appendLimited(task.Logs, timestampTaskLogLine("正在启动下载核心", now))
	task.StartedAt = &now
	task.FinishedAt = nil
	a.emitTaskLocked(task)
	_ = a.saveStateLocked()
	a.mu.Unlock()

	go a.runTask(id)
	return nil
}

func (a *App) StopTask(id string) error {
	a.mu.Lock()
	running := a.running[id]
	task := a.tasks[id]
	if running == nil && task != nil && task.Status == StatusPending && task.Queued {
		task.Queued = false
		task.ElapsedText = ""
		task.RemainingText = ""
		task.LastMessage = "已取消排队"
		task.Logs = appendLimited(task.Logs, timestampTaskLogLine("已取消排队。", time.Now()))
		_ = a.saveStateLocked()
		a.emitTaskLocked(task)
		a.mu.Unlock()
		return nil
	}
	a.mu.Unlock()
	if running == nil {
		return nil
	}
	running.cancel()
	a.appendTaskLog(id, "已请求停止当前任务。")
	return nil
}

func (a *App) RetryTask(id string) error {
	a.mu.Lock()
	task := a.tasks[id]
	if task == nil {
		a.mu.Unlock()
		return errors.New("任务不存在")
	}
	if _, ok := a.running[id]; ok {
		a.mu.Unlock()
		return errors.New("任务正在运行")
	}
	task.Status = StatusPending
	task.Queued = true
	task.Progress = 0
	task.ProgressText = ""
	task.SpeedText = ""
	task.ElapsedText = ""
	task.RemainingText = ""
	task.LastMessage = "等待重新开始"
	task.FailureMessage = ""
	task.ExitCode = 0
	task.FinishedAt = nil
	task.Logs = appendLimited(task.Logs, timestampTaskLogLine("任务已重新排队。", time.Now()))
	_ = a.saveStateLocked()
	a.emitTaskLocked(task)
	a.mu.Unlock()
	return a.StartTask(id)
}

func (a *App) StartPendingTasks() (int, error) {
	ids := a.taskIDsByStatus(StatusPending)
	return a.startTasksByID(ids)
}

func (a *App) StopRunningTasks() (int, error) {
	ids := a.runningTaskIDs()
	for _, id := range ids {
		if err := a.StopTask(id); err != nil {
			return len(ids), err
		}
	}
	return len(ids), nil
}

func (a *App) RetryFailedTasks() (int, error) {
	ids := a.taskIDsByStatus(StatusFailed)
	count := 0
	for _, id := range ids {
		if err := a.RetryTask(id); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (a *App) RemoveTask(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.running[id]; ok {
		return errors.New("任务正在运行，不能移除")
	}
	if a.tasks[id] == nil {
		return nil
	}
	delete(a.tasks, id)
	a.order = removeID(a.order, id)
	return a.saveStateLocked()
}

func (a *App) startTasksByID(ids []string) (int, error) {
	count := 0
	for _, id := range ids {
		if err := a.StartTask(id); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (a *App) taskIDsByStatus(statuses ...string) []string {
	wanted := map[string]bool{}
	for _, status := range statuses {
		wanted[status] = true
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	ids := make([]string, 0, len(a.order))
	for i := len(a.order) - 1; i >= 0; i-- {
		id := a.order[i]
		task := a.tasks[id]
		if task != nil && wanted[task.Status] {
			ids = append(ids, id)
		}
	}
	return ids
}

func (a *App) runningTaskIDs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	ids := make([]string, 0, len(a.order))
	for i := len(a.order) - 1; i >= 0; i-- {
		id := a.order[i]
		if _, ok := a.running[id]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func (a *App) hasTaskSlotLocked() bool {
	return a.activeTaskCountLocked() < a.maxActiveTasksLocked()
}

func (a *App) activeTaskCountLocked() int {
	count := 0
	for _, task := range a.tasks {
		if task != nil && task.Status == StatusRunning {
			count++
		}
	}
	return count
}

func (a *App) maxActiveTasksLocked() int {
	if a.settings.MaxActiveTasks <= 0 {
		return 2
	}
	return a.settings.MaxActiveTasks
}

func (a *App) scheduleQueuedTasks() {
	for {
		a.mu.Lock()
		if !a.hasTaskSlotLocked() {
			a.mu.Unlock()
			return
		}
		task := a.nextQueuedTaskLocked()
		if task == nil {
			a.mu.Unlock()
			return
		}
		id := task.ID
		task.Status = StatusRunning
		task.Queued = false
		task.Progress = 0
		task.ProgressText = ""
		task.SpeedText = ""
		task.ElapsedText = "0秒"
		task.RemainingText = ""
		task.LastMessage = "正在启动下载核心"
		task.FailureMessage = ""
		task.ExitCode = 0
		task.Files = nil
		now := time.Now()
		task.Logs = appendLimited(task.Logs, timestampTaskLogLine("正在启动下载核心", now))
		task.StartedAt = &now
		task.FinishedAt = nil
		a.emitTaskLocked(task)
		_ = a.saveStateLocked()
		a.mu.Unlock()
		go a.runTask(id)
	}
}

func (a *App) nextQueuedTaskLocked() *Task {
	for i := len(a.order) - 1; i >= 0; i-- {
		task := a.tasks[a.order[i]]
		if task != nil && task.Status == StatusPending && task.Queued {
			return task
		}
	}
	return nil
}

func (a *App) ClearFinishedTasks() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, task := range a.tasks {
		if task.Status == StatusCompleted || task.Status == StatusFailed || task.Status == StatusStopped {
			if _, running := a.running[id]; !running {
				delete(a.tasks, id)
				a.order = removeID(a.order, id)
			}
		}
	}
	return a.saveStateLocked()
}

func (a *App) ListTaskFiles(id string) ([]TaskFile, error) {
	a.mu.Lock()
	task := a.tasks[id]
	if task == nil {
		a.mu.Unlock()
		return nil, errors.New("任务不存在")
	}
	task.Files = scanTaskFiles(task)
	files := append([]TaskFile(nil), task.Files...)
	_ = a.saveStateLocked()
	a.mu.Unlock()
	return files, nil
}

func (a *App) ExportTaskLog(id string) (TaskFile, error) {
	a.mu.Lock()
	task := a.tasks[id]
	if task == nil {
		a.mu.Unlock()
		return TaskFile{}, errors.New("任务不存在")
	}
	snapshot := taskSnapshot(task)
	a.mu.Unlock()

	saveDir := strings.TrimSpace(snapshot.Request.SaveDir)
	if saveDir == "" {
		return TaskFile{}, errors.New("任务输出目录为空")
	}
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return TaskFile{}, err
	}

	now := time.Now()
	fileName := fmt.Sprintf("m3u8dl-go_%s_%s.log", safeLogFileComponent(snapshot.Title), now.Format("20060102_150405"))
	path := uniqueDesktopOutputPath(filepath.Join(saveDir, fileName))
	content := buildTaskLogExport(snapshot, now)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return TaskFile{}, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return TaskFile{}, err
	}
	exported := taskFileFromInfo(path, info)
	a.appendTaskLog(id, "已导出日志: "+filepath.Base(path))

	a.mu.Lock()
	task = a.tasks[id]
	if task != nil {
		task.Files = scanTaskFiles(task)
		files := append([]TaskFile(nil), task.Files...)
		_ = a.saveStateLocked()
		a.emitTaskLocked(task)
		a.mu.Unlock()
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "task:files", map[string]interface{}{"taskId": id, "files": files})
		}
		return exported, nil
	}
	a.mu.Unlock()
	return exported, nil
}

func (a *App) OpenTaskFolder(id string) error {
	a.mu.Lock()
	task := a.tasks[id]
	a.mu.Unlock()
	if task == nil {
		return errors.New("任务不存在")
	}
	return openPath(task.Request.SaveDir)
}

func (a *App) RevealPath(path string) error {
	return openPath(path)
}

func (a *App) DeleteTaskFile(taskID string, path string) error {
	a.mu.Lock()
	task := a.tasks[taskID]
	a.mu.Unlock()
	if task == nil {
		return errors.New("任务不存在")
	}
	if !pathUnderDir(path, task.Request.SaveDir) {
		return errors.New("只能删除任务输出目录内的文件")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("不能删除目录")
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	a.appendTaskLog(taskID, "已删除文件: "+filepath.Base(path))
	a.mu.Lock()
	task.Files = scanTaskFiles(task)
	files := append([]TaskFile(nil), task.Files...)
	_ = a.saveStateLocked()
	a.emitTaskLocked(task)
	a.mu.Unlock()
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "task:files", map[string]interface{}{"taskId": taskID, "files": files})
	}
	return nil
}

func (a *App) StartDownload(req DownloadRequest) (Task, error) {
	task, err := a.CreateTask(req)
	if err != nil {
		return task, err
	}
	return task, a.StartTask(task.ID)
}

func (a *App) StartDownloads(req DownloadRequest) ([]Task, error) {
	tasks, err := a.CreateTasks(req)
	if err != nil {
		return tasks, err
	}
	for _, task := range tasks {
		if err := a.StartTask(task.ID); err != nil {
			return tasks, err
		}
	}
	return tasks, nil
}

func expandDownloadRequests(req DownloadRequest) ([]DownloadRequest, error) {
	lines := splitDownloadLines(req.URL)
	if len(lines) == 0 {
		return nil, errors.New("请先填写 m3u8 地址")
	}
	separator := strings.TrimSpace(req.LinkNameSeparator)
	requests := make([]DownloadRequest, 0, len(lines))
	for index, line := range lines {
		item := req
		item.URL = line
		// long: 批量创建时没有逐行资源名的任务不能复用同一个全局保存名，否则多个下载会写向同一文件名。
		if len(lines) > 1 {
			item.SaveName = ""
		}
		if separator != "" {
			saveName, url, ok, err := splitNamedDownloadLine(line, separator)
			if err != nil {
				return nil, fmt.Errorf("第 %d 行地址格式无效: %w", index+1, err)
			}
			if ok {
				item.URL = url
				item.SaveName = saveName
			}
		}
		requests = append(requests, item)
	}
	return requests, nil
}

func splitDownloadLines(raw string) []string {
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		if value := strings.TrimSpace(line); value != "" {
			lines = append(lines, value)
		}
	}
	return lines
}

func splitNamedDownloadLine(line string, separator string) (string, string, bool, error) {
	index := strings.Index(line, separator)
	if index < 0 {
		return "", "", false, nil
	}
	saveName := strings.TrimSpace(line[:index])
	url := strings.TrimSpace(line[index+len(separator):])
	if saveName == "" || url == "" {
		return "", "", true, errors.New("名称和地址不能为空")
	}
	return saveName, url, true, nil
}

func (a *App) applyRequestDefaults(req DownloadRequest) DownloadRequest {
	a.mu.Lock()
	settings := a.settings
	a.mu.Unlock()

	if strings.TrimSpace(req.SaveDir) == "" {
		req.SaveDir = settings.DefaultSaveDir
	}
	if strings.TrimSpace(req.SaveDir) == "" {
		req.SaveDir = defaultSaveDir()
	}
	if strings.TrimSpace(req.FFmpegPath) == "" {
		req.FFmpegPath = settings.FFmpegPath
	}
	if req.ThreadCount <= 0 {
		req.ThreadCount = settings.ThreadCount
	}
	if req.RetryCount < 0 {
		req.RetryCount = settings.RetryCount
	}
	if strings.TrimSpace(req.MaxSpeed) == "" {
		req.MaxSpeed = settings.MaxSpeed
	}
	if strings.TrimSpace(req.CustomProxy) == "" {
		req.CustomProxy = settings.CustomProxy
	}
	return req
}

func (a *App) runTask(id string) {
	a.mu.Lock()
	task := a.tasks[id]
	if task == nil {
		a.mu.Unlock()
		return
	}
	req := task.Request
	a.mu.Unlock()

	cliPath, err := findCLIPath()
	if err != nil {
		a.finishTask(id, StatusFailed, -1, err.Error())
		return
	}
	args := buildCLIArgs(req)
	cmdText := taskCommandLine(cliPath, args)
	a.appendTaskLog(id, "$ "+cmdText)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, cliPath, args...)
	cmd.Dir = req.SaveDir
	cmd.Env = desktopEnvironment()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		a.finishTask(id, StatusFailed, -1, err.Error())
		return
	}
	cmd.Stderr = cmd.Stdout

	a.mu.Lock()
	a.running[id] = &runningTask{cancel: cancel, cmd: cmd}
	task = a.tasks[id]
	if task != nil {
		task.Command = cliPath
		task.Args = append([]string(nil), args...)
		task.CommandLine = cmdText
		a.emitTaskLocked(task)
		_ = a.saveStateLocked()
	}
	a.mu.Unlock()

	if err := cmd.Start(); err != nil {
		cancel()
		a.removeRunning(id)
		a.finishTask(id, StatusFailed, -1, err.Error())
		return
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		a.appendTaskLog(id, scanner.Text())
	}
	scanErr := scanner.Err()
	waitErr := cmd.Wait()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	wasCanceled := ctx.Err() != nil
	a.removeRunning(id)
	cancel()

	if scanErr != nil {
		a.finishTask(id, StatusFailed, exitCode, scanErr.Error())
		return
	}
	if waitErr != nil {
		if wasCanceled {
			a.finishTask(id, StatusStopped, exitCode, "下载任务已停止")
			return
		}
		a.finishTask(id, StatusFailed, exitCode, fmt.Sprintf("下载核心退出，退出码：%d", exitCode))
		return
	}
	a.finishTask(id, StatusCompleted, exitCode, "下载任务已完成")
}

func (a *App) removeRunning(id string) {
	a.mu.Lock()
	delete(a.running, id)
	a.mu.Unlock()
}

func (a *App) finishTask(id string, status string, exitCode int, message string) {
	a.mu.Lock()
	task := a.tasks[id]
	if task == nil {
		a.mu.Unlock()
		return
	}
	now := time.Now()
	task.Status = status
	task.Queued = false
	task.ExitCode = exitCode
	if status == StatusFailed && strings.TrimSpace(task.FailureMessage) != "" {
		message = task.FailureMessage
	}
	task.LastMessage = message
	task.FinishedAt = &now
	if status == StatusCompleted {
		task.Progress = 1
		task.ProgressText = "完成"
		task.Files = mergeTaskFiles(task.Files, scanTaskFiles(task))
	}
	updateTaskTiming(task, now)
	logLine := timestampTaskLogLine(message, now)
	task.Logs = appendLimited(task.Logs, logLine)
	_ = a.saveStateLocked()
	snapshot := taskSnapshot(task)
	a.mu.Unlock()

	a.emitLog(id, logLine)
	a.emitTask(snapshot)
	a.scheduleQueuedTasks()
}

func (a *App) appendTaskLog(id string, line string) {
	now := time.Now()
	progressEvent, hasProgressEvent := parseProgressJSON(line)
	summaryEvent, hasSummaryEvent := parseSummaryJSON(line)
	errorEvent, hasErrorEvent := parseErrorJSON(line)
	if hasProgressEvent {
		line = progressEventLogLine(line, progressEvent)
	} else if hasSummaryEvent {
		line = summaryEventLogLine(summaryEvent)
	} else if hasErrorEvent {
		line = errorEventLogLine(errorEvent)
	}
	logLine := timestampTaskLogLine(line, now)
	a.mu.Lock()
	task := a.tasks[id]
	if task == nil {
		a.mu.Unlock()
		return
	}
	task.Logs = appendLimited(task.Logs, logLine)
	task.LastMessage = line
	if hasProgressEvent {
		applyProgressEvent(task, progressEvent)
	} else if hasSummaryEvent {
		task.Files = mergeTaskFiles(task.Files, taskFilesFromSummary(task, summaryEvent))
	} else if hasErrorEvent {
		task.FailureMessage = line
	} else if current, total, speed, ok := parseProgress(line); ok {
		applyProgressValues(task, current, total, speed)
	}
	updateTaskTiming(task, now)
	_ = a.saveStateLocked()
	snapshot := taskSnapshot(task)
	a.mu.Unlock()

	a.emitLog(id, logLine)
	a.emitTask(snapshot)
}

func parseProgressJSON(line string) (cliProgressEvent, bool) {
	payload := stripTaskLogTimestamp(line)
	if !strings.HasPrefix(payload, "{") {
		return cliProgressEvent{}, false
	}
	var event cliProgressEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return cliProgressEvent{}, false
	}
	if event.Type != "progress" || event.Total <= 0 || event.Current < 0 {
		return cliProgressEvent{}, false
	}
	if event.Percent <= 0 {
		event.Percent = float64(event.Current) / float64(event.Total)
	}
	return event, true
}

func parseSummaryJSON(line string) (cliSummaryEvent, bool) {
	payload := stripTaskLogTimestamp(line)
	if !strings.HasPrefix(payload, "{") {
		return cliSummaryEvent{}, false
	}
	var event cliSummaryEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return cliSummaryEvent{}, false
	}
	if event.Type != "summary" {
		return cliSummaryEvent{}, false
	}
	return event, true
}

func parseErrorJSON(line string) (cliErrorEvent, bool) {
	payload := stripTaskLogTimestamp(line)
	if !strings.HasPrefix(payload, "{") {
		return cliErrorEvent{}, false
	}
	var event cliErrorEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return cliErrorEvent{}, false
	}
	if event.Type != "error" || strings.TrimSpace(event.Message) == "" {
		return cliErrorEvent{}, false
	}
	return event, true
}

func stripTaskLogTimestamp(line string) string {
	line = strings.TrimSpace(line)
	if match := taskLogTimestampRE.FindString(line); match != "" {
		line = strings.TrimSpace(line[len(match):])
	}
	return line
}

func progressEventLogLine(original string, event cliProgressEvent) string {
	message := progressEventMessage(event)
	if match := taskLogTimestampRE.FindString(strings.TrimSpace(original)); match != "" {
		return match + " " + message
	}
	return message
}

func progressEventMessage(event cliProgressEvent) string {
	prefix := strings.TrimSpace(event.Stream)
	if prefix == "" {
		prefix = "下载"
	}
	if strings.TrimSpace(event.Speed) != "" {
		return fmt.Sprintf("%s 下载进度 %d/%d，速度 %s", prefix, event.Current, event.Total, strings.TrimSpace(event.Speed))
	}
	return fmt.Sprintf("%s 下载进度 %d/%d", prefix, event.Current, event.Total)
}

func summaryEventLogLine(event cliSummaryEvent) string {
	names := make([]string, 0, len(event.Outputs))
	for _, output := range event.Outputs {
		if name := filepath.Base(strings.TrimSpace(output.Path)); name != "." && name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "下载完成，未收到输出文件摘要"
	}
	if len(names) > 3 {
		names = append(names[:3], fmt.Sprintf("等 %d 个文件", len(event.Outputs)))
	}
	return "下载完成，输出文件: " + strings.Join(names, "、")
}

func errorEventLogLine(event cliErrorEvent) string {
	message := strings.TrimSpace(event.Message)
	if message == "" {
		return "下载失败"
	}
	return "下载失败: " + message
}

func applyProgressEvent(task *Task, event cliProgressEvent) {
	if task == nil {
		return
	}
	progress := event.Percent
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	task.Progress = progress
	applyProgressText(task, event.Current, event.Total, strings.TrimSpace(event.Speed))
}

func applyProgressValues(task *Task, current int, total int, speed string) {
	if task == nil || total <= 0 {
		return
	}
	task.Progress = float64(current) / float64(total)
	if task.Progress > 1 {
		task.Progress = 1
	}
	applyProgressText(task, current, total, speed)
}

func applyProgressText(task *Task, current int, total int, speed string) {
	task.SpeedText = strings.TrimSpace(speed)
	if task.SpeedText != "" {
		task.ProgressText = fmt.Sprintf("%d/%d · %s", current, total, task.SpeedText)
	} else {
		task.ProgressText = fmt.Sprintf("%d/%d", current, total)
	}
}

func taskFilesFromSummary(task *Task, event cliSummaryEvent) []TaskFile {
	if task == nil {
		return nil
	}
	root := strings.TrimSpace(task.Request.SaveDir)
	files := make([]TaskFile, 0, len(event.Outputs))
	for _, output := range event.Outputs {
		path := strings.TrimSpace(output.Path)
		if path == "" {
			continue
		}
		if root != "" && !pathUnderDir(path, root) {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !isManagedOutputExt(ext) {
			continue
		}
		files = append(files, taskFileFromInfo(path, info))
	}
	return files
}

func timestampTaskLogLine(line string, now time.Time) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return now.Format("[2006-01-02 15:04:05]")
	}
	if taskLogTimestampRE.MatchString(line) {
		return line
	}
	return fmt.Sprintf("[%s] %s", now.Format("2006-01-02 15:04:05"), line)
}

func parseProgress(line string) (int, int, string, bool) {
	matches := progressRegex.FindStringSubmatch(line)
	if len(matches) < 3 {
		return 0, 0, "", false
	}
	var current, total int
	_, err1 := fmt.Sscanf(matches[1], "%d", &current)
	_, err2 := fmt.Sscanf(matches[2], "%d", &total)
	if err1 != nil || err2 != nil || total <= 0 {
		return 0, 0, "", false
	}
	speed := ""
	if len(matches) >= 4 {
		speed = strings.TrimSpace(matches[3])
	}
	return current, total, speed, true
}

func updateTaskTiming(task *Task, now time.Time) {
	if task == nil || task.StartedAt == nil {
		if task != nil {
			task.ElapsedText = ""
			task.RemainingText = ""
		}
		return
	}
	end := now
	if task.FinishedAt != nil {
		end = *task.FinishedAt
	}
	elapsed := end.Sub(*task.StartedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	task.ElapsedText = formatTaskDuration(elapsed)
	task.RemainingText = ""
	if task.Status == StatusCompleted || (task.Status == StatusRunning && task.Progress >= 1) {
		task.RemainingText = "0秒"
		return
	}
	if task.Status != StatusRunning || task.Progress <= 0 || task.Progress >= 1 {
		return
	}
	remaining := time.Duration(float64(elapsed) * (1 - task.Progress) / task.Progress)
	if remaining < 0 {
		remaining = 0
	}
	task.RemainingText = formatTaskDuration(remaining)
}

func formatTaskDuration(duration time.Duration) string {
	if duration < 0 {
		duration = 0
	}
	seconds := int(duration.Round(time.Second) / time.Second)
	if seconds < 60 {
		return fmt.Sprintf("%d秒", seconds)
	}
	minutes := seconds / 60
	seconds = seconds % 60
	if minutes < 60 {
		if seconds == 0 {
			return fmt.Sprintf("%d分", minutes)
		}
		return fmt.Sprintf("%d分%02d秒", minutes, seconds)
	}
	hours := minutes / 60
	minutes = minutes % 60
	if hours < 24 {
		if minutes == 0 {
			return fmt.Sprintf("%d小时", hours)
		}
		return fmt.Sprintf("%d小时%02d分", hours, minutes)
	}
	days := hours / 24
	hours = hours % 24
	if hours == 0 {
		return fmt.Sprintf("%d天", days)
	}
	return fmt.Sprintf("%d天%d小时", days, hours)
}

func (a *App) emitLog(taskID string, line string) {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "task:log", taskLogEvent{TaskID: taskID, Line: line})
	}
}

func (a *App) emitTask(task Task) {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "task:update", task)
	}
}

func (a *App) emitTaskLocked(task *Task) {
	if a.ctx != nil && task != nil {
		wailsRuntime.EventsEmit(a.ctx, "task:update", taskSnapshot(task))
	}
}

func buildCLIArgs(req DownloadRequest) []string {
	args := []string{
		strings.TrimSpace(req.URL),
		"--save-dir", strings.TrimSpace(req.SaveDir),
		"--auto-select", fmt.Sprintf("%t", req.AutoSelect),
		"--ui-language", "zh-CN",
		"--disable-update-check", "true",
		"--progress-json", "true",
		"--thread-count", fmt.Sprintf("%d", req.ThreadCount),
		"--download-retry-count", fmt.Sprintf("%d", req.RetryCount),
		"--use-system-proxy", fmt.Sprintf("%t", req.UseSystemProxy),
	}
	if value := strings.TrimSpace(req.SaveName); value != "" {
		args = append(args, "--save-name", value)
	}
	if value := strings.TrimSpace(req.CustomRange); value != "" {
		args = append(args, "--custom-range", value)
	}
	if value := strings.TrimSpace(req.MaxSpeed); value != "" {
		args = append(args, "-R", value)
	}
	if req.ConcurrentDownload {
		args = append(args, "-mt", "true")
	}
	if req.BinaryMerge {
		args = append(args, "--binary-merge", "true")
	}
	if req.MuxMP4 {
		args = append(args, "-M", "format=mp4:muxer=ffmpeg")
	}
	if value := strings.TrimSpace(req.CustomProxy); value != "" {
		args = append(args, "--custom-proxy", value)
	}
	if value := strings.TrimSpace(req.FFmpegPath); value != "" {
		args = append(args, "--ffmpeg-binary-path", value)
	}
	for _, header := range req.Headers {
		if value := strings.TrimSpace(header); value != "" {
			args = append(args, "-H", value)
		}
	}
	return args
}

func defaultCLICommandName() string {
	if runtime.GOOS == "windows" {
		return "m3u8dl-go-cli.exe"
	}
	return "m3u8dl-go-cli"
}

func taskCommandLine(command string, args []string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		command = defaultCLICommandName()
	}
	// long: 任务详情里的复制命令必须和实际启动命令共用同一套 shell 转义，避免 URL、路径或 Header 中的空格导致用户复现失败。
	parts := append([]string{command}, args...)
	return shellPreview(parts)
}

func taskTitle(req DownloadRequest) string {
	if value := strings.TrimSpace(req.SaveName); value != "" {
		return value
	}
	value := strings.TrimSpace(req.URL)
	if value == "" {
		return "未命名任务"
	}
	base := filepath.Base(strings.TrimRight(value, "/"))
	if base == "." || base == "/" || base == "" {
		return value
	}
	return base
}

func newTaskID() string {
	return fmt.Sprintf("task-%d", time.Now().UnixNano())
}

func appendLimited(lines []string, line string) []string {
	lines = append(lines, line)
	if len(lines) > maxTaskLogs {
		return append([]string(nil), lines[len(lines)-maxTaskLogs:]...)
	}
	return lines
}

func removeID(ids []string, target string) []string {
	next := ids[:0]
	for _, id := range ids {
		if id != target {
			next = append(next, id)
		}
	}
	return next
}

func taskSnapshot(task *Task) Task {
	snapshot := *task
	snapshot.Args = append([]string(nil), task.Args...)
	snapshot.Logs = append([]string(nil), task.Logs...)
	snapshot.Files = append([]TaskFile(nil), task.Files...)
	snapshot.Request.Headers = append([]string(nil), task.Request.Headers...)
	return snapshot
}

func (a *App) taskSnapshotsLocked() []Task {
	tasks := make([]Task, 0, len(a.tasks))
	for _, id := range a.order {
		if task := a.tasks[id]; task != nil {
			tasks = append(tasks, taskSnapshot(task))
		}
	}
	return tasks
}

func scanTaskFiles(task *Task) []TaskFile {
	root := strings.TrimSpace(task.Request.SaveDir)
	if root == "" {
		return nil
	}
	var since time.Time
	if task.StartedAt != nil {
		since = task.StartedAt.Add(-2 * time.Second)
	} else {
		since = task.CreatedAt.Add(-2 * time.Second)
	}

	var files []TaskFile
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.ModTime().Before(since) {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !isManagedOutputExt(ext) {
			return nil
		}
		files = append(files, taskFileFromInfo(path, info))
		return nil
	})
	sort.Slice(files, func(i, j int) bool {
		return files[i].Modified.After(files[j].Modified)
	})
	return files
}

func mergeTaskFiles(groups ...[]TaskFile) []TaskFile {
	byPath := map[string]TaskFile{}
	for _, files := range groups {
		for _, file := range files {
			path := strings.TrimSpace(file.Path)
			if path == "" {
				continue
			}
			file.Path = path
			byPath[path] = file
		}
	}
	merged := make([]TaskFile, 0, len(byPath))
	for _, file := range byPath {
		merged = append(merged, file)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Modified.After(merged[j].Modified)
	})
	return merged
}

func taskFileFromInfo(path string, info fs.FileInfo) TaskFile {
	return TaskFile{
		Path:      path,
		Name:      filepath.Base(path),
		Size:      info.Size(),
		Modified:  info.ModTime(),
		Extension: strings.ToLower(filepath.Ext(path)),
	}
}

func isManagedOutputExt(ext string) bool {
	switch ext {
	case ".mp4", ".mkv", ".ts", ".m4a", ".aac", ".ac3", ".eac3", ".vtt", ".srt", ".ttml", ".ass", ".m3u8", ".json", ".log":
		return true
	default:
		return false
	}
}

func buildTaskLogExport(task Task, exportedAt time.Time) string {
	var builder strings.Builder
	writeExportLine := func(label string, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			value = "-"
		}
		builder.WriteString(label)
		builder.WriteString(": ")
		builder.WriteString(value)
		builder.WriteByte('\n')
	}

	builder.WriteString("# m3u8dl-go 任务日志\n\n")
	writeExportLine("导出时间", exportedAt.Format("2006-01-02 15:04:05"))
	writeExportLine("任务ID", task.ID)
	writeExportLine("任务名称", task.Title)
	writeExportLine("状态", taskStatusForExport(task.Status))
	writeExportLine("进度", task.ProgressText)
	writeExportLine("耗时", task.ElapsedText)
	writeExportLine("预计剩余", task.RemainingText)
	writeExportLine("输出目录", task.Request.SaveDir)
	writeExportLine("保存名", task.Request.SaveName)
	writeExportLine("地址", task.Request.URL)
	writeExportLine("创建时间", formatExportTime(task.CreatedAt))
	writeExportLine("开始时间", formatExportTimePtr(task.StartedAt))
	writeExportLine("结束时间", formatExportTimePtr(task.FinishedAt))
	if task.CommandLine != "" {
		writeExportLine("命令", redactTaskLogLine(task.CommandLine, task.Request))
	}
	if len(task.Request.Headers) > 0 {
		writeExportLine("请求头", "已脱敏，不在日志文件中保留原文")
	}
	if proxy := strings.TrimSpace(task.Request.CustomProxy); proxy != "" {
		writeExportLine("代理", redactProxyForDisplay(proxy))
	}
	builder.WriteString("\n## 日志\n")
	for _, line := range task.Logs {
		builder.WriteString(redactTaskLogLine(line, task.Request))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func taskStatusForExport(status string) string {
	switch status {
	case StatusPending:
		return "等待"
	case StatusRunning:
		return "下载中"
	case StatusCompleted:
		return "完成"
	case StatusFailed:
		return "失败"
	case StatusStopped:
		return "已停止"
	default:
		return status
	}
}

func formatExportTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02 15:04:05")
}

func formatExportTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatExportTime(*value)
}

func redactTaskLogLine(text string, req DownloadRequest) string {
	return redactPreflightError(text, req)
}

func redactTaskLogLines(lines []string, req DownloadRequest) []string {
	if len(lines) == 0 {
		return nil
	}
	redacted := make([]string, len(lines))
	for i, line := range lines {
		redacted[i] = redactTaskLogLine(line, req)
	}
	return redacted
}

func safeLogFileComponent(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "task"
	}
	value = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`/\?%*:|"<>`, r) {
			return '_'
		}
		return r
	}, value)
	value = strings.Trim(value, " ._")
	if value == "" {
		value = "task"
	}
	runes := []rune(value)
	if len(runes) > 64 {
		value = string(runes[:64])
	}
	return value
}

func uniqueDesktopOutputPath(path string) string {
	if _, err := os.Stat(path); err != nil {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for index := 2; ; index++ {
		next := fmt.Sprintf("%s_%d%s", base, index, ext)
		if _, err := os.Stat(next); err != nil {
			return next
		}
	}
}

func pathUnderDir(path string, root string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel))
}

func (r *PreflightReport) addCheck(name string, label string, status string, message string, detail string) {
	r.Checks = append(r.Checks, PreflightCheck{
		Name:    name,
		Label:   label,
		Status:  status,
		Message: message,
		Detail:  detail,
	})
}

func (r *PreflightReport) finalize() {
	status := "ready"
	for _, check := range r.Checks {
		switch check.Status {
		case "error":
			r.Status = "error"
			r.Message = "预检查发现问题"
			return
		case "warning":
			status = "warning"
		}
	}
	r.Status = status
	if status == "warning" {
		r.Message = "预检查有提醒"
	} else {
		r.Message = "预检查通过"
	}
}

func ensureWritableDirectory(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return errors.New("输出目录为空")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("读取输出目录失败: %w", err)
	}
	if !info.IsDir() {
		return errors.New("输出路径不是目录")
	}
	// long: 桌面端预检查要复用真实下载的输出目录权限边界，用临时探针文件确认写入能力，避免任务启动后才发现沙盒/权限问题。
	probe, err := os.CreateTemp(dir, ".m3u8dl-go-write-test-*")
	if err != nil {
		return fmt.Errorf("写入输出目录失败: %w", err)
	}
	name := probe.Name()
	closeErr := probe.Close()
	removeErr := os.Remove(name)
	if closeErr != nil {
		return fmt.Errorf("关闭写入探针失败: %w", closeErr)
	}
	if removeErr != nil {
		return fmt.Errorf("清理写入探针失败: %w", removeErr)
	}
	return nil
}

func openPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("路径为空")
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func (a *App) loadState() error {
	b, err := os.ReadFile(a.statePath)
	if err != nil {
		return nil
	}
	var state persistedState
	if err := json.Unmarshal(b, &state); err != nil {
		return err
	}
	a.settings = normalizeSettings(state.Settings)
	for _, task := range state.Tasks {
		if task == nil || task.ID == "" {
			continue
		}
		sanitizeLoadedTask(task)
		a.tasks[task.ID] = task
	}
	a.order = append([]string(nil), state.Order...)
	for id := range a.tasks {
		found := false
		for _, orderedID := range a.order {
			if orderedID == id {
				found = true
				break
			}
		}
		if !found {
			a.order = append(a.order, id)
		}
	}
	return nil
}

func sanitizeLoadedTask(task *Task) {
	if task.Status == StatusRunning {
		task.Status = StatusStopped
		task.LastMessage = "上次退出时任务仍在运行，已标记为停止"
	}
	task.Request.CustomProxy = clearRedactedProxyForRuntime(task.Request.CustomProxy)
}

func (a *App) saveStateLocked() error {
	if err := os.MkdirAll(filepath.Dir(a.statePath), 0755); err != nil {
		return err
	}
	tasks := make([]*Task, 0, len(a.order))
	for _, id := range a.order {
		if task := a.tasks[id]; task != nil {
			snapshot := taskSnapshot(task)
			// long: 请求头经常包含 Cookie，历史任务只持久化任务参数和状态，不把敏感 Header 或带 Header 的复现命令/日志写入磁盘。
			snapshot.LastMessage = redactTaskLogLine(snapshot.LastMessage, task.Request)
			snapshot.Logs = redactTaskLogLines(snapshot.Logs, task.Request)
			snapshot.Request.Headers = nil
			snapshot.Request.CustomProxy = redactProxyForDisplay(snapshot.Request.CustomProxy)
			snapshot.Args = nil
			snapshot.CommandLine = ""
			tasks = append(tasks, &snapshot)
		}
	}
	state := persistedState{
		Settings: settingsForPersistence(a.settings),
		Order:    append([]string(nil), a.order...),
		Tasks:    tasks,
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.statePath, b, 0644)
}

func findCLIPath() (string, error) {
	names := []string{"m3u8dl-go-cli", "m3u8dl-go"}
	if runtime.GOOS == "windows" {
		names = []string{"m3u8dl-go-cli.exe", "m3u8dl-go.exe"}
	}
	if value := strings.TrimSpace(os.Getenv("M3U8DL_GO_CLI")); value != "" {
		if isExecutable(value) {
			return value, nil
		}
		return "", fmt.Errorf("M3U8DL_GO_CLI 指向的文件不可执行: %s", value)
	}

	var candidates []string
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		for _, name := range names {
			candidates = append(candidates,
				filepath.Join(exeDir, name),
				filepath.Join(filepath.Dir(exeDir), "Resources", name),
			)
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		for _, name := range names {
			candidates = append(candidates,
				filepath.Join(cwd, name),
				filepath.Join(cwd, "..", name),
				filepath.Join(cwd, "..", "build", name),
			)
		}
	}
	for _, candidate := range candidates {
		if isExecutable(candidate) {
			return candidate, nil
		}
	}
	for _, name := range names {
		if found, err := exec.LookPath(name); err == nil {
			return found, nil
		}
	}
	return "", errors.New("找不到下载核心，请确认 m3u8dl-go-cli 与桌面应用在同一目录，或设置 M3U8DL_GO_CLI")
}

func readCLIVersion(cliPath string) (CoreInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cliPath, "--version-json")
	cmd.Env = desktopEnvironment()
	out, err := cmd.Output()
	if err == nil {
		var payload cliVersionInfo
		if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &payload); jsonErr == nil && payload.FullVersion != "" {
			return CoreInfo{
				Status:      "ready",
				Version:     strings.TrimSpace(payload.Version),
				FullVersion: strings.TrimSpace(payload.FullVersion),
			}, nil
		}
	}

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cliPath, "--version")
	cmd.Env = desktopEnvironment()
	out, err = cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return CoreInfo{}, errors.New("读取下载核心版本超时")
		}
		return CoreInfo{}, fmt.Errorf("读取下载核心版本失败: %w", err)
	}
	fullVersion := strings.TrimSpace(string(out))
	return CoreInfo{
		Status:      "ready",
		Version:     strings.TrimSpace(strings.TrimPrefix(fullVersion, "m3u8dl-go")),
		FullVersion: fullVersion,
	}, nil
}

func desktopToolCheckSpecs(ffmpegPath string) []toolCheckSpec {
	ffmpegPath = strings.TrimSpace(ffmpegPath)
	ffmpegCommands := []string{"ffmpeg"}
	if ffmpegPath != "" {
		ffmpegCommands = []string{ffmpegPath}
	}
	ffprobeCommands := []string{"ffprobe"}
	if candidate := ffprobeCommandFromFFmpegPath(ffmpegPath); candidate != "" {
		ffprobeCommands = append([]string{candidate}, ffprobeCommands...)
	}
	return []toolCheckSpec{
		{Name: "ffmpeg", Label: "FFmpeg", Commands: ffmpegCommands, Args: []string{"-version"}},
		{Name: "ffprobe", Label: "FFprobe", Commands: ffprobeCommands, Args: []string{"-version"}},
		{Name: "mkvmerge", Label: "mkvmerge", Commands: []string{"mkvmerge"}, Args: []string{"--version"}},
		{Name: "mp4decrypt", Label: "mp4decrypt", Commands: []string{"mp4decrypt"}, Args: []string{"--version"}},
		{Name: "shaka-packager", Label: "Shaka Packager", Commands: []string{"shaka-packager", "packager-linux-x64", "packager-osx-x64", "packager-win-x64"}, Args: []string{"--version"}},
	}
}

func ffprobeCommandFromFFmpegPath(ffmpegPath string) string {
	if ffmpegPath == "" {
		return ""
	}
	dir := filepath.Dir(ffmpegPath)
	base := filepath.Base(ffmpegPath)
	if strings.HasPrefix(base, "ffmpeg") && dir != "." {
		return filepath.Join(dir, strings.Replace(base, "ffmpeg", "ffprobe", 1))
	}
	return ""
}

func probeToolVersion(path string, fallbackName string, args []string) (ToolInfo, error) {
	command := strings.TrimSpace(path)
	if command == "" {
		command = fallbackName
	}
	return probeToolCandidates([]string{command}, args)
}

func probeToolCandidates(commands []string, args []string) (ToolInfo, error) {
	var lastErr error
	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		resolved, err := lookPathDesktop(command)
		if err != nil {
			lastErr = err
			continue
		}
		return runToolVersionProbe(resolved, args)
	}
	if lastErr != nil {
		return ToolInfo{}, lastErr
	}
	return ToolInfo{}, errors.New("工具名称为空")
}

func runToolVersionProbe(resolved string, args []string) (ToolInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, resolved, args...)
	cmd.Env = desktopEnvironment()
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return ToolInfo{Path: resolved}, errors.New("读取工具版本超时")
	}
	if err != nil {
		text := firstNonEmptyLine(string(out))
		if text != "" {
			return ToolInfo{Path: resolved}, fmt.Errorf("读取工具版本失败: %w: %s", err, text)
		}
		return ToolInfo{Path: resolved}, fmt.Errorf("读取工具版本失败: %w", err)
	}
	return ToolInfo{
		Status:  "ready",
		Path:    resolved,
		Version: firstNonEmptyLine(string(out)),
	}, nil
}

func validateCLIArguments(cliPath string, args []string) error {
	cliArgs := append([]string(nil), args...)
	cliArgs = append(cliArgs, "--print-effective-options")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cliPath, cliArgs...)
	cmd.Env = desktopEnvironment()
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return errors.New("读取参数校验结果超时")
	}
	if err != nil {
		text := strings.TrimSpace(string(out))
		if text == "" {
			return fmt.Errorf("下载核心参数校验失败: %w", err)
		}
		return fmt.Errorf("下载核心参数校验失败: %w: %s", err, firstNonEmptyLine(text))
	}
	return nil
}

func redactPreflightError(text string, req DownloadRequest) string {
	for _, header := range req.Headers {
		name, value, ok := strings.Cut(header, ":")
		if !ok || !sensitiveHeaderName(name) {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			text = strings.ReplaceAll(text, value, "<redacted>")
		}
		text = strings.ReplaceAll(text, header, strings.TrimSpace(name)+": <redacted>")
	}
	if proxy := strings.TrimSpace(req.CustomProxy); proxy != "" {
		text = strings.ReplaceAll(text, proxy, redactProxyForDisplay(proxy))
	}
	return text
}

func sensitiveHeaderName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return name == "cookie" ||
		name == "authorization" ||
		name == "proxy-authorization" ||
		name == "x-api-key" ||
		name == "x-auth-token"
}

func redactProxyForDisplay(raw string) string {
	if raw == "" {
		return ""
	}
	if before, after, ok := strings.Cut(raw, "@"); ok {
		if scheme, userPart, hasScheme := strings.Cut(before, "://"); hasScheme {
			if user, _, hasPassword := strings.Cut(userPart, ":"); hasPassword {
				return scheme + "://" + user + ":redacted@" + after
			}
		}
	}
	return raw
}

func settingsForPersistence(settings Settings) Settings {
	settings.CustomProxy = redactProxyForDisplay(settings.CustomProxy)
	return settings
}

func clearRedactedProxyForRuntime(raw string) string {
	if strings.Contains(raw, ":redacted@") {
		// long: 状态文件只保留代理密码的脱敏形态；重新启动后不能把 redacted 当成真实密码继续发请求。
		return ""
	}
	return raw
}

func lookPathDesktop(command string) (string, error) {
	if resolved, err := exec.LookPath(command); err == nil {
		return resolved, nil
	}
	if hasPathSeparator(command) {
		return "", fmt.Errorf("exec: %q: executable file not found", command)
	}
	for _, dir := range filepath.SplitList(desktopSearchPath()) {
		if dir == "" {
			continue
		}
		for _, candidate := range executableCandidates(filepath.Join(dir, command)) {
			if isExecutable(candidate) {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("exec: %q: executable file not found in PATH", command)
}

func executableCandidates(path string) []string {
	if runtime.GOOS != "windows" || filepath.Ext(path) != "" {
		return []string{path}
	}
	extensions := strings.Split(os.Getenv("PATHEXT"), ";")
	if len(extensions) == 0 || strings.Join(extensions, "") == "" {
		extensions = []string{".COM", ".EXE", ".BAT", ".CMD"}
	}
	candidates := []string{path}
	for _, ext := range extensions {
		ext = strings.TrimSpace(ext)
		if ext != "" {
			candidates = append(candidates, path+ext)
		}
	}
	return candidates
}

func hasPathSeparator(command string) bool {
	return strings.Contains(command, "/") || strings.Contains(command, `\`)
}

func firstNonEmptyLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if value := strings.TrimSpace(line); value != "" {
			return value
		}
	}
	return ""
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0111 != 0
}

func desktopEnvironment() []string {
	env := os.Environ()
	env = append(env, "PATH="+desktopSearchPath())
	// long: 桌面应用从 Finder 启动时通常没有用户 shell 环境，补齐中文 locale 和常见命令路径可以减少 ffmpeg 查找失败。
	if os.Getenv("LC_ALL") == "" {
		env = append(env, "LC_ALL=zh_CN.UTF-8")
	}
	if os.Getenv("LANG") == "" {
		env = append(env, "LANG=zh_CN.UTF-8")
	}
	return env
}

func desktopSearchPath() string {
	pathValue := os.Getenv("PATH")
	finderSafePath := "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
	if pathValue == "" {
		return finderSafePath
	}
	return finderSafePath + string(os.PathListSeparator) + pathValue
}

func shellPreview(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.ContainsAny(arg, " \t\n\"'") {
			quoted = append(quoted, "'"+strings.ReplaceAll(arg, "'", "'\\''")+"'")
			continue
		}
		quoted = append(quoted, arg)
	}
	return strings.Join(quoted, " ")
}
