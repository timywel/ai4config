#!/bin/sh
# cfg4ai 安装脚本（Linux/macOS/麒麟）
# 用法：curl -fsSL https://raw.githubusercontent.com/timywel/ai4config/main/scripts/install.sh | sh
# 或：  sh install.sh [版本] [安装目录]
set -eu

VERSION="${1:-latest}"
DEST="${2:-/usr/local/bin}"
REPO="timywel/ai4config"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  loongarch64) ARCH="loong64" ;;
  *) echo "不支持的架构: $ARCH"; exit 1 ;;
esac

if [ "$VERSION" = "latest" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')"
fi

URL="https://github.com/$REPO/releases/download/v${VERSION}/cfg4ai_${VERSION}_${OS}_${ARCH}.tar.gz"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "下载 cfg4ai v$VERSION ($OS/$ARCH)..."
curl -fsSL "$URL" -o "$TMP/cfg4ai.tar.gz"
tar -xzf "$TMP/cfg4ai.tar.gz" -C "$TMP"

if [ -w "$DEST" ]; then
  install -m 0755 "$TMP/cfg4ai" "$DEST/cfg4ai"
else
  echo "需要 sudo 写入 $DEST"
  sudo install -m 0755 "$TMP/cfg4ai" "$DEST/cfg4ai"
fi
echo "已安装：$DEST/cfg4ai"
"$DEST/cfg4ai" version