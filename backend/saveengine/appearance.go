package saveengine

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Slot-data layout of the confirmed appearance block, shared by PC and PS4. The
// block has no fixed position inside a slot — everything in front of it depends
// on how much the character carries — so it is located through its own confirmed
// header and read forwards from there. Only the fields listed here are read.
const (
	// faceDataSize is the size of the whole confirmed appearance block.
	faceDataSize = 0x12F

	// The header declares a fixed alignment and inner size. Both are verified
	// before a single appearance byte is decoded, so a block that merely starts
	// like an appearance block is rejected instead of being interpreted.
	faceDataAlignmentOffset = 0x08
	faceDataInnerSizeOffset = 0x0C
	faceDataAlignment       = 4
	faceDataInnerSize       = 0x120

	// The decoded fields, counted from the start of the block. Each model ID is
	// a little-endian uint32; the three parameter blocks are raw bytes.
	faceDataModelIDsOffset  = 0x10
	faceDataModelIDCount    = 8
	faceDataFaceShapeOffset = 0x30
	faceDataFaceShapeSize   = 64
	faceDataBodyOffset      = 0xB0
	faceDataBodySize        = 7
	faceDataSkinOffset      = 0xB7
	faceDataSkinSize        = 91

	// Gender and voice type are not part of the appearance block. They sit in
	// the player data and are read backwards from the confirmed marker that
	// follows it.
	appearanceGenderOffset    = -249
	appearanceVoiceTypeOffset = -245

	// appearancePlayerAnchorLead is how many bytes an anchor needs in front of it
	// for both fields to lie inside the same slot. It is the distance of the more
	// distant field, so an anchor closer to the start of the slot data is not a
	// usable anchor and the search skips past it.
	appearancePlayerAnchorLead = -appearanceGenderOffset
)

// faceDataHeader is the confirmed start of the appearance block: the 0xFFFFFFFF
// marker followed by the four-byte "FACE" magic.
var faceDataHeader = []byte{0xFF, 0xFF, 0xFF, 0xFF, 'F', 'A', 'C', 'E'}

