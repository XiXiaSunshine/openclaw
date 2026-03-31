# Interactive Build Script 改造计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 `build-portable.bat` 改造为逐步确认的交互式构建脚本，自动检测已有产物并提示跳过。

**Architecture:** 保留现有 8 步构建流程不变，在每个可跳过步骤前加入产物检测和 `choice` 交互确认。使用 `if exist` 检测已有产物，存在时默认跳过（`[y/N]`），不存在时默认执行（`[Y/n]`）。

**Tech Stack:** Windows Batch Script（`choice`, `if exist`, `setlocal enabledelayedexpansion`）

---

## 概念设计

```
[1/8] Creating directories        → 始终执行（mkdir 幂等）
[2/8] Checking Node.js            → 检测 target\node\node.exe
[3/8] Building OpenClaw           → 检测项目根 dist\ 目录
[4/8] Copying artifacts           → 检测 target\app\dist\ 目录
[5/8] Production dependencies     → 检测 target\app\node_modules\tslog\
[6/8] Building launcher           → 检测 target\OpenClaw.exe
[7/8] Creating README             → 始终执行（覆盖写，代价极低）
[8/8] Done
```

每个可跳过步骤的交互模式：

| 产物状态 | 提示语 | 默认 | `choice` 键位 |
|---------|--------|------|---------------|
| 不存在 | `Step N: <描述>? [Y/n]` | Y（执行） | Y=执行, N=跳过 |
| 已存在 | `[SKIP] <产物> already exists. Redo? [y/N]` | N（跳过） | Y=重做, N=跳过 |

---

### Task 1: 添加 ask_step 子程序

**Files:**
- Modify: `portable/build-portable.bat` (脚本末尾，`:fail` 标签之前)

**Step 1: 在 `:fail` 标签之前添加 `:ask_step` 子程序**

此子程序封装产物检测 + 交互确认逻辑，供每个步骤调用。

```bat
REM ============================================================
REM Subroutine: ask_step
REM   %~1 = step number (e.g. "2")
REM   %~2 = step description (e.g. "Copy Node.js")
REM   %~3 = artifact path to check (e.g. "%TARGET%\node\node.exe")
REM   %~4 = skip message (e.g. "node.exe already in target")
REM
REM Sets SKIP_STEP=1 if user chooses to skip, otherwise SKIP_STEP=0
REM ============================================================
:ask_step
set "SKIP_STEP=0"
set "AStep_Num=%~1"
set "AStep_Desc=%~2"
set "AStep_Artifact=%~3"
set "AStep_SkipMsg=%~4"

if exist "%AStep_Artifact%" (
    echo [SKIP] %AStep_SkipMsg%
    choice /c YN /n /m "  Redo step %AStep_Num%: %AStep_Desc%? [y/N] "
    if errorlevel 2 (
        set "SKIP_STEP=1"
        echo        Skipped.
        echo.
        goto :eof
    )
) else (
    echo Step %AStep_Num%: %AStep_Desc%
    choice /c YN /n /m "  Execute? [Y/n] "
    if errorlevel 2 (
        set "SKIP_STEP=1"
        echo        Skipped.
        echo.
        goto :eof
    )
)
goto :eof
```

**Step 2: 手动验证**

运行 `build-portable.bat` 确认语法无误（`choice` 在现代 Windows 中始终可用）。

---

### Task 2: 改造 Step 2 — Node.js 复制

**Files:**
- Modify: `portable/build-portable.bat` (Step 2 区域，约第 40-55 行)

**Step 1: 替换 Step 2 原有代码**

将当前的 Step 2 硬编码逻辑替换为交互式版本：

