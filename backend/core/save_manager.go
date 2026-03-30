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
	Unk1              int32
	SteamID           uint64
	Unk2              []byte
	Menu              CSMenuSystemSaveLoad
	ActiveSlots       [10]bool
	ProfileSummaries  [10]ProfileSummary
	UserData10Padding []byte
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
	// 1. Skip Header (0x300)
	save.Header, _ = r.ReadBytes(0x300)

	// 2. Read 10 Slots
	for i := 0; i < 10; i++ {
		slotStart := r.Pos()
		// PC slots have a 16-byte MD5 checksum prefix
		r.ReadBytes(0x10)

		if err := save.Slots[i].Read(r, "PC"); err != nil {
			return nil, fmt.Errorf("failed to read slot %d: %w", i, err)
		}
		// Each slot is exactly 0x280000 bytes (excluding checksum). Skip remainder.
		r.Seek(int64(slotStart+0x10+0x280000), 0)
	}

	// 3. Read UserData10
	// PC UserData10 also has a 16-byte MD5 checksum
	udStart := r.Pos()
	r.ReadBytes(0x10)

	save.Unk1, _ = r.ReadI32()
	save.SteamID, _ = r.ReadU64()
	save.Unk2, _ = r.ReadBytes(0x140)

	save.Menu.Read(r)

	// Active Slots
	for i := 0; i < 10; i++ {
		b, _ := r.ReadU8()
		save.ActiveSlots[i] = b == 1
	}

	// Profile Summaries
	for i := 0; i < 10; i++ {
		save.ProfileSummaries[i].Read(r)
	}

	// 4. Read UserData10 Padding
	// UserData10 is exactly 0x60000 bytes (excluding MD5 checksum)
	currentPos := r.Pos()
	remainingUD := (udStart + 0x10 + 0x60000) - currentPos
	if remainingUD > 0 {
		save.UserData10Padding, _ = r.ReadBytes(int(remainingUD))
	}

	// 5. Read UserData11 (Regulation)
	remaining := r.Len() - r.Pos()
	if remaining > 0 {
		save.UserData11, _ = r.ReadBytes(int(remaining))
	}

	return save, nil
}

func loadPSSequential(r *Reader, save *SaveFile) (*SaveFile, error) {
	// 1. Skip Header (0x70)
	save.Header, _ = r.ReadBytes(0x70)

	// 2. Read 10 Slots
	for i := 0; i < 10; i++ {
		slotStart := r.Pos()
		if err := save.Slots[i].Read(r, "PS4"); err != nil {
			return nil, fmt.Errorf("failed to read slot %d: %w", i, err)
		}
		// Each slot is exactly 0x280000 bytes. Skip remainder.
		r.Seek(int64(slotStart+0x280000), 0)
	}

	// 3. Read UserData10
	udStart := r.Pos()
	save.Unk1, _ = r.ReadI32()
	save.SteamID, _ = r.ReadU64()
	save.Unk2, _ = r.ReadBytes(0x140)

	save.Menu.Read(r)

	// Active Slots
	for i := 0; i < 10; i++ {
		b, _ := r.ReadU8()
		save.ActiveSlots[i] = b == 1
	}

	// Profile Summaries
	for i := 0; i < 10; i++ {
		save.ProfileSummaries[i].Read(r)
	}

	// 4. Read UserData10 Padding
	currentPos := r.Pos()
	remainingUD := (udStart + 0x60000) - currentPos
	if remainingUD > 0 {
		save.UserData10Padding, _ = r.ReadBytes(int(remainingUD))
	}

	// 5. Read UserData11 (Regulation)
	remaining := r.Len() - r.Pos()
	if remaining > 0 {
		save.UserData11, _ = r.ReadBytes(int(remaining))
	}

	return save, nil
}

