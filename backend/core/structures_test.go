package core

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestPlayerGameDataMapping(t *testing.T) {
	// Create a dummy SaveSlot
	slot := &SaveSlot{}
	
	// Check size
	size := binary.Size(&PlayerGameData{})
	t.Logf("PlayerGameData size: %d (0x%x)", size, size)
	
	// Create dummy PlayerGameData
	pgd := &PlayerGameData{
		Level:  150,
		Vigor:  60,
		Mind:   30,
		Souls:  1000000,
	}
	copy(pgd.CharacterName[:], []byte{0x47, 0x00, 0x65, 0x00, 0x6d, 0x00, 0x69, 0x00, 0x6e, 0x00, 0x69, 0x00}) // "Gemini" in UTF-16LE

	// Write to slot
	err := slot.SetPlayerGameData(pgd)
	if err != nil {
		t.Fatalf("Failed to set PlayerGameData: %v", err)
	}

	// Read back from slot
	readPgd, err := slot.GetPlayerGameData()
	if err != nil {
		t.Fatalf("Failed to get PlayerGameData: %v", err)
	}

	// Verify values
	if readPgd.Level != 150 {
		t.Errorf("Expected level 150, got %d", readPgd.Level)
	}
	if readPgd.Vigor != 60 {
		t.Errorf("Expected vigor 60, got %d", readPgd.Vigor)
	}
	if readPgd.Souls != 1000000 {
		t.Errorf("Expected souls 1000000, got %d", readPgd.Souls)
	}
	if !bytes.Equal(readPgd.CharacterName[:12], pgd.CharacterName[:12]) {
		t.Errorf("Character name mismatch")
	}
}

func TestGaItemInventory(t *testing.T) {
	slot := &SaveSlot{}
	
	itemIDs := []uint32{100, 200, 300}
	added := slot.AddBulkItems(itemIDs)
	
	if added != 3 {
		t.Errorf("Expected 3 items added, got %d", added)
	}
	
	// Verify items are in GaItems
	foundCount := 0
	for _, item := range slot.GaItems {
		if item.Handle != 0 {
			foundCount++
		}
	}
	
	if foundCount != 3 {
		t.Errorf("Expected 3 items in inventory, found %d", foundCount)
	}
	
	// Test duplicate prevention
	addedAgain := slot.AddBulkItems(itemIDs)
	if addedAgain != 0 {
		t.Errorf("Expected 0 items added (duplicates), got %d", addedAgain)
	}
}
