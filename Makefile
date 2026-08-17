# cfg4ai 构建任务（Unix/麒麟；Windows 用 build.ps1）
# 纪律：CGO_ENABLED=0 强制（静态分发，ARCHITECTURE §8）
export CGO_ENABLED=0

.PHONY: build test lint fmt clean

build:
	go build ./...

test:
	go test ./...

lint: fmt-check vet

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt 未通过：" && gofmt -l . && exit 1)

vet:
	go vet ./...

clean:
	rm -f cfg4ai cfg4ai.exe coverage.out
