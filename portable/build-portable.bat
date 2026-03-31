@echo off
chcp 65001 >nul 2>&1
setlocal enabledelayedexpansion

echo ============================================
echo   OpenClaw Portable - Build Tool
echo ============================================
echo.

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

set "SCRIPT_DIR=%~dp0"
if "%SCRIPT_DIR:~-1%"=="\" set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"
set "PROJECT_ROOT=%SCRIPT_DIR%\.."

echo Target: %TARGET%
echo.

REM ============================================================
REM [1/8] Creating directories (always runs, mkdir is idempotent)
REM ============================================================
echo [1/8] Creating directories...
if not exist "%TARGET%\node" mkdir "%TARGET%\node"
if not exist "%TARGET%\app" mkdir "%TARGET%\app"
if not exist "%TARGET%\data" mkdir "%TARGET%\data"
if not exist "%TARGET%\license" mkdir "%TARGET%\license"
echo        Done.
echo.

REM ============================================================
REM [2/8] Checking Node.js
REM ============================================================
set "NODEJS_DIR=%SCRIPT_DIR%\nodejs"
if not exist "%NODEJS_DIR%\node.exe" (
    echo [ERROR] Node.js not found at: %NODEJS_DIR%
    echo         Please copy node.exe and npm/npx into that directory.
    goto :fail
)
echo        Source: %NODEJS_DIR%

call :ask_step "2" "Copy Node.js to target" "%TARGET%\node\node.exe" "node.exe already in target"
if not "!SKIP_STEP!"=="0" goto :after_step2

echo [2/8] Copying Node.js...
set "PATH=%NODEJS_DIR%;%PATH%"
xcopy /E /Y /Q /I "%NODEJS_DIR%\*" "%TARGET%\node\" >nul 2>&1
echo        Done.
echo.

:after_step2

REM ============================================================
REM [3/8] Building OpenClaw (pnpm install + build:docker)
REM ============================================================
call :ask_step "3" "Build OpenClaw (pnpm install + build:docker)" "%PROJECT_ROOT%\dist\index.js" "dist/ already exists from previous build"
if not "!SKIP_STEP!"=="0" goto :after_step3

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
echo.

:after_step3

REM ============================================================
REM [4/8] Copying OpenClaw artifacts
REM ============================================================
call :ask_step "4" "Copy OpenClaw artifacts to target" "%TARGET%\app\dist\index.js" "Artifacts already in target/app/dist/"
if not "!SKIP_STEP!"=="0" goto :after_step4

echo [4/8] Copying OpenClaw artifacts...
if exist "%PROJECT_ROOT%\dist" xcopy /E /Y /Q /I "%PROJECT_ROOT%\dist" "%TARGET%\app\dist\" >nul 2>&1
copy /Y "%PROJECT_ROOT%\openclaw.mjs" "%TARGET%\app\" >nul 2>&1
copy /Y "%PROJECT_ROOT%\package.json" "%TARGET%\app\" >nul 2>&1
if exist "%PROJECT_ROOT%\skills" xcopy /E /Y /Q /I "%PROJECT_ROOT%\skills" "%TARGET%\app\skills\" >nul 2>&1
echo        Done.
echo.

:after_step4

REM ============================================================
REM [5/8] Installing production dependencies (pnpm deploy)
REM ============================================================
call :ask_step "5" "Install production dependencies (pnpm deploy)" "%TARGET%\app\node_modules\tslog\package.json" "node_modules already in target"
if not "!SKIP_STEP!"=="0" goto :after_step5

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
echo.

:after_step5

REM ============================================================
REM [6/8] Building launcher (Go)
REM ============================================================
call :ask_step "6" "Build Go launcher (OpenClaw.exe)" "%TARGET%\OpenClaw.exe" "OpenClaw.exe already exists"
if not "!SKIP_STEP!"=="0" goto :after_step6

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
echo.

:after_step6

REM ============================================================
REM [7/8] Creating README (always runs, trivial cost)
REM ============================================================
echo [7/8] Creating README...
> "%TARGET%\README.txt" (
    echo OpenClaw Portable Edition
    echo =========================
    echo.
    echo Usage:
    echo   1. Insert USB into PC
    echo   2. Double-click OpenClaw.exe
    echo   3. First run starts setup wizard
    echo   4. Later runs start gateway automatically
    echo.
    echo All data stays on the USB drive.
    echo Support: Windows 10/11 (64-bit)
)
echo        Done.
echo.

REM ============================================================
REM [8/8] All done!
REM ============================================================
echo [8/8] All done!
echo.
echo ============================================
echo   BUILD SUCCESSFUL
echo ============================================
echo.
dir "%TARGET%"
echo.
pause
exit /b 0

REM ============================================================
REM Subroutine: ask_step
REM   %~1 = step number
REM   %~2 = step description
REM   %~3 = artifact path to check
REM   %~4 = skip message when artifact exists
REM
REM Sets SKIP_STEP=1 if user chooses to skip, otherwise 0
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

REM ============================================================
:fail
echo.
echo ============================================
echo   BUILD FAILED - see errors above
echo ============================================
echo.
pause
exit /b 1
