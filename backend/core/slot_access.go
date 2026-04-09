package core

import (
	"encoding/binary"
	"fmt"
)

// SlotAccessor provides bounds-checked read/write access to a save slot's raw byte buffer.
// It collects non-fatal warnings (e.g. clamped dynamic sizes) separately from fatal errors.
type SlotAccessor struct {
	Data     []byte
	Warnings []string
}

func NewSlotAccessor(data []byte) *SlotAccessor {
	return &SlotAccessor{Data: data}
}

// ReadU32 reads a little-endian uint32 at the given offset with bounds checking.
func (sa *SlotAccessor) ReadU32(off int) (uint32, error) {
	if off < 0 || off+4 > len(sa.Data) {
		return 0, fmt.Errorf("ReadU32: offset %d (0x%X) out of bounds [0, %d)",
			off, off, len(sa.Data))
	}
	return binary.LittleEndian.Uint32(sa.Data[off:]), nil
}

// ReadU64 reads a little-endian uint64 at the given offset with bounds checking.
func (sa *SlotAccessor) ReadU64(off int) (uint64, error) {
	if off < 0 || off+8 > len(sa.Data) {
		return 0, fmt.Errorf("ReadU64: offset %d (0x%X) out of bounds [0, %d)",
			off, off, len(sa.Data))
	}
	return binary.LittleEndian.Uint64(sa.Data[off:]), nil
}

// ReadU16 reads a little-endian uint16 at the given offset with bounds checking.
func (sa *SlotAccessor) ReadU16(off int) (uint16, error) {
	if off < 0 || off+2 > len(sa.Data) {
		return 0, fmt.Errorf("ReadU16: offset %d (0x%X) out of bounds [0, %d)",
			off, off, len(sa.Data))
	}
	return binary.LittleEndian.Uint16(sa.Data[off:]), nil
}

// ReadU8 reads a single byte at the given offset with bounds checking.
func (sa *SlotAccessor) ReadU8(off int) (uint8, error) {
	if off < 0 || off >= len(sa.Data) {
		return 0, fmt.Errorf("ReadU8: offset %d (0x%X) out of bounds [0, %d)",
			off, off, len(sa.Data))
	}
	return sa.Data[off], nil
}

// WriteU32 writes a little-endian uint32 at the given offset with bounds checking.
func (sa *SlotAccessor) WriteU32(off int, val uint32) error {
	if off < 0 || off+4 > len(sa.Data) {
		return fmt.Errorf("WriteU32: offset %d (0x%X) out of bounds [0, %d)",
			off, off, len(sa.Data))
	}
	binary.LittleEndian.PutUint32(sa.Data[off:], val)
	return nil
}

// WriteU64 writes a little-endian uint64 at the given offset with bounds checking.
func (sa *SlotAccessor) WriteU64(off int, val uint64) error {
	if off < 0 || off+8 > len(sa.Data) {
		return fmt.Errorf("WriteU64: offset %d (0x%X) out of bounds [0, %d)",
			off, off, len(sa.Data))
	}
	binary.LittleEndian.PutUint64(sa.Data[off:], val)
	return nil
}

// WriteU16 writes a little-endian uint16 at the given offset with bounds checking.
func (sa *SlotAccessor) WriteU16(off int, val uint16) error {
	if off < 0 || off+2 > len(sa.Data) {
		return fmt.Errorf("WriteU16: offset %d (0x%X) out of bounds [0, %d)",
			off, off, len(sa.Data))
	}
	binary.LittleEndian.PutUint16(sa.Data[off:], val)
	return nil
}

// WriteU8 writes a single byte at the given offset with bounds checking.
func (sa *SlotAccessor) WriteU8(off int, val uint8) error {
	if off < 0 || off >= len(sa.Data) {
		return fmt.Errorf("WriteU8: offset %d (0x%X) out of bounds [0, %d)",
			off, off, len(sa.Data))
	}
	sa.Data[off] = val
	return nil
}

// ReadDynamicSize reads a uint32 size value from untrusted save data and clamps it
// to a sane maximum. Returns 0 (not error) when clamped, but appends a warning.
// This is the correct behavior for PS4 saves which often have garbage in size fields.
func (sa *SlotAccessor) ReadDynamicSize(off int, maxSize int, name string) (int, error) {
	raw, err := sa.ReadU32(off)
	if err != nil {
		return 0, fmt.Errorf("cannot read %s: %w", name, err)
	}
	size := int(raw)
	if size < 0 || size > maxSize {
		sa.Warnings = append(sa.Warnings,
			fmt.Sprintf("%s: raw value %d (0x%X) exceeds max %d, clamped to 0",
				name, size, size, maxSize))
		return 0, nil
	}
	return size, nil
}

// CheckBounds validates that a write of `size` bytes at `off` is safe.
func (sa *SlotAccessor) CheckBounds(off, size int, label string) error {
	if off < 0 || off+size > len(sa.Data) {
		return fmt.Errorf("%s: offset %d + size %d = %d exceeds buffer length %d",
			label, off, size, off+size, len(sa.Data))
	}
	return nil
}
