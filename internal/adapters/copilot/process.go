package copilot

import (
	"os/exec"
	"runtime"
	"strings"
)

// detectRunning best-effort 探测 VS Code 进程是否运行中（热重载提示依据）。
func detectRunning() bool {
	switch runtime.GOOS {
	case "windows":
		out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq Code.exe", "/NH").Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), "Code.exe")
	default:
		out, err := exec.Command("pgrep", "-f", "Visual Studio Code").Output()
		if err == nil && len(strings.TrimSpace(string(out))) > 0 {
			return true
		}
		out, err = exec.Command("pgrep", "-x", "code").Output()
		return err == nil && len(strings.TrimSpace(string(out))) > 0
	}
}
