package saveengine

import "testing"

func TestGetItemCapacityReportsCostsWithoutMutatingSession(t *testing.T) {
	content := addItemTestFixture{
		platform: PlatformPC,
		slot:     2,
		common: []addItemTestRow{{
			index: 4, handle: addItemTestGoodsHandle, rawQuantity: 3, acquisition: 969,
		}},
		commonCount:     1,
		nextEquipIndex:  12,
		nextAcquisition: 970,
		gaItemData:      []uint32{addItemTestGoodsID},
	}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	sessionID := loaded.SaveSessionID
	before := addItemTestSlotData(t, engine, sessionID, PlatformPC, content.slot)

	topUp, err := engine.GetItemCapacity(
		sessionID, content.slot, ItemCapacityDestinationInventory,
		addItemTestGoodsID, 5, false, 40, 600)
	if err != nil {
		t.Fatalf("GetItemCapacity top-up: %v", err)
	}
	if !topUp.Active || !topUp.CanFit || topUp.LimitingFactor != "" ||
		topUp.CurrentQuantity != 3 || topUp.QuantityAfter != 8 ||
		topUp.PhysicalRecordsRequired != 0 || topUp.GaItemDataEntriesRequired != 0 {
		t.Errorf("top-up capacity = %+v", topUp)
	}

	created, err := engine.GetItemCapacity(
		sessionID, content.slot, ItemCapacityDestinationStorage,
		addItemTestOtherID, 9, false, 600, 600)
	if err != nil {
		t.Fatalf("GetItemCapacity new Storage record: %v", err)
	}
	if !created.CanFit || created.PhysicalRecordsRequired != 1 ||
		created.GaItemDataEntriesRequired != 1 || created.FreePhysicalRecords != 0x780 ||
		created.FreeGaItemDataEntries != addItemTestGaItemMaxCount-1 {
		t.Errorf("new-record capacity = %+v", created)
	}

	after := addItemTestSlotData(t, engine, sessionID, PlatformPC, content.slot)
	if ranges := addItemTestChangedRanges(t, before, after); len(ranges) != 0 {
		t.Fatalf("GetItemCapacity changed slot ranges %v", ranges)
	}
	engine.mutex.Lock()
	session := engine.sessions[sessionID].session
	if session.revision != 0 || session.dirty || len(session.ownedByID) != 0 ||
		len(session.ownedByLocator) != 0 || session.ownedSeq != 0 {
		t.Errorf("getter mutated session state: revision=%d dirty=%v ids=%d locators=%d seq=%d",
			session.revision, session.dirty, len(session.ownedByID),
			len(session.ownedByLocator), session.ownedSeq)
	}
	engine.mutex.Unlock()
}

func TestGetItemCapacityReportsLimitsAndInactiveSlots(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, addItemTestFixture{
		platform: PlatformPS4,
		slot:     1,
		common: []addItemTestRow{{
			index: 0, handle: addItemTestGoodsHandle, rawQuantity: 39, acquisition: 969,
		}},
		commonCount: 1,
		gaItemData:  []uint32{addItemTestGoodsID},
	}), string(PlatformPS4))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	limited, err := engine.GetItemCapacity(
		loaded.SaveSessionID, 1, ItemCapacityDestinationInventory,
		addItemTestGoodsID, 2, false, 40, 600)
	if err != nil {
		t.Fatalf("GetItemCapacity limited: %v", err)
	}
	if limited.CanFit || limited.LimitingFactor != ItemCapacityLimitPerRecord {
		t.Errorf("limited capacity = %+v", limited)
	}

	inactiveEngine := New()
	inactive, err := inactiveEngine.LoadSave(writeAddItemFixture(t, addItemTestFixture{
		platform: PlatformPC, slot: 4, inactive: true,
	}), string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave inactive: %v", err)
	}
	result, err := inactiveEngine.GetItemCapacity(
		inactive.SaveSessionID, 4, ItemCapacityDestinationInventory,
		addItemTestGoodsID, 1, false, 40, 600)
	if err != nil {
		t.Fatalf("GetItemCapacity inactive: %v", err)
	}
	if result.Active || result.CanFit || result.SaveRevision != "0" {
		t.Errorf("inactive capacity = %+v", result)
	}
}
