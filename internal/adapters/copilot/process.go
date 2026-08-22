package copilot

import (
	"github.com/timywel/ai4config/internal/platform/hidecmd"
	"os/exec"
	"runtime"
	"strings"
)

// detectRunning best-effort 探测 VS Code 进程是否运行中（热重载提示依据）。
func detectRunning() bool {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq Code.exe", "/NH")
		hidecmd.Hide(cmd)
		out, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), "Code.exe")
	default:
		cmd := exec.Command("pgrep", "-f", "Visual Studio Code")
		hidecmd.Hide(cmd)
		out, err := cmd.Output()
		if err == nil && len(strings.TrimSpace(string(out))) > 0 {
			return true
		}
		cmd = exec.Command("pgrep", "-x", "code")
		hidecmd.Hide(cmd)
		out, err = cmd.Output()
		return err == nil && len(strings.TrimSpace(string(out))) > 0
	}
}
