//go:build !windows

package main

import (
	"errors"
	"os/exec"
	"syscall"
	"time"
)

func prepareManagedCommand(cmd *exec.Cmd) {
	// long: 停止下载任务时要同时结束下载核心拉起的 ffmpeg、解密器或脚本子进程，否则 UI 会显示已请求停止但任务仍被子进程拖住。
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 2 * time.Second
	cmd.Cancel = func() error {
		pid := 0
		if cmd.Process != nil {
			pid = cmd.Process.Pid
		}
		if pid <= 0 {
			return nil
		}
		termErr := signalProcessGroup(pid, syscall.SIGTERM)
		time.Sleep(120 * time.Millisecond)
		killErr := signalProcessGroup(pid, syscall.SIGKILL)
		if termErr != nil {
			return termErr
		}
		return killErr
	}
}

func signalProcessGroup(pid int, signal syscall.Signal) error {
	err := syscall.Kill(-pid, signal)
	if err == nil || errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
