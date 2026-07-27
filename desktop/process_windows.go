//go:build windows

package main

import (
	"os/exec"
	"time"
)

func prepareManagedCommand(cmd *exec.Cmd) {
	// long: Windows 下先沿用 Go 对主进程的取消逻辑，并限制管道等待时间，避免外部工具残留输出句柄让任务一直挂在运行态。
	cmd.WaitDelay = 2 * time.Second
}
