package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// PruneBackups removes oldest timestamped backups for a given file, keeping at most max versions.
// Backup filenames are expected to match the pattern: <path>.<YYYYMMDD_HHMMSS>.bak
func PruneBackups(path string, max int) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	prefix := base + "."

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".bak") {
			backups = append(backups, filepath.Join(dir, name))
		}
	}

	if len(backups) <= max {
		return nil
	}

	sort.Strings(backups) // timestamp format → alphabetical = chronological
	for _, p := range backups[:len(backups)-max] {
		if err := os.Remove(p); err != nil {
			return fmt.Errorf("failed to remove old backup %s: %w", p, err)
		}
	}
	return nil
}
