package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestTaskLifecycleWithFakeCLI(t *testing.T) {
	app := newTestApp(t)
	fakeCLI := writeFakeCLI(t)
	t.Setenv("M3U8DL_GO_CLI", fakeCLI)

	task, err := app.StartDownload(DownloadRequest{
		URL:            "https://example.com/index.m3u8",
		SaveDir:        filepath.Join(t.TempDir(), "out"),
		SaveName:       "sample",
		AutoSelect:     true,
		MuxMP4:         true,
		ThreadCount:    4,
		RetryCount:     2,
		UseSystemProxy: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	task = waitTaskStatus(t, app, task.ID, StatusCompleted, 5*time.Second)
	if task.Progress != 1 {
		t.Fatalf("completed task should have full progress, got %.2f", task.Progress)
	}
	if len(task.Files) == 0 || !strings.HasSuffix(task.Files[0].Name, ".mp4") {
		t.Fatalf("completed task should scan output mp4 files, got %#v", task.Files)
	}
	if !strings.Contains(strings.Join(task.Logs, "\n"), "下载进度 2/2") {
		t.Fatalf("task logs should capture downloader output: %#v", task.Logs)
	}
	if task.SpeedText != "2.0 MB/s" || task.ProgressText != "完成" {
		t.Fatalf("task should parse progress speed, progress=%q speed=%q", task.ProgressText, task.SpeedText)
	}
	if !taskLogTimestampRE.MatchString(task.Logs[0]) {
		t.Fatalf("task logs should include timestamps: %#v", task.Logs)
	}
	if task.CommandLine == "" || !strings.Contains(task.CommandLine, fakeCLI) || !strings.Contains(task.CommandLine, "--save-name sample") {
		t.Fatalf("completed task should expose reproducible command line, got %q", task.CommandLine)
	}
}

func TestCreateTaskCommandPreviewDoesNotPersistSensitiveHeaders(t *testing.T) {
	app := newTestApp(t)
	task, err := app.CreateTask(DownloadRequest{
		URL:            "https://example.com/index.m3u8",
		SaveDir:        filepath.Join(t.TempDir(), "out dir"),
		Headers:        []string{"Cookie: session=secret"},
		AutoSelect:     true,
		ThreadCount:    4,
		RetryCount:     2,
		UseSystemProxy: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.CommandLine == "" || !strings.Contains(task.CommandLine, "m3u8dl-go-cli") || !strings.Contains(task.CommandLine, "'Cookie: session=secret'") {
		t.Fatalf("task should expose command preview for current session, got %q", task.CommandLine)
	}
	state, err := os.ReadFile(app.statePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(state), "session=secret") || strings.Contains(string(state), "commandLine") {
		t.Fatalf("persisted state should not contain sensitive command data:\n%s", state)
	}
}

func TestSaveSettingsDoesNotPersistProxyPassword(t *testing.T) {
	app := newTestApp(t)
	saved, err := app.SaveSettings(Settings{
		DefaultSaveDir: filepath.Join(t.TempDir(), "downloads"),
		ThreadCount:    8,
		RetryCount:     3,
		MaxActiveTasks: 2,
		CustomProxy:    "http://user:secret@127.0.0.1:8888",
		UseSystemProxy: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.CustomProxy != "http://user:secret@127.0.0.1:8888" {
		t.Fatalf("runtime settings should keep proxy for current session, got %q", saved.CustomProxy)
	}

	stateBytes, err := os.ReadFile(app.statePath)
	if err != nil {
		t.Fatal(err)
	}
	stateText := string(stateBytes)
	if strings.Contains(stateText, "user:secret") {
		t.Fatalf("persisted settings should not contain proxy password:\n%s", stateText)
	}
	if !strings.Contains(stateText, "http://user:redacted@127.0.0.1:8888") {
		t.Fatalf("persisted settings should retain redacted proxy for auditability:\n%s", stateText)
	}

	reloaded := &App{
		tasks:     map[string]*Task{},
		running:   map[string]*runningTask{},
		settings:  defaultSettings(),
		statePath: app.statePath,
	}
	if err := reloaded.loadState(); err != nil {
		t.Fatal(err)
	}
	if reloaded.settings.CustomProxy != "" {
		t.Fatalf("redacted proxy should be cleared before runtime reuse, got %q", reloaded.settings.CustomProxy)
	}
}

func TestLoadedTaskClearsRedactedProxyBeforeRetry(t *testing.T) {
	app := newTestApp(t)
	task, err := app.CreateTask(DownloadRequest{
		URL:            "https://example.com/index.m3u8",
		SaveDir:        filepath.Join(t.TempDir(), "out"),
		CustomProxy:    "http://user:secret@127.0.0.1:8888",
		AutoSelect:     true,
		ThreadCount:    4,
		RetryCount:     2,
		UseSystemProxy: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Request.CustomProxy != "http://user:secret@127.0.0.1:8888" {
		t.Fatalf("runtime task should keep proxy for current session, got %q", task.Request.CustomProxy)
	}

	stateBytes, err := os.ReadFile(app.statePath)
	if err != nil {
		t.Fatal(err)
	}
	stateText := string(stateBytes)
	if strings.Contains(stateText, "user:secret") {
		t.Fatalf("persisted task should not contain proxy password:\n%s", stateText)
	}
	if !strings.Contains(stateText, "http://user:redacted@127.0.0.1:8888") {
		t.Fatalf("persisted task should keep redacted proxy for display:\n%s", stateText)
	}

	reloaded := &App{
		tasks:     map[string]*Task{},
		running:   map[string]*runningTask{},
		settings:  defaultSettings(),
		statePath: app.statePath,
	}
	if err := reloaded.loadState(); err != nil {
		t.Fatal(err)
	}
	loadedTask := reloaded.tasks[task.ID]
	if loadedTask == nil {
		t.Fatalf("loaded task missing: %#v", reloaded.tasks)
	}
	if loadedTask.Request.CustomProxy != "" {
		t.Fatalf("redacted task proxy should be cleared before retry, got %q", loadedTask.Request.CustomProxy)
	}
}

func TestExportTaskLogWritesRedactedLogFile(t *testing.T) {
	app := newTestApp(t)
	out := filepath.Join(t.TempDir(), "out")
	task, err := app.CreateTask(DownloadRequest{
		URL:            "https://example.com/index.m3u8",
		SaveDir:        out,
		SaveName:       "Movie:01",
		Headers:        []string{"Cookie: session=secret", "Authorization: Bearer secret-token"},
		CustomProxy:    "http://user:secret@127.0.0.1:8888",
		AutoSelect:     true,
		ThreadCount:    4,
		RetryCount:     2,
		UseSystemProxy: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	app.appendTaskLog(task.ID, "$ m3u8dl-go-cli -H 'Cookie: session=secret' -H 'Authorization: Bearer secret-token' --custom-proxy http://user:secret@127.0.0.1:8888 https://example.com/index.m3u8")
	app.appendTaskLog(task.ID, "下载进度 1/1，速度 1 MB/s")

	file, err := app.ExportTaskLog(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if file.Extension != ".log" || !strings.HasPrefix(file.Name, "m3u8dl-go_Movie_01_") {
		t.Fatalf("exported log file metadata mismatch: %#v", file)
	}
	contentBytes, err := os.ReadFile(file.Path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(contentBytes)
	for _, secret := range []string{"session=secret", "secret-token", "user:secret"} {
		if strings.Contains(content, secret) {
			t.Fatalf("exported log should redact %q:\n%s", secret, content)
		}
	}
	for _, want := range []string{"Cookie: <redacted>", "Authorization: <redacted>", "http://user:redacted@127.0.0.1:8888", "下载进度 1/1"} {
		if !strings.Contains(content, want) {
			t.Fatalf("exported log missing %q:\n%s", want, content)
		}
	}

	files, err := app.ListTaskFiles(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, item := range files {
		if item.Path == file.Path && item.Extension == ".log" {
			found = true
		}
	}
	if !found {
		t.Fatalf("exported log should appear in task files, got %#v", files)
	}

	state, err := os.ReadFile(app.statePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"session=secret", "secret-token", "user:secret", "commandLine"} {
		if strings.Contains(string(state), secret) {
			t.Fatalf("persisted state should not contain sensitive exported log data %q:\n%s", secret, state)
		}
	}
}

func TestPreviewCommandsExpandsBatchWithoutCreatingTasks(t *testing.T) {
	app := newTestApp(t)
	commands, err := app.PreviewCommands(DownloadRequest{
		URL:               "第一集|https://example.com/one.m3u8\n第二集|https://example.com/two.m3u8",
		SaveDir:           filepath.Join(t.TempDir(), "out dir"),
		LinkNameSeparator: "|",
		AutoSelect:        true,
		ThreadCount:       4,
		RetryCount:        2,
		UseSystemProxy:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(commands, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two preview commands, got %q", commands)
	}
	if !strings.Contains(lines[0], "--save-name") || !strings.Contains(lines[0], "第一集") || !strings.Contains(lines[1], "第二集") {
		t.Fatalf("preview commands should preserve batch names, got %q", commands)
	}
	if len(app.ListTasks()) != 0 {
		t.Fatalf("preview should not create tasks, got %#v", app.ListTasks())
	}
}

func TestPreflightDownloadChecksCoreOutputAndFFmpeg(t *testing.T) {
	app := newTestApp(t)
	fakeCLI := writeFakeCLI(t)
	fakeFFmpeg := writeFakeFFmpeg(t)
	t.Setenv("M3U8DL_GO_CLI", fakeCLI)

	report := app.PreflightDownload(DownloadRequest{
		URL:            "https://example.com/index.m3u8",
		SaveDir:        filepath.Join(t.TempDir(), "out"),
		FFmpegPath:     fakeFFmpeg,
		AutoSelect:     true,
		MuxMP4:         true,
		ThreadCount:    4,
		RetryCount:     2,
		UseSystemProxy: true,
	})
	if report.Status != "ready" || report.TaskCount != 1 || len(report.CommandLines) != 1 {
		t.Fatalf("preflight should pass, got %#v", report)
	}
	for _, name := range []string{"input", "core", "output", "ffmpeg"} {
		if check := preflightCheckByName(report, name); check == nil || check.Status != "ready" {
			t.Fatalf("preflight check %s should be ready, got %#v in %#v", name, check, report.Checks)
		}
	}
	if !strings.Contains(report.CommandLines[0], "--ffmpeg-binary-path") || !strings.Contains(report.CommandLines[0], fakeFFmpeg) {
		t.Fatalf("preflight should expose preview command with ffmpeg path, got %#v", report.CommandLines)
	}
}

func TestPreflightDownloadReportsUnwritableOutputPath(t *testing.T) {
	app := newTestApp(t)
	fakeCLI := writeFakeCLI(t)
	t.Setenv("M3U8DL_GO_CLI", fakeCLI)
	filePath := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	report := app.PreflightDownload(DownloadRequest{
		URL:            "https://example.com/index.m3u8",
		SaveDir:        filePath,
		AutoSelect:     true,
		MuxMP4:         false,
		ThreadCount:    4,
		RetryCount:     2,
		UseSystemProxy: true,
	})
	if report.Status != "error" {
		t.Fatalf("preflight should fail for file output path, got %#v", report)
	}
	check := preflightCheckByName(report, "output")
	if check == nil || check.Status != "error" || !strings.Contains(check.Message, "输出目录") {
		t.Fatalf("preflight should report output error, got %#v", report.Checks)
	}
}

func TestPreflightDownloadReportsCoreArgumentValidationError(t *testing.T) {
	app := newTestApp(t)
	fakeCLI := writeEffectiveOptionsRejectingFakeCLI(t)
	t.Setenv("M3U8DL_GO_CLI", fakeCLI)

	report := app.PreflightDownload(DownloadRequest{
		URL:            "https://example.com/index.m3u8",
		SaveDir:        filepath.Join(t.TempDir(), "out"),
		Headers:        []string{"Cookie: session=secret"},
		AutoSelect:     true,
		MuxMP4:         false,
		ThreadCount:    4,
		RetryCount:     2,
		UseSystemProxy: true,
		CustomProxy:    "http://user:secret@127.0.0.1:8888",
	})
	if report.Status != "error" {
		t.Fatalf("preflight should fail for core argument validation, got %#v", report)
	}
	check := preflightCheckByName(report, "args")
	if check == nil || check.Status != "error" || !strings.Contains(check.Detail, "bad option") {
		t.Fatalf("preflight should expose sanitized argument validation error, got %#v", report.Checks)
	}
	if strings.Contains(check.Detail, "session=secret") || strings.Contains(check.Detail, "user:secret") {
		t.Fatalf("preflight argument error should be redacted, got %#v", check)
	}
}

func TestParseProgressWithSpeed(t *testing.T) {
	current, total, speed, ok := parseProgress("[2026-06-25 12:00:00] Vid 下载进度 3/7，速度 1.5 MB/s")
	if !ok || current != 3 || total != 7 || speed != "1.5 MB/s" {
		t.Fatalf("progress with speed parsed wrong: current=%d total=%d speed=%q ok=%t", current, total, speed, ok)
	}

	current, total, speed, ok = parseProgress("Vid download progress 4/9, speed 900 KB/s")
	if !ok || current != 4 || total != 9 || speed != "900 KB/s" {
		t.Fatalf("english progress with speed parsed wrong: current=%d total=%d speed=%q ok=%t", current, total, speed, ok)
	}
}

func TestParseProgressJSONWithTimestamp(t *testing.T) {
	event, ok := parseProgressJSON(`[2026-06-25 12:00:00] {"type":"progress","stream":"Vid","current":3,"total":7,"speed":"1.5 MB/s","bytes":1048576,"percent":0.42}`)
	if !ok || event.Stream != "Vid" || event.Current != 3 || event.Total != 7 || event.Speed != "1.5 MB/s" || event.Bytes != 1048576 || event.Percent != 0.42 {
		t.Fatalf("progress json parsed wrong: %#v ok=%t", event, ok)
	}
	line := progressEventLogLine(`[2026-06-25 12:00:00] {"type":"progress","stream":"Vid","current":3,"total":7,"speed":"1.5 MB/s"}`, event)
	if line != "[2026-06-25 12:00:00] Vid 下载进度 3/7，速度 1.5 MB/s" {
		t.Fatalf("progress json log line mismatch: %q", line)
	}
}

func TestAppendTaskLogHandlesProgressJSON(t *testing.T) {
	app := newTestApp(t)
	task, err := app.CreateTask(DownloadRequest{
		URL:            "https://example.com/index.m3u8",
		SaveDir:        filepath.Join(t.TempDir(), "out"),
		AutoSelect:     true,
		ThreadCount:    4,
		RetryCount:     2,
		UseSystemProxy: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	app.appendTaskLog(task.ID, `[2026-06-25 12:00:00] {"type":"progress","stream":"Vid","current":2,"total":5,"speed":"2.0 MB/s","bytes":2097152,"percent":0.4}`)
	updated := app.tasks[task.ID]
	if updated.Progress != 0.4 || updated.ProgressText != "2/5 · 2.0 MB/s" || updated.SpeedText != "2.0 MB/s" {
		t.Fatalf("task should parse progress json, got progress=%.2f text=%q speed=%q", updated.Progress, updated.ProgressText, updated.SpeedText)
	}
	if !strings.Contains(strings.Join(updated.Logs, "\n"), "Vid 下载进度 2/5，速度 2.0 MB/s") {
		t.Fatalf("task logs should convert progress json to readable text: %#v", updated.Logs)
	}
}

func TestAppendTaskLogHandlesSummaryJSON(t *testing.T) {
	app := newTestApp(t)
	out := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	mediaPath := filepath.Join(out, "movie.mp4")
	if err := os.WriteFile(mediaPath, []byte("media"), 0644); err != nil {
		t.Fatal(err)
	}
	task, err := app.CreateTask(DownloadRequest{
		URL:            "https://example.com/index.m3u8",
		SaveDir:        out,
		AutoSelect:     true,
		ThreadCount:    4,
		RetryCount:     2,
		UseSystemProxy: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	app.appendTaskLog(task.ID, `{"type":"summary","timestamp":"2026-06-25 12:00:00","status":"completed","outputs":[{"path":"`+mediaPath+`","size":5}]}`)
	updated := app.tasks[task.ID]
	if len(updated.Files) != 1 || updated.Files[0].Path != mediaPath || updated.Files[0].Size != 5 {
		t.Fatalf("task should use summary files, got %#v", updated.Files)
	}
	if !strings.Contains(strings.Join(updated.Logs, "\n"), "下载完成，输出文件: movie.mp4") {
		t.Fatalf("task logs should convert summary json to readable text: %#v", updated.Logs)
	}
}

func TestTaskFailureUsesErrorJSONMessage(t *testing.T) {
	app := newTestApp(t)
	fakeCLI := writeFailingFakeCLI(t)
	t.Setenv("M3U8DL_GO_CLI", fakeCLI)

	task, err := app.StartDownload(DownloadRequest{
		URL:            "https://example.com/index.m3u8",
		SaveDir:        filepath.Join(t.TempDir(), "out"),
		AutoSelect:     true,
		ThreadCount:    4,
		RetryCount:     2,
		UseSystemProxy: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	task = waitTaskStatus(t, app, task.ID, StatusFailed, 5*time.Second)
	if task.ExitCode != 2 {
		t.Fatalf("failed task exit code mismatch: %#v", task)
	}
	if !strings.Contains(task.LastMessage, "下载失败: playlist parse failed") || task.FailureMessage != task.LastMessage {
		t.Fatalf("failed task should keep structured error message, got last=%q failure=%q logs=%#v", task.LastMessage, task.FailureMessage, task.Logs)
	}
	if !strings.Contains(strings.Join(task.Logs, "\n"), "下载失败: playlist parse failed") {
		t.Fatalf("task logs should include structured error message: %#v", task.Logs)
	}
}

func TestBuildCLIArgsEnablesProgressJSON(t *testing.T) {
	args := buildCLIArgs(DownloadRequest{
		URL:            "https://example.com/index.m3u8",
		SaveDir:        "/tmp/out",
		AutoSelect:     true,
		ThreadCount:    4,
		RetryCount:     2,
		UseSystemProxy: true,
	})
	joined := " " + strings.Join(args, " ") + " "
	if !strings.Contains(joined, " --progress-json true ") {
		t.Fatalf("desktop CLI args should enable progress json: %#v", args)
	}
}

func TestTaskTimingEstimatesRemaining(t *testing.T) {
	started := time.Date(2026, 6, 27, 4, 0, 0, 0, time.Local)
	task := &Task{
		Status:    StatusRunning,
		Progress:  0.25,
		StartedAt: &started,
	}
	updateTaskTiming(task, started.Add(30*time.Second))
	if task.ElapsedText != "30秒" || task.RemainingText != "1分30秒" {
		t.Fatalf("running task timing mismatch elapsed=%q remaining=%q", task.ElapsedText, task.RemainingText)
	}

	finished := started.Add(2*time.Minute + 5*time.Second)
	task.Status = StatusCompleted
	task.Progress = 1
	task.FinishedAt = &finished
	updateTaskTiming(task, finished.Add(time.Minute))
	if task.ElapsedText != "2分05秒" || task.RemainingText != "0秒" {
		t.Fatalf("completed task timing mismatch elapsed=%q remaining=%q", task.ElapsedText, task.RemainingText)
	}
}

func TestFormatTaskDuration(t *testing.T) {
	cases := map[time.Duration]string{
		0:                             "0秒",
		59 * time.Second:              "59秒",
		60 * time.Second:              "1分",
		90 * time.Second:              "1分30秒",
		2*time.Hour + 5*time.Minute:   "2小时05分",
		25*time.Hour + 10*time.Second: "1天1小时",
	}
	for input, want := range cases {
		if got := formatTaskDuration(input); got != want {
			t.Fatalf("formatTaskDuration(%s)=%q want %q", input, got, want)
		}
	}
}

func TestGetCoreInfoUsesVersionJSON(t *testing.T) {
	app := newTestApp(t)
	fakeCLI := writeFakeCLI(t)
	t.Setenv("M3U8DL_GO_CLI", fakeCLI)

	info := app.GetCoreInfo()
	if info.Status != "ready" || info.CLIPath != fakeCLI || info.Version != "9.9.9" || info.FullVersion != "m3u8dl-go 9.9.9" || info.Error != "" {
		t.Fatalf("core info should read fake CLI version json, got %#v", info)
	}
}

func TestGetCoreInfoFallsBackToPlainVersion(t *testing.T) {
	app := newTestApp(t)
	fakeCLI := writeVersionOnlyFakeCLI(t)
	t.Setenv("M3U8DL_GO_CLI", fakeCLI)

	info := app.GetCoreInfo()
	if info.Status != "ready" || info.CLIPath != fakeCLI || info.Version != "8.8.8" || info.FullVersion != "m3u8dl-go 8.8.8" || info.Error != "" {
		t.Fatalf("core info should fall back to plain --version, got %#v", info)
	}
}

func TestCheckFFmpegReadsVersion(t *testing.T) {
	app := newTestApp(t)
	fakeFFmpeg := writeFakeFFmpeg(t)

	info := app.CheckFFmpeg(fakeFFmpeg)
	if info.Status != "ready" || info.Path != fakeFFmpeg || info.Version != "ffmpeg version 9.9.9" || info.Error != "" {
		t.Fatalf("ffmpeg check mismatch: %#v", info)
	}
}

func TestCheckToolsDetectsFFprobeAndShakaAlias(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake shell tool test is only used on Unix-like CI runners")
	}
	app := newTestApp(t)
	dir := t.TempDir()
	fakeFFmpeg := writeFakeTool(t, dir, "ffmpeg", "-version", "ffmpeg version 9.9.9")
	writeFakeTool(t, dir, "ffprobe", "-version", "ffprobe version 9.9.9")
	writeFakeTool(t, dir, "mkvmerge", "--version", "mkvmerge v99")
	writeFakeTool(t, dir, "mp4decrypt", "--version", "mp4decrypt v99")
	writeFakeTool(t, dir, "packager-osx-x64", "--version", "packager version 99")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	tools := app.CheckTools(fakeFFmpeg)
	byName := map[string]ToolInfo{}
	for _, tool := range tools {
		byName[tool.Name] = tool
	}
	for _, name := range []string{"ffmpeg", "ffprobe", "mkvmerge", "mp4decrypt", "shaka-packager"} {
		if byName[name].Status != "ready" {
			t.Fatalf("tool %s should be ready, got %#v in %#v", name, byName[name], tools)
		}
	}
	if byName["ffprobe"].Path != filepath.Join(dir, "ffprobe") {
		t.Fatalf("ffprobe should be resolved next to custom ffmpeg, got %#v", byName["ffprobe"])
	}
	if byName["shaka-packager"].Path != filepath.Join(dir, "packager-osx-x64") {
		t.Fatalf("shaka alias should be detected, got %#v", byName["shaka-packager"])
	}
}

func TestExpandDownloadRequestsSupportsBatchAndNamedLines(t *testing.T) {
	requests, err := expandDownloadRequests(DownloadRequest{
		URL:               "第一集=>https://example.com/one.m3u8\nhttps://example.com/two.m3u8\n第三集=>https://example.com/three.m3u8",
		SaveName:          "global-name",
		LinkNameSeparator: "=>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 3 {
		t.Fatalf("expected 3 expanded requests, got %d", len(requests))
	}
	if requests[0].SaveName != "第一集" || requests[0].URL != "https://example.com/one.m3u8" {
		t.Fatalf("first named line was not parsed correctly: %#v", requests[0])
	}
	if requests[1].SaveName != "" || requests[1].URL != "https://example.com/two.m3u8" {
		t.Fatalf("unnamed batch line should not reuse global save name: %#v", requests[1])
	}
	if requests[2].SaveName != "第三集" || requests[2].URL != "https://example.com/three.m3u8" {
		t.Fatalf("third named line was not parsed correctly: %#v", requests[2])
	}

	single, err := expandDownloadRequests(DownloadRequest{
		URL:               "电影|https://example.com/movie.m3u8",
		SaveName:          "global-name",
		LinkNameSeparator: "|",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(single) != 1 || single[0].SaveName != "电影" || single[0].URL != "https://example.com/movie.m3u8" {
		t.Fatalf("single named line should override global save name: %#v", single)
	}

	if _, err := expandDownloadRequests(DownloadRequest{
		URL:               "缺少地址|",
		LinkNameSeparator: "|",
	}); err == nil {
		t.Fatal("invalid named line should be rejected")
	}
}

func TestStartDownloadsCreatesAndStartsBatch(t *testing.T) {
	app := newTestApp(t)
	fakeCLI := writeFakeCLI(t)
	t.Setenv("M3U8DL_GO_CLI", fakeCLI)

	out := filepath.Join(t.TempDir(), "out")
	tasks, err := app.StartDownloads(DownloadRequest{
		URL:               "第一集|https://example.com/one.m3u8\n第二集|https://example.com/two.m3u8",
		SaveDir:           out,
		LinkNameSeparator: "|",
		AutoSelect:        true,
		MuxMP4:            true,
		ThreadCount:       4,
		RetryCount:        2,
		UseSystemProxy:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 created tasks, got %d", len(tasks))
	}
	if tasks[0].Title != "第一集" || tasks[1].Title != "第二集" {
		t.Fatalf("batch tasks should keep line order and names: %#v", tasks)
	}

	for _, task := range tasks {
		waitTaskStatus(t, app, task.ID, StatusCompleted, 5*time.Second)
	}
	for _, name := range []string{"第一集.mp4", "第二集.mp4"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("expected batch output %s: %v", name, err)
		}
	}

	list := app.ListTasks()
	if len(list) != 2 || list[0].Title != "第一集" || list[1].Title != "第二集" {
		t.Fatalf("task list should preserve pasted line order, got %#v", list)
	}
}

func TestBulkTaskActions(t *testing.T) {
	app := newTestApp(t)
	fakeCLI := writeFakeCLI(t)
	t.Setenv("M3U8DL_GO_CLI", fakeCLI)

	out := filepath.Join(t.TempDir(), "out")
	tasks, err := app.CreateTasks(DownloadRequest{
		URL:               "第一集|https://example.com/one.m3u8\n第二集|https://example.com/two.m3u8",
		SaveDir:           out,
		LinkNameSeparator: "|",
		AutoSelect:        true,
		MuxMP4:            true,
		ThreadCount:       4,
		RetryCount:        2,
		UseSystemProxy:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if count, err := app.StartPendingTasks(); err != nil || count != 2 {
		t.Fatalf("start pending mismatch count=%d err=%v", count, err)
	}
	for _, task := range tasks {
		waitTaskStatus(t, app, task.ID, StatusCompleted, 5*time.Second)
	}

	app.mu.Lock()
	app.tasks[tasks[0].ID].Status = StatusFailed
	app.tasks[tasks[0].ID].LastMessage = "人为标记失败"
	app.mu.Unlock()
	if count, err := app.RetryFailedTasks(); err != nil || count != 1 {
		t.Fatalf("retry failed mismatch count=%d err=%v", count, err)
	}
	waitTaskStatus(t, app, tasks[0].ID, StatusCompleted, 5*time.Second)
}

func TestStopRunningTasks(t *testing.T) {
	app := newTestApp(t)
	fakeCLI := writeSlowFakeCLI(t)
	t.Setenv("M3U8DL_GO_CLI", fakeCLI)

	task, err := app.StartDownload(DownloadRequest{
		URL:            "https://example.com/slow.m3u8",
		SaveDir:        filepath.Join(t.TempDir(), "out"),
		SaveName:       "slow",
		AutoSelect:     true,
		ThreadCount:    4,
		RetryCount:     2,
		UseSystemProxy: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitUntilTaskRunning(t, app, task.ID)
	count, err := app.StopRunningTasks()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one running task to stop, got %d", count)
	}
	stopped := waitTaskStatus(t, app, task.ID, StatusStopped, 5*time.Second)
	if !strings.Contains(strings.Join(stopped.Logs, "\n"), "已请求停止当前任务") {
		t.Fatalf("stop request should be logged, logs=%#v", stopped.Logs)
	}
}

func TestMaxActiveTasksQueuesAndSchedulesNextTask(t *testing.T) {
	app := newTestApp(t)
	app.settings.MaxActiveTasks = 1
	fakeCLI := writeSlowFakeCLI(t)
	t.Setenv("M3U8DL_GO_CLI", fakeCLI)

	tasks, err := app.StartDownloads(DownloadRequest{
		URL:               "第一集|https://example.com/one.m3u8\n第二集|https://example.com/two.m3u8",
		SaveDir:           filepath.Join(t.TempDir(), "out"),
		LinkNameSeparator: "|",
		AutoSelect:        true,
		ThreadCount:       4,
		RetryCount:        2,
		UseSystemProxy:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected two tasks, got %d", len(tasks))
	}
	waitUntilTaskRunning(t, app, tasks[0].ID)
	queued := waitTaskQueued(t, app, tasks[1].ID)
	if queued.LastMessage != "已加入队列，等待空闲任务槽" {
		t.Fatalf("queued task message mismatch: %#v", queued)
	}

	if err := app.StopTask(tasks[0].ID); err != nil {
		t.Fatal(err)
	}
	waitTaskStatus(t, app, tasks[0].ID, StatusStopped, 5*time.Second)
	waitUntilTaskRunning(t, app, tasks[1].ID)
	if err := app.StopTask(tasks[1].ID); err != nil {
		t.Fatal(err)
	}
	waitTaskStatus(t, app, tasks[1].ID, StatusStopped, 5*time.Second)
}

func TestDesktopRealSampleDownload(t *testing.T) {
	if os.Getenv("M3U8DL_GO_REAL_SAMPLE") != "1" {
		t.Skip("set M3U8DL_GO_REAL_SAMPLE=1 and M3U8DL_GO_CLI to run the real sample")
	}
	if os.Getenv("M3U8DL_GO_CLI") == "" {
		t.Fatal("M3U8DL_GO_CLI is required for the real sample test")
	}

	app := newTestApp(t)
	task, err := app.StartDownload(DownloadRequest{
		URL:            "https://hd.ijycnd.com/play/dL9Zywje/index.m3u8",
		SaveDir:        filepath.Join(t.TempDir(), "out"),
		SaveName:       "ijycnd-desktop-range",
		CustomRange:    "0-2",
		AutoSelect:     true,
		MuxMP4:         true,
		ThreadCount:    8,
		RetryCount:     3,
		UseSystemProxy: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	task = waitTaskStatus(t, app, task.ID, StatusCompleted, 90*time.Second)
	var foundMP4 bool
	for _, file := range task.Files {
		if strings.HasSuffix(strings.ToLower(file.Name), ".mp4") && file.Size > 0 {
			foundMP4 = true
		}
	}
	if !foundMP4 {
		t.Fatalf("real sample should produce an mp4 file, files=%#v logs=%s", task.Files, strings.Join(task.Logs, "\n"))
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	tmp := t.TempDir()
	settings := defaultSettings()
	settings.DefaultSaveDir = filepath.Join(tmp, "downloads")
	return &App{
		tasks:     map[string]*Task{},
		running:   map[string]*runningTask{},
		settings:  settings,
		statePath: filepath.Join(tmp, "state.json"),
	}
}

func writeFakeCLI(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake shell CLI test is only used on Unix-like CI runners")
	}
	path := filepath.Join(t.TempDir(), "m3u8dl-go-cli")
	script := `#!/bin/sh
set -eu
if [ "${1:-}" = "--version-json" ]; then
  echo '{"name":"m3u8dl-go","version":"9.9.9","fullVersion":"m3u8dl-go 9.9.9"}'
  exit 0
fi
if [ "${1:-}" = "--version" ]; then
  echo "m3u8dl-go 9.9.9"
  exit 0
fi
for arg in "$@"; do
  if [ "$arg" = "--print-effective-options" ]; then
    echo '{"version":{"name":"m3u8dl-go","version":"9.9.9","fullVersion":"m3u8dl-go 9.9.9"}}'
    exit 0
  fi
done
out="."
name="sample"
while [ "$#" -gt 0 ]; do
  case "$1" in
    --save-dir) out="$2"; shift 2 ;;
    --save-name) name="$2"; shift 2 ;;
    *) shift ;;
  esac
done
mkdir -p "$out"
echo "开始下载...Vid"
echo '{"type":"progress","stream":"Vid","current":1,"total":2,"speed":"1.0 MB/s","bytes":1048576,"percent":0.5}'
echo '{"type":"progress","stream":"Vid","current":2,"total":2,"speed":"2.0 MB/s","bytes":2097152,"percent":1}'
printf "ok" > "$out/$name.mp4"
echo "输出: $out/$name.mp4"
`
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeSlowFakeCLI(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake shell CLI test is only used on Unix-like CI runners")
	}
	path := filepath.Join(t.TempDir(), "m3u8dl-go-cli")
	script := `#!/bin/sh
set -eu
if [ "${1:-}" = "--version-json" ]; then
  echo '{"name":"m3u8dl-go","version":"9.9.9","fullVersion":"m3u8dl-go 9.9.9"}'
  exit 0
fi
if [ "${1:-}" = "--version" ]; then
  echo "m3u8dl-go 9.9.9"
  exit 0
fi
for arg in "$@"; do
  if [ "$arg" = "--print-effective-options" ]; then
    echo '{"version":{"name":"m3u8dl-go","version":"9.9.9","fullVersion":"m3u8dl-go 9.9.9"}}'
    exit 0
  fi
done
echo "开始下载...Slow"
echo "Slow 下载进度 1/10，速度 1.0 MB/s"
sleep 30
`
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeEffectiveOptionsRejectingFakeCLI(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake shell CLI test is only used on Unix-like CI runners")
	}
	path := filepath.Join(t.TempDir(), "m3u8dl-go-cli")
	script := `#!/bin/sh
set -eu
if [ "${1:-}" = "--version-json" ]; then
  echo '{"name":"m3u8dl-go","version":"9.9.9","fullVersion":"m3u8dl-go 9.9.9"}'
  exit 0
fi
for arg in "$@"; do
  if [ "$arg" = "--print-effective-options" ]; then
    echo '错误: bad option Cookie: session=secret http://user:secret@127.0.0.1:8888' >&2
    exit 2
  fi
done
exit 0
`
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFailingFakeCLI(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake shell CLI test is only used on Unix-like CI runners")
	}
	path := filepath.Join(t.TempDir(), "m3u8dl-go-cli")
	script := `#!/bin/sh
set -eu
if [ "${1:-}" = "--version-json" ]; then
  echo '{"name":"m3u8dl-go","version":"9.9.9","fullVersion":"m3u8dl-go 9.9.9"}'
  exit 0
fi
if [ "${1:-}" = "--version" ]; then
  echo "m3u8dl-go 9.9.9"
  exit 0
fi
for arg in "$@"; do
  if [ "$arg" = "--print-effective-options" ]; then
    echo '{"version":{"name":"m3u8dl-go","version":"9.9.9","fullVersion":"m3u8dl-go 9.9.9"}}'
    exit 0
  fi
done
echo '{"type":"error","timestamp":"2026-06-25 12:00:00","status":"failed","message":"playlist parse failed"}'
exit 2
`
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeVersionOnlyFakeCLI(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake shell CLI test is only used on Unix-like CI runners")
	}
	path := filepath.Join(t.TempDir(), "m3u8dl-go-cli")
	script := `#!/bin/sh
set -eu
if [ "${1:-}" = "--version" ]; then
  echo "m3u8dl-go 8.8.8"
  exit 0
fi
exit 2
`
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFakeFFmpeg(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake shell tool test is only used on Unix-like CI runners")
	}
	path := filepath.Join(t.TempDir(), "ffmpeg")
	script := `#!/bin/sh
set -eu
if [ "${1:-}" = "-version" ]; then
  echo "ffmpeg version 9.9.9"
  exit 0
fi
exit 2
`
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFakeTool(t *testing.T, dir string, name string, versionArg string, output string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	script := `#!/bin/sh
set -eu
if [ "${1:-}" = "` + versionArg + `" ]; then
  echo "` + output + `"
  exit 0
fi
exit 2
`
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func preflightCheckByName(report PreflightReport, name string) *PreflightCheck {
	for i := range report.Checks {
		if report.Checks[i].Name == name {
			return &report.Checks[i]
		}
	}
	return nil
}

func waitTaskStatus(t *testing.T, app *App, id string, want string, timeout time.Duration) Task {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, task := range app.ListTasks() {
			if task.ID != id {
				continue
			}
			if task.Status == want {
				return task
			}
			if task.Status == StatusFailed || task.Status == StatusStopped {
				t.Fatalf("task ended as %s, want %s, logs=%s", task.Status, want, strings.Join(task.Logs, "\n"))
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("task did not reach %s before timeout", want)
	return Task{}
}

func waitUntilTaskRunning(t *testing.T, app *App, id string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		app.mu.Lock()
		task := app.tasks[id]
		_, hasRunningProcess := app.running[id]
		ready := task != nil && task.Status == StatusRunning && hasRunningProcess
		app.mu.Unlock()
		if ready {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("task did not enter cancellable running status")
}

func waitTaskQueued(t *testing.T, app *App, id string) Task {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, task := range app.ListTasks() {
			if task.ID == id && task.Status == StatusPending && task.Queued {
				return task
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("task did not enter queued status")
	return Task{}
}
