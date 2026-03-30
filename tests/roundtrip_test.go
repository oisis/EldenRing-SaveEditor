package tests

import (
	"bytes"
	"github.com/oisis/EldenRing-SaveEditor/backend/core"
	"os"
	"testing"
)

func TestRoundTripPS4(t *testing.T) {
	// Use the provided test save file
	path := "../tmp/save/oisis_pl-org.txt"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("Test save file not found in tmp/save/")
	}

	// 1. Load original
	save, err := core.LoadSave(path)
	if err != nil {
		t.Fatalf("Failed to load save: %v", err)
	}

	if save.Platform != core.PlatformPS {
		t.Fatalf("Expected PS4 platform, got %s", save.Platform)
	}

	// 2. Write to a temporary file
	tmpPath := "data/ps4/roundtrip_test.dat"
	os.MkdirAll("data/ps4", 0755)
	if err := save.Write(tmpPath, save.Platform); err != nil {
		t.Fatalf("Failed to write save: %v", err)
	}
	defer os.Remove(tmpPath)

	// 3. Compare bytes bit-per-bit
	originalData, _ := os.ReadFile(path)
	newData, _ := os.ReadFile(tmpPath)

	if !bytes.Equal(originalData, newData) {
		t.Errorf("Byte mismatch! Round-trip failed to preserve data integrity.")
		
		// Find first mismatch for debugging
		for i := 0; i < len(originalData); i++ {
			if i >= len(newData) {
				t.Logf("New data is shorter than original (len: %d vs %d)", len(newData), len(originalData))
				break
			}
			if originalData[i] != newData[i] {
				t.Logf("First mismatch at offset 0x%x: expected %02x, got %02x", i, originalData[i], newData[i])
				break
			}
		}
	} else {
		t.Logf("Round-trip successful! Data integrity preserved.")
	}
}

func TestRoundTripPC(t *testing.T) {
	path := "../tmp/save/ER0000.sl2"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("Test save file not found in tmp/save/")
	}

	// 1. Load original
	save, err := core.LoadSave(path)
	if err != nil {
		t.Fatalf("Failed to load save: %v", err)
	}

	if save.Platform != core.PlatformPC {
		t.Fatalf("Expected PC platform, got %s", save.Platform)
	}

	// 2. Write to a temporary file
	tmpPath := "data/pc/roundtrip_test.sl2"
	os.MkdirAll("data/pc", 0755)
	if err := save.Write(tmpPath, save.Platform); err != nil {
		t.Fatalf("Failed to write save: %v", err)
	}
	defer os.Remove(tmpPath)

	// 3. Compare bytes bit-per-bit
	originalData, _ := os.ReadFile(path)
	newData, _ := os.ReadFile(tmpPath)

	if !bytes.Equal(originalData, newData) {
		t.Errorf("Byte mismatch! Round-trip failed to preserve data integrity.")
		
		// Find first mismatch for debugging
		for i := 0; i < len(originalData); i++ {
			if i >= len(newData) {
				t.Logf("New data is shorter than original (len: %d vs %d)", len(newData), len(originalData))
				break
			}
			if originalData[i] != newData[i] {
				t.Logf("First mismatch at offset 0x%x: expected %02x, got %02x", i, originalData[i], newData[i])
				break
			}
		}
	} else {
		t.Logf("Round-trip successful! Data integrity preserved.")
	}
}

func TestConversionPS4ToPC(t *testing.T) {
	path := "../tmp/save/oisis_pl-org.txt"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("Test save file not found in tmp/save/")
	}

	// 1. Load original PS4
	ps4Save, err := core.LoadSave(path)
	if err != nil {
		t.Fatalf("Failed to load PS4 save: %v", err)
	}

	// 2. Convert to PC
	tmpPath := "data/pc/conversion_test.sl2"
	os.MkdirAll("data/pc", 0755)
	if err := ps4Save.Write(tmpPath, core.PlatformPC); err != nil {
		t.Fatalf("Failed to write as PC: %v", err)
	}
	defer os.Remove(tmpPath)

	// 3. Load the new PC save
	pcSave, err := core.LoadSave(tmpPath)
	if err != nil {
		t.Fatalf("Failed to load converted PC save: %v", err)
	}

	if pcSave.Platform != core.PlatformPC {
		t.Errorf("Expected PC platform after conversion, got %s", pcSave.Platform)
	}

	// 4. Verify data preservation (e.g., Name of first active slot)
	for i := 0; i < 10; i++ {
		if ps4Save.ActiveSlots[i] {
			ps4Name := core.UTF16ToString(ps4Save.Slots[i].PlayerGameData.CharacterName[:])
			pcName := core.UTF16ToString(pcSave.Slots[i].PlayerGameData.CharacterName[:])
			if ps4Name != pcName {
				t.Errorf("Name mismatch after conversion at slot %d: expected %s, got %s", i, ps4Name, pcName)
			}
			if ps4Save.Slots[i].PlayerGameData.Level != pcSave.Slots[i].PlayerGameData.Level {
				t.Errorf("Level mismatch after conversion at slot %d", i)
			}
		}
	}
}

func TestConversionPCToPS4(t *testing.T) {
	path := "../tmp/save/ER0000.sl2"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("Test save file not found in tmp/save/")
	}

	// 1. Load original PC
	pcSave, err := core.LoadSave(path)
	if err != nil {
		t.Fatalf("Failed to load PC save: %v", err)
	}

	// 2. Convert to PS4
	tmpPath := "data/ps4/conversion_test.dat"
	os.MkdirAll("data/ps4", 0755)
	if err := pcSave.Write(tmpPath, core.PlatformPS); err != nil {
		t.Fatalf("Failed to write as PS4: %v", err)
	}
	defer os.Remove(tmpPath)

	// 3. Load the new PS4 save
	ps4Save, err := core.LoadSave(tmpPath)
	if err != nil {
		t.Fatalf("Failed to load converted PS4 save: %v", err)
	}

	if ps4Save.Platform != core.PlatformPS {
		t.Errorf("Expected PS4 platform after conversion, got %s", ps4Save.Platform)
	}

	// 4. Verify data preservation
	for i := 0; i < 10; i++ {
		if pcSave.ActiveSlots[i] {
			pcName := core.UTF16ToString(pcSave.Slots[i].PlayerGameData.CharacterName[:])
			ps4Name := core.UTF16ToString(ps4Save.Slots[i].PlayerGameData.CharacterName[:])
			if pcName != ps4Name {
				t.Errorf("Name mismatch after conversion at slot %d: expected %s, got %s", i, pcName, ps4Name)
			}
		}
	}
}
