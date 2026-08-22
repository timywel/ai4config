//go:build windows

// Package hidecmd 隐藏 Windows 子进程控制台窗口（GUI 应用防止弹 cmd 黑窗）。
package hidecmd

import (
	"os/exec"
	"syscall"
)

// Hide 设置命令隐藏控制台窗口（CREATE_NO_WINDOW + HideWindow）。
// GUI 子系统（-H windowsgui）下必须对每个外部子进程调用，否则每个子进程弹一个 cmd 窗口。
func Hide(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags = 0x08000000 // CREATE_NO_WINDOW
}
