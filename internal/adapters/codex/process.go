package codex

import (
	"os/exec"
	"runtime"
	"strings"
)

// detectRunning best-effort 探测 codex 进程是否运行中。
func detectRunning() bool {
	switch runtime.GOOS {
	case "windows":
		out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq codex.exe", "/NH").Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), "codex.exe")
	default:
		out, err := exec.Command("pgrep", "-f", "codex").Output()
		return err == nil && len(strings.TrimSpace(string(out))) > 0
	}
}
