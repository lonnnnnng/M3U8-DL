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

var (
	consoleOutputMu sync.Mutex
	rawOutputMu     sync.Mutex
	rawStdout       *os.File
	rawProgressLog  *rawProgressLogWriter
)

type rawProgressLogWriter struct {
	dst   io.Writer
	mu    *sync.Mutex
	level logLevel
}

func setupLogging(opt Options, argv []string) (logCleanup, string, error) {
	var file *os.File
	var path string
	var err error
	if !opt.NoLog {
		path, err = resolveLogFilePath(opt.LogFilePath)
		if err != nil {
			return nil, "", err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, "", err
		}
		file, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
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
	}

	oldStdout, oldStderr := os.Stdout, os.Stderr
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		if file != nil {
			_ = file.Close()
		}
		return nil, "", err
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
		if file != nil {
			_ = file.Close()
		}
		return nil, "", err
	}

	var wg sync.WaitGroup
	writeMu := &sync.Mutex{}
	stdoutConsole := &timestampConsoleWriter{dst: oldStdout, mu: &consoleOutputMu, atLineStart: true}
	stderrConsole := &timestampConsoleWriter{dst: oldStderr, mu: &consoleOutputMu, atLineStart: true}
	var stdoutLog, stderrLog *filteredLogWriter
	if file != nil {
		stdoutLog = &filteredLogWriter{dst: file, mu: writeMu, level: parseLogLevel(opt.LogLevel)}
		stderrLog = &filteredLogWriter{dst: file, mu: writeMu, level: parseLogLevel(opt.LogLevel)}
	}
	restoreRawOutput := setRawProgressOutput(oldStdout, stdoutLog)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(combinedLogWriter(stdoutConsole, stdoutLog), stdoutReader)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(combinedLogWriter(stderrConsole, stderrLog), stderrReader)
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
		restoreRawOutput()
		if stdoutLog != nil {
			_ = stdoutLog.Flush()
		}
		if stderrLog != nil {
			_ = stderrLog.Flush()
		}
		_ = stdoutReader.Close()
		_ = stderrReader.Close()
		if file != nil {
			return file.Close()
		}
		return nil
	}, path, nil
}

func setRawProgressOutput(stdout *os.File, log *filteredLogWriter) func() {
	var progressLog *rawProgressLogWriter
	if log != nil {
		progressLog = &rawProgressLogWriter{dst: log.dst, mu: log.mu, level: log.level}
	}
	rawOutputMu.Lock()
	previous := rawStdout
	previousLog := rawProgressLog
	rawStdout = stdout
	rawProgressLog = progressLog
	rawOutputMu.Unlock()
	return func() {
		rawOutputMu.Lock()
		rawStdout = previous
		rawProgressLog = previousLog
		rawOutputMu.Unlock()
	}
}

func writeRawEventLine(data []byte) bool {
	rawOutputMu.Lock()
	dst := rawStdout
	log := rawProgressLog
	rawOutputMu.Unlock()
	consoleOutputMu.Lock()
	defer consoleOutputMu.Unlock()
	wroteStdout := false
	if dst != nil {
		_, err := dst.Write(data)
		wroteStdout = err == nil
	}
	if log != nil && shouldWriteLogLine(string(data), log.level) {
		log.mu.Lock()
		_, _ = log.dst.Write(data)
		log.mu.Unlock()
	}
	return wroteStdout
}

func combinedLogWriter(console io.Writer, fileLog *filteredLogWriter) io.Writer {
	if fileLog == nil {
		return console
	}
	return io.MultiWriter(console, fileLog)
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

type timestampConsoleWriter struct {
	dst         io.Writer
	mu          *sync.Mutex
	atLineStart bool
}

func (w *timestampConsoleWriter) Write(p []byte) (int, error) {
	if w == nil || w.dst == nil {
		return len(p), nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, b := range p {
		if b == '\r' {
			if !w.atLineStart {
				if _, err := w.dst.Write([]byte("\n")); err != nil {
					return 0, err
				}
			}
			w.atLineStart = true
			continue
		}
		if b == '\n' {
			if _, err := w.dst.Write([]byte{b}); err != nil {
				return 0, err
			}
			w.atLineStart = true
			continue
		}
		if w.atLineStart {
			// long: 终端看到的是实时运行日志，统一在转发层补时间戳，业务代码仍然保持原本的 fmt.Print/Println 调用方式。
			if _, err := fmt.Fprintf(w.dst, "[%s] ", time.Now().Format("2006-01-02 15:04:05")); err != nil {
				return 0, err
			}
			w.atLineStart = false
		}
		if _, err := w.dst.Write([]byte{b}); err != nil {
			return 0, err
		}
	}
	return len(p), nil
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
	_, err := w.dst.Write(timestampLogPayload(line, time.Now()))
	return err
}

func timestampLogPayload(line string, now time.Time) []byte {
	text := strings.TrimRight(line, "\r\n")
	if text == "" {
		return []byte("\n")
	}
	parts := strings.Split(strings.ReplaceAll(text, "\r", "\n"), "\n")
	var b strings.Builder
	stamp := now.Format("2006-01-02 15:04:05")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// long: CLI 里很多业务输出仍走 fmt.Println，写日志时集中补时间戳，避免每个调用点重复拼接且漏掉桌面集成的输出。
		b.WriteString("[")
		b.WriteString(stamp)
		b.WriteString("] ")
		b.WriteString(part)
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func timestampConsoleMessage(line string, now time.Time) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return now.Format("[2006-01-02 15:04:05]")
	}
	// long: main 的最终错误输出发生在 stdout/stderr 恢复之后，单独补一次时间戳，保证 CLI 终端日志出口一致。
	return fmt.Sprintf("[%s] %s", now.Format("2006-01-02 15:04:05"), line)
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
