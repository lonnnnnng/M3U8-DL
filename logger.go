package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type logCleanup func() error
type logLevel int

const (
	logLevelOff logLevel = iota
	logLevelError
	logLevelWarn
	logLevelInfo
	logLevelDebug
)

func setupLogging(opt Options, argv []string) (logCleanup, string, error) {
	if opt.NoLog {
		return func() error { return nil }, "", nil
	}
	path, err := resolveLogFilePath(opt.LogFilePath)
	if err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, "", err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, "", err
	}
	now := time.Now()
	header := fmt.Sprintf("LOG %s\nSave Path: %s\nTask Start: %s\nTask CommandLine: %s\n\n",
		now.Format("2006/01/02"),
		filepath.Dir(path),
		now.Format("2006/01/02 15:04:05"),
		strings.Join(argv, " "),
	)
	if _, err := file.WriteString(header); err != nil {
		_ = file.Close()
		return nil, "", err
	}

	oldStdout, oldStderr := os.Stdout, os.Stderr
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		_ = file.Close()
		return nil, "", err
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
		_ = file.Close()
		return nil, "", err
	}

	var wg sync.WaitGroup
	writeMu := &sync.Mutex{}
	stdoutLog := &filteredLogWriter{dst: file, mu: writeMu, level: parseLogLevel(opt.LogLevel)}
	stderrLog := &filteredLogWriter{dst: file, mu: writeMu, level: parseLogLevel(opt.LogLevel)}
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(io.MultiWriter(oldStdout, stdoutLog), stdoutReader)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(io.MultiWriter(oldStderr, stderrLog), stderrReader)
	}()
	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	return func() error {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		// long: 先关闭写端让复制协程自然读到 EOF，再关闭文件，避免最后一行日志丢在缓冲通道里。
		_ = stdoutWriter.Close()
		_ = stderrWriter.Close()
		wg.Wait()
		_ = stdoutLog.Flush()
		_ = stderrLog.Flush()
		_ = stdoutReader.Close()
		_ = stderrReader.Close()
		return file.Close()
	}, path, nil
}

func resolveLogFilePath(input string) (string, error) {
	if input != "" {
		return filepath.Abs(input)
	}
	exe, err := os.Executable()
	if err != nil || exe == "" {
		exe = "."
	}
	logDir := filepath.Join(filepath.Dir(exe), "Logs")
	base := time.Now().Format("2006-01-02_15-04-05-000") + ".log"
	path := filepath.Join(logDir, base)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path, nil
	}
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	for i := 1; ; i++ {
		candidate := filepath.Join(logDir, fmt.Sprintf("%s-%d%s", name, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		}
	}
}

type filteredLogWriter struct {
	dst   io.Writer
	mu    *sync.Mutex
	level logLevel
	buf   []byte
}

func (w *filteredLogWriter) Write(p []byte) (int, error) {
	for _, b := range p {
		w.buf = append(w.buf, b)
		if b == '\n' {
			if err := w.flushLine(); err != nil {
				return 0, err
			}
		}
	}
	return len(p), nil
}

func (w *filteredLogWriter) Flush() error {
	if len(w.buf) == 0 {
		return nil
	}
	return w.flushLine()
}

func (w *filteredLogWriter) flushLine() error {
	line := string(w.buf)
	w.buf = nil
	if !shouldWriteLogLine(line, w.level) {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err := w.dst.Write([]byte(line))
	return err
}

func parseLogLevel(input string) logLevel {
	switch strings.ToUpper(strings.TrimSpace(input)) {
	case "OFF":
		return logLevelOff
	case "ERROR":
		return logLevelError
	case "WARN":
		return logLevelWarn
	case "DEBUG":
		return logLevelDebug
	case "INFO", "":
		return logLevelInfo
	default:
		return logLevelInfo
	}
}

func shouldWriteLogLine(line string, current logLevel) bool {
	if current == logLevelOff {
		return false
	}
	return current >= detectLogLineLevel(line)
}

func detectLogLineLevel(line string) logLevel {
	upper := strings.ToUpper(line)
	switch {
	case strings.Contains(upper, "DEBUG"):
		return logLevelDebug
	case strings.Contains(upper, "ERROR"):
		return logLevelError
	case strings.Contains(upper, "WARN"):
		return logLevelWarn
	case strings.Contains(upper, "INFO"):
		return logLevelInfo
	default:
		// long: Go 版仍有直接 fmt.Println 的业务输出，按 INFO 处理可让 WARN/ERROR/OFF 等级像上游一样减少日志噪声。
		return logLevelInfo
	}
}
