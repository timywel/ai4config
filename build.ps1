# cfg4ai 构建任务（Windows）：.\build.ps1 build|test|lint|fmt|clean
# 便携 Go 路径按本机实际安装位置（AGENTS.md 构建节）
param([string]$Task = "build")

$env:CGO_ENABLED = "0"
$env:PATH = "C:\Users\Wel\dev\go\bin;$env:PATH"
# 本机 proxy.golang.org 不可达，使用国内镜像
$env:GOPROXY = "https://goproxy.cn,direct"

switch ($Task) {
  "build" { go build ./... }
  "test"  { go test ./... }
  "fmt"   { gofmt -w . }
  "lint"  { $f = gofmt -l .; if ($f) { "gofmt 未通过："; $f; exit 1 }; go vet ./... }
  "clean" { Remove-Item cfg4ai, cfg4ai.exe, coverage.out -ErrorAction SilentlyContinue }
  default { "未知任务：$Task（可用：build|test|lint|fmt|clean）"; exit 2 }
}
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
