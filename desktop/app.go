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

var progressRegex = regexp.MustCompile(`(?i)(?:下载进度|download progress)\s+(\d+)\s*/\s*(\d+)`)

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
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Status       string          `json:"status"`
	Progress     float64         `json:"progress"`
	ProgressText string          `json:"progressText"`
	LastMessage  string          `json:"lastMessage"`
	ExitCode     int             `json:"exitCode"`
	CreatedAt    time.Time       `json:"createdAt"`
	StartedAt    *time.Time      `json:"startedAt,omitempty"`
	FinishedAt   *time.Time      `json:"finishedAt,omitempty"`
	Request      DownloadRequest `json:"request"`
	Args         []string        `json:"args"`
	Command      string          `json:"command"`
	Logs         []string        `json:"logs"`
	Files        []TaskFile      `json:"files"`
}

type TaskFile struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Modified  time.Time `json:"modified"`
	Extension string    `json:"extension"`
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
	if strings.TrimSpace(a.settings.DefaultSaveDir) == "" {
		a.settings.DefaultSaveDir = defaultSaveDir()
	}
	return a.settings
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
	if settings.ThreadCount <= 0 {
		settings.ThreadCount = 8
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
	req = a.applyRequestDefaults(req)
	if strings.TrimSpace(req.URL) == "" {
		return Task{}, errors.New("请先填写 m3u8 地址")
	}
	if err := os.MkdirAll(req.SaveDir, 0755); err != nil {
		return Task{}, fmt.Errorf("创建输出目录失败: %w", err)
	}

	task := &Task{
		ID:        newTaskID(),
		Title:     taskTitle(req),
		Status:    StatusPending,
		CreatedAt: time.Now(),
		Request:   req,
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
	task.Status = StatusRunning
	task.Progress = 0
	task.ProgressText = ""
	task.LastMessage = "正在启动下载核心"
	task.ExitCode = 0
	task.Files = nil
	task.Logs = appendLimited(task.Logs, "正在启动下载核心")
	now := time.Now()
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
	task.Progress = 0
	task.ProgressText = ""
	task.LastMessage = "等待重新开始"
	task.ExitCode = 0
	task.FinishedAt = nil
	task.Logs = appendLimited(task.Logs, "任务已重新排队。")
	_ = a.saveStateLocked()
	a.emitTaskLocked(task)
	a.mu.Unlock()
	return a.StartTask(id)
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
	cmdText := cliPath + " " + shellPreview(args)
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
	a.removeRunning(id)
	cancel()

	if scanErr != nil {
		a.finishTask(id, StatusFailed, exitCode, scanErr.Error())
		return
	}
	if waitErr != nil {
		if ctx.Err() != nil {
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
	task.ExitCode = exitCode
	task.LastMessage = message
	task.FinishedAt = &now
	if status == StatusCompleted {
		task.Progress = 1
		task.ProgressText = "完成"
		task.Files = scanTaskFiles(task)
	}
	task.Logs = appendLimited(task.Logs, message)
	_ = a.saveStateLocked()
	snapshot := taskSnapshot(task)
	a.mu.Unlock()

	a.emitLog(id, message)
	a.emitTask(snapshot)
}

func (a *App) appendTaskLog(id string, line string) {
	a.mu.Lock()
	task := a.tasks[id]
	if task == nil {
		a.mu.Unlock()
		return
	}
	task.Logs = appendLimited(task.Logs, line)
	task.LastMessage = line
	if current, total, ok := parseProgress(line); ok {
		task.Progress = float64(current) / float64(total)
		task.ProgressText = fmt.Sprintf("%d/%d", current, total)
	}
	_ = a.saveStateLocked()
	snapshot := taskSnapshot(task)
	a.mu.Unlock()

	a.emitLog(id, line)
	a.emitTask(snapshot)
}

func parseProgress(line string) (int, int, bool) {
	matches := progressRegex.FindStringSubmatch(line)
	if len(matches) != 3 {
		return 0, 0, false
	}
	var current, total int
	_, err1 := fmt.Sscanf(matches[1], "%d", &current)
	_, err2 := fmt.Sscanf(matches[2], "%d", &total)
	if err1 != nil || err2 != nil || total <= 0 {
		return 0, 0, false
	}
	return current, total, true
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
		files = append(files, TaskFile{
			Path:      path,
			Name:      filepath.Base(path),
			Size:      info.Size(),
			Modified:  info.ModTime(),
			Extension: ext,
		})
		return nil
	})
	sort.Slice(files, func(i, j int) bool {
		return files[i].Modified.After(files[j].Modified)
	})
	return files
}

func isManagedOutputExt(ext string) bool {
	switch ext {
	case ".mp4", ".mkv", ".ts", ".m4a", ".aac", ".ac3", ".eac3", ".vtt", ".srt", ".ttml", ".ass", ".m3u8", ".json":
		return true
	default:
		return false
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
		if task.Status == StatusRunning {
			task.Status = StatusStopped
			task.LastMessage = "上次退出时任务仍在运行，已标记为停止"
		}
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

func (a *App) saveStateLocked() error {
	if err := os.MkdirAll(filepath.Dir(a.statePath), 0755); err != nil {
		return err
	}
	tasks := make([]*Task, 0, len(a.order))
	for _, id := range a.order {
		if task := a.tasks[id]; task != nil {
			snapshot := taskSnapshot(task)
			// long: 请求头经常包含 Cookie，历史任务只持久化任务参数和状态，不把敏感 Header 写入磁盘。
			snapshot.Request.Headers = nil
			tasks = append(tasks, &snapshot)
		}
	}
	state := persistedState{
		Settings: a.settings,
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
	pathValue := os.Getenv("PATH")
	finderSafePath := "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
	if pathValue == "" {
		env = append(env, "PATH="+finderSafePath)
	} else {
		env = append(env, "PATH="+finderSafePath+":"+pathValue)
	}
	// long: 桌面应用从 Finder 启动时通常没有用户 shell 环境，补齐中文 locale 和常见命令路径可以减少 ffmpeg 查找失败。
	if os.Getenv("LC_ALL") == "" {
		env = append(env, "LC_ALL=zh_CN.UTF-8")
	}
	if os.Getenv("LANG") == "" {
		env = append(env, "LANG=zh_CN.UTF-8")
	}
	return env
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
