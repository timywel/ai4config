//go:build !windows

// Package hidecmd 非 Windows 平台 no-op（控制台窗口概念仅 Windows 有）。
package hidecmd

import "os/exec"

// Hide 非 Windows 平台无需处理。
func Hide(cmd *exec.Cmd) {}
