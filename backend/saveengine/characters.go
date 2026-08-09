package saveengine

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf16"
)

// UserData10 layout shared by PC and PS4, counted from the start of the
// UserData10 data. Only the two confirmed fields of a profile summary are read:
// the character name and the character level. Everything else in the summary is
// left untouched.
const (
	characterSlotCount = 10

	// userData10ActiveFlagsOffset is the first of the ten slot activity flags.
	// A slot counts as active only when its flag is exactly 1.
	userData10ActiveFlagsOffset = 0x1954
	userData10ActiveFlagValue   = 1

	// userData10SummaryOffset is the first profile summary; summaries follow one
	// another with a fixed stride.
	userData10SummaryOffset = 0x195E
	userData10SummaryStride = 0x24C

	// summaryNameOffset holds 16 UTF-16LE values, cut at the first NUL.
	summaryNameOffset = 0x00
	summaryNameUnits  = 16
	summaryNameSize   = summaryNameUnits * 2

	// summaryLevelOffset holds the level as a little-endian uint32.
	summaryLevelOffset = 0x22
	summaryLevelSize   = 4
)

// CharacterSummary is the safe public summary of one physical save slot. Name
// and Level are reported for an active slot only: an inactive slot always
// reports an empty name and level 0, even when UserData10 still carries the
// residual values of a deleted character.
type CharacterSummary struct {
	CharacterID int    `json:"characterID"`
	Active      bool   `json:"active"`
	Name        string `json:"name"`
	Level       uint32 `json:"level"`
}

// SaveCharacters is the result of GetSaveCharacters: the session that was read
// and one summary per physical slot, always ten of them in slot order.
type SaveCharacters struct {
	SaveSessionID string             `json:"saveSessionID"`
	Characters    []CharacterSummary `json:"characters"`
}

// GetSaveCharacters returns the summary of all ten physical character slots of
// an existing session. It reads the session's private snapshot through the
// codec only, opens no file, writes nothing, changes no session and returns no
// snapshot byte.
//
// saveSessionID is matched exactly. It is never trimmed, normalised or guessed,
// so an empty, unknown or already closed identifier is rejected instead of
// resolving to a session.
//
// CharacterID is the slot index 0..9, so the result is positional and needs no
// separate slot field. Activity comes from the slot's UserData10 flag, and only
// the confirmed name and level of an active slot are decoded.
func (engine *Engine) GetSaveCharacters(saveSessionID string) (SaveCharacters, error) {
	if saveSessionID == "" {
		return SaveCharacters{}, errors.New("saveSessionID is required")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return SaveCharacters{}, fmt.Errorf("unknown save session %q", saveSessionID)
	}

	// The PC and PS4 layouts differ only in where the UserData10 data starts:
	// the PC container puts an MD5 prefix in front of it.
	base := int64(pcUserData10DataOffset)
	if loaded.session.platform == PlatformPS4 {
		base = ps4UserData10DataOffset
	}

	flags, err := loaded.snapshot.readAt(base+userData10ActiveFlagsOffset, characterSlotCount)
	if err != nil {
		return SaveCharacters{}, fmt.Errorf("cannot read character slot activity: %w", err)
	}

	characters := make([]CharacterSummary, characterSlotCount)
	for slot := range characters {
		characters[slot] = CharacterSummary{CharacterID: slot}
		if flags[slot] != userData10ActiveFlagValue {
			continue
		}
		summary := base + userData10SummaryOffset + int64(slot)*userData10SummaryStride

		rawName, err := loaded.snapshot.readAt(summary+summaryNameOffset, summaryNameSize)
		if err != nil {
			return SaveCharacters{}, fmt.Errorf("cannot read name of character %d: %w", slot, err)
		}
		rawLevel, err := loaded.snapshot.readAt(summary+summaryLevelOffset, summaryLevelSize)
		if err != nil {
			return SaveCharacters{}, fmt.Errorf("cannot read level of character %d: %w", slot, err)
		}

		characters[slot].Active = true
		characters[slot].Name = decodeCharacterName(rawName)
		characters[slot].Level = binary.LittleEndian.Uint32(rawLevel)
	}

	return SaveCharacters{SaveSessionID: saveSessionID, Characters: characters}, nil
}

// decodeCharacterName decodes the fixed-size UTF-16LE name field. The name ends
// at the first NUL value; a field without one is decoded in full.
func decodeCharacterName(raw []byte) string {
	units := make([]uint16, 0, summaryNameUnits)
	for index := 0; index+1 < len(raw); index += 2 {
		unit := binary.LittleEndian.Uint16(raw[index:])
		if unit == 0 {
			break
		}
		units = append(units, unit)
	}
	return string(utf16.Decode(units))
}
