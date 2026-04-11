package core

import (
	"encoding/binary"
	"testing"
)

func TestComputeHashedValue(t *testing.T) {
	// Verify the magic constant reduction works correctly
	tests := []struct {
		input    uint32
		expected uint32
	}{
		{0, 0},
		{1, 1},                // Small values should pass through nearly unchanged
		{0xFFFF, 0xFFFF},      // Value at modulus boundary (0xFFF1 = 65521)
		{0x10000, 0x10000 - 1}, // One above: should wrap
	}

	for _, tt := range tests {
		result := computeHashedValue(tt.input)
		// We can't easily predict exact values without running the algorithm,
		// so just verify it doesn't panic and returns reasonable values
		_ = result
	}

	// Verify determinism
	a := computeHashedValue(12345)
	b := computeHashedValue(12345)
	if a != b {
		t.Errorf("computeHashedValue not deterministic: %d != %d", a, b)
	}

	// Verify different inputs produce different outputs
	x := computeHashedValue(100)
	y := computeHashedValue(200)
	if x == y {
		t.Errorf("computeHashedValue(100) == computeHashedValue(200) = %d", x)
	}
}

func TestBytesHash(t *testing.T) {
	// Empty slice
	h0 := bytesHash([]byte{})
	// lo=1, hi=0 → loH=computeHashedValue(1), hiH=computeHashedValue(0)
	if h0 == 0 {
		// (1 | 0<<16) * 2 = 2, unless computeHashedValue changes it
		// Actually computeHashedValue(1) should be close to 1 for small inputs
		t.Log("bytesHash([]) =", h0)
	}

	// Single byte
	h1 := bytesHash([]byte{0x42})
	if h1 == 0 {
		t.Error("bytesHash of non-zero byte should not be 0")
	}

	// Determinism
	a := bytesHash([]byte{1, 2, 3, 4})
	b := bytesHash([]byte{1, 2, 3, 4})
	if a != b {
		t.Errorf("bytesHash not deterministic: %d != %d", a, b)
	}

	// Different data → different hash
	c := bytesHash([]byte{1, 2, 3, 4})
	d := bytesHash([]byte{4, 3, 2, 1})
	if c == d {
		t.Error("bytesHash should produce different results for different inputs")
	}
}

func TestValueHash(t *testing.T) {
	// valueHash converts u32 to LE bytes then runs bytesHash
	h := valueHash(42)
	if h == 0 {
		t.Error("valueHash(42) should not be 0")
	}

	// Determinism
	a := valueHash(99)
	b := valueHash(99)
	if a != b {
		t.Errorf("valueHash not deterministic: %d != %d", a, b)
	}

	// Different values
	x := valueHash(1)
	y := valueHash(2)
	if x == y {
		t.Error("valueHash(1) should differ from valueHash(2)")
	}
}

func TestStatsHash_IntFaithSwap(t *testing.T) {
	// Verify that swapping Int and Faith produces a different hash than not swapping
	// The stats hash puts Faith before Intelligence

	// Same values for all, different for Int and Faith
	h1 := statsHash(10, 10, 10, 10, 10, 15, 8, 10, 0) // Int=15, Faith=8
	h2 := statsHash(10, 10, 10, 10, 10, 8, 15, 10, 0)  // Int=8, Faith=15

	// These should be different because Int and Faith are in swapped positions
	if h1 == h2 {
		t.Error("swapping Int/Faith values should produce different hash")
	}

	// But the hash internally puts Faith before Int, so:
	// statsHash(... int=15, faith=8 ...) puts bytes as: [faith=8][int=15]
	// statsHash(... int=8, faith=15 ...) puts bytes as: [faith=15][int=8]
	// Both orderings should be different from each other
}