var (
	// DefaultPCHeader is a template BND4 header for Elden Ring PC saves
	DefaultPCHeader = []byte{
		0x42, 0x4e, 0x44, 0x34, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
		0x0c, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x30, 0x30, 0x30, 0x30,
		0x30, 0x30, 0x30, 0x31, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	// DefaultPSHeader is a template header for PS4 saves (decrypted)
	DefaultPSHeader = make([]byte, 0x70)
)

func (s *SaveFile) Write(path string, targetPlatform Platform) error {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	// 1. Header
	header := s.Header
	if targetPlatform != s.Platform {
		if targetPlatform == PlatformPC {
			// Convert to PC: Use default BND4 header
			header = make([]byte, 0x300)
			copy(header, DefaultPCHeader)
		} else {
			// Convert to PS4: Use default empty header
			header = make([]byte, 0x70)
			copy(header, DefaultPSHeader)
		}
	}
	w.WriteBytes(header)

	// 2. Slots
	for i := 0; i < 10; i++ {
		if targetPlatform == PlatformPC {
			// PC: MD5(slot_data) + slot_data
			var slotBuf bytes.Buffer
			sw := NewWriter(&slotBuf)
			s.Slots[i].Write(sw, "PC")

			hash := ComputeMD5(slotBuf.Bytes())
			w.WriteBytes(hash[:])
			w.WriteBytes(slotBuf.Bytes())
		} else {
			// PS4: just slot_data
			s.Slots[i].Write(w, "PS4")
		}
	}

	// 3. UserData10
	if targetPlatform == PlatformPC {
		var udBuf bytes.Buffer
		uw := NewWriter(&udBuf)

		uw.WriteI32(s.Unk1)
		uw.WriteU64(s.SteamID)
		uw.WriteBytes(s.Unk2)
		s.Menu.Write(uw)
		for i := 0; i < 10; i++ {
			if s.ActiveSlots[i] {
				uw.WriteU8(1)
			} else {
				uw.WriteU8(0)
			}
		}
		for i := 0; i < 10; i++ {
			s.ProfileSummaries[i].Write(uw)
		}
		uw.WriteBytes(s.UserData10Padding)

		hash := ComputeMD5(udBuf.Bytes())
		w.WriteBytes(hash[:])
		w.WriteBytes(udBuf.Bytes())
	} else {
		w.WriteI32(s.Unk1)
		w.WriteU64(s.SteamID)
		w.WriteBytes(s.Unk2)
		s.Menu.Write(w)
		for i := 0; i < 10; i++ {
			if s.ActiveSlots[i] {
				w.WriteU8(1)
			} else {
				w.WriteU8(0)
			}
		}
		for i := 0; i < 10; i++ {
			s.ProfileSummaries[i].Write(w)
		}
		w.WriteBytes(s.UserData10Padding)
	}

	// 4. UserData11
	w.WriteBytes(s.UserData11)

	// 5. Finalize
	data := buf.Bytes()

	if targetPlatform == PlatformPC && s.Encrypted {
		// PC saves are typically encrypted with AES-128-CBC
		iv := s.IV
		if len(iv) != 16 {
			iv = make([]byte, 16)
		}

		encrypted, err := EncryptSave(data, iv)
		if err != nil {
			return fmt.Errorf("failed to encrypt save: %w", err)
		}
		data = encrypted
	}

	return os.WriteFile(path, data, 0644)
}

// ImportSlot copies a slot and its metadata from another SaveFile.
func (s *SaveFile) ImportSlot(source *SaveFile, srcIdx, destIdx int) error {
	if srcIdx < 0 || srcIdx >= 10 || destIdx < 0 || destIdx >= 10 {
		return fmt.Errorf("invalid slot index")
	}

	// Copy slot data
	s.Slots[destIdx] = source.Slots[srcIdx]

	// Copy activity status
	s.ActiveSlots[destIdx] = source.ActiveSlots[srcIdx]

	// Copy profile summary
	s.ProfileSummaries[destIdx] = source.ProfileSummaries[srcIdx]

	return nil
}
