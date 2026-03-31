#!/usr/bin/env bash
set -euo pipefail

# ─────────────────────────────────────────────────────────────
# OpenClaw U 盘便携版 — 预装制作脚本
#
# 用法:
#   ./build-portable.sh /path/to/usb-drive
#
# 前置条件:
#   - Go 1.22+ (编译启动器)
#   - Node.js 22+ / pnpm (构建 OpenClaw)
#   - curl / unzip
# ─────────────────────────────────────────────────────────────

TARGET="${1:?用法: $0 <目标目录>}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

NODE_VERSION="v24.0.0"
NODE_ARCH="x64"
NODE_URL="https://nodejs.org/dist/${NODE_VERSION}/node-${NODE_VERSION}-win-${NODE_ARCH}.zip"
NODE_ZIP="/tmp/openclaw-node-portable.zip"

echo "=== OpenClaw 便携版制作工具 ==="
echo "目标目录: $TARGET"
echo ""

# 1. 创建目录结构
echo "[1/7] 创建目录结构..."
mkdir -p "$TARGET"/{node,app,data/.openclaw,license}

# 2. 下载 Node.js 便携版
if [ ! -f "$NODE_ZIP" ]; then
  echo "[2/7] 下载 Node.js ${NODE_VERSION}..."
  curl -fSL -o "$NODE_ZIP" "$NODE_URL"
else
  echo "[2/7] Node.js 已缓存，跳过下载"
fi

echo "        解压 Node.js..."
rm -rf /tmp/openclaw-node-portable-extract
unzip -qo "$NODE_ZIP" -d /tmp/openclaw-node-portable-extract
# node-zip 解压后带有版本号子目录
NODE_EXTRACTED="$(find /tmp/openclaw-node-portable-extract -maxdepth 1 -type d -name "node-*" | head -1)"
if [ -z "$NODE_EXTRACTED" ]; then
  echo "错误: 解压后找不到 node 目录"
  exit 1
fi
cp -r "$NODE_EXTRACTED"/* "$TARGET/node/"

# 3. 构建 OpenClaw
echo "[3/7] 构建 OpenClaw..."
cd "$PROJECT_ROOT"
pnpm install --frozen-lockfile
pnpm build

# 4. 复制 OpenClaw 产物
echo "[4/7] 复制 OpenClaw 产物..."
cp -r dist/ "$TARGET/app/dist/"
cp -r node_modules/ "$TARGET/app/node_modules/"
cp openclaw.mjs "$TARGET/app/"
cp package.json "$TARGET/app/"

# 复制 skills（如果有）
if [ -d "skills" ]; then
  cp -r skills/ "$TARGET/app/skills/"
fi

# 5. 编译 Go 启动器
echo "[5/7] 编译启动器..."
cd "$SCRIPT_DIR/launcher"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
  -ldflags="-s -w -X main.version=$(git -C "$PROJECT_ROOT" describe --tags --always 2>/dev/null || echo dev)" \
  -o "$TARGET/OpenClaw.exe"

# 6. 创建 README
echo "[6/7] 创建 README..."
cat > "$TARGET/README.txt" << 'READMEEOF'
OpenClaw Portable Edition
=========================

使用方法:
1. 将 U 盘插入电脑
2. 双击 OpenClaw.exe
3. 首次运行会进入配置向导
4. 之后每次启动自动运行网关

配置数据保存在 U 盘上，拔出后不留痕迹。

支持: Windows 10/11 (64位)
READMEEOF

# 7. 完成
echo "[7/7] 完成!"
echo ""
echo "=== 制作完成 ==="
echo "U 盘内容:"
du -sh "$TARGET"/* 2>/dev/null || true
echo ""
echo "总大小:"
du -sh "$TARGET"
echo ""
echo "请将 U 盘插入 Windows 电脑测试。"
