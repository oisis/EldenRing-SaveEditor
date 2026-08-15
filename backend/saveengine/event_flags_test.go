package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Synthetic container layout used only by this test. The offsets are restated
// literally instead of reused from the implementation, so a changed base, stride
// or section distance fails here.
const (
	eventFlagTestPCSlotDataBase  = 0x300 + 0x10 // first PC slot data, behind its MD5 prefix
	eventFlagTestPCSlotStride    = 0x280010
	eventFlagTestPS4SlotDataBase = 0x70 // first PS4 slot data, no MD5 prefix
	eventFlagTestPS4SlotStride   = 0x280000
	eventFlagTestSlotDataSize    = 0x280000

	// Distance from the anchor to the uint32 that declares how many
	// acquired-projectile records follow it: SpEffect, EquipedItemIndex,
	// ActiveEquipedItems, EquipedItemsID, ActiveEquipedItemsGa, InventoryHeld,
	// EquippedSpells, EquipItemData and EquippedGestures.
	eventFlagTestProjectileCountAt = 0xD0 + 0x58 + 0x1C + 0x58 + 0x58 + 0x9011 + 0x74 + 0x8C + 0x18

	// The fixed blocks between the projectile records and the unlocked regions:
	// the equipped armaments, EquipPhysicsData, the face data, the Storage Box
	// and GestureGameData, each restated as its own confirmed size.
	eventFlagTestBlocksBeforeStorage = 0x9C + 0x0C + 0x12F
	eventFlagTestStorageBoxSize      = 4 + 0x780*12 + 4 + 0x80*12 + 8
	eventFlagTestGestureSectionSize  = 64 * 4

	// Torrent plus its control byte, and the blood stain plus its padding.
	eventFlagTestHorseSize      = 0x28 + 1
	eventFlagTestBloodStainSize = 0x44 + 8

	// The two variable blocks are stored as u16 + u16 + u32 size, then payload.
	eventFlagTestDynamicHeader = 2 + 2 + 4

	// TrophyEquipData, and GaItemGameData with its int64 count in front of 7000
	// sixteen-byte entries.
	eventFlagTestTrophySize = 0x34
	eventFlagTestGaItemSize = 8 + 7000*16

	// The scalar block between the end of the tutorial data and the bitfield.
	eventFlagTestScalarsSize = 3 + 4 + 4 + 1 + 4 + 4 + 1 + 4 + 4

	eventFlagTestBlockSize = 125

	// Distances from the anchor to the two variable headers, valid for a fixture
	// that declares no projectile, no unlocked region and an empty menu payload.
	// They exist so a fixture can place one header exactly across the end of the
	// slot while every earlier step of the chain still lies inside it.
	eventFlagTestMenuHeaderAt = eventFlagTestProjectileCountAt + 4 +
		eventFlagTestBlocksBeforeStorage + eventFlagTestStorageBoxSize +
		eventFlagTestGestureSectionSize + 4 +
		eventFlagTestHorseSize + eventFlagTestBloodStainSize
	eventFlagTestTutorialHeaderAt = eventFlagTestMenuHeaderAt + eventFlagTestDynamicHeader +
		eventFlagTestTrophySize + eventFlagTestGaItemSize
)

