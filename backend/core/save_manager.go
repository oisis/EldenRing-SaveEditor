package core

import (
	"bytes"
	"fmt"
	"os"
)

type Platform string

const (
	PlatformPC Platform = "PC"
	PlatformPS Platform = "PS4"
)

// SaveFile represents the entire save file state in memory.
type SaveFile struct {
	Platform          Platform
	Encrypted         bool
	IV                []byte
	Header            []byte
	Slots             [10]SaveSlot
	SteamID           uint64
	UserData10        CSMenuSystemSaveLoad
	ActiveSlots       [10]bool
	ProfileSummaries  [10]ProfileSummary
	UserData11        []byte
}

// LoadSave loads a save file from the given path, auto-detecting the platform.
func LoadSave(path string) (*SaveFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	save := &SaveFile{}

	// 1. Detect Platform & Decrypt if needed
	if bytes.HasPrefix(data, []byte("BND4")) {
		save.Platform = PlatformPC
		save.Encrypted = false
		return loadPCSequential(NewReader(data), save)
	}

	// Try decrypting (PC saves start with 16-byte IV)
	decrypted, err := DecryptSave(data)
	if err == nil && bytes.HasPrefix(decrypted, []byte("BND4")) {
		save.Platform = PlatformPC
		save.Encrypted = true
		save.IV = data[:16]
		return loadPCSequential(NewReader(decrypted), save)
	}

	// Default to PS4
	save.Platform = PlatformPS
	return loadPSSequential(NewReader(data), save)
}

func loadPCSequential(r *Reader, save *SaveFile) (*SaveFile, error) {
	// 1. Header (0x300 bytes)
	save.Header, _ = r.ReadBytes(0x300)

	// 2. Read 10 Slots
	for i := 0; i < 10; i++ {
		slotStart := r.Pos()
		// PC slots have a 16-byte MD5 checksum prefix
		r.ReadBytes(0x10)

		if err := save.Slots[i].Read(r, "PC"); err != nil {
			// If a slot fails to parse (e.g. empty), we just continue
			fmt.Printf("Warning: failed to parse slot %d: %v\n", i, err)
		}
		r.Seek(int64(slotStart+0x10+0x280000), 0)
	}

	// 3. Read UserData10 (0x60000 bytes + 0x10 MD5)
	r.ReadBytes(0x10) // MD5
	save.UserData10.Read(r)

	// 4. Read UserData11 (Regulation)
	remaining := r.Len() - r.Pos()
	if remaining > 0 {
		save.UserData11, _ = r.ReadBytes(int(remaining))
	}

	return save, nil
}

func loadPSSequential(r *Reader, save *SaveFile) (*SaveFile, error) {
	// 1. Header (0x70 bytes)
	save.Header, _ = r.ReadBytes(0x70)

	// 2. Read 10 Slots
	for i := 0; i < 10; i++ {
		slotStart := r.Pos()
		if err := save.Slots[i].Read(r, "PS4"); err != nil {
			fmt.Printf("Warning: failed to parse slot %d: %v\n", i, err)
		}
		r.Seek(int64(slotStart+0x280000), 0)
	}

	// 3. Read UserData10
	save.UserData10.Read(r)

	// 4. Read UserData11
	remaining := r.Len() - r.Pos()
	if remaining > 0 {
		save.UserData11, _ = r.ReadBytes(int(remaining))
	}

	return save, nil
}

// SaveFile writes the current state back to a file.
func (s *SaveFile) SaveFile(path string) error {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	if s.Platform == PlatformPC {
		w.WriteBytes(s.Header)
		for i := 0; i < 10; i++ {
			slotData := s.Slots[i].Write("PC")
			checksum := ComputeMD5(slotData)
			w.WriteBytes(checksum[:])
			w.WriteBytes(slotData)
		}
		
		udData := s.UserData10.Data
		checksum := ComputeMD5(udData)
		w.WriteBytes(checksum[:])
		w.WriteBytes(udData)
		
		w.WriteBytes(s.UserData11)
	} else {
		w.WriteBytes(s.Header)
		for i := 0; i < 10; i++ {
			w.WriteBytes(s.Slots[i].Write("PS4"))
		}
		w.WriteBytes(s.UserData10.Data)
		w.WriteBytes(s.UserData11)
	}

	finalData := buf.Bytes()
	if s.Encrypted && s.Platform == PlatformPC {
		var err error
		finalData, err = EncryptSave(finalData, s.IV)
		if err != nil {
			return fmt.Errorf("failed to encrypt save: %w", err)
		}
	}

	// Create backup
	if _, err := os.Stat(path); err == nil {
		backupPath := path + ".bak"
		_ = os.WriteFile(backupPath, finalData, 0644)
	}

	return os.WriteFile(path, finalData, 0644)
}
