package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	setPouchInventoryAt           = 505
	setPouchInventoryRowSize      = 12
	setPouchSlotVersion           = 82
	setPouchAnchorAt              = 0x10020
	setPouchTestGameID1           = uint32(0x40000064)
	setPouchTestGameID2           = uint32(0x40000065)
	setPouchTestGameID3           = uint32(0x40000066)
	setPouchTestAccessoryID       = uint32(0x20000064)
	setPouchProjectileCountOffset = 0x931D
	setPouchProjectileRecordSize  = 8
)

func writeSetPouchItemsFixture(
	t *testing.T,
	platform Platform,
	slot int,
	active bool,
	ownedInventory []struct {
		gameID   uint32
		quantity uint32
	},
) (string, string) {
	t.Helper()

	content := pouchItemsFixture{
		platform: platform,
		slot:     slot,
		flag:     1,
		anchorAt: setPouchAnchorAt,
		items:    [6]PouchItemSlot{},
	}
	if !active {
		content.flag = 0
	}
	path := writePouchItemsFixture(t, content)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var slotBase int64
	if platform == PlatformPS4 {
		slotBase = pouchItemsPS4SlotDataBase + int64(slot)*pouchItemsPS4SlotStride
	} else {
		slotBase = pouchItemsPCSlotDataBase + int64(slot)*pouchItemsPCSlotStride
	}
	binary.LittleEndian.PutUint32(data[slotBase:], setPouchSlotVersion)

	pairAt := slotBase + setPouchAnchorAt + pouchItemsSectionOffset
	for i := 0; i < pouchItemSlotCount; i++ {
		binary.LittleEndian.PutUint32(data[pairAt+int64(i)*8:], pouchEmptyItemID)
		binary.LittleEndian.PutUint32(data[pairAt+int64(i)*8+4:], pouchEmptyEquipIndex)
	}

	countAt := slotBase + setPouchAnchorAt + setPouchProjectileCountOffset
	binary.LittleEndian.PutUint32(data[countAt:], 17)

	armamentsOff := countAt + 4 + 17*setPouchProjectileRecordSize
	tailAt := armamentsOff + 0x80
	for i := 0; i < pouchItemSlotCount; i++ {
		binary.LittleEndian.PutUint32(data[tailAt+int64(i)*4:], PouchEmptyGameID)
	}

	inventoryAt := slotBase + setPouchAnchorAt + setPouchInventoryAt
	binary.LittleEndian.PutUint32(data[inventoryAt-4:], uint32(len(ownedInventory)))

	for index, item := range ownedInventory {
		rowAt := inventoryAt + int64(index*setPouchInventoryRowSize)
		handle, err := gaItemHandleForGameID(item.gameID)
		if err != nil {
			t.Fatalf("gaItemHandleForGameID(0x%08X): %v", item.gameID, err)
		}
		binary.LittleEndian.PutUint32(data[rowAt:], handle)
		binary.LittleEndian.PutUint32(data[rowAt+4:], item.quantity)
		binary.LittleEndian.PutUint32(data[rowAt+8:], uint32(index+1))
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}
	return path, string(platform)
}

func mockPouchValidator(gameID uint32) error {
	switch gameID {
	case setPouchTestGameID1, setPouchTestGameID2, setPouchTestGameID3:
		return nil
	default:
		return fmt.Errorf("item 0x%08X has no pouch capability", gameID)
	}
}