// eventFlagTestAnchor is the 65-byte anchor the chain is measured from, restated
// here independently of the implementation.
var eventFlagTestAnchor = []byte{
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

// eventFlagTestFixture describes the synthetic slot content one test save is
// built from. Every declared length is written into the file, so a reader that
// ignores one of them lands on a shifted position.
type eventFlagTestFixture struct {
	platform     Platform
	slot         int
	flag         byte
	anchorAt     int64
	projectiles  uint32
	regions      uint32
	menuSize     uint32
	tutorialSize uint32
	set          []uint32
	noAnchor     bool
}

// eventFlagTestPosition places one identifier inside the bitfield, restating the
// confirmed formula independently of the implementation.
func eventFlagTestPosition(t *testing.T, id uint32) (int64, uint8) {
	t.Helper()

	position := map[uint32]int64{
		9:  9,
		60: 10, 62: 12, 65: 15, 67: 17, 68: 18, 71: 21, 72: 22, 73: 23, 74: 24, 76: 26,
		670: 107, 11109: 11129,
	}[id/1000]
	if position == 0 {
		t.Fatalf("test fixture cannot place event flag %d", id)
	}
	index := int64(id % 1000)
	return position*eventFlagTestBlockSize + index/8, uint8(7 - index%8)
}

// writeEventFlagFixture builds a synthetic save and returns its path inside
// t.TempDir(). Only the activity flag, the anchor, the declared lengths and the
// requested set flags are written; the rest of the container stays zeroed.
func writeEventFlagFixture(t *testing.T, content eventFlagTestFixture) string {
	t.Helper()

	var data []byte
	var userData10Base, slotBase int64
	switch content.platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = pcUserData10DataOffset
		slotBase = eventFlagTestPCSlotDataBase + int64(content.slot)*eventFlagTestPCSlotStride
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = ps4UserData10DataOffset
		slotBase = eventFlagTestPS4SlotDataBase + int64(content.slot)*eventFlagTestPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", content.platform)
	}

	data[userData10Base+userData10ActiveFlagsOffset+int64(content.slot)] = content.flag

	path := filepath.Join(t.TempDir(), "event_flags.sl2")
	write := func() string {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		return path
	}

	if content.noAnchor {
		return write()
	}
	copy(data[slotBase+content.anchorAt:], eventFlagTestAnchor)

	put := func(at int64, value uint32) {
		if at+4 <= eventFlagTestSlotDataSize {
			binary.LittleEndian.PutUint32(data[slotBase+at:], value)
		}
	}

	at := content.anchorAt + eventFlagTestProjectileCountAt
	put(at, content.projectiles)
	at += 4 + int64(content.projectiles)*8 +
		eventFlagTestBlocksBeforeStorage + eventFlagTestStorageBoxSize +
		eventFlagTestGestureSectionSize

	put(at, content.regions)
	at += 4 + int64(content.regions)*4 + eventFlagTestHorseSize + eventFlagTestBloodStainSize

	put(at+4, content.menuSize)
	at += eventFlagTestDynamicHeader + int64(content.menuSize) +
		eventFlagTestTrophySize + eventFlagTestGaItemSize

	put(at+4, content.tutorialSize)
	sectionAt := at + eventFlagTestDynamicHeader + int64(content.tutorialSize) +
		eventFlagTestScalarsSize

	for _, id := range content.set {
		offset, bit := eventFlagTestPosition(t, id)
		if sectionAt+offset >= eventFlagTestSlotDataSize {
			continue
		}
		data[slotBase+sectionAt+offset] |= 1 << bit
	}
	return write()
}

// eventFlagTestContent is the slot content both platform fixtures share: a full
// chain with a non-zero projectile count, a non-zero region count and a tutorial
// payload that is not the legacy 0x400 bytes long.
func eventFlagTestContent(platform Platform) eventFlagTestFixture {
	return eventFlagTestFixture{
		platform: platform, slot: 3, flag: 1, anchorAt: 0x1A7,
		projectiles: 37, regions: 91, menuSize: 0x800, tutorialSize: 0x321,
		set: []uint32{67000, 67999, 68500},
	}
}

func TestGetEventFlagsReadsTheActiveSlotOfBothPlatforms(t *testing.T) {
	// The two containers carry identical slot content, so only the platform base
	// differs and a reader that mixes the two bases cannot pass both cases.
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := eventFlagTestContent(platform)
			content.set = append(content.set, 11109710, 670500)

			engine := New()
			loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.GetEventFlags(
				loaded.SaveSessionID, content.slot,
				[]uint32{67000, 67110, 11109710, 11109711, 670500, 670501})
			if err != nil {
				t.Fatalf("GetEventFlags: %v", err)
			}

			want := CharacterEventFlags{
				SaveSessionID: loaded.SaveSessionID,
				CharacterID:   content.slot,
				Active:        true,
				Flags: map[uint32]bool{
					67000: true, 67110: false, 11109710: true, 11109711: false,
					670500: true, 670501: false,
				},
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}
		})
	}
}

