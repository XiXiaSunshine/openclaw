package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// 内嵌加密密钥（编译到 EXE 中）
var hwidSecretKey = []byte("OpenClaw-Portable-HWID-Binding-Key")

// getDiskSerialFromWmic 通过 wmic 获取物理磁盘序列号
func getDiskSerialFromWmic() (string, error) {
	out, err := exec.Command("wmic", "diskdrive", "get", "serialnumber").Output()
	if err != nil {
		return "", fmt.Errorf("wmic diskdrive: %w", err)
	}
	return parseWmicOutput(string(out)), nil
}

// getVolumeSerialFromWmic 通过 wmic 获取卷序列号
func getVolumeSerialFromWmic(drive string) (string, error) {
	out, err := exec.Command("wmic", "logicaldisk", "where",
		fmt.Sprintf("DeviceID='%s'", drive),
		"get", "volumeserialnumber").Output()
	if err != nil {
		return "", fmt.Errorf("wmic logicaldisk: %w", err)
	}
	return parseWmicOutput(string(out)), nil
}

// parseWmicOutput 解析 wmic 命令的表格输出，提取第一行有效值
func parseWmicOutput(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// 跳过表头行
		if strings.EqualFold(trimmed, "SerialNumber") ||
			strings.EqualFold(trimmed, "VolumeSerialNumber") {
			continue
		}
		return trimmed
	}
	return ""
}

// generateHWID 根据磁盘序列号和卷序列号生成硬件指纹
func generateHWID(diskSerial, volumeSerial string) string {
	h := sha256.New()
	h.Write([]byte(diskSerial + ":" + volumeSerial))
	h.Write(hwidSecretKey)
	return hex.EncodeToString(h.Sum(nil))
}

// saveHWID 将 HWID 加密后保存到文件
func saveHWID(filePath string, hwid string) error {
	encrypted, err := encrypt([]byte(hwid))
	if err != nil {
		return fmt.Errorf("encrypt hwid: %w", err)
	}
	return os.WriteFile(filePath, encrypted, 0600)
}

// loadHWID 从文件读取并解密 HWID
func loadHWID(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read hwid file: %w", err)
	}
	decrypted, err := decrypt(data)
	if err != nil {
		return "", fmt.Errorf("decrypt hwid: %w", err)
	}
	return string(decrypted), nil
}

// verifyHWID 验证当前 HWID 是否与文件中保存的匹配
func verifyHWID(hwidFile string, currentHWID string) bool {
	saved, err := loadHWID(hwidFile)
	if err != nil {
		return false
	}
	return saved == currentHWID
}

// encrypt 使用 AES-GCM 加密
func encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(deriveKey())
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decrypt 使用 AES-GCM 解密
func decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(deriveKey())
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// deriveKey 从密钥派生 32 字节 AES 密钥
func deriveKey() []byte {
	h := sha256.New()
	h.Write(hwidSecretKey)
	return h.Sum(nil)
}
