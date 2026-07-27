package main

import "os"

func applyConsoleRedirectDefaults(opt *Options, stdout *os.File, stderr *os.File) bool {
	if opt == nil {
		return false
	}
	if !isConsoleRedirected(stdout) && !isConsoleRedirected(stderr) {
		return false
	}
	// long: 上游在 stdout/stderr 被重定向时强制走无颜色输出，避免日志文件或管道里混入 ANSI 颜色控制序列。
	opt.ForceANSIConsole = true
	opt.NoANSIColor = true
	return true
}

func isConsoleRedirected(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice == 0
}