func TestGetEventFlagsResolvesSupportedBlocksAtTheirBoundaries(t *testing.T) {
	content := eventFlagTestContent(PlatformPC)
	content.set = append(content.set,
		60000, 60999, 65000, 65999, 670000, 670999, 11109000, 11109999)

	engine := New()
	loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// The first and the last flag of every supported block, so an off-by-one in
	// the byte or in the bit direction lands on a neighbour that is set
	// differently.
	requested := []uint32{
		60000, 60999, 65000, 65999, 67000, 67999, 68000, 68999,
		670000, 670001, 670998, 670999, 11109000, 11109999,
	}
	want := map[uint32]bool{
		60000: true, 60999: true, 65000: true, 65999: true,
		67000: true, 67999: true, 68000: false, 68999: false,
		670000: true, 670001: false, 670998: false, 670999: true,
		11109000: true, 11109999: true,
	}

	result, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot, requested)
	if err != nil {
		t.Fatalf("GetEventFlags: %v", err)
	}
	if !reflect.DeepEqual(result.Flags, want) {
		t.Errorf("flags = %+v, want %+v", result.Flags, want)
	}
}

// The curated Graces table uses the five blocks 71 to 74 and 76. Block 75 has a
// bitfield position of its own but carries no grace, so the reader must keep
// rejecting it instead of answering it from a neighbouring position.
func TestGetEventFlagsResolvesTheGraceBlocksOfBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := eventFlagTestContent(platform)
			content.set = append(content.set, 71000, 71999, 72000, 73000, 74000, 76000)

			engine := New()
			loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			// One set flag and its clear neighbour per block, so a block resolved
			// to a wrong position or a mirrored bit lands on a different value.
			requested := []uint32{
				71000, 71001, 71999, 72000, 72001, 73000, 73001, 74000, 74001, 76000, 76001,
			}
			want := map[uint32]bool{
				71000: true, 71001: false, 71999: true,
				72000: true, 72001: false,
				73000: true, 73001: false,
				74000: true, 74001: false,
				76000: true, 76001: false,
			}

			result, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot, requested)
			if err != nil {
				t.Fatalf("GetEventFlags: %v", err)
			}
			if !reflect.DeepEqual(result.Flags, want) {
				t.Errorf("flags = %+v, want %+v", result.Flags, want)
			}

			if _, err := engine.GetEventFlags(
				loaded.SaveSessionID, content.slot, []uint32{75000}); err == nil {
				t.Error("the unused block 75 was accepted")
			}
		})
	}
}

// The curated Bosses table uses block 9 alone. Its neighbours 8 and 10 carry no
// curated resource, so the reader must keep rejecting them instead of answering
// them from an adjacent position.
func TestGetEventFlagsResolvesTheBossBlockOfBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := eventFlagTestContent(platform)
			content.set = append(content.set, 9000, 9100, 9281, 9999)

			engine := New()
			loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			// Both block boundaries plus the first and the last curated boss flag,
			// each with its clear neighbour, so a wrong block position or a
			// mirrored bit lands on a different value.
			requested := []uint32{9000, 9001, 9100, 9101, 9280, 9281, 9998, 9999}
			want := map[uint32]bool{
				9000: true, 9001: false, 9100: true, 9101: false,
				9280: false, 9281: true, 9998: false, 9999: true,
			}

			result, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot, requested)
			if err != nil {
				t.Fatalf("GetEventFlags: %v", err)
			}
			if !reflect.DeepEqual(result.Flags, want) {
				t.Errorf("flags = %+v, want %+v", result.Flags, want)
			}

			for _, unsupported := range []uint32{8999, 10000} {
				if _, err := engine.GetEventFlags(
					loaded.SaveSessionID, content.slot, []uint32{unsupported}); err == nil {
					t.Errorf("the unsupported neighbouring flag %d was accepted", unsupported)
				}
			}
		})
	}
}

