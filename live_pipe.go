package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	envLivePipeOptions     = "RE_LIVE_PIPE_OPTIONS"
	envLivePipeTmpDir      = "RE_LIVE_PIPE_TMP_DIR"
	livePipeConnectTimeout = 15 * time.Second
)

type livePipeEnv struct {
	Options string
	TmpDir  string
}

type livePipeSession struct {
	Pipes      []*os.File
	PipeNames  []string
	PipePaths  []string
	OutputPath string
	Cmd        *exec.Cmd
}

func runLivePipeMuxOutputs(outs []outputFile, opt Options) ([]outputFile, error) {
	pipeInputs, subtitleOutputs := splitLivePipeMuxOutputs(outs)
	if len(pipeInputs) == 0 {
		return outs, nil
	}
	session, err := prepareLivePipeMux(opt.FFmpegBinaryPath, len(pipeInputs), pipeInputs[0].Path, currentLivePipeEnv())
	if err != nil {
		return outs, err
	}
	return finishLivePipeMuxOutputs(pipeInputs, subtitleOutputs, session, opt.LiveKeepSegments)
}

func finishLivePipeMuxOutputs(pipeInputs []outputFile, subtitleOutputs []outputFile, session *livePipeSession, keepSegments bool) ([]outputFile, error) {
	if session == nil {
		return nil, fmt.Errorf("live pipe session is nil")
	}
	trackFiles := make([][]string, 0, len(pipeInputs))
	for _, out := range pipeInputs {
		if out.Path == "" {
			return nil, fmt.Errorf("live pipe input path is empty")
		}
		if info, err := os.Stat(out.Path); err != nil {
			return nil, err
		} else if info.IsDir() {
			return nil, fmt.Errorf("live pipe input is a directory: %s", out.Path)
		}
		trackFiles = append(trackFiles, []string{out.Path})
	}
	if err := writeLivePipeMuxFiles(session, trackFiles, keepSegments); err != nil {
		return nil, err
	}
	// long: 上游 pipe mux 已经把非字幕轨道实时混成一个 TS，后续输出列表只保留这个 mux 结果和原本单独处理的字幕。
	merged := []outputFile{{Path: session.OutputPath}}
	merged = append(merged, subtitleOutputs...)
	return merged, nil
}

func splitLivePipeMuxOutputs(outs []outputFile) ([]outputFile, []outputFile) {
	var pipeInputs []outputFile
	var subtitleOutputs []outputFile
	for _, out := range outs {
		if out.MediaType != nil && *out.MediaType == MediaSubtitles {
			subtitleOutputs = append(subtitleOutputs, out)
			continue
		}
		pipeInputs = append(pipeInputs, out)
	}
	return pipeInputs, subtitleOutputs
}

