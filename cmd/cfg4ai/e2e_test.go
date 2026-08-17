// e2e 脚手架：编译真实二进制并黑盒驱动（TEST-PLAN §3）。
// 技术选型：标准 testing + os/exec（testscript 仓库已下线，见该节变更记录）。
package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "cfg4ai-e2e-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	name := "cfg4ai"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binPath = filepath.Join(dir, name)

	// 测试工作目录即包目录（cmd/cfg4ai），直接编译当前包
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		panic("构建 e2e 二进制失败: " + err.Error() + "\n" + string(out))
	}

	os.Exit(m.Run())
}

// e2eCase 用例表驱动结构（后续命令按此扩充）。
type e2eCase struct {
	name     string
	args     []string
	wantOut  *regexp.Regexp // 匹配合并输出（stdout+stderr）
	wantCode int
}

func runE2E(t *testing.T, tc e2eCase) {
	t.Helper()
	cmd := exec.Command(binPath, tc.args...)
	out, err := cmd.CombinedOutput()
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	if code != tc.wantCode {
		t.Fatalf("%s: 退出码 = %d, 期望 %d；输出: %s", tc.name, code, tc.wantCode, out)
	}
	if tc.wantOut != nil && !tc.wantOut.Match(out) {
		t.Fatalf("%s: 输出不匹配 %v；实际: %s", tc.name, tc.wantOut, out)
	}
}

// TestVersion 验证 --version 与无参用法错误（CLI-SPEC §0 退出码 0/2）。
func TestVersion(t *testing.T) {
	runE2E(t, e2eCase{
		name:     "version",
		args:     []string{"--version"},
		wantOut:  regexp.MustCompile(`cfg4ai 0\.0\.1-dev`),
		wantCode: 0,
	})
	runE2E(t, e2eCase{
		name:     "no-args 为用法错误",
		args:     nil,
		wantOut:  regexp.MustCompile(`CLI-SPEC`),
		wantCode: 2,
	})
}

// TestUnknownArg 骨架期任何非 version 参数均为用法错误。
func TestUnknownArg(t *testing.T) {
	runE2E(t, e2eCase{
		name:     "unknown",
		args:     []string{"frobnicate"},
		wantOut:  regexp.MustCompile(`cfg4ai`),
		wantCode: 2,
	})
}
