package zhanlu

import (
	"os/exec"
	"runtime"
	"strings"
)

// detectRunning best-effort 探测 zhanlu 进程是否运行中。
func detectRunning() bool {
	switch runtime.GOOS {
	case "windows":
		out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq zhanlu.exe", "/NH").Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), "zhanlu.exe")
	default:
		out, err := exec.Command("pgrep", "-f", "zhanlu").Output()
		return err == nil && len(strings.TrimSpace(string(out))) > 0
	}
}
