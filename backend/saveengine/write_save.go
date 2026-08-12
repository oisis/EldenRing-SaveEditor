package saveengine

import (
	"crypto/md5"
	"errors"
	"fmt"
)

// serializeContainer prepares an independent candidate for a future WriteSave.
// It performs no I/O and never changes the session snapshot.
//
// PC containers store an MD5 prefix before each of the ten character slots and
// before UserData10. SaveForge 1.5.8 and 1.6.8 recalculated all eleven prefixes
// on every save, including prefixes that had previously been all zero. PS4
// containers have no corresponding checksum layer, so their candidate is an
// unchanged copy.
func serializeContainer(loaded *loadedSave) ([]byte, error) {
	if loaded == nil || loaded.session == nil || loaded.snapshot == nil {
		return nil, errors.New("loaded save is incomplete")
	}

	candidate := &codec{data: append([]byte(nil), loaded.snapshot.data...)}
	switch loaded.session.platform {
	case PlatformPC:
		for slot := 0; slot < pcSlotCount; slot++ {
			checksumAt := pcSlotsOffset + int64(slot)*pcSlotBlockSize
			if err := writeMD5Prefix(candidate, checksumAt, pcSlotBlockSize-int64(md5.Size)); err != nil {
				return nil, fmt.Errorf("cannot checksum PC slot %d: %w", slot, err)
			}
		}
		if err := writeMD5Prefix(
			candidate,
			pcUserData10Offset,
			pcUserData10BlockSize-int64(md5.Size),
		); err != nil {
			return nil, fmt.Errorf("cannot checksum PC UserData10: %w", err)
		}
	case PlatformPS4:
		// PS4 has no per-entry MD5 prefixes.
	default:
		return nil, fmt.Errorf("cannot serialize unknown save platform %q", loaded.session.platform)
	}
	return candidate.data, nil
}

// writeMD5Prefix replaces the 16 bytes at checksumAt with the MD5 of the data
// immediately following them. The complete prefix-and-data block is checked
// before any byte is changed.
func writeMD5Prefix(target *codec, checksumAt int64, dataLength int64) error {
	blockLength := int64(md5.Size) + dataLength
	if !target.covers(checksumAt, blockLength) {
		return fmt.Errorf(
			"checksum block [0x%X, 0x%X) is outside the snapshot (0x%X bytes)",
			checksumAt,
			checksumAt+blockLength,
			target.length(),
		)
	}
	dataAt := checksumAt + int64(md5.Size)
	checksum := md5.Sum(target.data[dataAt : dataAt+dataLength])
	copy(target.data[checksumAt:dataAt], checksum[:])
	return nil
}
