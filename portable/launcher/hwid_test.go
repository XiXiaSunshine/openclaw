package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetDiskSerialFromWmic(t *testing.T) {
	serial, err := getDiskSerialFromWmic()
	if err != nil {
		t.Skipf("wmic not available in this environment: %v", err)
	}
	if serial == "" {
		t.Fatal("expected non-empty disk serial")
	}
	t.Logf("disk serial: %s", serial)
}

func TestGetVolumeSerialFromWmic(t *testing.T) {
	serial, err := getVolumeSerialFromWmic("C:")
	if err != nil {
		t.Skipf("wmic not available in this environment: %v", err)
	}
	if serial == "" {
		t.Fatal("expected non-empty volume serial")
	}
	t.Logf("volume serial: %s", serial)
}

func TestGenerateHWID(t *testing.T) {
	hwid := generateHWID("DISK-SN-12345", "VOL-SN-67890")
	if hwid == "" {
		t.Fatal("expected non-empty hwid")
	}
	// 相同输入应产生相同输出
	hwid2 := generateHWID("DISK-SN-12345", "VOL-SN-67890")
	if hwid != hwid2 {
		t.Fatalf("expected same hwid, got %q and %q", hwid, hwid2)
	}
	// 不同输入应产生不同输出
	hwid3 := generateHWID("DIFFERENT", "VOL-SN-67890")
	if hwid == hwid3 {
		t.Fatal("expected different hwid for different input")
	}
}

func TestSaveAndLoadHWID(t *testing.T) {
	dir := t.TempDir()
	hwidFile := filepath.Join(dir, "hwid.dat")
	expectedHWID := "test-hwid-value"

	// Save
	err := saveHWID(hwidFile, expectedHWID)
	if err != nil {
		t.Fatalf("saveHWID failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(hwidFile); os.IsNotExist(err) {
		t.Fatal("hwid file was not created")
	}

	// Load
	loaded, err := loadHWID(hwidFile)
	if err != nil {
		t.Fatalf("loadHWID failed: %v", err)
	}
	if loaded != expectedHWID {
		t.Fatalf("expected %q, got %q", expectedHWID, loaded)
	}
}

func TestLoadHWIDFileNotFound(t *testing.T) {
	_, err := loadHWID(filepath.Join(t.TempDir(), "nonexistent.dat"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestVerifyHWIDMatch(t *testing.T) {
	dir := t.TempDir()
	hwidFile := filepath.Join(dir, "hwid.dat")
	hwid := generateHWID("DISK-SN", "VOL-SN")

	// 首次保存
	err := saveHWID(hwidFile, hwid)
	if err != nil {
		t.Fatal(err)
	}

	// 相同值应匹配
	if !verifyHWID(hwidFile, hwid) {
		t.Fatal("expected hwid to match")
	}

	// 不同值应不匹配
	if verifyHWID(hwidFile, "wrong-hwid") {
		t.Fatal("expected hwid not to match")
	}
}
