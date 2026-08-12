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

// validateSerialized reloads one serialized candidate through the same
// platform recognition and structural validation as LoadSave, then validates
// every active slot through the Inventory, Storage and GaItem readers that own
// the only mutable save surface implemented by SaveForge 2.0 today.
//
// The reload is isolated in a private engine and session. It therefore mints no
// identity in the active session, changes no revision or dirty flag and performs
// no I/O. Inactive and residual slots are intentionally answered from their
// activity flag alone, exactly as the public readers answer them.
func validateSerialized(candidate []byte, expectedPlatform Platform) error {
	if expectedPlatform != PlatformPC && expectedPlatform != PlatformPS4 {
		return fmt.Errorf("cannot validate unknown expected platform %q", expectedPlatform)
	}

	source := &codec{data: candidate}
	head, err := source.readAt(0, magicLength)
	if err != nil {
		return errors.New("serialized candidate is too short to identify its platform")
	}

	var platform Platform
	var format string
	switch {
	case pcRecognises(head):
		platform, format = PlatformPC, pcContainerFormat
	case ps4Recognises(head):
		platform, format = PlatformPS4, ps4ContainerFormat
	default:
		return errors.New("serialized candidate is neither a native PC nor a native PS4 container")
	}
	if platform != expectedPlatform {
		return fmt.Errorf("serialized candidate is a %s save, expected %s", platform, expectedPlatform)
	}

	switch platform {
	case PlatformPC:
		err = pcValidate(source)
	case PlatformPS4:
		err = ps4Validate(source)
	}
	if err != nil {
		return fmt.Errorf("serialized %s container is invalid: %w", platform, err)
	}

	validationSession := &Session{
		id:             "write-validation",
		platform:       platform,
		format:         format,
		ownedByLocator: make(map[ownedItemLocator]string),
		ownedByID:      make(map[string]ownedItemLocator),
	}
	loaded := &loadedSave{session: validationSession, snapshot: source}
	validationEngine := &Engine{sessions: map[string]*loadedSave{validationSession.id: loaded}}

	// The private readers mutate only the temporary identity registry and require
	// their owning engine lock. This engine is local and unreachable elsewhere,
	// but taking its lock preserves the same calling contract as the live path.
	validationEngine.mutex.Lock()
	defer validationEngine.mutex.Unlock()

	flags, err := source.readAt(
		userData10Base(platform)+userData10ActiveFlagsOffset,
		characterSlotCount,
	)
	if err != nil {
		return fmt.Errorf("cannot reload character slot activity: %w", err)
	}
	for characterID, flag := range flags {
		if flag != userData10ActiveFlagValue {
			continue
		}

		byHandle, err := readGaItemMap(source, platform, characterID)
		if err != nil {
			return fmt.Errorf("cannot reload items of character %d: %w", characterID, err)
		}
		inventory, err := readInventoryRecords(loaded, characterID)
		if err != nil {
			return fmt.Errorf("cannot reload inventory of character %d: %w", characterID, err)
		}
		storage, err := readStorageRecords(loaded, characterID)
		if err != nil {
			return fmt.Errorf("cannot reload storage of character %d: %w", characterID, err)
		}

		for _, record := range inventory {
			if _, err := resolveGaItemHandle(byHandle, record.GaItemHandle); err != nil {
				return fmt.Errorf(
					"character %d inventory record %s/%d: %w",
					characterID,
					record.ContainerSection,
					record.PhysicalIndex,
					err,
				)
			}
		}
		for _, record := range storage {
			if _, err := resolveGaItemHandle(byHandle, record.GaItemHandle); err != nil {
				return fmt.Errorf(
					"character %d storage record %s/%d: %w",
					characterID,
					record.ContainerSection,
					record.PhysicalIndex,
					err,
				)
			}
		}
	}
	return nil
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
