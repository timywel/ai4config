package codex

import (
	"github.com/timywel/ai4config/internal/platform/hidecmd"
	"os/exec"
	"runtime"
	"strings"
)

// detectRunning best-effort 探测 codex 进程是否运行中。
func detectRunning() bool {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq codex.exe", "/NH")
		hidecmd.Hide(cmd)
		out, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), "codex.exe")
	default:
		cmd := exec.Command("pgrep", "-f", "codex")
		hidecmd.Hide(cmd)
		out, err := cmd.Output()
		return err == nil && len(strings.TrimSpace(string(out))) > 0
	}
}
