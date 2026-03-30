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
	if err := save.Write(tmpPath); err != nil {
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
	if err := save.Write(tmpPath); err != nil {
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