// appearancePlayerAnchor is the confirmed 65-byte marker that follows the player
// data holding gender and voice type: one leading 0x00 byte, then four full
// repetitions of a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed by twelve
// 0x00 bytes.
var appearancePlayerAnchor = []byte{
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

// CharacterAppearance is the raw appearance of one physical save slot. Every
// field is reported exactly as stored: no value is validated, normalised,
// clamped, recomputed, rejected or mapped to a name, and nothing here is derived
// from anything else. ModelIDs stay the uint32 values the save holds; the three
// parameter blocks stay raw bytes and serialise as JSON number arrays.
//
// Every field except SaveSessionID and CharacterID describes an active slot
// only. An inactive slot — including a residual one, whose deleted character's
// appearance is still in the file — reports Active false and zero values, and
// its slot data is never searched or read.
type CharacterAppearance struct {
	SaveSessionID string    `json:"saveSessionID"`
	CharacterID   int       `json:"characterID"`
	Active        bool      `json:"active"`
	Gender        uint8     `json:"gender"`
	VoiceType     uint8     `json:"voiceType"`
	ModelIDs      [8]uint32 `json:"modelIDs"`
	FaceShape     [64]uint8 `json:"faceShape"`
	Body          [7]uint8  `json:"body"`
	Skin          [91]uint8 `json:"skin"`
}

// GetCharacterAppearance returns the raw appearance stored in one physical
// character slot of an existing session. Like the other character readers it
// reads the session's private snapshot through the codec only: it opens no file,
// writes nothing, changes no session and returns no snapshot byte.
//
// saveSessionID is matched exactly. It is never trimmed, normalised or guessed,
// so an empty, unknown or already closed identifier is rejected instead of
// resolving to a session.
//
// characterID is the slot index 0..9. An inactive or residual slot is a normal
// result, not an error, and its slot data is never read.
//
// For an active slot the appearance is the first confirmed appearance block of
// its slot data. A missing block, a first block whose header declares anything
// but the confirmed alignment and inner size, and a first block that does not fit
// inside the slot are hard errors. There is no fallback position, no later block
// is tried, no partial result is returned and nothing is guessed.
func (engine *Engine) GetCharacterAppearance(saveSessionID string, characterID int) (CharacterAppearance, error) {
	if saveSessionID == "" {
		return CharacterAppearance{}, errors.New("saveSessionID is required")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterAppearance{}, fmt.Errorf("unknown save session %q", saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterAppearance{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return CharacterAppearance{}, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}

	appearance := CharacterAppearance{SaveSessionID: saveSessionID, CharacterID: characterID}
	if flag[0] != userData10ActiveFlagValue {
		// An inactive slot is reported from its flag alone, so the residual
		// appearance of a deleted character is never located or decoded.
		return appearance, nil
	}

	anchor, err := findAppearancePlayerAnchor(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return CharacterAppearance{}, err
	}
	gender, err := loaded.snapshot.readAt(anchor+appearanceGenderOffset, 1)
	if err != nil {
		return CharacterAppearance{}, fmt.Errorf("cannot read gender of character %d: %w", characterID, err)
	}
	voiceType, err := loaded.snapshot.readAt(anchor+appearanceVoiceTypeOffset, 1)
	if err != nil {
		return CharacterAppearance{}, fmt.Errorf("cannot read voice type of character %d: %w", characterID, err)
	}

	block, err := readFaceData(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return CharacterAppearance{}, err
	}

	appearance.Active = true
	appearance.Gender = gender[0]
	appearance.VoiceType = voiceType[0]
	for index := range appearance.ModelIDs {
		appearance.ModelIDs[index] = binary.LittleEndian.Uint32(block[faceDataModelIDsOffset+index*4:])
	}
	copy(appearance.FaceShape[:], block[faceDataFaceShapeOffset:])
	copy(appearance.Body[:], block[faceDataBodyOffset:])
	copy(appearance.Skin[:], block[faceDataSkinOffset:])
	return appearance, nil
}

// findAppearancePlayerAnchor locates the player-data anchor of one slot and
// returns its absolute offset. The search is limited to the data of that slot
// and starts appearancePlayerAnchorLead bytes into it, so an anchor without room
// for gender and voice type in front of it is skipped instead of being read
// across the slot start.
//
// A missing anchor in an active slot is a hard error: there is no fallback
// position, no default offset and no guess.
func findAppearancePlayerAnchor(source *codec, platform Platform, characterID int) (int64, error) {
	base := slotDataBase(platform, characterID)
	anchor, err := source.indexIn(base+appearancePlayerAnchorLead,
		characterSlotDataSize-appearancePlayerAnchorLead, appearancePlayerAnchor)
	if err != nil {
		return 0, fmt.Errorf("cannot search the player data of character %d: %w", characterID, err)
	}
	if anchor < 0 {
		return 0, fmt.Errorf("character %d carries no appearance player anchor", characterID)
	}
	return anchor, nil
}

// readFaceData returns a copy of the appearance block of one slot: the first
// confirmed block of that slot's data. Later copies of the same header exist in
// a healthy slot — they sit behind the sections this getter never touches — and
// are ignored; the first block is the appearance the game reads.
//
// A missing block, a first block whose header declares anything but the
// confirmed alignment and inner size, and a first block reaching past the end of
// the slot are all hard errors. No later block is tried as a fallback.
func readFaceData(source *codec, platform Platform, characterID int) ([]byte, error) {
	base := slotDataBase(platform, characterID)
	start, err := source.indexIn(base, characterSlotDataSize, faceDataHeader)
	if err != nil {
		return nil, fmt.Errorf("cannot search the appearance of character %d: %w", characterID, err)
	}
	if start < 0 {
		return nil, fmt.Errorf("character %d carries no appearance block", characterID)
	}
	if start+faceDataSize > base+characterSlotDataSize {
		return nil, fmt.Errorf("appearance block of character %d does not fit into its slot", characterID)
	}

	block, err := source.readAt(start, faceDataSize)
	if err != nil {
		return nil, fmt.Errorf("cannot read appearance of character %d: %w", characterID, err)
	}
	alignment := binary.LittleEndian.Uint32(block[faceDataAlignmentOffset:])
	innerSize := binary.LittleEndian.Uint32(block[faceDataInnerSizeOffset:])
	if alignment != faceDataAlignment || innerSize != faceDataInnerSize {
		return nil, fmt.Errorf("appearance block of character %d declares alignment %d and inner size 0x%X, want %d and 0x%X",
			characterID, alignment, innerSize, faceDataAlignment, faceDataInnerSize)
	}
	return block, nil
}
