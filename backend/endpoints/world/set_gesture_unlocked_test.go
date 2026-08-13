package world

import (
	"fmt"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func gestureKeyForSlot(t *testing.T, slotID uint32) string {
	t.Helper()
	for _, resource := range storedGestureResources(t) {
		if resource.Item == nil || resource.Item.Gesture == nil {
			continue
		}
		for _, slot := range resource.Item.Gesture.Slots {
			if slot.SlotID.Known && slot.SlotID.Value == slotID {
				return resource.Key
			}
		}
	}
	t.Fatalf("catalog gesture slot %d not found", slotID)
	return ""
}

func setGestureEndpointRecords(values ...uint32) []uint32 {
	records := make([]uint32, getGesturesSlotCount)
	for index := range records {
		records[index] = getGesturesEmptySentinel
	}
	copy(records, values)
	return records
}

func gestureEntryBySlot(t *testing.T, result GetGesturesResult, slotID uint32) GestureEntry {
	t.Helper()
	for _, entry := range result.Gestures {
		if entry.SlotID == slotID {
			return entry
		}
	}
	t.Fatalf("GetGestures carries no slot %d", slotID)
	return GestureEntry{}
}

func TestSetGestureUnlockedAssignsCatalogGestureState(t *testing.T) {
	const slotID = uint32(229)
	engine, sessionID := loadGesturesSession(
		t, setGestureEndpointRecords(slotID-1, 4242, 0), true)
	gameCatalog := newGesturesCatalog(t)
	key := gestureKeyForSlot(t, slotID)

	result, err := SetGestureUnlocked(
		engine, gameCatalog, sessionID, getGesturesSlot,
		"item", key, true, "0")
	if err != nil {
		t.Fatalf("SetGestureUnlocked: %v", err)
	}
	want := SetGestureUnlockedResult{
		SaveSessionID: sessionID,
		SaveRevision:  "1",
		CharacterID:   getGesturesSlot,
		GestureKind:   schema.ResourceKindItem,
		GestureKey:    key,
		Unlocked:      true,
	}
	if result != want {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	gestures, err := GetGestures(engine, gameCatalog, sessionID, getGesturesSlot, "")
	if err != nil {
		t.Fatalf("GetGestures after unlock: %v", err)
	}
	if !gestureEntryBySlot(t, gestures, slotID).Unlocked {
		t.Errorf("gesture slot %d is locked after unlock", slotID)
	}

	result, err = SetGestureUnlocked(
		engine, gameCatalog, sessionID, getGesturesSlot,
		"item", key, false, "1")
	if err != nil {
		t.Fatalf("SetGestureUnlocked lock: %v", err)
	}
	if result.SaveRevision != "2" || result.Unlocked {
		t.Errorf("lock result = %+v", result)
	}
	gestures, err = GetGestures(engine, gameCatalog, sessionID, getGesturesSlot, "")
	if err != nil {
		t.Fatalf("GetGestures after lock: %v", err)
	}
	if gestureEntryBySlot(t, gestures, slotID).Unlocked {
		t.Errorf("gesture slot %d is unlocked after lock", slotID)
	}
}

func TestSetGestureUnlockedTreatsRingOfMiquellaAsOneLogicalResource(t *testing.T) {
	t.Run("unlocks earned alias when pre-order grant is absent", func(t *testing.T) {
		engine, sessionID := loadGesturesSession(
			t, setGestureEndpointRecords(232, 4242), true)
		gameCatalog := newGesturesCatalog(t)

		result, err := SetGestureUnlocked(
			engine, gameCatalog, sessionID, getGesturesSlot,
			"item", getGesturesMultiSlotKey, true, "0")
		if err != nil {
			t.Fatalf("SetGestureUnlocked: %v", err)
		}
		if result.GestureKey != getGesturesMultiSlotKey || !result.Unlocked {
			t.Errorf("result = %+v", result)
		}

		gestures, err := GetGestures(engine, gameCatalog, sessionID, getGesturesSlot, "")
		if err != nil {
			t.Fatalf("GetGestures: %v", err)
		}
		if gestureEntryBySlot(t, gestures, getGesturesMultiSlotFirst).Unlocked {
			t.Error("unlock created protected pre-order slot 227")
		}
		if !gestureEntryBySlot(t, gestures, getGesturesMultiSlotSecond).Unlocked {
			t.Error("unlock did not create earned slot 233")
		}
	})

	t.Run("cannot lock pre-order grant", func(t *testing.T) {
		engine, sessionID := loadGesturesSession(
			t, setGestureEndpointRecords(227, 233, 4242), true)
		_, err := SetGestureUnlocked(
			engine, newGesturesCatalog(t), sessionID, getGesturesSlot,
			"item", getGesturesMultiSlotKey, false, "0")
		if err == nil || err.Error() !=
			"Ring of Miquella is granted by pre-order slot 227 and cannot be locked" {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestSetGestureUnlockedRejectsMissingBackendsAndInvalidResources(t *testing.T) {
	engine, sessionID := loadGesturesSession(
		t, setGestureEndpointRecords(228), true)
	gameCatalog := newGesturesCatalog(t)
	validKey := gestureKeyForSlot(t, 229)

	cases := map[string]struct {
		engine      *saveengine.Engine
		gameCatalog *gamecatalog.Catalog
		kind        string
		key         string
		want        string
	}{
		"nil engine": {nil, gameCatalog, "item", validKey,
			"save engine is not available"},
		"nil catalog": {engine, nil, "item", validKey,
			"game catalog is not available"},
		"unknown kind": {engine, gameCatalog, "gesture", validKey,
			`unknown resource kind "gesture"`},
		"unknown key": {engine, gameCatalog, "item", "UNKNOWN",
			`unknown resource key "UNKNOWN" in kind "item"`},
		"non-gesture item": {engine, gameCatalog, "item", "000F4240",
			`resource kind "item" key "000F4240" has item family "weapon", want "gesture"`},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := SetGestureUnlocked(
				testCase.engine,
				testCase.gameCatalog,
				sessionID,
				getGesturesSlot,
				testCase.kind,
				testCase.key,
				true,
				"0",
			)
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
		})
	}

	info, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Errorf("session after rejections = %+v, want clean", info)
	}
}

func TestGestureMutationSlotIDRejectsAnUnconfirmedAliasSet(t *testing.T) {
	resource, err := newGesturesCatalog(t).ResourceByKindAndKey(
		schema.ResourceKindItem, gestureKeyForSlot(t, 229))
	if err != nil {
		t.Fatalf("ResourceByKindAndKey: %v", err)
	}
	resource.Item.Gesture.Slots = append(
		resource.Item.Gesture.Slots,
		schema.GestureSlotRecord{SlotID: schema.Fact[uint32]{Known: true, Value: 231}},
	)

	_, err = gestureMutationSlotID(resource)
	want := fmt.Sprintf(
		"gesture %q declares 2 save slots; only one canonical slot or the alias pair 227/233 is supported",
		resource.Key)
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v", err)
	}
}
