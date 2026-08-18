package gemini

import (
	"os/exec"
	"runtime"
	"strings"
)

func detectRunning() bool {
	switch runtime.GOOS {
	case "windows":
		out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq gemini.exe", "/NH").Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), "gemini.exe")
	default:
		out, err := exec.Command("pgrep", "-f", "gemini").Output()
		return err == nil && len(strings.TrimSpace(string(out))) > 0
	}
}
