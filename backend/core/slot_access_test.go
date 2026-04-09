package core

import (
	"encoding/binary"
	"testing"
)

func TestSlotAccessorReadU32OutOfBounds(t *testing.T) {
	sa := NewSlotAccessor(make([]byte, 10))
	_, err := sa.ReadU32(8) // needs 4 bytes at offset 8, but buffer is only 10
	if err == nil {
		t.Fatal("expected error for out-of-bounds read")
	}
}

func TestSlotAccessorReadU32Negative(t *testing.T) {
	sa := NewSlotAccessor(make([]byte, 100))
	_, err := sa.ReadU32(-1)
	if err == nil {
		t.Fatal("expected error for negative offset")
	}
}

func TestSlotAccessorReadU32Valid(t *testing.T) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint32(data[4:], 0xDEADBEEF)
	sa := NewSlotAccessor(data)
	val, err := sa.ReadU32(4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 0xDEADBEEF {
		t.Fatalf("expected 0xDEADBEEF, got 0x%X", val)
	}
}

func TestSlotAccessorReadU16OutOfBounds(t *testing.T) {
	sa := NewSlotAccessor(make([]byte, 5))
	_, err := sa.ReadU16(4) // needs 2 bytes at offset 4, but buffer is only 5
	if err == nil {
		t.Fatal("expected error for out-of-bounds read")
	}
}

func TestSlotAccessorReadU8OutOfBounds(t *testing.T) {
	sa := NewSlotAccessor(make([]byte, 5))
	_, err := sa.ReadU8(5)
	if err == nil {
		t.Fatal("expected error for out-of-bounds read")
	}
}

func TestSlotAccessorWriteU32OutOfBounds(t *testing.T) {
	sa := NewSlotAccessor(make([]byte, 6))
	err := sa.WriteU32(4, 42) // needs 4 bytes at offset 4, but buffer is only 6
	if err == nil {
		t.Fatal("expected error for out-of-bounds write")
	}
}

func TestSlotAccessorWriteU32Valid(t *testing.T) {
	data := make([]byte, 8)
	sa := NewSlotAccessor(data)
	if err := sa.WriteU32(0, 0xCAFEBABE); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := binary.LittleEndian.Uint32(data[0:])
	if got != 0xCAFEBABE {
		t.Fatalf("expected 0xCAFEBABE, got 0x%X", got)
	}
}

func TestSlotAccessorReadDynamicSizeClamp(t *testing.T) {
	data := make([]byte, 100)
	binary.LittleEndian.PutUint32(data[0:], 99999) // absurd value
	sa := NewSlotAccessor(data)
	size, err := sa.ReadDynamicSize(0, 256, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 0 {
		t.Fatalf("expected clamped to 0, got %d", size)
	}
	if len(sa.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(sa.Warnings))
	}
}

func TestSlotAccessorReadDynamicSizeValid(t *testing.T) {
	data := make([]byte, 100)
	binary.LittleEndian.PutUint32(data[0:], 128)
	sa := NewSlotAccessor(data)
	size, err := sa.ReadDynamicSize(0, 256, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 128 {
		t.Fatalf("expected 128, got %d", size)
	}
	if len(sa.Warnings) != 0 {
		t.Fatalf("expected 0 warnings, got %d", len(sa.Warnings))
	}
}

func TestSlotAccessorReadDynamicSizeOutOfBounds(t *testing.T) {
	sa := NewSlotAccessor(make([]byte, 2))
	_, err := sa.ReadDynamicSize(0, 256, "test")
	if err == nil {
		t.Fatal("expected error for out-of-bounds dynamic size read")
	}
}

func TestSlotAccessorCheckBounds(t *testing.T) {
	sa := NewSlotAccessor(make([]byte, 100))
	if err := sa.CheckBounds(0, 100, "full"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := sa.CheckBounds(50, 51, "overflow"); err == nil {
		t.Fatal("expected error for overflow bounds check")
	}
	if err := sa.CheckBounds(-1, 10, "negative"); err == nil {
		t.Fatal("expected error for negative offset bounds check")
	}
}

func TestSlotAccessorReadU64(t *testing.T) {
	data := make([]byte, 16)
	binary.LittleEndian.PutUint64(data[8:], 0x0102030405060708)
	sa := NewSlotAccessor(data)

	val, err := sa.ReadU64(8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 0x0102030405060708 {
		t.Fatalf("expected 0x0102030405060708, got 0x%X", val)
	}

	_, err = sa.ReadU64(10) // out of bounds
	if err == nil {
		t.Fatal("expected error for out-of-bounds ReadU64")
	}
}

func TestSlotAccessorWriteU64(t *testing.T) {
	data := make([]byte, 16)
	sa := NewSlotAccessor(data)
	if err := sa.WriteU64(0, 0xAABBCCDDEEFF0011); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := binary.LittleEndian.Uint64(data[0:])
	if got != 0xAABBCCDDEEFF0011 {
		t.Fatalf("expected 0xAABBCCDDEEFF0011, got 0x%X", got)
	}

	err := sa.WriteU64(10, 0) // out of bounds
	if err == nil {
		t.Fatal("expected error for out-of-bounds WriteU64")
	}
}