func TestSetPouchItemsWritesBothPlatformsAndReloads(t *testing.T) {
	cases := []struct {
		platform Platform
		slot     int
	}{
		{PlatformPC, 0},
		{PlatformPS4, 5},
	}

	for _, tc := range cases {
		t.Run(string(tc.platform), func(t *testing.T) {
			source, platformStr := writeSetPouchItemsFixture(t, tc.platform, tc.slot, true, []struct {
				gameID   uint32
				quantity uint32
			}{
				{setPouchTestGameID1, 1},
				{setPouchTestGameID2, 1},
				{setPouchTestGameID3, 1},
			})

			engine := New()
			loaded, err := engine.LoadSave(source, platformStr, "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			sess := engine.sessions[loaded.SaveSessionID]

			quickBefore, err := engine.GetQuickItems(loaded.SaveSessionID, tc.slot)
			if err != nil {
				t.Fatalf("GetQuickItems before: %v", err)
			}

			invBefore, err := engine.GetInventory(loaded.SaveSessionID, tc.slot, "common", 1, 50)
			if err != nil || len(invBefore.Records) < 3 {
				t.Fatalf("GetInventory: %v, len=%d", err, len(invBefore.Records))
			}

			tok1 := invBefore.Records[0].OwnedItemID
			tok2 := invBefore.Records[1].OwnedItemID
			tok3 := invBefore.Records[2].OwnedItemID

			assignments := [6]*string{&tok1, nil, &tok2, nil, &tok3, nil}

			snapshotBefore := make([]byte, len(sess.snapshot.data))
			copy(snapshotBefore, sess.snapshot.data)

			result, err := engine.SetPouchItems(
				loaded.SaveSessionID, tc.slot, assignments, "0", mockPouchValidator)
			if err != nil {
				t.Fatalf("SetPouchItems: %v", err)
			}
			if result.SaveRevision != "1" {
				t.Fatalf("result.SaveRevision = %q, want 1", result.SaveRevision)
			}
			wantGameIDs := [6]uint32{
				setPouchTestGameID1, PouchEmptyGameID,
				setPouchTestGameID2, PouchEmptyGameID,
				setPouchTestGameID3, PouchEmptyGameID,
			}
			if result.GameIDs != wantGameIDs {
				t.Fatalf("result.GameIDs = %v, want %v", result.GameIDs, wantGameIDs)
			}

			quickAfter, err := engine.GetQuickItems(loaded.SaveSessionID, tc.slot)
			if err != nil {
				t.Fatalf("GetQuickItems after: %v", err)
			}
			quickBefore.SaveRevision = quickAfter.SaveRevision
			if !reflect.DeepEqual(quickBefore, quickAfter) {
				t.Errorf("QuickItems changed after SetPouchItems; before=%+v, after=%+v", quickBefore, quickAfter)
			}

			invAfter, err := engine.GetInventory(loaded.SaveSessionID, tc.slot, "common", 1, 50)
			if err != nil {
				t.Fatalf("GetInventory after: %v", err)
			}
			if len(invBefore.Records) != len(invAfter.Records) {
				t.Errorf("Inventory record count changed: before=%d, after=%d", len(invBefore.Records), len(invAfter.Records))
			} else {
				for i := range invBefore.Records {
					b := invBefore.Records[i]
					a := invAfter.Records[i]
					if b.ContainerSection != a.ContainerSection ||
						b.PhysicalIndex != a.PhysicalIndex ||
						b.GaItemHandle != a.GaItemHandle ||
						b.Quantity != a.Quantity ||
						b.AcquisitionIndex != a.AcquisitionIndex {
						t.Errorf("Inventory record [%d] changed: before=%+v, after=%+v", i, b, a)
					}
				}
			}
			if len(invBefore.Records) > 0 && invBefore.Records[0].OwnedItemID == invAfter.Records[0].OwnedItemID {
				t.Errorf("OwnedItemID of first record did not change after revision bump: before=%q, after=%q", invBefore.Records[0].OwnedItemID, invAfter.Records[0].OwnedItemID)
			}

			// Verify scope of modified bytes
			var slotBase int64
			if tc.platform == PlatformPS4 {
				slotBase = pouchItemsPS4SlotDataBase + int64(tc.slot)*pouchItemsPS4SlotStride
			} else {
				slotBase = pouchItemsPCSlotDataBase + int64(tc.slot)*pouchItemsPCSlotStride
			}
			pairAt := slotBase + setPouchAnchorAt + pouchItemsSectionOffset
			countAt := slotBase + setPouchAnchorAt + setPouchProjectileCountOffset
			armamentsOff := countAt + 4 + 17*setPouchProjectileRecordSize
			tailAt := armamentsOff + 0x80

			for i := int64(0); i < int64(len(sess.snapshot.data)); i++ {
				inPair := i >= pairAt && i < pairAt+pouchItemsReadSize
				inTail := i >= tailAt && i < tailAt+24
				if sess.snapshot.data[i] != snapshotBefore[i] {
					if !inPair && !inTail {
						t.Errorf("unexpected byte modified at offset 0x%X (pairAt=0x%X, tailAt=0x%X)", i, pairAt, tailAt)
					}
				}
			}

			// Verify GetPouchItems readback
			pouchState, err := engine.GetPouchItems(loaded.SaveSessionID, tc.slot)
			if err != nil {
				t.Fatalf("GetPouchItems: %v", err)
			}
			if pouchState.Items[1].ItemID != 0 || pouchState.Items[1].EquipIndex != 0xFFFFFFFF {
				t.Errorf("pouchState slot 1 = %+v, want empty", pouchState.Items[1])
			}

			// Write save and reload
			target := filepath.Join(t.TempDir(), "written_pouch.sl2")
			written, err := engine.WriteSave(loaded.SaveSessionID, "1", target)
			if err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			if written.SaveRevision != "2" {
				t.Fatalf("WriteSave revision = %q, want 2", written.SaveRevision)
			}

			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, platformStr, "local")
			if err != nil {
				t.Fatalf("reload written save: %v", err)
			}
			reloadedPouch, err := reloadedEngine.GetPouchItems(reloaded.SaveSessionID, tc.slot)
			if err != nil {
				t.Fatalf("GetPouchItems after reload: %v", err)
			}
			if !reflect.DeepEqual(reloadedPouch.Items, pouchState.Items) {
				t.Errorf("reloaded pouch items = %+v, want %+v", reloadedPouch.Items, pouchState.Items)
			}
		})
	}
}