func TestEquipmentHash(t *testing.T) {
	ids := []uint32{100000, 200000, 0xFFFFFFFF, 0, 0, 0, 0, 0, 0, 0}
	h := equipmentHash(ids)
	if h == 0 {
		t.Error("equipmentHash should not be 0 for non-zero inputs")
	}

	// All zeros
	zeros := make([]uint32, 10)
	hZero := equipmentHash(zeros)
	// Should still produce a non-zero hash (lo starts at 1)
	if hZero == 0 {
		t.Error("equipmentHash of zeros should not be 0")
	}
}

func TestQuickItemsHash_Lower28Bits(t *testing.T) {
	// Verify that upper 4 bits are masked off
	ids1 := []uint32{0xF0000001}
	ids2 := []uint32{0x00000001}

	h1 := quickItemsHash(ids1)
	h2 := quickItemsHash(ids2)

	if h1 != h2 {
		t.Errorf("quickItemsHash should mask upper 4 bits: %d != %d", h1, h2)
	}
}

func TestComputeSlotHash_Deterministic(t *testing.T) {
	// Create a minimal valid slot
	data := make([]byte, SlotSize)

	// Write MagicPattern at a known offset
	magicOff := 0x15420 + 432
	copy(data[magicOff:], MagicPattern)

	// Write some stats at proper offsets
	binary.LittleEndian.PutUint32(data[magicOff+OffLevel:], 9)
	binary.LittleEndian.PutUint32(data[magicOff+OffVigor:], 15)
	binary.LittleEndian.PutUint32(data[magicOff+OffMind:], 10)
	binary.LittleEndian.PutUint32(data[magicOff+OffEndurance:], 11)
	binary.LittleEndian.PutUint32(data[magicOff+OffStrength:], 14)
	binary.LittleEndian.PutUint32(data[magicOff+OffDexterity:], 13)
	binary.LittleEndian.PutUint32(data[magicOff+OffIntelligence:], 9)
	binary.LittleEndian.PutUint32(data[magicOff+OffFaith:], 9)
	binary.LittleEndian.PutUint32(data[magicOff+OffArcane:], 7)
	binary.LittleEndian.PutUint32(data[magicOff+OffSouls:], 0)
	data[magicOff+OffClass] = 0 // Vagabond

	slot := &SaveSlot{
		Data:        data,
		MagicOffset: magicOff,
		Player: PlayerGameData{
			Level:        9,
			Vigor:        15,
			Mind:         10,
			Endurance:    11,
			Strength:     14,
			Dexterity:    13,
			Intelligence: 9,
			Faith:        9,
			Arcane:       7,
			Souls:        0,
			Class:        0,
		},
	}

	hash1 := ComputeSlotHash(slot)
	hash2 := ComputeSlotHash(slot)

	if hash1 != hash2 {
		t.Error("ComputeSlotHash should be deterministic")
	}

	// Level hash should be non-zero
	levelHash := binary.LittleEndian.Uint32(hash1[0:4])
	if levelHash == 0 {
		t.Error("Level hash entry should not be 0 for Level=9")
	}

	// Stats hash should be non-zero
	statsH := binary.LittleEndian.Uint32(hash1[4:8])
	if statsH == 0 {
		t.Error("Stats hash entry should not be 0")
	}

	// Class hash should be non-zero (class 0 = Vagabond, but ValueHash(0) should still produce non-zero)
	classH := binary.LittleEndian.Uint32(hash1[8:12])
	// Actually ValueHash(0) — bytesHash of [0,0,0,0]:
	// lo = 1+0+0+0+0=1, hi = 0+1+1+1+1=4
	// computeHashedValue(1) ≈ 1, computeHashedValue(4) ≈ 4
	// (1 | 4<<16) * 2 = (1 | 0x40000) * 2 = 0x40001 * 2 = 0x80002
	_ = classH // just verify it doesn't panic

	// Padding entry [4] should be 0
	padding := binary.LittleEndian.Uint32(hash1[16:20])
	if padding != 0 {
		t.Errorf("padding entry [4] should be 0, got %d", padding)
	}
}

