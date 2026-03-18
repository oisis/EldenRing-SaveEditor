package core

import (
	"fmt"
	"os"
	"time"
)

type SaveManager struct {
	CurrentSave *PCSave
	FilePath    string
}

func NewSaveManager() *SaveManager {
	return &SaveManager{}
}

// LoadSave loads, decrypts and parses an Elden Ring PC save file (.sl2)
func (m *SaveManager) LoadSave(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Decrypt the save file
	decrypted, err := DecryptSave(data)
	if err != nil {
		return fmt.Errorf("failed to decrypt save: %v", err)
	}

	// Parse the decrypted data into Go structures
	save := &PCSave{}
	if err := save.Read(decrypted); err != nil {
		return fmt.Errorf("failed to parse save structures: %v", err)
	}

	m.CurrentSave = save
	m.FilePath = path
	return nil
}

// SaveFile encrypts and writes the current save back to disk with a backup
func (m *SaveManager) SaveFile() error {
	if m.CurrentSave == nil {
		return fmt.Errorf("no save file loaded")
	}

	// 1. Create backup
	if err := m.createBackup(); err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}

	// 2. Update all MD5 checksums
	if err := m.CurrentSave.UpdateChecksums(); err != nil {
		return fmt.Errorf("failed to update checksums: %v", err)
	}

	// 3. Serialize structures to bytes
	decrypted, err := m.CurrentSave.Write()
	if err != nil {
		return fmt.Errorf("failed to serialize save: %v", err)
	}

	// 4. Encrypt data (using a new random IV or original one)
	// For now, we use the first 16 bytes of the original file as IV
	originalData, _ := os.ReadFile(m.FilePath)
	iv := originalData[:16]
	
	encrypted, err := EncryptSave(decrypted, iv)
	if err != nil {
		return fmt.Errorf("failed to encrypt save: %v", err)
	}

	// 5. Write to disk
	if err := os.WriteFile(m.FilePath, encrypted, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	// 6. Round-trip Validation
	return m.validateSavedFile()
}

func (m *SaveManager) createBackup() error {
	backupPath := fmt.Sprintf("%s.bak.%s", m.FilePath, time.Now().Format("20060102150405"))
	data, err := os.ReadFile(m.FilePath)
	if err != nil {
		return err
	}
	return os.WriteFile(backupPath, data, 0644)
}

func (m *SaveManager) validateSavedFile() error {
	tempManager := NewSaveManager()
	if err := tempManager.LoadSave(m.FilePath); err != nil {
		return fmt.Errorf("post-write validation failed: %v", err)
	}
	// Verify checksums of the newly written file
	return tempManager.CurrentSave.ValidateChecksums()
}
