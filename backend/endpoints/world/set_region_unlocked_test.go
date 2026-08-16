package world

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	setRegionValidKey = "limgrave_the_first_step"
)

var setRegionAnchor = []byte{
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

func writeSetRegionFixture(t *testing.T, active bool, initialRegions []uint32) string {
	t.Helper()

	const (
		fixtureSize                 = 0x1A00000
		pcUserData10DataOffset      = 0x19003B0
		userData10ActiveFlagsOffset = 0x1954
		slotBase                    = 0x00000310
		characterSlotDataSize       = 0x280000
		slotFixedDlcOffset          = characterSlotDataSize - 128 - 50
		slotFixedHashOffset         = characterSlotDataSize - 128
	)

	data := make([]byte, fixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[0x0C:], 12)

	if active {
		data[pcUserData10DataOffset+userData10ActiveFlagsOffset] = 1
	}

	put32 := func(at int64, v uint32) {
		binary.LittleEndian.PutUint32(data[slotBase+at:], v)
	}

	put32(0, 230) // version

	const (
		anchorAt               = int64(0x01A7)
		projectileCountOffset  = 0x93DC
		blocksBeforeStorage    = 0x1D7
		storageBoxSize         = 0x6010
		gestureSectionSize     = 0x100
		worldHeadSize          = 117
		menuProfilePayloadSize = 0x20
		trophyEquipSize        = 52
		gaItemSize             = 112008
		tutorialPayloadSize    = 0x20
		scalarsSize            = 29
		eventFlagsSize         = 0x1BF99F + 1
		coordinatesSize        = 61
		spawnPointSize         = 15
		netManSize             = 4 + 0x20000
		trailingFixedSize      = 130
		playerHashSize         = 128
	)

	copy(data[slotBase+anchorAt:], setRegionAnchor)

	projAt := anchorAt + projectileCountOffset
	put32(projAt, 0)

	countAt := projAt + 4 + blocksBeforeStorage + storageBoxSize + gestureSectionSize
	put32(countAt, uint32(len(initialRegions)))
	for i, id := range initialRegions {
		put32(countAt+4+int64(i)*4, id)
	}

	pos := countAt + 4 + int64(len(initialRegions))*4
	pos += worldHeadSize
	put32(pos+4, menuProfilePayloadSize)
	pos += 8 + menuProfilePayloadSize
	pos += trophyEquipSize
	pos += gaItemSize
	put32(pos+4, tutorialPayloadSize)
	pos += 8 + tutorialPayloadSize
	pos += scalarsSize
	pos += eventFlagsSize

	// WorldGeom 5 blocks
	for i := 0; i < 5; i++ {
		put32(pos, 0x10)
		pos += 4 + 0x10
	}

	pos += coordinatesSize
	pos += spawnPointSize
	pos += netManSize
	pos += trailingFixedSize
	pos += playerHashSize

	// Fixed DLC & Hash
	for i := int64(0); i < 50; i++ {
		data[slotBase+slotFixedDlcOffset+i] = 0xAA
	}
	for i := int64(0); i < 128; i++ {
		data[slotBase+slotFixedHashOffset+i] = 0xBB
	}

	path := filepath.Join(t.TempDir(), "set-regions.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func loadSetRegionSession(t *testing.T, active bool, initial []uint32) (*saveengine.Engine, string) {
	t.Helper()
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeSetRegionFixture(t, active, initial), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func TestSetRegionUnlockedSuccess(t *testing.T) {
	engine, sessionID := loadSetRegionSession(t, true, []uint32{6200000})
	gameCatalog := newCookbooksCatalog(t)

	result, err := SetRegionUnlocked(
		engine,
		gameCatalog,
		sessionID,
		0,
		"region",
		setRegionValidKey,
		true,
		"0",
	)
	if err != nil {
		t.Fatalf("SetRegionUnlocked: %v", err)
	}

	want := SetRegionUnlockedResult{
		SaveSessionID: sessionID,
		SaveRevision:  "1",
		CharacterID:   0,
		RegionKind:    schema.ResourceKindRegion,
		RegionKey:     setRegionValidKey,
		Unlocked:      true,
	}
	if result != want {
		t.Fatalf("result = %+v, want %+v", result, want)
	}

	// Verify state via GetRegions
	regionsResult, err := GetRegions(engine, gameCatalog, sessionID, 0)
	if err != nil {
		t.Fatalf("GetRegions: %v", err)
	}
	found := false
	for _, reg := range regionsResult.Regions {
		if reg.Key == setRegionValidKey {
			found = true
			if !reg.Unlocked {
				t.Fatalf("region %q is not unlocked", setRegionValidKey)
			}
		}
	}
	if !found {
		t.Fatalf("region %q not found in GetRegions result", setRegionValidKey)
	}
}

func TestSetRegionUnlockedRejectsInvalidRequests(t *testing.T) {
	engine, sessionID := loadSetRegionSession(t, true, []uint32{6100000})
	gameCatalog := newCookbooksCatalog(t)

	// nil Engine
	if _, err := SetRegionUnlocked(nil, gameCatalog, sessionID, 0, "region", setRegionValidKey, true, "0"); err == nil ||
		err.Error() != "save engine is not available" {
		t.Fatalf("nil Engine error = %v", err)
	}

	// nil GameCatalog
	if _, err := SetRegionUnlocked(engine, nil, sessionID, 0, "region", setRegionValidKey, true, "0"); err == nil ||
		err.Error() != "game catalog is not available" {
		t.Fatalf("nil GameCatalog error = %v", err)
	}

	// Wrong kind
	if _, err := SetRegionUnlocked(engine, gameCatalog, sessionID, 0, "map_region", setRegionValidKey, true, "0"); err == nil ||
		err.Error() != `resource kind "map_region" is not "region"` {
		t.Fatalf("wrong kind error = %v", err)
	}

	// Unknown key
	if _, err := SetRegionUnlocked(engine, gameCatalog, sessionID, 0, "region", "unknown_region_key", true, "0"); err == nil ||
		err.Error() != `unknown resource key "unknown_region_key" in kind "region"` {
		t.Fatalf("unknown key error = %v", err)
	}
}
