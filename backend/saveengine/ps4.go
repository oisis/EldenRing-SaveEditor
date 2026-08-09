package saveengine

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// PS4 container layout used by the first stage. A PS4 save starts with a fixed
// 0x70-byte header holding twelve entry descriptors, followed by ten slots and
// UserData10 stored without the MD5 prefix the PC container uses.
const (
	ps4ContainerFormat = "ps4-container"

	ps4HeaderSize       = 0x70
	ps4EntryTableOffset = 0x10
	ps4EntryCount       = 12
	ps4EntryStride      = 8
	ps4FirstEntryIndex  = 7
	ps4EntryMarker      = 0x7F7F7F7F
	ps4SlotCount        = 10
	ps4SlotSize         = 0x280000
	ps4UserData10Size   = 0x60000

	ps4SlotsOffset      = int64(ps4HeaderSize)
	ps4SlotsSize        = int64(ps4SlotCount) * ps4SlotSize
	ps4UserData10Offset = ps4SlotsOffset + ps4SlotsSize

	// ps4UserData10DataOffset is where the UserData10 data itself starts. The PS4
	// container stores it without the MD5 prefix the PC container uses, so the
	// data begins at the block offset.
	ps4UserData10DataOffset = ps4UserData10Offset

	// ps4SlotDataOffset is where the data of the first character slot starts. The
	// PS4 container stores slots without the MD5 prefix the PC container uses, so
	// the data begins at the block offset. The following slots repeat with
	// ps4SlotSize.
	ps4SlotDataOffset = ps4SlotsOffset
)

// ps4Magic marks a raw PS4 container.
var ps4Magic = []byte{0xCB, 0x01, 0x9C, 0x2C}

// ps4Recognises reports whether the leading bytes of a file are a PS4 container.
func ps4Recognises(head []byte) bool {
	return bytes.HasPrefix(head, ps4Magic)
}

// ps4Validate checks only the structure the first stage depends on: the fixed
// header with its twelve entry descriptors and the bounds of the ten slots and
// UserData10. It parses no slot content, no SteamID, no UserData11 and no MD5.
//
// The header is validated structurally — consecutive entry indices 0x07..0x12,
// each followed by the 0x7F7F7F7F marker — instead of by byte equality against
// one captured header, so a native save is never rejected over a field this
// stage does not interpret.
func ps4Validate(source *codec) error {
	header, err := source.readAt(0, ps4HeaderSize)
	if err != nil {
		return fmt.Errorf("PS4 container is too small for its 0x%X-byte header (0x%X bytes)",
			ps4HeaderSize, source.length())
	}
	for entry := 0; entry < ps4EntryCount; entry++ {
		offset := ps4EntryTableOffset + entry*ps4EntryStride
		index := binary.LittleEndian.Uint32(header[offset:])
		marker := binary.LittleEndian.Uint32(header[offset+4:])
		if index != uint32(ps4FirstEntryIndex+entry) || marker != ps4EntryMarker {
			return fmt.Errorf("PS4 header entry %d is 0x%08X/0x%08X, want 0x%08X/0x%08X",
				entry, index, marker, ps4FirstEntryIndex+entry, ps4EntryMarker)
		}
	}
	if !source.covers(ps4SlotsOffset, ps4SlotsSize) {
		return fmt.Errorf("PS4 container is too small for %d slots (0x%X bytes)",
			ps4SlotCount, source.length())
	}
	if !source.covers(ps4UserData10Offset, ps4UserData10Size) {
		return fmt.Errorf("PS4 container is too small for UserData10 (0x%X bytes)", source.length())
	}
	return nil
}
