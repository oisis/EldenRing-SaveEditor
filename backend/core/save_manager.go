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

type SaveFile struct {
	Platform         Platform
	Encrypted        bool
	IV               []byte
	Header           []byte
	Slots            [10]SaveSlot
	SteamID          uint64
	UserData10       CSMenuSystemSaveLoad
	ActiveSlots      [10]bool
	ProfileSummaries [10]ProfileSummary
	UserData11       []byte
}

func LoadSave(path string) (*SaveFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	save := &SaveFile{}

	if bytes.HasPrefix(data, []byte("BND4")) {
		save.Platform = PlatformPC
		save.Encrypted = false
		return loadPCSequential(NewReader(data), save)
	}

	decrypted, err := DecryptSave(data)
	if err == nil && bytes.HasPrefix(decrypted, []byte("BND4")) {
		save.Platform = PlatformPC
		save.Encrypted = true
		save.IV = data[:16]
		return loadPCSequential(NewReader(decrypted), save)
	}

	save.Platform = PlatformPS
	return loadPSSequential(NewReader(data), save)
}

func loadPCSequential(r *Reader, save *SaveFile) (*SaveFile, error) {
	save.Header, _ = r.ReadBytes(0x300)

	for i := 0; i < 10; i++ {
		slotStart := r.Pos()
		r.ReadBytes(0x10) // MD5
		if err := save.Slots[i].Read(r, "PC"); err != nil {
			fmt.Printf("Warning: failed to parse slot %d: %v\n", i, err)
		}
		r.Seek(int64(slotStart+0x10+0x280000), 0)
	}

	// UserData10
	udStart := r.Pos()
	r.ReadBytes(0x10) // MD5
	save.UserData10.Data, _ = r.ReadBytes(0x60000)

	udReader := NewReader(save.UserData10.Data)

	// SteamID is at the beginning of UserData10 data on PC
	save.SteamID, _ = udReader.ReadU64()

	// Active Slots are at 0x310
	udReader.Seek(0x310, 0)
	for i := 0; i < 10; i++ {
		b, _ := udReader.ReadU8()
		save.ActiveSlots[i] = b == 1
	}

	// Profile Summaries start at 0x31A (ActiveSlots + 10 bytes)
	// Each summary is 0x100 bytes
	for i := 0; i < 10; i++ {
		save.ProfileSummaries[i].Read(udReader)
	}

	r.Seek(int64(udStart+0x10+0x60000), 0)
	remaining := r.Len() - r.Pos()
	if remaining > 0 {
		save.UserData11, _ = r.ReadBytes(int(remaining))
	}

	return save, nil
}

func loadPSSequential(r *Reader, save *SaveFile) (*SaveFile, error) {
	save.Header, _ = r.ReadBytes(0x70)

	for i := 0; i < 10; i++ {
		slotStart := r.Pos()
		if err := save.Slots[i].Read(r, "PS4"); err != nil {
			fmt.Printf("Warning: failed to parse slot %d: %v\n", i, err)
		}
		r.Seek(int64(slotStart+0x280000), 0)
	}

	save.UserData10.Data, _ = r.ReadBytes(0x60000)
	udReader := NewReader(save.UserData10.Data)

	// PS4: Active Slots are at 0x300, Summaries at 0x30A
	udReader.Seek(0x300, 0)
	for i := 0; i < 10; i++ {
		b, _ := udReader.ReadU8()
		save.ActiveSlots[i] = b == 1
	}
	for i := 0; i < 10; i++ {
		save.ProfileSummaries[i].Read(udReader)
	}

	remaining := r.Len() - r.Pos()
	if remaining > 0 {
		save.UserData11, _ = r.ReadBytes(int(remaining))
	}

	return save, nil
}

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

	if _, err := os.Stat(path); err == nil {
		backupPath := path + ".bak"
		_ = os.WriteFile(backupPath, finalData, 0644)
	}

	return os.WriteFile(path, finalData, 0644)
}
