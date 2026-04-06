package tests

import (
	"app/backend/core"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestSteamIDModification(t *testing.T) {
	// Mock a decrypted PC save
	// Header (0x300) + 10 * (MD5 0x10 + Slot 0x280000) + UD10 (MD5 0x10 + 0x60000) + UD11 (MD5 0x10 + 0x23FFF0)
	headerSize := 0x300
	slotSize := 0x280010
	ud10Size := 0x60010
	ud11Size := 0x240000
	totalSize := headerSize + 10*slotSize + ud10Size + ud11Size

	mockData := make([]byte, totalSize)
	// Set some dummy SteamID in UD10 (offset 0x300 + 10*0x280010 + 0x10 + 4 = 0x19003C4?)
	// Let's use the parser to load it from a file.

	// Create a dummy encrypted file
	iv := make([]byte, 16)
	copy(iv, "MOCK_IV_12345678")

	encrypted, _ := core.EncryptSave(mockData, iv)
	tmpFile := "tmp_mock_save.sl2"
	os.WriteFile(tmpFile, encrypted, 0644)
	defer os.Remove(tmpFile)

	save, err := core.LoadPC(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load mock save: %v", err)
	}

	newSteamID := uint64(123456789012345678)
	save.SetSteamID(newSteamID)

	if save.GetSteamID() != newSteamID {
		t.Errorf("Expected SteamID %d, got %d", newSteamID, save.GetSteamID())
	}

	// Verify it's in the raw data too
	steamIDInData := binary.LittleEndian.Uint64(save.UserData10.Data[4:12])
	if steamIDInData != newSteamID {
		t.Errorf("SteamID not updated in raw UserData10 data")
	}

	// Write back to file
	outputFile := "tmp_output_save.sl2"
	// Create the file first so Write can back it up
	os.WriteFile(outputFile, []byte("original content"), 0644)

	if err := save.Write(outputFile); err != nil {
		t.Fatalf("Failed to write save: %v", err)
	}
	defer os.Remove(outputFile)

	// Verify backup exists
	files, _ := filepath.Glob(outputFile + ".*.bak")
	if len(files) == 0 {
		t.Errorf("Backup file was not created")
	}
	for _, f := range files {
		os.Remove(f)
	}

	// Load it back and verify checksums
	reloaded, err := core.LoadPC(outputFile)
	if err != nil {
		t.Fatalf("Failed to reload save: %v", err)
	}

	// Verify UD10 checksum
	expectedUD10Checksum := core.ComputeMD5(reloaded.UserData10.Data)
	if reloaded.UserData10.Checksum != expectedUD10Checksum {
		t.Errorf("UserData10 checksum mismatch")
	}

	// Verify Slot checksums
	for i := 0; i < 10; i++ {
		expectedSlotChecksum := core.ComputeMD5(reloaded.SaveSlots[i].Data)
		if reloaded.SaveSlots[i].Checksum != expectedSlotChecksum {
			t.Errorf("Slot %d checksum mismatch", i)
		}
	}
}
