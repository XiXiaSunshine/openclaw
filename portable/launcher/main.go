package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	appName     = "OpenClaw Portable"
	version     = "1.0.0"
	defaultPort = "18789"
)

func main() {
	exePath, err := os.Executable()
	if err != nil {
		fatalError("无法获取程序路径: %v", err)
	}

	root := resolveRootDir(exePath)

	// 检查必要文件是否存在
	nodeExe := nodeExePath(root)
	if _, err := os.Stat(nodeExe); os.IsNotExist(err) {
		fatalError("找不到 Node.js: %s\n请确认 U 盘结构完整", nodeExe)
	}
	appEntry := appEntryPath(root)
	if _, err := os.Stat(appEntry); os.IsNotExist(err) {
		fatalError("找不到 OpenClaw: %s\n请确认 U 盘结构完整", appEntry)
	}

	// 硬件绑定验证
	drive := determineDriveLetter(root)
	if err := verifyOrBindHardware(root, drive); err != nil {
		fatalError("硬件验证失败: %v\n\n此副本可能与原始 U 盘不匹配。", err)
	}

	// 确保数据目录存在
	if err := ensureDir(filepath.Join(dataDir(root), ".openclaw")); err != nil {
		fatalError("创建数据目录失败: %v", err)
	}

	// 判断首次运行还是常规启动
	firstRun := isFirstRun(dataDir(root))
	var args []string
	if firstRun {
		fmt.Printf("[%s] 首次运行，启动配置向导...\n", appName)
		args = buildOnboardArgs()
	} else {
		fmt.Printf("[%s] 启动网关...\n", appName)
		args = buildGatewayArgs(false)
	}

	// 构建并执行命令
	cmd := exec.Command(nodeExe, append([]string{appEntry}, args...)...)
	cmd.Env = buildEnv(root, os.Environ())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("[%s] 执行: %s %s\n", appName,
		filepath.Base(nodeExe),
		strings.Join(append([]string{filepath.Base(appEntry)}, args...), " "))

	if err := cmd.Run(); err != nil {
		fatalError("运行失败: %v", err)
	}
}

// determineDriveLetter 从路径中提取盘符（如 "E:"）
func determineDriveLetter(p string) string {
	vol := filepath.VolumeName(p)
	if vol != "" {
		return vol
	}
	return "C:"
}

// collectHardwareInfo 收集硬件信息（磁盘序列号 + 卷序列号）
func collectHardwareInfo(drive string) (diskSN string, volSN string, err error) {
	diskSN, diskErr := getDiskSerialFromWmic()
	volSN, volErr := getVolumeSerialFromWmic(drive)

	if diskErr != nil && volErr != nil {
		return "", "", fmt.Errorf("无法获取硬件信息: disk=%v, vol=%v", diskErr, volErr)
	}

	return diskSN, volSN, nil
}

// verifyOrBindHardware 验证或绑定硬件
// 首次运行：自动绑定当前 U 盘
// 后续运行：验证是否匹配
func verifyOrBindHardware(root string, drive string) error {
	// 开发模式跳过
	if os.Getenv("OPENCLAW_PORTABLE_SKIP_HWID") == "1" {
		fmt.Println("[警告] 跳过硬件验证（开发模式）")
		return nil
	}

	hwidFile := hwidFilePath(root)

	diskSN, volSN, err := collectHardwareInfo(drive)
	if err != nil {
		return err
	}

	currentHWID := generateHWID(diskSN, volSN)

	// 检查是否已有绑定
	if _, err := os.Stat(hwidFile); os.IsNotExist(err) {
		// 首次运行 → 绑定
		if err := ensureDir(filepath.Dir(hwidFile)); err != nil {
			return fmt.Errorf("创建 license 目录失败: %w", err)
		}
		if err := saveHWID(hwidFile, currentHWID); err != nil {
			return fmt.Errorf("保存硬件绑定失败: %w", err)
		}
		fmt.Printf("[%s] 已绑定到当前 U 盘\n", appName)
		return nil
	}

	// 已有绑定 → 验证
	if !verifyHWID(hwidFile, currentHWID) {
		return fmt.Errorf("此副本与当前 U 盘不匹配")
	}

	return nil
}

// buildGatewayArgs 构建网关启动参数
func buildGatewayArgs(verbose bool) []string {
	args := []string{"gateway", "--port", defaultPort}
	if verbose {
		args = append(args, "--verbose")
	}
	return args
}

// buildOnboardArgs 构建首次引导参数
func buildOnboardArgs() []string {
	return []string{"onboard", "--skip-daemon"}
}

// fatalError 输出错误信息并退出
func fatalError(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "\n[%s] 错误: %s\n\n按任意键退出...\n", appName, msg)
	os.Exit(1)
}
