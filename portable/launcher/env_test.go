package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRootDir(t *testing.T) {
	dir := t.TempDir()
	root := resolveRootDir(filepath.Join(dir, "OpenClaw.exe"))
	if root != dir {
		t.Fatalf("expected %q, got %q", dir, root)
	}
}

func TestResolveRootDirTrailingSep(t *testing.T) {
	dir := t.TempDir() + string(os.PathSeparator)
	root := resolveRootDir(filepath.Join(dir, "OpenClaw.exe"))
	if root != filepath.Clean(dir) {
		t.Fatalf("expected %q, got %q", filepath.Clean(dir), root)
	}
}

func TestEnsureDir(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a", "b", "c")
	err := ensureDir(target)
	if err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !stat.IsDir() {
		t.Fatal("expected directory")
	}
}

func TestIsFirstRun(t *testing.T) {
	dir := t.TempDir()
	// 无配置文件 → 首次运行
	if !isFirstRun(dir) {
		t.Fatal("expected first run when config missing")
	}

	// 创建配置文件
	cfgDir := filepath.Join(dir, ".openclaw")
	os.MkdirAll(cfgDir, 0755)
	os.WriteFile(filepath.Join(cfgDir, "openclaw.json"), []byte("{}"), 0644)

	if isFirstRun(dir) {
		t.Fatal("expected not first run when config exists")
	}
}

func TestBuildEnv(t *testing.T) {
	root := "/test/root"
	env := buildEnv(root, os.Environ())

	expectedVars := map[string]string{
		"OPENCLAW_HOME":      filepath.Join(root, "data"),
		"OPENCLAW_STATE_DIR": filepath.Join(root, "data", ".openclaw"),
		"HOME":               filepath.Join(root, "data"),
	}

	for key, expected := range expectedVars {
		found := false
		for _, e := range env {
			if strings.HasPrefix(e, key+"=") {
				actual := e[len(key)+1:]
				if actual != expected {
					t.Fatalf("%s: expected %q, got %q", key, expected, actual)
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected env var %s to be set", key)
		}
	}
}

func TestBuildEnvPrependsNodeToPath(t *testing.T) {
	root := "/test/root"
	env := buildEnv(root, os.Environ())

	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			pathVal := e[5:]
			expected := filepath.Join(root, "node")
			if !strings.HasPrefix(pathVal, expected) {
				t.Fatalf("PATH should start with %q, got %q", expected, pathVal[:len(expected)+1])
			}
			return
		}
	}
	t.Fatal("PATH not found in env")
}

func TestBuildEnvOverridesExisting(t *testing.T) {
	root := "/new/root"
	existing := []string{
		"HOME=/old/home",
		"OPENCLAW_HOME=/old/openclaw",
		"PATH=/usr/bin",
	}
	env := buildEnv(root, existing)

	homeCount := 0
	for _, e := range env {
		if strings.HasPrefix(e, "HOME=") {
			homeCount++
			if e != "HOME="+filepath.Join(root, "data") {
				t.Fatalf("expected HOME override, got %q", e)
			}
		}
	}
	if homeCount != 1 {
		t.Fatalf("expected exactly 1 HOME entry, got %d", homeCount)
	}
}

func TestNodeExePath(t *testing.T) {
	root := "/test/root"
	expected := filepath.Join(root, "node", "node.exe")
	if nodeExePath(root) != expected {
		t.Fatalf("expected %q", expected)
	}
}

func TestAppEntryPath(t *testing.T) {
	root := "/test/root"
	expected := filepath.Join(root, "app", "openclaw.mjs")
	if appEntryPath(root) != expected {
		t.Fatalf("expected %q", expected)
	}
}

func TestDataDir(t *testing.T) {
	root := "/test/root"
	expected := filepath.Join(root, "data")
	if dataDir(root) != expected {
		t.Fatalf("expected %q", expected)
	}
}

func TestHwidFilePath(t *testing.T) {
	root := "/test/root"
	expected := filepath.Join(root, "license", "hwid.dat")
	if hwidFilePath(root) != expected {
		t.Fatalf("expected %q", expected)
	}
}
