package saveengine

import (
	"bytes"
	"fmt"
)

// PC container layout used by the first stage. A PC save is a raw BND4 archive
// of twelve entries: ten character slots, UserData10 and UserData11. Each slot
// and UserData10 are stored as an MD5 prefix followed by fixed-size data, so
// their block sizes are constant and only UserData11 varies in length.
const (
	pcContainerFormat = "bnd4"

	pcHeaderSize          = 0x300
	pcEntryCountOffset    = 0x0C
	pcEntryCount          = 12
	pcSlotCount           = 10
	pcSlotBlockSize       = 0x280010 // 0x10 MD5 + 0x280000 slot data
	pcUserData10BlockSize = 0x60010  // 0x10 MD5 + 0x60000 UserData10 data

	pcSlotsOffset      = int64(pcHeaderSize)
	pcSlotsSize        = int64(pcSlotCount) * pcSlotBlockSize
	pcUserData10Offset = pcSlotsOffset + pcSlotsSize

	// pcUserData10DataOffset is where the UserData10 data itself starts: the PC
	// container stores it behind a 0x10-byte MD5 prefix, which is never parsed.
	pcUserData10DataOffset = pcUserData10Offset + 0x10

	// pcSlotDataOffset is where the data of the first character slot starts: like
	// UserData10, every PC slot sits behind a 0x10-byte MD5 prefix, which is never
	// parsed. The following slots repeat with pcSlotBlockSize.
	pcSlotDataOffset = pcSlotsOffset + 0x10
)

// pcMagic marks a raw, unencrypted PC container. An AES-encrypted container
// does not carry it and is never decrypted to find out what it holds.
var pcMagic = []byte("BND4")

// pcRecognises reports whether the leading bytes of a file are a PC container.
func pcRecognises(head []byte) bool {
	return bytes.HasPrefix(head, pcMagic)
}

// pcValidate checks only the structure the first stage depends on: the header,
// the declared BND4 entry count and the bounds of the ten slots and UserData10.
// It parses no slot content, no SteamID, no UserData11 and no MD5.
func pcValidate(source *codec) error {
	if !source.covers(0, pcHeaderSize) {
		return fmt.Errorf("PC container is too small for its 0x%X-byte header (0x%X bytes)",
			pcHeaderSize, source.length())
	}
	entries, err := source.uint32At(pcEntryCountOffset)
	if err != nil {
		return err
	}
	if entries != pcEntryCount {
		return fmt.Errorf("PC container declares %d BND4 entries, want %d", entries, pcEntryCount)
	}
	if !source.covers(pcSlotsOffset, pcSlotsSize) {
		return fmt.Errorf("PC container is too small for %d slots (0x%X bytes)",
			pcSlotCount, source.length())
	}
	if !source.covers(pcUserData10Offset, pcUserData10BlockSize) {
		return fmt.Errorf("PC container is too small for UserData10 (0x%X bytes)", source.length())
	}
	return nil
}
