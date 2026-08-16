package saveengine

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
)

// PC layout of the account identifier. A PC save stores the same Steam identity
// twice: once globally in UserData10 and once inside every active character
// slot. SaveForge 1.5.8 updated the global copy alone, and 1.6.8 confirmed that
// the resulting disagreement makes the game treat the save as corrupt, so both
// copies are always written together here.
const (
	// accountIDSize is the width of both copies: one little-endian uint64.
	accountIDSize = 8

	// accountIDGlobalOffset is the position of the global copy, counted from the
	// start of the UserData10 data. The four bytes in front of it are metadata and
	// are never part of the identifier.
	accountIDGlobalOffset = 0x04

	// The slot copy has no fixed position. It lives in the trailing fixed block of
	// the slot, which follows the event-flag bitfield across five size-prefixed
	// world blocks, the player coordinates, the version-gated spawn fields and
	// NetMan. Every distance below is one step of that confirmed chain.
	//
	// playerCoordinatesSize is the real 61-byte block: coordinates (12),
	// map ID (4), angle (16), one game-manager byte, unknown coordinates (12) and
	// an unknown angle (16).
	playerCoordinatesSize = 12 + 4 + 16 + 1 + 12 + 16

	// spawnPointFixedSize covers the two padding bytes behind the coordinates
	// and the two always-present spawn identifiers. The two trailing spawn fields
	// exist only from the slot versions below, so the slot declares them itself.
	spawnPointFixedSize          = 2 + 4 + 4
	spawnPointTempVersion        = 65
	spawnPointGameManByteVersion = 66

	// netManSectionSize is the network-manager block: one uint32 and a 128 KB
	// opaque payload, neither of which is parsed here.
	netManSectionSize = 4 + 0x20000

	// accountIDTrailingOffset is the distance from the first byte of the trailing
	// fixed block to the identifier: WorldAreaWeather (12), WorldAreaTime (12) and
	// BaseVersion (16). The blocks behind it are never read.
	accountIDTrailingOffset = 12 + 12 + 16
)

// worldBlockLimits are the five size-prefixed blocks between the
// event-flag terminator and the player coordinates, with the confirmed ceiling
// of each: field area, world area, world geometry, its second block and the
// renderer block. A declared size outside its ceiling is treated as corrupt, so
// a slot is never walked from a guessed position.
var worldBlockLimits = [...]int64{0x10000, 0x10000, 0x100000, 0x100000, 0x100000}

// SetSaveAccountIDResult reports one committed account-identifier change. It
// carries no identifier: the value is private account data and never leaves the
// package, not in a result, an error or a log.
type SetSaveAccountIDResult struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
}

