# OpenClaw Portable Edition

将 OpenClaw 打包为 Windows U 盘即插即用商业产品。

## 目录结构

```
E:\OpenClaw\
├── OpenClaw.exe            # Go 启动器（单 EXE，无运行时依赖）
├── node\                   # 便携 Node.js v24
├── app\                    # OpenClaw 预编译产物
│   ├── dist\
│   ├── node_modules\
│   ├── openclaw.mjs
│   └── package.json
├── data\                   # 用户数据（配置、凭据、会话）
│   └── .openclaw\
│       ├── openclaw.json
│       ├── credentials\
│       └── workspace\
├── license\
│   └── hwid.dat            # 硬件绑定数据（AES-GCM 加密）
└── README.txt
```

## 制作 U 盘

前置条件：Go 1.22+, Node.js 22+, pnpm, curl, unzip

```bash
# 在项目根目录执行
./portable/build-portable.sh /path/to/usb-drive
```

## 开发启动器

```bash
cd portable/launcher
go test ./...          # 运行测试
go build -o test.exe   # 编译测试
```

### 跳过硬件验证（开发模式）

设置环境变量 `OPENCLAW_PORTABLE_SKIP_HWID=1` 可跳过硬件绑定验证。

## 环境变量

启动器自动设置以下环境变量，无需用户手动配置：

| 变量 | 说明 | 值 |
|---|---|---|
| `OPENCLAW_PORTABLE` | 便携模式标识 | `1` |
| `OPENCLAW_PORTABLE_ROOT` | U 盘根路径 | EXE 所在目录 |
| `OPENCLAW_HOME` | Home 基准 | `<root>/data` |
| `OPENCLAW_STATE_DIR` | 状态目录 | `<root>/data/.openclaw` |
| `HOME` | Node.js home | `<root>/data` |
| `PATH` | 可执行路径 | `<root>/node` + 原PATH |

## 防复制机制

1. 启动器通过 Windows WMI 获取 U 盘物理序列号（`Win32_DiskDrive.SerialNumber`）
2. 同时获取卷序列号（`Win32_LogicalDisk.VolumeSerialNumber`）
3. 两者组合生成 SHA-256 硬件指纹
4. 指纹用 AES-256-GCM 加密存储到 `license/hwid.dat`
5. 首次运行自动绑定当前 U 盘
6. 后续运行验证指纹是否匹配

注意：此方案可防止普通用户直接复制，但无法防专业逆向工程。

## 升级

替换 `app/` 目录即可，用户数据在 `data/` 中不受影响。

## 故障排除

| 问题 | 解决 |
|---|---|
| 启动器提示"找不到 Node.js" | 确认 `node/node.exe` 存在 |
| 启动器提示"硬件验证失败" | 使用原始 U 盘，或设置 `OPENCLAW_PORTABLE_SKIP_HWID=1`（仅开发） |
| 网关启动失败 | 检查 `data/.openclaw/openclaw.json` 配置是否正确 |
