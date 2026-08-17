package claudecode

import (
	"os/exec"
	"runtime"
	"strings"
)

// detectRunning best-effort 探测 claude 进程是否运行中（热重载提示依据，ARCHITECTURE §5.3）。
// 失败/不确定时返回 false（宁可不提示，不误报）。
func detectRunning() bool {
	switch runtime.GOOS {
	case "windows":
		out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq claude.exe", "/NH").Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), "claude.exe")
	default:
		out, err := exec.Command("pgrep", "-f", "claude").Output()
		return err == nil && len(strings.TrimSpace(string(out))) > 0
	}
}