func TestSetPouchItemsRejectsInconsistentExistingState(t *testing.T) {
	source, platformStr := writeSetPouchItemsFixture(t, PlatformPC, 0, true, []struct {
		gameID   uint32
		quantity uint32
	}{
		{setPouchTestGameID1, 1},
	})

	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	slotBase := pouchItemsPCSlotDataBase
	pairAt := slotBase + setPouchAnchorAt + pouchItemsSectionOffset
	handle, err := gaItemHandleForGameID(setPouchTestGameID1)
	if err != nil {
		t.Fatalf("gaItemHandleForGameID: %v", err)
	}
	// Corrupt slot 0 pair by placing valid handle and index, but leave tail corrupted (e.g. 0x12345678)
	binary.LittleEndian.PutUint32(data[pairAt:], handle)
	binary.LittleEndian.PutUint32(data[pairAt+4:], 0x180)

	if err := os.WriteFile(source, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	engine := New()
	loaded, err := engine.LoadSave(source, platformStr, "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	sess := engine.sessions[loaded.SaveSessionID]

	inv, err := engine.GetInventory(loaded.SaveSessionID, 0, "common", 1, 50)
	if err != nil || len(inv.Records) < 1 {
		t.Fatalf("GetInventory: %v", err)
	}
	tok := inv.Records[0].OwnedItemID

	snapshotBefore := make([]byte, len(sess.snapshot.data))
	copy(snapshotBefore, sess.snapshot.data)

	assignments := [6]*string{&tok, nil, nil, nil, nil, nil}
	_, err = engine.SetPouchItems(loaded.SaveSessionID, 0, assignments, "0", mockPouchValidator)
	if err == nil || !strings.Contains(err.Error(), "inconsistent existing save state") {
		t.Fatalf("expected inconsistent existing save state error; got %v", err)
	}

	if !bytes.Equal(sess.snapshot.data, snapshotBefore) {
		t.Errorf("snapshot bytes modified after rejected mutation")
	}

	info, infoErr := engine.GetSessionInfo(loaded.SaveSessionID)
	if infoErr != nil {
		t.Fatalf("GetSessionInfo: %v", infoErr)
	}
	if info.UnsavedChanges {
		t.Error("rejected mutation marked session dirty")
	}
	if sess.session.revisionString() != "0" {
		t.Errorf("revision = %q, want 0", sess.session.revisionString())
	}
}

func TestSetPouchItemsRejectsInvalidPlansWithoutMutation(t *testing.T) {
	source, platformStr := writeSetPouchItemsFixture(t, PlatformPC, 0, true, []struct {
		gameID   uint32
		quantity uint32
	}{
		{setPouchTestGameID1, 1},
		{setPouchTestGameID2, 0}, // 0 quantity item
		{setPouchTestAccessoryID, 1},
	})

	engine := New()
	loaded, err := engine.LoadSave(source, platformStr, "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	sess := engine.sessions[loaded.SaveSessionID]

	inv, err := engine.GetInventory(loaded.SaveSessionID, 0, "common", 1, 50)
	if err != nil || len(inv.Records) < 3 {
		t.Fatalf("GetInventory: %v", err)
	}

	tokValid := inv.Records[0].OwnedItemID
	tokZeroQty := inv.Records[1].OwnedItemID
	tokAccessory := inv.Records[2].OwnedItemID
	tokInvalid := "oi-" + loaded.SaveSessionID + "-0-12345"

	// Mint a Storage item token to test storage rejection
	tokStorage := sess.session.mintOwnedItemID(ownedItemLocator{
		characterID:      0,
		container:        ownedContainerStorage,
		containerSection: StorageSectionCommon,
		physicalIndex:    0,
	})

	snapshotBefore := make([]byte, len(sess.snapshot.data))
	copy(snapshotBefore, sess.snapshot.data)

	tests := []struct {
		name        string
		assignments [6]*string
		revision    string
		validator   func(gameID uint32) error
		wantError   string
	}{
		{
			name:        "stale revision",
			assignments: [6]*string{&tokValid, nil, nil, nil, nil, nil},
			revision:    "1",
			validator:   mockPouchValidator,
			wantError:   "does not match",
		},
		{
			name:        "duplicate OwnedItemID",
			assignments: [6]*string{&tokValid, &tokValid, nil, nil, nil, nil},
			revision:    "0",
			validator:   mockPouchValidator,
			wantError:   "assigned to both slot 0 and slot 1",
		},
		{
			name:        "unknown OwnedItemID",
			assignments: [6]*string{&tokInvalid, nil, nil, nil, nil, nil},
			revision:    "0",
			validator:   mockPouchValidator,
			wantError:   "unknown ownedItemID",
		},
		{
			name:        "zero quantity item",
			assignments: [6]*string{&tokZeroQty, nil, nil, nil, nil, nil},
			revision:    "0",
			validator:   mockPouchValidator,
			wantError:   "has 0 quantity",
		},
		{
			name:        "storage record",
			assignments: [6]*string{&tokStorage, nil, nil, nil, nil, nil},
			revision:    "0",
			validator:   mockPouchValidator,
			wantError:   "active Inventory record required",
		},
		{
			name:        "non-goods handle",
			assignments: [6]*string{&tokAccessory, nil, nil, nil, nil, nil},
			revision:    "0",
			wantError:   "has non-goods handle 0xA0000064",
		},
		{
			name:        "catalog validator rejection",
			assignments: [6]*string{&tokValid, nil, nil, nil, nil, nil},
			revision:    "0",
			validator: func(gameID uint32) error {
				return errors.New("no pouch capability")
			},
			wantError: "no pouch capability",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.SetPouchItems(
				loaded.SaveSessionID, 0, tt.assignments, tt.revision, tt.validator)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want error containing %q", err, tt.wantError)
			}
			if !reflect.DeepEqual(result, SetPouchItemsResult{}) {
				t.Errorf("result = %+v, want zero value", result)
			}
			if !bytes.Equal(sess.snapshot.data, snapshotBefore) {
				t.Errorf("snapshot bytes modified after rejected mutation")
			}
			info, infoErr := engine.GetSessionInfo(loaded.SaveSessionID)
			if infoErr != nil {
				t.Fatalf("GetSessionInfo: %v", infoErr)
			}
			if info.UnsavedChanges {
				t.Error("rejected mutation marked session dirty")
			}
			if sess.session.revisionString() != "0" {
				t.Errorf("revision = %q, want 0", sess.session.revisionString())
			}
		})
	}
}