// The curated map visibility table uses block 62 alone. Blocks 61 and 63 are
// not part of this getter contract, so they must remain unsupported.
func TestGetEventFlagsResolvesTheMapRegionBlockOfBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := eventFlagTestContent(platform)
			content.set = append(content.set, 62000, 62010, 62999)

			engine := New()
			loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			requested := []uint32{62000, 62001, 62010, 62011, 62998, 62999}
			want := map[uint32]bool{
				62000: true, 62001: false, 62010: true,
				62011: false, 62998: false, 62999: true,
			}
			result, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot, requested)
			if err != nil {
				t.Fatalf("GetEventFlags: %v", err)
			}
			if !reflect.DeepEqual(result.Flags, want) {
				t.Errorf("flags = %+v, want %+v", result.Flags, want)
			}

			for _, unsupported := range []uint32{61999, 63000, 82001} {
				if _, err := engine.GetEventFlags(
					loaded.SaveSessionID, content.slot, []uint32{unsupported}); err == nil {
					t.Errorf("the unsupported map flag %d was accepted", unsupported)
				}
			}
		})
	}
}

func TestGetEventFlagsReportsAResidualSlotAsInactive(t *testing.T) {
	content := eventFlagTestContent(PlatformPC)
	content.flag = 0

	engine := New()
	loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot, []uint32{67000})
	if err != nil {
		t.Fatalf("GetEventFlags: %v", err)
	}

	// The bitfield of the deleted character is still written into the fixture and
	// carries 67000 as set, so a false result proves the slot data was never
	// searched or decoded.
	want := CharacterEventFlags{
		SaveSessionID: loaded.SaveSessionID,
		CharacterID:   content.slot,
		Flags:         map[uint32]bool{},
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
	if result.Flags == nil {
		t.Error("flags is nil, want an empty map")
	}
}

