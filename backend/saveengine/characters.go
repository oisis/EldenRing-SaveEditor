package saveengine

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf16"
)

// UserData10 layout shared by PC and PS4, counted from the start of the
// UserData10 data. Only the confirmed fields of a profile summary are read: the
// character name, level, play time, gender and starting class. Everything else
// in the summary is left untouched.
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

	// summarySecondsPlayedOffset holds the raw play time in seconds as a
	// little-endian uint32. It is never converted into a formatted duration here.
	summarySecondsPlayedOffset = 0x26
	summarySecondsPlayedSize   = 4

	// summaryGenderOffset and summaryStartingClassOffset hold two adjacent raw
	// identifiers. They are reported as stored: the confirmed gender values are
	// 0 for Type B and 1 for Type A, and no identifier is mapped to a name or
	// rejected for being unknown.
	summaryGenderOffset        = 0x242
	summaryStartingClassOffset = 0x243
	summaryIdentifierSize      = 1
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

	base := userData10Base(loaded.session.platform)

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

// userData10Base is the single source of truth for where the UserData10 data
// starts. The PC and PS4 layouts of UserData10 itself are identical; they differ
// only in this base, because the PC container puts an MD5 prefix in front of the
// data and the PS4 container does not.
func userData10Base(platform Platform) int64 {
	if platform == PlatformPS4 {
		return ps4UserData10DataOffset
	}
	return pcUserData10DataOffset
}

// CharacterProfile is the safe public profile of one physical save slot.
//
// StartingClassID and Gender are raw identifiers reported exactly as stored: the
// confirmed gender values are 0 for Type B and 1 for Type A, and an unknown
// identifier is passed through instead of being mapped to a name or rejected.
// SecondsPlayed is the raw number of seconds.
//
// Every field except SaveSessionID and CharacterID describes an active slot
// only. An inactive slot — including a residual one, whose deleted character's
// values are still in the file — reports Active false and zero values, and its
// summary is never read.
type CharacterProfile struct {
	SaveSessionID   string `json:"saveSessionID"`
	CharacterID     int    `json:"characterID"`
	Active          bool   `json:"active"`
	Name            string `json:"name"`
	Level           uint32 `json:"level"`
	StartingClassID uint8  `json:"startingClassID"`
	Gender          uint8  `json:"gender"`
	SecondsPlayed   uint32 `json:"secondsPlayed"`
}

// GetCharacterProfile returns the confirmed profile of one physical character
// slot of an existing session. Like GetSaveCharacters it reads the session's
// private snapshot through the codec only: it opens no file, writes nothing,
// changes no session and returns no snapshot byte.
//
// saveSessionID is matched exactly. It is never trimmed, normalised or guessed,
// so an empty, unknown or already closed identifier is rejected instead of
// resolving to a session.
//
// characterID is the slot index 0..9. An inactive or residual slot is a normal
// result, not an error.
func (engine *Engine) GetCharacterProfile(saveSessionID string, characterID int) (CharacterProfile, error) {
	if saveSessionID == "" {
		return CharacterProfile{}, errors.New("saveSessionID is required")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterProfile{}, fmt.Errorf("unknown save session %q", saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterProfile{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	base := userData10Base(loaded.session.platform)

	flag, err := loaded.snapshot.readAt(base+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return CharacterProfile{}, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}

	profile := CharacterProfile{SaveSessionID: saveSessionID, CharacterID: characterID}
	if flag[0] != userData10ActiveFlagValue {
		// An inactive slot is reported from its flag alone, so residual summary
		// values of a deleted character are never decoded or returned.
		return profile, nil
	}

	summary := base + userData10SummaryOffset + int64(characterID)*userData10SummaryStride

	rawName, err := loaded.snapshot.readAt(summary+summaryNameOffset, summaryNameSize)
	if err != nil {
		return CharacterProfile{}, fmt.Errorf("cannot read name of character %d: %w", characterID, err)
	}
	rawLevel, err := loaded.snapshot.readAt(summary+summaryLevelOffset, summaryLevelSize)
	if err != nil {
		return CharacterProfile{}, fmt.Errorf("cannot read level of character %d: %w", characterID, err)
	}
	rawSeconds, err := loaded.snapshot.readAt(summary+summarySecondsPlayedOffset, summarySecondsPlayedSize)
	if err != nil {
		return CharacterProfile{}, fmt.Errorf("cannot read play time of character %d: %w", characterID, err)
	}
	rawGender, err := loaded.snapshot.readAt(summary+summaryGenderOffset, summaryIdentifierSize)
	if err != nil {
		return CharacterProfile{}, fmt.Errorf("cannot read gender of character %d: %w", characterID, err)
	}
	rawClass, err := loaded.snapshot.readAt(summary+summaryStartingClassOffset, summaryIdentifierSize)
	if err != nil {
		return CharacterProfile{}, fmt.Errorf("cannot read starting class of character %d: %w", characterID, err)
	}

	profile.Active = true
	profile.Name = decodeCharacterName(rawName)
	profile.Level = binary.LittleEndian.Uint32(rawLevel)
	profile.SecondsPlayed = binary.LittleEndian.Uint32(rawSeconds)
	profile.Gender = rawGender[0]
	profile.StartingClassID = rawClass[0]
	return profile, nil
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
