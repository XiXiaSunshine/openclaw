package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveRootDir 获取 EXE 所在目录作为 U 盘根路径
func resolveRootDir(exePath string) string {
	return filepath.Dir(exePath)
}

// ensureDir 确保目录存在
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// isFirstRun 检测是否首次运行（配置文件不存在）
func isFirstRun(dataDir string) bool {
	configPath := filepath.Join(dataDir, ".openclaw", "openclaw.json")
	_, err := os.Stat(configPath)
	return os.IsNotExist(err)
}

// buildEnv 构建环境变量列表，注入便携化所需的路径
func buildEnv(root string, existingEnv []string) []string {
	dataDir := filepath.Join(root, "data")
	nodeDir := filepath.Join(root, "node")

	// 需要覆盖/新增的环境变量
	portableVars := map[string]string{
		"OPENCLAW_HOME":      dataDir,
		"OPENCLAW_STATE_DIR": filepath.Join(dataDir, ".openclaw"),
		"HOME":               dataDir,
	}

	result := make([]string, 0, len(existingEnv)+len(portableVars))

	var existingPath string
	for _, e := range existingEnv {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			result = append(result, e)
			continue
		}
		key := parts[0]
		// 跳过被覆盖的变量
		if _, ok := portableVars[key]; ok {
			continue
		}
		if key == "PATH" {
			existingPath = parts[1]
			continue
		}
		result = append(result, e)
	}

	// 注入便携化变量
	for k, v := range portableVars {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}

	// PATH: 便携 node 目录优先
	newPath := nodeDir + string(os.PathListSeparator) + existingPath
	result = append(result, fmt.Sprintf("PATH=%s", newPath))

	return result
}

// nodeExePath 返回便携 Node.js 可执行文件路径
func nodeExePath(root string) string {
	return filepath.Join(root, "node", "node.exe")
}

// appEntryPath 返回 OpenClaw 入口文件路径
func appEntryPath(root string) string {
	return filepath.Join(root, "app", "openclaw.mjs")
}

// dataDir 返回数据目录路径
func dataDir(root string) string {
	return filepath.Join(root, "data")
}

// hwidFilePath 返回硬件绑定文件路径
func hwidFilePath(root string) string {
	return filepath.Join(root, "license", "hwid.dat")
}
