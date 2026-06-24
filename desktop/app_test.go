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
echo "Vid 下载进度 1/2"
echo "Vid 下载进度 2/2"
printf "ok" > "$out/$name.mp4"
echo "输出: $out/$name.mp4"
`
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return path
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
