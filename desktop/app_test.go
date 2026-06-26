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
echo "Vid 下载进度 1/2，速度 1.0 MB/s"
echo "Vid 下载进度 2/2，速度 2.0 MB/s"
printf "ok" > "$out/$name.mp4"
echo "输出: $out/$name.mp4"
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