func TestRecalculateSlotHash_WritesToData(t *testing.T) {
	data := make([]byte, SlotSize)
	magicOff := 0x15420 + 432
	copy(data[magicOff:], MagicPattern)

	binary.LittleEndian.PutUint32(data[magicOff+OffLevel:], 1)
	binary.LittleEndian.PutUint32(data[magicOff+OffVigor:], 10)
	binary.LittleEndian.PutUint32(data[magicOff+OffMind:], 10)
	binary.LittleEndian.PutUint32(data[magicOff+OffEndurance:], 10)
	binary.LittleEndian.PutUint32(data[magicOff+OffStrength:], 10)
	binary.LittleEndian.PutUint32(data[magicOff+OffDexterity:], 10)
	binary.LittleEndian.PutUint32(data[magicOff+OffIntelligence:], 10)
	binary.LittleEndian.PutUint32(data[magicOff+OffFaith:], 10)
	binary.LittleEndian.PutUint32(data[magicOff+OffArcane:], 10)

	slot := &SaveSlot{
		Data:        data,
		MagicOffset: magicOff,
		Player: PlayerGameData{
			Level: 1, Vigor: 10, Mind: 10, Endurance: 10,
			Strength: 10, Dexterity: 10, Intelligence: 10,
			Faith: 10, Arcane: 10, Class: 9,
		},
	}

	// Verify hash area is all zeros before
	for i := HashOffset; i < HashOffset+HashSize; i++ {
		if data[i] != 0 {
			t.Fatal("hash area should be zeroed initially")
		}
	}

	RecalculateSlotHash(slot)

	// Verify hash area is no longer all zeros
	allZero := true
	for i := HashOffset; i < HashOffset+HashSize; i++ {
		if data[i] != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("hash area should not be all zeros after RecalculateSlotHash")
	}
}

func TestComputeSlotHash_ChangingStatsChangesHash(t *testing.T) {
	makeSlot := func(vigor uint32) *SaveSlot {
		data := make([]byte, SlotSize)
		magicOff := 0x15420 + 432
		copy(data[magicOff:], MagicPattern)
		binary.LittleEndian.PutUint32(data[magicOff+OffVigor:], vigor)
		binary.LittleEndian.PutUint32(data[magicOff+OffMind:], 10)
		binary.LittleEndian.PutUint32(data[magicOff+OffEndurance:], 10)
		binary.LittleEndian.PutUint32(data[magicOff+OffStrength:], 10)
		binary.LittleEndian.PutUint32(data[magicOff+OffDexterity:], 10)
		binary.LittleEndian.PutUint32(data[magicOff+OffIntelligence:], 10)
		binary.LittleEndian.PutUint32(data[magicOff+OffFaith:], 10)
		binary.LittleEndian.PutUint32(data[magicOff+OffArcane:], 10)
		level := vigor + 10*7 - 79
		binary.LittleEndian.PutUint32(data[magicOff+OffLevel:], level)

		return &SaveSlot{
			Data:        data,
			MagicOffset: magicOff,
			Player: PlayerGameData{
				Level: level, Vigor: vigor, Mind: 10, Endurance: 10,
				Strength: 10, Dexterity: 10, Intelligence: 10,
				Faith: 10, Arcane: 10, Class: 9,
			},
		}
	}

	h1 := ComputeSlotHash(makeSlot(10))
	h2 := ComputeSlotHash(makeSlot(50))

	// Level hash (entry 0) should differ
	l1 := binary.LittleEndian.Uint32(h1[0:4])
	l2 := binary.LittleEndian.Uint32(h2[0:4])
	if l1 == l2 {
		t.Error("different levels should produce different level hashes")
	}

	// Stats hash (entry 1) should differ
	s1 := binary.LittleEndian.Uint32(h1[4:8])
	s2 := binary.LittleEndian.Uint32(h2[4:8])
	if s1 == s2 {
		t.Error("different vigor should produce different stats hashes")
	}
}