func (s *livePipeSession) Close() error {
	var firstErr error
	for _, pipe := range s.Pipes {
		if pipe == nil {
			continue
		}
		if err := pipe.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func currentLivePipeEnv() livePipeEnv {
	return livePipeEnv{
		Options: os.Getenv(envLivePipeOptions),
		TmpDir:  os.Getenv(envLivePipeTmpDir),
	}
}

func livePipePath(name string, env livePipeEnv, windows bool) string {
	if windows {
		return `\\.\pipe\` + name
	}
	pipeDir := env.TmpDir
	if pipeDir == "" {
		pipeDir = os.TempDir()
	}
	return filepath.Join(pipeDir, name)
}

func randomLivePipeName() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "RE_pipe_" + hex.EncodeToString(b[:]), nil
}

func prepareLivePipeMux(binary string, pipeCount int, outputPath string, env livePipeEnv) (*livePipeSession, error) {
	if pipeCount <= 0 {
		return nil, fmt.Errorf("pipe count must be greater than zero")
	}
	if binary == "" {
		binary = "ffmpeg"
	}
	session := &livePipeSession{
		OutputPath: strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".ts",
	}
	for i := 0; i < pipeCount; i++ {
		name, err := randomLivePipeName()
		if err != nil {
			_ = session.Close()
			return nil, err
		}
		pipe, path, err := createLivePipe(name, env)
		if err != nil {
			_ = session.Close()
			return nil, err
		}
		session.Pipes = append(session.Pipes, pipe)
		session.PipeNames = append(session.PipeNames, name)
		session.PipePaths = append(session.PipePaths, path)
	}
	args := buildLivePipeMuxArgs(session.PipeNames, session.OutputPath, time.Now(), env, runtime.GOOS == "windows")
	cmd := exec.Command(binary, args...)
	if err := cmd.Start(); err != nil {
		_ = session.Close()
		return nil, err
	}
	session.Cmd = cmd
	if err := connectLivePipesWithTimeout(session.Pipes, livePipeConnectTimeout); err != nil {
		_ = session.Close()
		if session.Cmd.Process != nil {
			_ = session.Cmd.Process.Kill()
		}
		_ = session.Cmd.Wait()
		return nil, err
	}
	return session, nil
}

func connectLivePipesWithTimeout(pipes []*os.File, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- connectLivePipes(pipes)
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		// long: Windows 命名管道连接依赖 ffmpeg 主动打开输入；超时后主动关闭，避免直播任务在启动 mux 阶段永久卡住。
		for _, pipe := range pipes {
			if pipe != nil {
				_ = pipe.Close()
			}
		}
		return fmt.Errorf("live pipe mux connect timeout after %s", timeout)
	}
}

func connectLivePipes(pipes []*os.File) error {
	for _, pipe := range pipes {
		if err := connectLivePipe(pipe); err != nil {
			return err
		}
	}
	return nil
}

func writeLivePipeMuxFiles(session *livePipeSession, trackFiles [][]string, keepSegments bool) error {
	if session == nil {
		return fmt.Errorf("live pipe session is nil")
	}
	if len(trackFiles) != len(session.Pipes) {
		return fmt.Errorf("track count %d does not match pipe count %d", len(trackFiles), len(session.Pipes))
	}
	errCh := make(chan error, len(trackFiles))
	var wg sync.WaitGroup
	for index := range trackFiles {
		index := index
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := copyFilesToLivePipe(session.Pipes[index], trackFiles[index], keepSegments); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	var firstErr error
	for err := range errCh {
		if firstErr == nil {
			firstErr = err
		}
	}
	for i := range session.Pipes {
		session.Pipes[i] = nil
	}
	if session.Cmd != nil {
		if err := session.Cmd.Wait(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func finishOpenLivePipeMuxSession(session *livePipeSession) error {
	if session == nil {
		return fmt.Errorf("live pipe session is nil")
	}
	var firstErr error
	for i, pipe := range session.Pipes {
		if pipe == nil {
			continue
		}
		if err := pipe.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		session.Pipes[i] = nil
	}
	if session.Cmd != nil {
		if err := session.Cmd.Wait(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func copyFilesToLivePipe(pipe *os.File, files []string, keepSegments bool) error {
	if pipe == nil {
		return fmt.Errorf("live pipe is nil")
	}
	firstErr := copyFilesToOpenLivePipe(pipe, files, keepSegments)
	if err := pipe.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func copyFilesToOpenLivePipe(pipe *os.File, files []string, keepSegments bool) error {
	if pipe == nil {
		return fmt.Errorf("live pipe is nil")
	}
	var firstErr error
	for _, file := range files {
		in, err := os.Open(file)
		if err != nil {
			firstErr = err
			break
		}
		_, copyErr := io.Copy(pipe, in)
		closeErr := in.Close()
		if copyErr != nil {
			firstErr = copyErr
			break
		}
		if closeErr != nil {
			firstErr = closeErr
			break
		}
		if !keepSegments && !strings.HasPrefix(filepath.Base(file), "_init") {
			_ = os.Remove(file)
		}
	}
	return firstErr
}

func buildLivePipeMuxArgs(pipeNames []string, outputPath string, now time.Time, env livePipeEnv, windows bool) []string {
	args := []string{"-y", "-fflags", "+genpts", "-loglevel", "quiet"}
	if strings.TrimSpace(env.Options) != "" {
		args = append(args, "-re")
	}
	for _, name := range pipeNames {
		args = append(args, "-i", livePipePath(name, env, windows))
	}
	for i := range pipeNames {
		args = append(args, "-map", strconv.Itoa(i))
	}
	args = append(args,
		"-strict", "unofficial",
		"-c", "copy",
		"-metadata", "date="+now.Format(time.RFC3339Nano),
		"-ignore_unknown", "-copy_unknown",
	)
	custom := strings.TrimSpace(env.Options)
	if custom != "" {
		if strings.HasPrefix(custom, "-") {
			// long: 上游允许 RE_LIVE_PIPE_OPTIONS 直接追加一整段 ffmpeg 参数；Go 版必须先按 shell 风格拆分，保住标题、URL 等包含空格的引号参数。
			args = append(args, splitLivePipeOptionArgs(custom)...)
		} else {
			args = append(args, "-f", "mpegts", "-shortest", custom)
		}
	} else {
		args = append(args, "-f", "mpegts", "-shortest", outputPath)
	}
	return args
}

func splitLivePipeOptionArgs(input string) []string {
	args, err := splitQuotedArgs(input)
	if err != nil {
		return strings.Fields(input)
	}
	return args
}

func splitQuotedArgs(input string) ([]string, error) {
	var args []string
	var b strings.Builder
	var quote rune
	inToken := false
	runes := []rune(input)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\\' {
			if i+1 < len(runes) && livePipeBackslashEscapes(runes[i+1], quote) {
				i++
				b.WriteRune(runes[i])
			} else {
				b.WriteRune(r)
			}
			inToken = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
				inToken = true
				continue
			}
			b.WriteRune(r)
			inToken = true
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			inToken = true
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if inToken {
				args = append(args, b.String())
				b.Reset()
				inToken = false
			}
			continue
		}
		b.WriteRune(r)
		inToken = true
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	if inToken {
		args = append(args, b.String())
	}
	return args, nil
}

func livePipeBackslashEscapes(next rune, quote rune) bool {
	if quote == '\'' {
		return false
	}
	if quote == '"' {
		return next == '"' || next == '\\'
	}
	return next == '\'' || next == '"' || next == '\\' || next == ' ' || next == '\t' || next == '\n' || next == '\r'
}

func startLivePipeMux(binary string, pipeNames []string, outputPath string) error {
	if binary == "" {
		binary = "ffmpeg"
	}
	args := buildLivePipeMuxArgs(pipeNames, outputPath, time.Now(), currentLivePipeEnv(), runtime.GOOS == "windows")
	cmd := exec.Command(binary, args...)
	return cmd.Run()
}