// SetSaveAccountID writes accountID into the global UserData10 copy and into the
// own copy of every active character slot of one PC session, as a single atomic
// plan with verification and rollback.
//
// accountID is the canonical decimal representation of a uint64: 0, or an
// unsigned number without leading zeros that fits into 64 bits. It is a string
// because JavaScript and JSON lose the precision of large identifiers. A
// rejected value is never echoed back.
//
// PC only. PS4 carries no confirmed Steam identity, so a PS4 session is rejected
// before any account-identifier field is read or written; no PSN or hypothetical
// PS4 identifier is invented for it.
//
// Nothing is persisted here. The mutation changes the session's private snapshot
// only; a later WriteSave serializes it, recalculates the PC MD5 prefixes and
// writes the file.
func (engine *Engine) SetSaveAccountID(
	saveSessionID string,
	accountID string,
	expectedRevision string,
) (SetSaveAccountIDResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetSaveAccountIDResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	account, err := strconv.ParseUint(accountID, 10, 64)
	if err != nil || strconv.FormatUint(account, 10) != accountID {
		return SetSaveAccountIDResult{}, errors.New(
			"accountID must be the canonical decimal representation of a uint64, " +
				"without a sign, prefix, padding or separator")
	}

	saveRevision, err := engine.commitRevision(saveSessionID, func(loaded *loadedSave) error {
		if loaded.session.platform != PlatformPC {
			return fmt.Errorf(
				"the account identifier is confirmed for PC saves only; this session is a %s save",
				loaded.session.platform)
		}
		current := loaded.session.revisionString()
		if expectedRevision != current {
			return fmt.Errorf(
				"expectedRevision %q does not match the current saveRevision %q",
				expectedRevision, current)
		}

		userDataBase := userData10Base(PlatformPC)
		flags, err := loaded.snapshot.readAt(
			userDataBase+userData10ActiveFlagsOffset, characterSlotCount)
		if err != nil {
			return fmt.Errorf("cannot read character slot activity: %w", err)
		}

		// Every target is resolved and range-checked before the first write, so an
		// unparseable active slot rejects the whole operation instead of leaving the
		// global copy and the slot copies disagreeing.
		value := make([]byte, accountIDSize)
		binary.LittleEndian.PutUint64(value, account)
		writes := []byteWrite{{
			at:   userDataBase + accountIDGlobalOffset,
			data: value,
		}}
		for characterID, flag := range flags {
			if flag != userData10ActiveFlagValue {
				continue
			}
			at, err := pcAccountIDFieldAt(loaded, characterID)
			if err != nil {
				return err
			}
			writes = append(writes, byteWrite{at: at, data: value})
		}
		if err := applyByteWrites(loaded.snapshot, writes); err != nil {
			return fmt.Errorf("cannot set the account identifier: %w", err)
		}
		return nil
	})
	if err != nil {
		return SetSaveAccountIDResult{}, err
	}
	return SetSaveAccountIDResult{SaveSessionID: saveSessionID, SaveRevision: saveRevision}, nil
}

// pcAccountIDFieldAt resolves the account identifier of one active PC slot by
// continuing the confirmed locator chain behind the event-flag bitfield. Every
// declared length is widened to int64 before it is added, and the field itself
// must lie completely inside the slot, so a corrupt slot is reported instead of
// producing a plausible-looking offset. The identifier is never found by
// scanning for its bytes and never assumed at a fixed distance from the end of
// the slot.
func pcAccountIDFieldAt(loaded *loadedSave, characterID int) (int64, error) {
	sectionAt, err := eventFlagSectionStart(loaded, characterID)
	if err != nil {
		return 0, err
	}
	slotBase, slotEnd := pcEventFlagSlotBounds(characterID)

	version, err := loaded.snapshot.uint32At(slotBase)
	if err != nil {
		return 0, fmt.Errorf("cannot read the slot version of character %d: %w", characterID, err)
	}
	if version == 0 {
		return 0, fmt.Errorf("character %d declares no slot version", characterID)
	}

	at := sectionAt + eventFlagSectionSize + eventFlagTerminatorSize
	for index, maximum := range worldBlockLimits {
		if at+4 > slotEnd {
			return 0, fmt.Errorf(
				"world block %d of character %d lies outside its slot", index, characterID)
		}
		declared, err := loaded.snapshot.uint32At(at)
		if err != nil {
			return 0, fmt.Errorf(
				"cannot read world block %d of character %d: %w", index, characterID, err)
		}
		size := int64(int32(declared))
		if size < 0 || size >= maximum {
			return 0, fmt.Errorf(
				"character %d declares a world block %d size of %d, want 0..%d",
				characterID, index, size, maximum-1)
		}
		at += 4 + size
	}

	at += playerCoordinatesSize + spawnPointFixedSize
	if version >= spawnPointTempVersion {
		at += 4
	}
	if version >= spawnPointGameManByteVersion {
		at++
	}
	at += netManSectionSize + accountIDTrailingOffset

	if at+accountIDSize > slotEnd {
		return 0, fmt.Errorf(
			"the account identifier of character %d does not fit into its slot", characterID)
	}
	return at, nil
}
