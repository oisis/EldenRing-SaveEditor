package saveengine

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// Slot-data layout of the confirmed character statistics, shared by PC and PS4.
// The block has no fixed position inside a slot, so it is located through a
// confirmed anchor and every field is read backwards from it. Only the fields
// listed here are read; nothing else in the slot is touched.
const (
	// characterSlotDataSize is the size of one character slot's data. It is the
	// same on both platforms; the PC container only prefixes each slot with an
	// MD5 block, which is why the two bases differ and the size does not.
	characterSlotDataSize = ps4SlotSize

	// Offsets of the confirmed fields, counted backwards from the anchor. Every
	// field is a little-endian uint32.
	statsHPOffset                     = -423
	statsMaxHPOffset                  = -419
	statsBaseMaxHPOffset              = -415
	statsFPOffset                     = -411
	statsMaxFPOffset                  = -407
	statsBaseMaxFPOffset              = -403
	statsSPOffset                     = -395
	statsMaxSPOffset                  = -391
	statsBaseMaxSPOffset              = -387
	statsVigorOffset                  = -379
	statsMindOffset                   = -375
	statsEnduranceOffset              = -371
	statsStrengthOffset               = -367
	statsDexterityOffset              = -363
	statsIntelligenceOffset           = -359
	statsFaithOffset                  = -355
	statsArcaneOffset                 = -351
	statsLevelOffset                  = -335
	statsMatchmakingWeaponLevelOffset = int64(-0xD5)

	// statsAnchorLead is how many bytes an anchor needs in front of it for all
	// confirmed fields to lie inside the same slot. It is the distance of the
	// most distant field, so an anchor closer to the start of the slot data is
	// not a usable anchor and the search skips past it.
	statsAnchorLead = -statsHPOffset
)

// statsAnchor is the confirmed 65-byte marker that follows the statistics block
// of a character slot: one leading 0x00 byte, then four full repetitions of a
// 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes.
var statsAnchor = []byte{
	0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

// CharacterStats is the raw statistics block of one physical save slot. Every
// numeric field is the uint32 stored in the save, reported exactly as read: no
// value is validated, normalised, clamped, recomputed or rejected, and nothing
// here is derived from anything else.
//
// Every field except SaveSessionID and CharacterID describes an active slot
// only. An inactive slot — including a residual one, whose deleted character's
// values are still in the file — reports Active false and zero values, and its
// slot data is never searched or read.
type CharacterStats struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
	Active        bool   `json:"active"`

	Vigor        uint32 `json:"vigor"`
	Mind         uint32 `json:"mind"`
	Endurance    uint32 `json:"endurance"`
	Strength     uint32 `json:"strength"`
	Dexterity    uint32 `json:"dexterity"`
	Intelligence uint32 `json:"intelligence"`
	Faith        uint32 `json:"faith"`
	Arcane       uint32 `json:"arcane"`
	Level        uint32 `json:"level"`

	HP        uint32 `json:"hp"`
	MaxHP     uint32 `json:"maxHP"`
	BaseMaxHP uint32 `json:"baseMaxHP"`
	FP        uint32 `json:"fp"`
	MaxFP     uint32 `json:"maxFP"`
	BaseMaxFP uint32 `json:"baseMaxFP"`
	SP        uint32 `json:"sp"`
	MaxSP     uint32 `json:"maxSP"`
	BaseMaxSP uint32 `json:"baseMaxSP"`
}

// GetCharacterStats returns the raw statistics stored in one physical character
// slot of an existing session. Like the other character readers it reads the
// session's private snapshot through the codec only: it opens no file, writes
// nothing, changes no session and returns no snapshot byte.
//
// saveSessionID is matched exactly. It is never trimmed, normalised or guessed,
// so an empty, unknown or already closed identifier is rejected instead of
// resolving to a session.
//
// characterID is the slot index 0..9. An inactive or residual slot is a normal
// result, not an error, and its slot data is never read.
func (engine *Engine) GetCharacterStats(saveSessionID string, characterID int) (CharacterStats, error) {
	if saveSessionID == "" {
		return CharacterStats{}, apperror.MissingField("saveSessionID")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterStats{}, apperror.UnknownSaveSession(saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterStats{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return CharacterStats{}, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}

	stats := CharacterStats{
		SaveSessionID: saveSessionID,
		SaveRevision:  loaded.session.revisionString(),
		CharacterID:   characterID,
	}
	if flag[0] != userData10ActiveFlagValue {
		// An inactive slot is reported from its flag alone, so the residual
		// statistics of a deleted character are never located or decoded.
		return stats, nil
	}

	anchor, err := findStatsAnchor(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return CharacterStats{}, err
	}

	fields := []struct {
		offset int64
		target *uint32
	}{
		{statsHPOffset, &stats.HP},
		{statsMaxHPOffset, &stats.MaxHP},
		{statsBaseMaxHPOffset, &stats.BaseMaxHP},
		{statsFPOffset, &stats.FP},
		{statsMaxFPOffset, &stats.MaxFP},
		{statsBaseMaxFPOffset, &stats.BaseMaxFP},
		{statsSPOffset, &stats.SP},
		{statsMaxSPOffset, &stats.MaxSP},
		{statsBaseMaxSPOffset, &stats.BaseMaxSP},
		{statsVigorOffset, &stats.Vigor},
		{statsMindOffset, &stats.Mind},
		{statsEnduranceOffset, &stats.Endurance},
		{statsStrengthOffset, &stats.Strength},
		{statsDexterityOffset, &stats.Dexterity},
		{statsIntelligenceOffset, &stats.Intelligence},
		{statsFaithOffset, &stats.Faith},
		{statsArcaneOffset, &stats.Arcane},
		{statsLevelOffset, &stats.Level},
	}
	for _, field := range fields {
		value, err := loaded.snapshot.uint32At(anchor + field.offset)
		if err != nil {
			return CharacterStats{}, fmt.Errorf("cannot read statistics of character %d: %w", characterID, err)
		}
		*field.target = value
	}

	stats.Active = true
	return stats, nil
}

// slotDataBase is the single source of truth for where the data of one character
// slot starts. The slot data itself is laid out identically on both platforms;
// the containers differ only in this base, because the PC container puts an MD5
// prefix in front of every slot and the PS4 container does not.
func slotDataBase(platform Platform, characterID int) int64 {
	if platform == PlatformPS4 {
		return ps4SlotDataOffset + int64(characterID)*ps4SlotSize
	}
	return pcSlotDataOffset + int64(characterID)*pcSlotBlockSize
}

// findStatsAnchor locates the statistics anchor of one slot and returns its
// absolute offset. The search is limited to the data of that slot and starts
// statsAnchorLead bytes into it, so an anchor without room for the confirmed
// fields in front of it is skipped instead of being read across the slot start.
//
// A missing anchor in an active slot is a hard error: there is no fallback
// position, no default offset and no guess.
func findStatsAnchor(source *codec, platform Platform, characterID int) (int64, error) {
	base := slotDataBase(platform, characterID)
	anchor, err := source.indexIn(base+statsAnchorLead, characterSlotDataSize-statsAnchorLead, statsAnchor)
	if err != nil {
		return 0, fmt.Errorf("cannot search the statistics of character %d: %w", characterID, err)
	}
	if anchor < 0 {
		return 0, fmt.Errorf("character %d carries no statistics anchor", characterID)
	}
	return anchor, nil
}
