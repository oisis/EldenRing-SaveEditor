package core

import (
	"fmt"
	"io"
	"os"
	"time"
)

// CreateBackup creates a copy of the file at the given path.
// The backup file is named: original_filename.YYYYMMDD_HHMMSS.bak
// Returns the path to the created backup file.
func CreateBackup(path string) (string, error) {
	sourceFile, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open source file for backup: %w", err)
	}
	defer sourceFile.Close()

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.%s.bak", path, timestamp)

	destFile, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return "", fmt.Errorf("failed to copy data to backup: %w", err)
	}

	// Ensure data is synced to disk
	err = destFile.Sync()
	if err != nil {
		return "", fmt.Errorf("failed to sync backup file to disk: %w", err)
	}

	return backupPath, nil
}
