package gemini

import (
	"github.com/timywel/ai4config/internal/platform/hidecmd"
	"os/exec"
	"runtime"
	"strings"
)

func detectRunning() bool {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq gemini.exe", "/NH")
		hidecmd.Hide(cmd)
		out, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), "gemini.exe")
	default:
		cmd := exec.Command("pgrep", "-f", "gemini")
		hidecmd.Hide(cmd)
		out, err := cmd.Output()
		return err == nil && len(strings.TrimSpace(string(out))) > 0
	}
}
