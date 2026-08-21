package store

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func loadLedger(path string) (Ledger, error) {
	primary, primaryErr := readLedgerFile(path)
	if primaryErr == nil {
		if err := validateLedger(primary); err == nil {
			return primary, nil
		}
	}
	backup, backupErr := readLedgerFile(path + ".bak")
	if backupErr == nil {
		if err := validateLedger(backup); err == nil {
			return backup, nil
		}
	}
	if os.IsNotExist(primaryErr) && os.IsNotExist(backupErr) {
		return NewLedger(), nil
	}
	if primaryErr != nil {
		return Ledger{}, fmt.Errorf("主账本和恢复副本均不可用: %w", primaryErr)
	}
	return Ledger{}, fmt.Errorf("主账本和恢复副本均不可用: %w", backupErr)
}

func readLedgerFile(path string) (Ledger, error) {
	file, err := os.Open(path)
	if err != nil {
		return Ledger{}, err
	}
	defer file.Close()
	contents, err := io.ReadAll(file)
	if err != nil {
		return Ledger{}, fmt.Errorf("读取账本失败: %w", err)
	}
	if len(contents) == 0 {
		return Ledger{}, fmt.Errorf("%w: 账本文件为空", ErrCorruptLedger)
	}
	var ledger Ledger
	if err := json.Unmarshal(contents, &ledger); err != nil {
		return Ledger{}, fmt.Errorf("%w: %v", ErrCorruptLedger, err)
	}
	return normalizeLedger(ledger), nil
}

func persistLedger(path string, ledger Ledger) error {
	encoded, err := json.MarshalIndent(ledger, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化账本失败: %w", err)
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("创建账本目录失败: %w", err)
	}
	if err := backupExisting(path, directory); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".ledger-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时账本失败: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("设置临时账本权限失败: %w", err)
	}
	if _, err := temporary.Write(encoded); err != nil {
		temporary.Close()
		return fmt.Errorf("写入临时账本失败: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("同步临时账本失败: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("关闭临时账本失败: %w", err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("原子替换账本失败: %w", err)
	}
	directoryFile, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("打开账本目录失败: %w", err)
	}
	defer directoryFile.Close()
	if err := directoryFile.Sync(); err != nil {
		return fmt.Errorf("同步账本目录失败: %w", err)
	}
	return nil
}

func backupExisting(path, directory string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("读取旧账本失败: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".ledger-backup-*.tmp")
	if err != nil {
		return fmt.Errorf("创建账本恢复副本失败: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("设置恢复副本权限失败: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("写入恢复副本失败: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("同步恢复副本失败: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("关闭恢复副本失败: %w", err)
	}
	if err := os.Rename(temporaryName, path+".bak"); err != nil {
		return fmt.Errorf("提交恢复副本失败: %w", err)
	}
	return nil
}
