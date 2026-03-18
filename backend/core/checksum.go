package core

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"fmt"
)

// ValidateChecksums checks if all MD5 hashes in the save file are correct
func (s *PCSave) ValidateChecksums() error {
	// Validate Slots
	for i, slot := range s.Slots {
		data, err := slot.Slot.Write()
		if err != nil {
			return fmt.Errorf("failed to write slot %d: %v", i, err)
		}
		hash := md5.Sum(data)
		if !bytes.Equal(hash[:], slot.Checksum[:]) {
			return fmt.Errorf("checksum mismatch for slot %d", i)
		}
	}

	// Validate UserData10
	ud10Data, _ := s.UserData10.Write()
	hash10 := md5.Sum(ud10Data)
	if !bytes.Equal(hash10[:], s.UserData10.Checksum[:]) {
		return fmt.Errorf("checksum mismatch for UserData10")
	}

	return nil
}

// UpdateChecksums recalculates all MD5 hashes
func (s *PCSave) UpdateChecksums() error {
	for i := range s.Slots {
		data, err := s.Slots[i].Slot.Write()
		if err != nil {
			return err
		}
		s.Slots[i].Checksum = md5.Sum(data)
	}

	ud10Data, _ := s.UserData10.Write()
	s.UserData10.Checksum = md5.Sum(ud10Data)

	ud11Data, _ := s.UserData11.Write()
	s.UserData11.Checksum = md5.Sum(ud11Data)

	return nil
}

func (s *SaveSlot) Write() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, s); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (u *UserData10) Write() ([]byte, error) {
	buf := new(bytes.Buffer)
	// Skip the first 16 bytes (checksum) when calculating hash
	if err := binary.Write(buf, binary.LittleEndian, u); err != nil {
		return nil, err
	}
	return buf.Bytes()[16:], nil
}

func (u *UserData11) Write() ([]byte, error) {
	buf := new(bytes.Buffer)
	// Skip the first 16 bytes (checksum) when calculating hash
	if err := binary.Write(buf, binary.LittleEndian, u); err != nil {
		return nil, err
	}
	return buf.Bytes()[16:], nil
}