```bat
echo [2/8] Checking Node.js...
set "NODEJS_DIR=%SCRIPT_DIR%\nodejs"
if not exist "%NODEJS_DIR%\node.exe" (
    echo [ERROR] Node.js not found at: %NODEJS_DIR%
    echo         Please copy node.exe and npm/npx into that directory.
    goto :fail
)
echo        Source: %NODEJS_DIR%

call :ask_step "2" "Copy Node.js to target" "%TARGET%\node\node.exe" "node.exe already in target"
if "!SKIP_STEP!"=="0" (
    set "PATH=%NODEJS_DIR%;%PATH%"
    xcopy /E /Y /Q /I "%NODEJS_DIR%\*" "%TARGET%\node\" >nul 2>&1
    echo        Done.
)
echo.
```

**Step 2: 验证**

- 目标目录无 node.exe 时：提示 `[Y/n]`，默认执行
- 目标目录已有 node.exe 时：提示 `[SKIP]` + `[y/N]`，默认跳过

---

### Task 3: 改造 Step 3 — 构建 OpenClaw

**Files:**
- Modify: `portable/build-portable.bat` (Step 3 区域，约第 57-104 行)

**Step 1: 替换 Step 3 原有代码**

保留 dist 清理重试逻辑，在外层包裹交互确认：

```bat
call :ask_step "3" "Build OpenClaw (pnpm install + build:docker)" "%PROJECT_ROOT%\dist\index.js" "dist/ already exists from previous build"
if "!SKIP_STEP!"=="0" (
    echo [3/8] Building OpenClaw...
    pushd "%PROJECT_ROOT%"

    REM Clean previous build artifacts to avoid file locking issues
    if exist "dist" (
        echo        Cleaning previous build...
        rmdir /S /Q "dist" 2>nul
        set "CLEAN_RETRY=0"
        :clean_retry
        if exist "dist" (
            if !CLEAN_RETRY! LSS 3 (
                set /a "CLEAN_RETRY+=1"
                echo        dist/ locked, retrying in 2s... [!CLEAN_RETRY!/3]
                timeout /t 2 /nobreak >nul 2>nul
                rmdir /S /Q "dist" 2>nul
                goto :clean_retry
            )
            echo [WARN] Cannot fully clean dist/ - proceeding anyway.
        )
    )

    REM Verify node is still accessible after pushd
    where node >nul 2>&1
    if errorlevel 1 (
        echo [ERROR] node not found after directory change.
        popd
        goto :fail
    )
    echo        node found at:
    where node 2>nul | findstr /n "." | findstr "^1:"
    echo.

    call pnpm install --frozen-lockfile
    if errorlevel 1 (
        echo [ERROR] pnpm install failed.
        popd
        goto :fail
    )
    call pnpm build:docker
    if errorlevel 1 (
        echo [ERROR] pnpm build failed.
        popd
        goto :fail
    )
    popd
    echo        Done.
)
echo.
```

---

### Task 4: 改造 Step 4 — 复制产物

**Files:**
- Modify: `portable/build-portable.bat` (Step 4 区域，约第 106-112 行)

**Step 1: 替换 Step 4 原有代码**

```bat
call :ask_step "4" "Copy OpenClaw artifacts to target" "%TARGET%\app\dist\index.js" "Artifacts already in target/app/dist/"
if "!SKIP_STEP!"=="0" (
    echo [4/8] Copying OpenClaw artifacts...
    if exist "%PROJECT_ROOT%\dist" xcopy /E /Y /Q /I "%PROJECT_ROOT%\dist" "%TARGET%\app\dist\" >nul 2>&1
    copy /Y "%PROJECT_ROOT%\openclaw.mjs" "%TARGET%\app\" >nul 2>&1
    copy /Y "%PROJECT_ROOT%\package.json" "%TARGET%\app\" >nul 2>&1
    if exist "%PROJECT_ROOT%\skills" xcopy /E /Y /Q /I "%PROJECT_ROOT%\skills" "%TARGET%\app\skills\" >nul 2>&1
    echo        Done.
)
echo.
```

---

### Task 5: 改造 Step 5 — 生产依赖

**Files:**
- Modify: `portable/build-portable.bat` (Step 5 区域，约第 114-139 行)