func TestGetEventFlagsRejectsWhatItCannotResolve(t *testing.T) {
	engine := New()

	load := func(content eventFlagTestFixture) string {
		t.Helper()
		loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		return loaded.SaveSessionID
	}

	present := load(eventFlagTestContent(PlatformPC))

	missingAnchor := eventFlagTestContent(PlatformPC)
	missingAnchor.noAnchor = true

	// The anchor sits so late in the slot that the bitfield and its terminator no
	// longer fit behind the chain.
	corruptRange := eventFlagTestContent(PlatformPC)
	corruptRange.anchorAt = 0x100000

	corruptCount := eventFlagTestContent(PlatformPC)
	corruptCount.projectiles = 200001

	corruptRegions := eventFlagTestContent(PlatformPC)
	corruptRegions.regions = 20001

	corruptTutorial := eventFlagTestContent(PlatformPC)
	corruptTutorial.tutorialSize = 0x10001

	cases := map[string]struct {
		saveSessionID string
		requested     []uint32
		want          string
	}{
		"block below the supported range": {present, []uint32{66999},
			"event flag 66999 lies in block 66, which this reader does not support"},
		"block above the supported range": {present, []uint32{69000},
			"event flag 69000 lies in block 69, which this reader does not support"},
		"no block at all": {present, []uint32{7},
			"event flag 7 lies in block 0, which this reader does not support"},
		"missing anchor": {load(missingAnchor), []uint32{67000},
			"character 3 carries no event flag anchor"},
		"corrupt event flag range": {load(corruptRange), []uint32{67000},
			"event flags of character 3 do not fit into their slot"},
		"corrupt projectile count": {load(corruptCount), []uint32{67000},
			"character 3 declares a projectile count of 200001, want at most 200000"},
		"corrupt region count": {load(corruptRegions), []uint32{67000},
			"character 3 declares a region count of 20001, want at most 20000"},
		"corrupt tutorial size": {load(corruptTutorial), []uint32{67000},
			"character 3 declares a tutorial size of 65537, want at most 65536"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.GetEventFlags(testCase.saveSessionID, 3, testCase.requested)
			if err == nil {
				t.Fatalf("GetEventFlags accepted %q", name)
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, CharacterEventFlags{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}

func TestGetEventFlagsRejectsInvalidRequests(t *testing.T) {
	engine := New()

	present, err := engine.LoadSave(writeEventFlagFixture(t, eventFlagTestContent(PlatformPC)), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	closed, err := engine.LoadSave(writeEventFlagFixture(t, eventFlagTestContent(PlatformPC)), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if err := engine.CloseSession(closed.SaveSessionID); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}

	cases := map[string]struct {
		saveSessionID string
		characterID   int
		want          string
	}{
		"empty session":   {"", 3, "saveSessionID is required"},
		"unknown session": {"missing", 3, `unknown save session "missing"`},
		"closed session": {closed.SaveSessionID, 3,
			"unknown save session " + strconv.Quote(closed.SaveSessionID)},
		"characterID -1": {present.SaveSessionID, -1, "characterID -1 is outside the range 0..9"},
		"characterID 10": {present.SaveSessionID, 10, "characterID 10 is outside the range 0..9"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.GetEventFlags(
				testCase.saveSessionID, testCase.characterID, []uint32{67000})
			if err == nil {
				t.Fatalf("GetEventFlags accepted %q", name)
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, CharacterEventFlags{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}

func TestGetEventFlagsResolvesEveryIdentifierBeforeTheSlotIsTouched(t *testing.T) {
	// The slot is residual: its activity flag is zero while the bitfield of the
	// deleted character is still written into the fixture. An inactive slot is
	// otherwise a normal empty result, so answering this request with one would
	// mean the unsupported identifier was never resolved.
	content := eventFlagTestContent(PlatformPC)
	content.flag = 0

	engine := New()
	loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.GetEventFlags(
		loaded.SaveSessionID, content.slot, []uint32{67000, 69000})
	if err == nil {
		t.Fatalf("GetEventFlags accepted an unsupported identifier: %+v", result)
	}
	want := "event flag 69000 lies in block 69, which this reader does not support"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
	if !reflect.DeepEqual(result, CharacterEventFlags{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}

func TestGetEventFlagsRejectsCorruptVariableBlocks(t *testing.T) {
	engine := New()

	load := func(content eventFlagTestFixture) string {
		t.Helper()
		loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		return loaded.SaveSessionID
	}

	oversizedMenu := eventFlagTestContent(PlatformPC)
	oversizedMenu.menuSize = 0x10001

	// A flat chain — no projectile, no unlocked region, no menu payload — placed
	// so that one variable header is cut off by the end of the slot while every
	// earlier step of the chain still lies inside it.
	flat := eventFlagTestContent(PlatformPC)
	flat.projectiles, flat.regions, flat.menuSize = 0, 0, 0

	incompleteMenuHeader := flat
	incompleteMenuHeader.anchorAt = eventFlagTestSlotDataSize - eventFlagTestMenuHeaderAt - 7

	incompleteTutorialHeader := flat
	incompleteTutorialHeader.anchorAt = eventFlagTestSlotDataSize - eventFlagTestTutorialHeaderAt - 7

	cases := map[string]struct {
		saveSessionID string
		want          string
	}{
		"menu profile size above the accepted maximum": {load(oversizedMenu),
			"character 3 declares a menu profile size of 65537, want at most 65536"},
		"incomplete menu profile header": {load(incompleteMenuHeader),
			"menu profile size of character 3 lies outside its slot"},
		"incomplete tutorial header": {load(incompleteTutorialHeader),
			"tutorial size of character 3 lies outside its slot"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.GetEventFlags(testCase.saveSessionID, 3, []uint32{67000})
			if err == nil {
				t.Fatalf("GetEventFlags accepted %q", name)
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, CharacterEventFlags{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}

func TestEventFlagSectionStartRejectsASnapshotShorterThanTheSlot(t *testing.T) {
	// A loaded container always covers its slots, so a snapshot that ends inside
	// the slot can only be built directly. The chain must refuse it before one
	// flag is read: a bitfield that is not completely present in the private
	// snapshot is an error, never a false.
	loaded := &loadedSave{
		session:  &Session{platform: PlatformPC},
		snapshot: &codec{data: make([]byte, pcSlotDataOffset+eventFlagTestSlotDataSize/2)},
	}

	sectionAt, err := eventFlagSectionStart(loaded, 0)
	if err == nil {
		t.Fatalf("eventFlagSectionStart accepted a truncated snapshot: 0x%X", sectionAt)
	}
	if sectionAt != 0 {
		t.Errorf("sectionAt = 0x%X, want 0", sectionAt)
	}
	if !strings.HasPrefix(err.Error(), "cannot search the event flags of character 0:") {
		t.Errorf("error = %q, want the search of character 0 to be reported", err)
	}
}

func TestGetEventFlagsReturnsOneEntryPerDistinctIdentifier(t *testing.T) {
	content := eventFlagTestContent(PlatformPC)

	engine := New()
	loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	cases := map[string]struct {
		requested []uint32
		want      map[uint32]bool
	}{
		"no identifier at all":  {[]uint32{}, map[uint32]bool{}},
		"a repeated set flag":   {[]uint32{67000, 67000}, map[uint32]bool{67000: true}},
		"a repeated clear flag": {[]uint32{68000, 68000}, map[uint32]bool{68000: false}},
		"repetition mixed with a second identifier": {
			[]uint32{67000, 67999, 67000}, map[uint32]bool{67000: true, 67999: true}},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.GetEventFlags(
				loaded.SaveSessionID, content.slot, testCase.requested)
			if err != nil {
				t.Fatalf("GetEventFlags: %v", err)
			}
			if !result.Active {
				t.Error("active = false, want the slot to stay active")
			}
			if result.Flags == nil {
				t.Fatal("flags is nil, want a map")
			}
			if !reflect.DeepEqual(result.Flags, testCase.want) {
				t.Errorf("flags = %+v, want %+v", result.Flags, testCase.want)
			}
		})
	}
}

func TestGetEventFlagsLeavesTheSaveAndTheSnapshotUntouched(t *testing.T) {
	content := eventFlagTestContent(PlatformPC)
	path := writeEventFlagFixture(t, content)

	engine := New()
	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	fileBefore, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	snapshotBefore := append([]byte(nil), engine.sessions[loaded.SaveSessionID].snapshot.data...)

	// Both an active and an inactive slot, so neither the located read nor the
	// activity-only path can write into the snapshot.
	for _, characterID := range []int{content.slot, content.slot + 1} {
		if _, err := engine.GetEventFlags(
			loaded.SaveSessionID, characterID, []uint32{67000, 67999, 68000}); err != nil {
			t.Fatalf("GetEventFlags(%d): %v", characterID, err)
		}
	}

	fileAfter, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reread fixture: %v", err)
	}
	if !bytes.Equal(fileBefore, fileAfter) {
		t.Error("the save file changed, want it untouched")
	}
	if !bytes.Equal(snapshotBefore, engine.sessions[loaded.SaveSessionID].snapshot.data) {
		t.Error("the private snapshot changed, want it untouched")
	}
	if info := engine.sessions[loaded.SaveSessionID].session.Info(); info.UnsavedChanges {
		t.Error("unsavedChanges = true, want a read to leave the session clean")
	}
}
