package core

import "testing"

// makeSlotWithOffsets returns a SaveSlot with a minimal Data buffer and the given offsets.
func makeSlotWithOffsets(magic, invEnd, playerData, faceData, storageBox, eventFlags int) *SaveSlot {
	return &SaveSlot{
		Data:              make([]byte, SlotSize),
		MagicOffset:       magic,
		InventoryEnd:      invEnd,
		PlayerDataOffset:  playerData,
		FaceDataOffset:    faceData,
		StorageBoxOffset:  storageBox,
		EventFlagsOffset:  eventFlags,
		IngameTimerOffset: 0,
	}
}

func TestValidateOffsetChain_Valid(t *testing.T) {
	s := makeSlotWithOffsets(
		0x10000,  // MagicOffset
		0x5000,   // InventoryEnd  (< MagicOffset)
		0x20000,  // PlayerData    (> MagicOffset)
		0x30000,  // FaceData      (> PlayerData)
		0x30000,  // StorageBox    (== FaceData, storage starts at face data end)
		0x100000, // EventFlags    (within SlotSize)
	)
	if err := s.validateOffsetChain(); err != nil {
		t.Fatalf("expected valid chain, got error: %v", err)
	}
	if len(s.Warnings) != 0 {
		t.Fatalf("expected 0 warnings, got %d: %v", len(s.Warnings), s.Warnings)
	}
}

func TestValidateOffsetChain_NonMonotonic(t *testing.T) {
	// FaceData < PlayerData → violates monotonicity
	s := makeSlotWithOffsets(
		0x10000, // MagicOffset
		0x5000,  // InventoryEnd
		0x30000, // PlayerData
		0x20000, // FaceData (before PlayerData — invalid)
		0x40000, // StorageBox
		0,
	)
	if err := s.validateOffsetChain(); err == nil {
		t.Fatal("expected error for non-monotonic offset chain")
	}
}

func TestValidateOffsetChain_MagicTooSmall(t *testing.T) {
	// MagicOffset below MinMagicOffset
	s := makeSlotWithOffsets(
		100,      // MagicOffset < MinMagicOffset(400)
		0x20,     // InventoryEnd
		0x20000,  // PlayerData
		0x30000,  // FaceData
		0x40000,  // StorageBox
		0x100000, // EventFlags
	)
	if err := s.validateOffsetChain(); err == nil {
		t.Fatal("expected error for MagicOffset below MinMagicOffset")
	}
}

func TestValidateOffsetChain_InventoryEndAboveMagic(t *testing.T) {
	// InventoryEnd >= MagicOffset → out of range
	s := makeSlotWithOffsets(
		0x10000, // MagicOffset
		0x10000, // InventoryEnd == MagicOffset (must be <)
		0x20000, // PlayerData
		0x30000, // FaceData
		0x40000, // StorageBox
		0,
	)
	if err := s.validateOffsetChain(); err == nil {
		t.Fatal("expected error for InventoryEnd >= MagicOffset")
	}
}

func TestValidateOffsetChain_EventFlagsExceedsSlotSize(t *testing.T) {
	// EventFlagsOffset >= SlotSize → clamped to 0 with warning
	s := makeSlotWithOffsets(
		0x10000,  // MagicOffset
		0x5000,   // InventoryEnd
		0x20000,  // PlayerData
		0x30000,  // FaceData
		0x40000,  // StorageBox
		SlotSize, // EventFlags == SlotSize (out of bounds)
	)
	if err := s.validateOffsetChain(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.EventFlagsOffset != 0 {
		t.Fatalf("expected EventFlagsOffset clamped to 0, got 0x%X", s.EventFlagsOffset)
	}
	if len(s.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(s.Warnings))
	}
}

func TestValidateOffsetChain_StorageBoxExceedsSlotSize(t *testing.T) {
	// StorageBoxOffset >= SlotSize → out of range error
	s := makeSlotWithOffsets(
		0x10000, // MagicOffset
		0x5000,  // InventoryEnd
		0x20000, // PlayerData
		0x30000, // FaceData
		SlotSize, // StorageBox == SlotSize (out of bounds)
		0,
	)
	if err := s.validateOffsetChain(); err == nil {
		t.Fatal("expected error for StorageBoxOffset >= SlotSize")
	}
}