**Step 1: 替换 Step 5 原有代码**

```bat
call :ask_step "5" "Install production dependencies (pnpm deploy)" "%TARGET%\app\node_modules\tslog\package.json" "node_modules already in target"
if "!SKIP_STEP!"=="0" (
    echo [5/8] Installing production dependencies...
    set "DEPLOY_TEMP=%TEMP%\openclaw-portable-deploy"
    if exist "!DEPLOY_TEMP!" rmdir /S /Q "!DEPLOY_TEMP!" 2>nul
    mkdir "!DEPLOY_TEMP!"

    pushd "%PROJECT_ROOT%"
    call pnpm deploy --prod --legacy --filter=openclaw "!DEPLOY_TEMP!"
    if errorlevel 1 (
        echo [ERROR] pnpm deploy failed.
        popd
        goto :fail
    )
    popd

    echo        Copying production node_modules...
    robocopy "!DEPLOY_TEMP!\node_modules" "%TARGET%\app\node_modules" /E /NJH /NJS /NDL /NFL /NC /NS /NP >nul
    if errorlevel 8 (
        echo [ERROR] Failed to copy node_modules.
        rmdir /S /Q "!DEPLOY_TEMP!" 2>nul
        goto :fail
    )

    echo        Cleaning up...
    rmdir /S /Q "!DEPLOY_TEMP!" 2>nul
    echo        Done.
)
echo.
```

---

### Task 6: 改造 Step 6 — 构建 Launcher

**Files:**
- Modify: `portable/build-portable.bat` (Step 6 区域，约第 141-159 行)

**Step 1: 替换 Step 6 原有代码**

```bat
call :ask_step "6" "Build Go launcher (OpenClaw.exe)" "%TARGET%\OpenClaw.exe" "OpenClaw.exe already exists"
if "!SKIP_STEP!"=="0" (
    echo [6/8] Building launcher...
    where go >nul 2>&1
    if errorlevel 1 (
        echo [ERROR] go not found. Install Go 1.22+ first.
        goto :fail
    )
    pushd "%SCRIPT_DIR%\launcher"
    set CGO_ENABLED=0
    set GOOS=windows
    set GOARCH=amd64
    go build -ldflags="-s -w" -o "%TARGET%\OpenClaw.exe"
    if errorlevel 1 (
        echo [ERROR] Go build failed.
        popd
        goto :fail
    )
    popd
    echo        Done.
)
echo.
```

---

### Task 7: 改造 Step 1 — 目录创建 + 目标目录交互输入

**Files:**
- Modify: `portable/build-portable.bat` (脚本开头，约第 10-30 行)

**Step 1: 替换参数检查逻辑，支持交互输入目标目录**

```bat
if "%~1"=="" (
    echo.
    set /p "TARGET=  Target directory: "
    if "!TARGET!"=="" (
        echo [ERROR] No target directory specified.
        pause
        exit /b 1
    )
) else (
    set "TARGET=%~1"
)
if "%TARGET:~-1%"=="\" set "TARGET=%TARGET:~0,-1%"
```

---

### Task 8: 端到端验证

**Step 1: 空目标目录测试**

```
build-portable.bat D:\openclaw-test
```

预期：每步提示 `[Y/n]`，全部 Y 应完成完整构建。

**Step 2: 已有产物测试**

再次运行同命令。预期：每步提示 `[SKIP]` + `[y/N]`，全部 N 应瞬间完成。

**Step 3: 选择性执行测试**

第三次运行，选择只重做 Step 6（Go launcher）。预期：仅重新编译 OpenClaw.exe。

---

## 实施约束

- `choice /c YN /n` — 无大小写区分，Y=errorlevel 1, N=errorlevel 2
- `%AStep_*` 变量前缀避免与主脚本变量冲突
- Step 1（创建目录）和 Step 7（README）始终执行，不加入交互
- Step 3 的 `:clean_retry` 标签需保持在 `if` 块内以避免脚本流混乱
