//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func createLivePipe(pipeName string, env livePipeEnv) (*os.File, string, error) {
	path := livePipePath(pipeName, env, false)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, path, err
	}
	if info, err := os.Stat(path); err == nil {
		if info.Mode()&os.ModeNamedPipe == 0 {
			return nil, path, fmt.Errorf("pipe path exists but is not a FIFO: %s", path)
		}
	} else if os.IsNotExist(err) {
		// long: 非 Windows 平台沿用上游 pipe mux 的 FIFO 模型，提前创建管道文件后交给 ffmpeg 作为实时输入。
		if out, mkErr := exec.Command("mkfifo", path).CombinedOutput(); mkErr != nil {
			return nil, path, fmt.Errorf("mkfifo %s failed: %v\n%s", path, mkErr, string(out))
		}
	} else {
		return nil, path, err
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		return nil, path, err
	}
	return file, path, nil
}

func connectLivePipe(pipe *os.File) error {
	if pipe == nil {
		return fmt.Errorf("live pipe is nil")
	}
	return nil
}
