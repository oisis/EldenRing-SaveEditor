package saveengine

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// The GaItem table is the private save-side lookup from an InventoryHeld or
// Storage handle to the catalog game ID. It is shared by both containers, but
// is deliberately separate from their readers: it owns no container layout and
// returns no raw slot data.
const (
	gaItemTableOffset        = 0x20
	gaItemRecordSize         = 8
	gaItemWeaponRecordSize   = 21
	gaItemArmorRecordSize    = 16
	gaItemOldRecordCount     = 5118
	gaItemCurrentRecordCount = 5120
	gaItemVersionBreak       = 81

	gaItemHandleTypeMask  uint32 = 0xF0000000
	gaItemWeaponHandle    uint32 = 0x80000000
	gaItemArmorHandle     uint32 = 0x90000000
	gaItemAccessoryHandle uint32 = 0xA0000000
	gaItemGoodsHandle     uint32 = 0xB0000000
	gaItemAshOfWarHandle  uint32 = 0xC0000000
)

// gaItemAnchor is the confirmed PlayerGameData marker. The table starts at
// offset 0x20 of the slot and ends immediately before this marker.
var gaItemAnchor = []byte{
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

// ResolveGaItemIDs resolves each supplied raw InventoryHeld/Storage handle to
// the catalog game ID of the item it denotes. It reads only the private session
// snapshot and is intended for endpoint joins with GameCatalog; it never makes
// a handle part of a public endpoint contract.
func (engine *Engine) ResolveGaItemIDs(
	saveSessionID string,
	characterID int,
	handles []uint32,
) ([]uint32, error) {
	if saveSessionID == "" {
		return nil, errors.New("saveSessionID is required")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return nil, fmt.Errorf("unknown save session %q", saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return nil, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}
	if len(handles) == 0 {
		return []uint32{}, nil
	}

	byHandle, err := readGaItemMap(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve items of character %d: %w", characterID, err)
	}

	resolved := make([]uint32, len(handles))
	for index, handle := range handles {
		gameID, err := resolveGaItemHandle(byHandle, handle)
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", index, err)
		}
		resolved[index] = gameID
	}
	return resolved, nil
}

func readGaItemMap(source *codec, platform Platform, characterID int) (map[uint32]uint32, error) {
	byHandle := make(map[uint32]uint32)
	err := walkGaItemRecords(source, platform, characterID, func(record gaItemRecord) error {
		handle, gameID := record.handle, record.gameID
		if handle == 0 || gameID == 0 || gameID == 0xFFFFFFFF {
			return nil
		}
		switch handle & gaItemHandleTypeMask {
		case gaItemWeaponHandle, gaItemArmorHandle, gaItemAccessoryHandle, gaItemGoodsHandle, gaItemAshOfWarHandle:
			if previous, duplicate := byHandle[handle]; duplicate && previous != gameID {
				return fmt.Errorf("GaItem handle 0x%08X maps to both 0x%08X and 0x%08X", handle, previous, gameID)
			}
			byHandle[handle] = gameID
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return byHandle, nil
}

type gaItemRecord struct {
	at     int64
	handle uint32
	gameID uint32
}

// walkGaItemRecords is the single parser of the variable-length GaItem table.
// It visits each record without materialising a second table in memory, while
// readers and mutations still share version counts, sizes and exact positions.
func walkGaItemRecords(
	source *codec,
	platform Platform,
	characterID int,
	visit func(gaItemRecord) error,
) error {
	base, slotEnd := inventorySlotBounds(platform, characterID)
	version, err := source.uint32At(base)
	if err != nil {
		return fmt.Errorf("cannot read slot version: %w", err)
	}
	if version == 0 {
		return errors.New("slot has no GaItem table")
	}

	anchor, err := source.indexIn(base, slotEnd-base, gaItemAnchor)
	if err != nil {
		return fmt.Errorf("cannot search GaItem marker: %w", err)
	}
	if anchor < 0 {
		return errors.New("slot carries no GaItem marker")
	}
	start := base + gaItemTableOffset
	if anchor < start {
		return errors.New("GaItem marker precedes the GaItem table")
	}

	table, err := source.readAt(start, int(anchor-start))
	if err != nil {
		return fmt.Errorf("cannot read GaItem table: %w", err)
	}
	count := gaItemCurrentRecordCount
	if version <= gaItemVersionBreak {
		count = gaItemOldRecordCount
	}

	position := 0
	for index := 0; index < count; index++ {
		if position+gaItemRecordSize > len(table) {
			return fmt.Errorf("GaItem record %d does not fit before the marker", index)
		}
		handle := binary.LittleEndian.Uint32(table[position:])
		gameID := binary.LittleEndian.Uint32(table[position+4:])
		recordSize := gaItemSize(gameID)
		if position+recordSize > len(table) {
			return fmt.Errorf("GaItem record %d exceeds the marker", index)
		}
		if err := visit(gaItemRecord{
			at: start + int64(position), handle: handle, gameID: gameID,
		}); err != nil {
			return err
		}
		position += recordSize
	}
	return nil
}

func gaItemSize(gameID uint32) int {
	if gameID == 0 || gameID == 0xFFFFFFFF {
		return gaItemRecordSize
	}
	switch gameID & gaItemHandleTypeMask {
	case 0x00000000:
		return gaItemWeaponRecordSize
	case 0x10000000:
		return gaItemArmorRecordSize
	default:
		return gaItemRecordSize
	}
}

// gaItemHandleForGameID derives the handle a new record of gameID has to carry.
//
// It is the exact inverse of the record-free branch of resolveGaItemHandle
// below, and it sits beside it so the two can never drift apart: only the two
// families whose handle is derived from the game ID alone — accessories and
// goods — have one, because only they need no record in the variable-length
// GaItem table. Every other family is rejected here rather than given a handle
// that would resolve to nothing.
func gaItemHandleForGameID(gameID uint32) (uint32, error) {
	lower := gameID & 0x0FFFFFFF
	switch gameID & gaItemHandleTypeMask {
	case 0x20000000:
		return gaItemAccessoryHandle | lower, nil
	case 0x40000000:
		return gaItemGoodsHandle | lower, nil
	default:
		return 0, fmt.Errorf(
			"game ID 0x%08X needs a record in the GaItem table, which this mutation never allocates",
			gameID)
	}
}

func resolveGaItemHandle(byHandle map[uint32]uint32, handle uint32) (uint32, error) {
	if gameID, exists := byHandle[handle]; exists {
		return gameID, nil
	}

	lower := handle & 0x0FFFFFFF
	switch handle & gaItemHandleTypeMask {
	case gaItemAccessoryHandle:
		return 0x20000000 | lower, nil
	case gaItemGoodsHandle:
		return 0x40000000 | lower, nil
	case gaItemWeaponHandle, gaItemArmorHandle, gaItemAshOfWarHandle:
		return 0, fmt.Errorf("GaItem handle 0x%08X has no record", handle)
	default:
		return 0, fmt.Errorf("GaItem handle 0x%08X has an unknown type", handle)
	}
}
