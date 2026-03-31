package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetermineDriveLetter(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{filepath.Join("C:", "test", "path"), "C:"},
		{filepath.Join("E:", "OpenClaw", "OpenClaw.exe"), "E:"},
		{"/no/drive/letter", "C:"}, // 非 Windows 路径回退到 C:
	}
	for _, tt := range tests {
		got := determineDriveLetter(tt.path)
		if got != tt.expected {
			t.Errorf("determineDriveLetter(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}

func TestCollectHardwareInfo(t *testing.T) {
	drive := determineDriveLetter(filepath.Join("C:", "test"))
	diskSN, volSN, err := collectHardwareInfo(drive)
	if err != nil {
		t.Skipf("hw info not available: %v", err)
	}
	if diskSN == "" && volSN == "" {
		t.Fatal("expected at least one serial number")
	}
	t.Logf("disk=%s vol=%s", diskSN, volSN)
}

func TestBuildGatewayArgs(t *testing.T) {
	args := buildGatewayArgs(false)
	if len(args) < 3 {
		t.Fatalf("expected at least 3 args, got %d", len(args))
	}
	if args[0] != "gateway" {
		t.Fatalf("expected first arg 'gateway', got %q", args[0])
	}
	foundPort := false
	for i, a := range args {
		if a == "--port" && i+1 < len(args) {
			foundPort = true
		}
	}
	if !foundPort {
		t.Fatal("expected --port flag in gateway args")
	}
}

func TestBuildGatewayArgsVerbose(t *testing.T) {
	args := buildGatewayArgs(true)
	foundVerbose := false
	for _, a := range args {
		if a == "--verbose" {
			foundVerbose = true
		}
	}
	if !foundVerbose {
		t.Fatal("expected --verbose flag when verbose=true")
	}
}

func TestBuildOnboardArgs(t *testing.T) {
	args := buildOnboardArgs()
	if len(args) < 2 {
		t.Fatalf("expected at least 2 args, got %d", len(args))
	}
	if args[0] != "onboard" {
		t.Fatalf("expected first arg 'onboard', got %q", args[0])
	}
}

func TestBuildCommandIntegration(t *testing.T) {
	dir := t.TempDir()
	nodeExe := filepath.Join(dir, "node", "node.exe")
	appEntry := filepath.Join(dir, "app", "openclaw.mjs")

	// 创建假 node.exe
	os.MkdirAll(filepath.Join(dir, "node"), 0755)
	os.WriteFile(nodeExe, []byte("echo hello"), 0755)

	os.MkdirAll(filepath.Join(dir, "app"), 0755)
	os.WriteFile(appEntry, []byte("// entry"), 0644)

	cmd := exec.Command(nodeExe, append([]string{appEntry}, buildGatewayArgs(false)...)...)
	env := buildEnv(dir, os.Environ())
	cmd.Env = env

	// 验证环境变量设置正确
	expectedHome := filepath.Join(dir, "data")
	for _, e := range env {
		if strings.HasPrefix(e, "HOME=") {
			if e != "HOME="+expectedHome {
				t.Fatalf("expected HOME=%s, got %s", expectedHome, e)
			}
		}
	}
}

func TestVerifyOrBindHardwareDevMode(t *testing.T) {
	dir := t.TempDir()
	// 设置开发模式跳过 HWID
	os.Setenv("OPENCLAW_PORTABLE_SKIP_HWID", "1")
	defer os.Unsetenv("OPENCLAW_PORTABLE_SKIP_HWID")

	err := verifyOrBindHardware(dir, "C:")
	if err != nil {
		t.Fatalf("expected no error in dev mode, got %v", err)
	}
}
