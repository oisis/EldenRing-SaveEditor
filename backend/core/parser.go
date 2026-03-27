package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

// Save interface for both PC and PS4 saves.
type Save interface {
	GetSteamID() uint64
	SetSteamID(steamID uint64)
	Write(path string) error
}

// PCSaveSlot wraps a SaveSlot with its MD5 checksum.
type PCSaveSlot struct {
	Checksum [16]byte
	Data     []byte // Raw SaveSlot data (0x280000 bytes)
}

// PCUserData10 wraps UserData10 with its MD5 checksum.
type PCUserData10 struct {
	Checksum [16]byte
	Data     []byte // Raw UserData10 data (0x60000 bytes)
}

// PCUserData11 wraps UserData11 with its MD5 checksum.
type PCUserData11 struct {
	Checksum [16]byte
	Data     []byte // Raw UserData11 data (0x23FFF0 bytes)
}

// PCSave represents the decrypted PC save file structure.
type PCSave struct {
	Header     []byte // 0x300 bytes
	SaveSlots  [10]PCSaveSlot
	UserData10 PCUserData10
	UserData11 PCUserData11
}

// LoadPC loads and decrypts a PC save file (.sl2).
func LoadPC(path string) (*PCSave, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	decrypted, err := DecryptSave(data)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(decrypted)
	save := &PCSave{}

	// Read Header (0x300 bytes)
	save.Header = make([]byte, 0x300)
	if _, err := reader.Read(save.Header); err != nil {
		return nil, err
	}

	// Read 10 Save Slots (0x280010 bytes each)
	for i := 0; i < 10; i++ {
		if _, err := reader.Read(save.SaveSlots[i].Checksum[:]); err != nil {
			return nil, err
		}
		save.SaveSlots[i].Data = make([]byte, 0x280000)
		if _, err := reader.Read(save.SaveSlots[i].Data); err != nil {
			return nil, err
		}
	}

	// Read UserData10 (0x60010 bytes)
	if _, err := reader.Read(save.UserData10.Checksum[:]); err != nil {
		return nil, err
	}
	save.UserData10.Data = make([]byte, 0x60000)
	if _, err := reader.Read(save.UserData10.Data); err != nil {
		return nil, err
	}

	// Read UserData11 (0x240000 bytes)
	if _, err := reader.Read(save.UserData11.Checksum[:]); err != nil {
		return nil, err
	}
	save.UserData11.Data = make([]byte, 0x23FFF0)
	if _, err := reader.Read(save.UserData11.Data); err != nil {
		return nil, err
	}

	return save, nil
}

// GetSteamID returns the SteamID from UserData10.
func (s *PCSave) GetSteamID() uint64 {
	// SteamID is at offset 0x10 in UserData10 (after 16-byte checksum and 4-byte padding)
	// Wait, UserData10 struct in Rust:
	// pub struct UserData10 {
	//     pub checksum: [u8; 0x10],
	//     _0x19003b4: i32,
	//     pub steam_id: u64,
	// ...
	// So it's 4 bytes after the checksum.
	return binary.LittleEndian.Uint64(s.UserData10.Data[4:12])
}

// SetSteamID updates the SteamID in UserData10 and all active SaveSlots.
func (s *PCSave) SetSteamID(steamID uint64) {
	// Update UserData10
	binary.LittleEndian.PutUint64(s.UserData10.Data[4:12], steamID)

	// Update SaveSlots
	// SteamID is at offset 0x27FF26 in the 0x280000 byte SaveSlot data.
	steamIDOffset := 0x27FF26
	for i := 0; i < 10; i++ {
		// Only update if the slot is not empty (we can check a marker or just update all)
		// The Rust code updates all slots.
		binary.LittleEndian.PutUint64(s.SaveSlots[i].Data[steamIDOffset:steamIDOffset+8], steamID)
	}
}

// Write encrypts and writes the save file to the given path.
// It automatically creates a backup of the original file if it exists.
func (s *PCSave) Write(path string) error {
	// Create backup if file exists
	if _, err := os.Stat(path); err == nil {
		backupPath := fmt.Sprintf("%s.%s.bak", path, time.Now().Format("20060102-150405"))
		if err := copyFile(path, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %v", err)
		}
	}

	var buf bytes.Buffer

	// Recalculate checksums
	s.UserData10.Checksum = ComputeMD5(s.UserData10.Data)
	s.UserData11.Checksum = ComputeMD5(s.UserData11.Data)
	for i := 0; i < 10; i++ {
		s.SaveSlots[i].Checksum = ComputeMD5(s.SaveSlots[i].Data)
	}

	// Write Header
	buf.Write(s.Header)

	// Write Slots
	for i := 0; i < 10; i++ {
		buf.Write(s.SaveSlots[i].Checksum[:])
		buf.Write(s.SaveSlots[i].Data)
	}

	// Write UserData10
	buf.Write(s.UserData10.Checksum[:])
	buf.Write(s.UserData10.Data)

	// Write UserData11
	buf.Write(s.UserData11.Checksum[:])
	buf.Write(s.UserData11.Data)

	// Encrypt
	// We need an IV. Usually it's the first 16 bytes of the original file.
	// For now, let's use a zero IV or reuse the one from the header if it's there.
	// Actually, DecryptSave says: iv := data[:16]
	iv := s.Header[:16]
	encrypted, err := EncryptSave(buf.Bytes(), iv)
	if err != nil {
		return err
	}

	return os.WriteFile(path, encrypted, 0644)
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

