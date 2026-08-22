package claudecode

import (
	"github.com/timywel/ai4config/internal/platform/hidecmd"
	"os/exec"
	"runtime"
	"strings"
)

// detectRunning best-effort 探测 claude 进程是否运行中（热重载提示依据，ARCHITECTURE §5.3）。
// 失败/不确定时返回 false（宁可不提示，不误报）。
func detectRunning() bool {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq claude.exe", "/NH")
		hidecmd.Hide(cmd)
		out, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), "claude.exe")
	default:
		cmd := exec.Command("pgrep", "-f", "claude")
		hidecmd.Hide(cmd)
		out, err := cmd.Output()
		return err == nil && len(strings.TrimSpace(string(out))) > 0
	}
}
