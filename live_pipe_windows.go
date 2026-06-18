//go:build windows

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func createLivePipe(pipeName string, env livePipeEnv) (*os.File, string, error) {
	path := livePipePath(pipeName, env, true)
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, path, err
	}
	// long: Windows 的 pipe mux 需要先创建服务端命名管道，ffmpeg 启动后作为客户端连接这个路径，后续直播分片才能持续写入同一个输入流。
	handle, err := windows.CreateNamedPipe(
		name,
		windows.PIPE_ACCESS_OUTBOUND,
		windows.PIPE_TYPE_BYTE|windows.PIPE_WAIT,
		1,
		64*1024,
		64*1024,
		0,
		nil,
	)
	if err != nil {
		return nil, path, err
	}
	return os.NewFile(uintptr(handle), path), path, nil
}

func connectLivePipe(pipe *os.File) error {
	if pipe == nil {
		return fmt.Errorf("live pipe is nil")
	}
	err := windows.ConnectNamedPipe(windows.Handle(pipe.Fd()), nil)
	if err == nil || err == windows.ERROR_PIPE_CONNECTED {
		return nil
	}
	return err
}
