package core

import (
	"encoding/binary"
	"testing"
)

// helper to build a minimal SaveSlot fixture with a Storage section of 1920 records
func buildStorageContractFixture(t *testing.T, storageRecords []InventoryItem, invRecords []InventoryItem, nextEquip, nextAcq uint32) *SaveSlot {
	t.Helper()
	storageBoxOff := 0x1000
	storageStart := storageBoxOff + StorageHeaderSkip
	nextEquipOff := storageStart + StorageNextEquipIdxRel
	nextAcqOff := storageStart + StorageNextAcqSortRel
	storageBufSize := nextAcqOff + 8

	invStart := storageBufSize + 0x100
	invNextEquipOff := invStart + CommonItemCount*InvRecordLen + InvKeyCountHeader + KeyItemCount*InvRecordLen
	invNextAcqOff := invNextEquipOff + 4
	totalSize := invNextAcqOff + 8

	slot := &SaveSlot{
		Version:          1,
		StorageBoxOffset: storageBoxOff,
		MagicOffset:      invStart - InvStartFromMagic,
		Data:             make([]byte, totalSize),
		GaMap:            make(map[uint32]uint32),
	}

	// Setup Storage binary
	nonEmptyStorage := 0
	for i, rec := range storageRecords {
		if rec.GaItemHandle != GaHandleEmpty && rec.GaItemHandle != GaHandleInvalid {
			nonEmptyStorage++
			off := storageStart + i*InvRecordLen
			binary.LittleEndian.PutUint32(slot.Data[off:], rec.GaItemHandle)
			binary.LittleEndian.PutUint32(slot.Data[off+4:], rec.Quantity)
			binary.LittleEndian.PutUint32(slot.Data[off+8:], rec.Index)
			slot.Storage.CommonItems = append(slot.Storage.CommonItems, rec)
		}
	}
	binary.LittleEndian.PutUint32(slot.Data[storageBoxOff:], uint32(nonEmptyStorage))
	binary.LittleEndian.PutUint32(slot.Data[nextEquipOff:], nextEquip)
	binary.LittleEndian.PutUint32(slot.Data[nextAcqOff:], nextAcq)
	slot.Storage.NextEquipIndex = nextEquip
	slot.Storage.NextAcquisitionSortId = nextAcq
	slot.Storage.nextEquipIndexOff = nextEquipOff
	slot.Storage.nextAcqSortIdOff = nextAcqOff

	// Setup Inventory binary
	slot.Inventory.CommonItems = make([]InventoryItem, CommonItemCount)
	for i := 0; i < CommonItemCount; i++ {
		slot.Inventory.CommonItems[i] = InventoryItem{GaItemHandle: 0, Quantity: 0, Index: uint32(i)}
	}
	nonEmptyInv := 0
	for i, rec := range invRecords {
		if rec.GaItemHandle != GaHandleEmpty && rec.GaItemHandle != GaHandleInvalid {
			nonEmptyInv++
			off := invStart + i*InvRecordLen
			binary.LittleEndian.PutUint32(slot.Data[off:], rec.GaItemHandle)
			binary.LittleEndian.PutUint32(slot.Data[off+4:], rec.Quantity)
			binary.LittleEndian.PutUint32(slot.Data[off+8:], rec.Index)
			slot.Inventory.CommonItems[i] = rec
		}
	}
	countOff := slot.MagicOffset + InvStartFromMagic - 4
	binary.LittleEndian.PutUint32(slot.Data[countOff:], uint32(nonEmptyInv))
	binary.LittleEndian.PutUint32(slot.Data[invNextEquipOff:], 500)
	binary.LittleEndian.PutUint32(slot.Data[invNextAcqOff:], 1000)
	slot.Inventory.NextEquipIndex = 500
	slot.Inventory.NextAcquisitionSortId = 1000
	slot.Inventory.nextEquipIndexOff = invNextEquipOff
	slot.Inventory.nextAcqSortIdOff = invNextAcqOff

	return slot
}

// Regression Test 1:
// Transfer do Storage na tablicy z niezerowymi rekordami przydziela indeks akwizycji
// zgodnie z polityka parzystosci (stride 2), zapewniajac zerowa liczbe kolizji kubelkow
// (acq>>1) w calej tablicy oraz ustawia NextAcquisitionSortId = acq/2 + 1.
//
// UWAGA: Kontraktem gry jest BRAK KOLIZJI KUBELKOW (acq>>1), a nie bezwzgledna parzystosc.
// Zmierzone na natywnym pliku gry (tmp/storage-test/t2-f4-refill.sl2, slot 1, 1230 rekordow):
//   - rekord fiz. 192   handle 0xB003A1B1  Index 1965  (nieparzysty)
//   - rekord fiz. 498   handle 0xC080009C  Index 2573  (nieparzysty)
//   - rekord fiz. 1008  handle 0x908001DB  Index 3603  (nieparzysty)
//   - kolizje kubelkow: 0
//
// Gra sporadycznie zapisuje nieparzyste indeksy, co jest stanem legalnym. Nasz kod stosuje
// polityke zapisu parzystych indeksow ze stride 2 jako bezpieczny podzbior stanow legalnych,
// gwarantujacy unikalnosc kubelkow.
func TestStorageContract_TransferToStorage_Stride2Policy_NoBucketCollision(t *testing.T) {
	// Storage ma 1 rekord z acq=1572 (bucket 786).
	// Inventory ma 1 bron z acq=434.
	storageRecords := []InventoryItem{
		{GaItemHandle: 0x80000001, Quantity: 1, Index: 1572},
	}
	invRecords := []InventoryItem{
		{GaItemHandle: 0x80000002, Quantity: 1, Index: 434},
	}
	slot := buildStorageContractFixture(t, storageRecords, invRecords, 128, 787)

	res, err := MoveItemsBetweenContainers(slot, []uint32{0x80000002}, TransferToStorage, nil)
	if err != nil {
		t.Fatalf("MoveItemsBetweenContainers: %v", err)
	}
	if res.Moved != 1 {
		t.Fatalf("MoveItemsBetweenContainers moved %d, want 1", res.Moved)
	}

	// Sprawdzenie w Storage
	storageStart := slot.StorageBoxOffset + StorageHeaderSkip
	var acqs []uint32
	buckets := make(map[uint32]int)
	for i := 0; i < StorageCommonCount; i++ {
		off := storageStart + i*InvRecordLen
		h := binary.LittleEndian.Uint32(slot.Data[off:])
		if h != GaHandleEmpty && h != GaHandleInvalid {
			acq := binary.LittleEndian.Uint32(slot.Data[off+8:])
			acqs = append(acqs, acq)
			b := acq >> 1
			buckets[b]++
			// Weryfikacja naszej polityki zapisu (stride-2 parzyste)
			if acq%2 != 0 {
				t.Errorf("Storage record %d has odd acquisition index %d (writer policy requires even stride-2)", i, acq)
			}
		}
	}

	// Weryfikacja kontraktu gry: brak kolizji kubelkow
	for b, count := range buckets {
		if count > 1 {
			t.Errorf("Bucket collision in bucket %d: %d records share it", b, count)
		}
	}

	// Nowy rekord na slocie 1 powinien miec acq=1574 (maxAcq 1572 + 2)
	newRecAcq := binary.LittleEndian.Uint32(slot.Data[storageStart+1*InvRecordLen+8:])
	if newRecAcq != 1574 {
		t.Errorf("New record acq: got %d, want 1574", newRecAcq)
	}

	// NextAcquisitionSortId powinien wynosic 1574/2 + 1 = 788
	wantNextAcq := uint32(788)
	if slot.Storage.NextAcquisitionSortId != wantNextAcq {
		t.Errorf("Storage.NextAcquisitionSortId: got %d, want %d", slot.Storage.NextAcquisitionSortId, wantNextAcq)
	}
	rawNextAcq := binary.LittleEndian.Uint32(slot.Data[slot.Storage.nextAcqSortIdOff:])
	if rawNextAcq != wantNextAcq {
		t.Errorf("Binary NextAcquisitionSortId: got %d, want %d", rawNextAcq, wantNextAcq)
	}
}

// Regression Test 2:
// Dodanie rekordu do tablicy Z DZIURA nie zmienia NextEquipIndex, jesli rekord wpadl w dziure
// ponizej dotychczasowego last_occupied_index.
func TestStorageContract_AddIntoHole_PreservesNextEquipIndex(t *testing.T) {
	// Storage ma rekordy na indeksie fizycznym 0 i 2 (dziura na 1).
	// last_occupied_index = 2, wiec NextEquipIndex = 128 + 2 = 130.
	storageRecords := make([]InventoryItem, 3)
	storageRecords[0] = InventoryItem{GaItemHandle: 0x80000001, Quantity: 1, Index: 2}
	storageRecords[1] = InventoryItem{GaItemHandle: 0, Quantity: 0, Index: 0} // dziura na 1
	storageRecords[2] = InventoryItem{GaItemHandle: 0x80000002, Quantity: 1, Index: 4}

	slot := buildStorageContractFixture(t, storageRecords, nil, 130, 3)

	const newHandle = uint32(0x80000003)
	if err := addToInventory(slot, newHandle, 1, true, false, false); err != nil {
		t.Fatalf("addToInventory: %v", err)
	}

	// Rekord powinien trafic w dziure na indeksie fizycznym 1
	storageStart := slot.StorageBoxOffset + StorageHeaderSkip
	h1 := binary.LittleEndian.Uint32(slot.Data[storageStart+1*InvRecordLen:])
	if h1 != newHandle {
		t.Fatalf("Record on physical slot 1: got handle 0x%08X, want 0x%08X", h1, newHandle)
	}

	// Poniewaz last_occupied_index to nadal 2, NextEquipIndex musi pozostac 130 (128 + 2)
	const wantEquip = uint32(130)
	if slot.Storage.NextEquipIndex != wantEquip {
		t.Errorf("Storage.NextEquipIndex: got %d, want %d", slot.Storage.NextEquipIndex, wantEquip)
	}
	rawEquip := binary.LittleEndian.Uint32(slot.Data[slot.Storage.nextEquipIndexOff:])
	if rawEquip != wantEquip {
		t.Errorf("Binary NextEquipIndex: got %d, want %d", rawEquip, wantEquip)
	}
}

// Regression Test 3:
// Dodanie rekordu przedluzajace tablice podnosi NextEquipIndex do 128 + nowy last_occupied_index.
func TestStorageContract_AddExtendingArray_AdvancesNextEquipIndex(t *testing.T) {
	// Storage ma rekordy na indeksach fizycznych 0, 1, 2 (gesta tablica).
	// last_occupied_index = 2, NextEquipIndex = 130.
	storageRecords := []InventoryItem{
		{GaItemHandle: 0x80000001, Quantity: 1, Index: 2},
		{GaItemHandle: 0x80000002, Quantity: 1, Index: 4},
		{GaItemHandle: 0x80000003, Quantity: 1, Index: 6},
	}
	slot := buildStorageContractFixture(t, storageRecords, nil, 130, 4)

	const newHandle = uint32(0x80000004)
	if err := addToInventory(slot, newHandle, 1, true, false, false); err != nil {
		t.Fatalf("addToInventory: %v", err)
	}

	// Rekord trafia na indeks fizyczny 3 (nowy last_occupied_index = 3).
	// NextEquipIndex = 128 + 3 = 131.
	const wantEquip = uint32(131)
	if slot.Storage.NextEquipIndex != wantEquip {
		t.Errorf("Storage.NextEquipIndex: got %d, want %d", slot.Storage.NextEquipIndex, wantEquip)
	}
	rawEquip := binary.LittleEndian.Uint32(slot.Data[slot.Storage.nextEquipIndexOff:])
	if rawEquip != wantEquip {
		t.Errorf("Binary NextEquipIndex: got %d, want %d", rawEquip, wantEquip)
	}
}

// Regression Test 4:
// Wzrost qty istniejacego rekordu Storage odswieza acq i podnosi licznik.
func TestStorageContract_QuantityIncrease_RefreshesAcquisitionIndex(t *testing.T) {
	t.Run("addToInventory_topup", func(t *testing.T) {
		storageRecords := []InventoryItem{
			{GaItemHandle: 0xB0000100, Quantity: 5, Index: 2},
			{GaItemHandle: 0x80000001, Quantity: 1, Index: 10},
		}
		slot := buildStorageContractFixture(t, storageRecords, nil, 129, 6)

		// Zwiekszenie qty z 5 do 15 dla 0xB0000100
		if err := addToInventory(slot, 0xB0000100, 15, true, false, false); err != nil {
			t.Fatalf("addToInventory: %v", err)
		}

		storageStart := slot.StorageBoxOffset + StorageHeaderSkip
		// Pozycja fizyczna slot 0 bez zmian
		h0 := binary.LittleEndian.Uint32(slot.Data[storageStart:])
		qty0 := binary.LittleEndian.Uint32(slot.Data[storageStart+4:])
		acq0 := binary.LittleEndian.Uint32(slot.Data[storageStart+8:])

		if h0 != 0xB0000100 || qty0 != 15 {
			t.Fatalf("Record 0: got handle=0x%08X qty=%d, want handle=0xB0000100 qty=15", h0, qty0)
		}
		// Max acq byl 10, wiec nowy acq to 12
		if acq0 != 12 {
			t.Errorf("Refreshed record acq: got %d, want 12 (maxAcq 10 + 2)", acq0)
		}
		// NextAcquisitionSortId = 12/2 + 1 = 7
		if slot.Storage.NextAcquisitionSortId != 7 {
			t.Errorf("Storage.NextAcquisitionSortId: got %d, want 7", slot.Storage.NextAcquisitionSortId)
		}
	})

	t.Run("transfer_merge_to_storage", func(t *testing.T) {
		storageRecords := []InventoryItem{
			{GaItemHandle: 0xB0000200, Quantity: 5, Index: 2},
			{GaItemHandle: 0x80000001, Quantity: 1, Index: 10},
		}
		invRecords := []InventoryItem{
			{GaItemHandle: 0xB0000200, Quantity: 5, Index: 434},
		}
		slot := buildStorageContractFixture(t, storageRecords, invRecords, 129, 6)

		opts := &TransferOptions{
			DestCaps: map[uint32]uint32{0xB0000200: 99},
		}
		res, err := MoveItemsBetweenContainers(slot, []uint32{0xB0000200}, TransferToStorage, opts)
		if err != nil {
			t.Fatalf("MoveItemsBetweenContainers: %v", err)
		}
		if res.Moved != 1 {
			t.Fatalf("Moved %d, want 1", res.Moved)
		}

		storageStart := slot.StorageBoxOffset + StorageHeaderSkip
		h0 := binary.LittleEndian.Uint32(slot.Data[storageStart:])
		qty0 := binary.LittleEndian.Uint32(slot.Data[storageStart+4:])
		acq0 := binary.LittleEndian.Uint32(slot.Data[storageStart+8:])

		if h0 != 0xB0000200 || qty0 != 10 {
			t.Fatalf("Record 0: got handle=0x%08X qty=%d, want handle=0xB0000200 qty=10", h0, qty0)
		}
		if acq0 != 12 {
			t.Errorf("Refreshed record acq after transfer: got %d, want 12", acq0)
		}
		if slot.Storage.NextAcquisitionSortId != 7 {
			t.Errorf("Storage.NextAcquisitionSortId after transfer: got %d, want 7", slot.Storage.NextAcquisitionSortId)
		}
	})
}

// Regression Test 5:
// Spadek qty rekordu Storage NIE zmienia acq (regula 4 kontraktu).
func TestStorageContract_QuantityDecrease_PreservesAcquisitionIndex(t *testing.T) {
	storageRecords := []InventoryItem{
		{GaItemHandle: 0xB0000300, Quantity: 10, Index: 1572},
	}
	slot := buildStorageContractFixture(t, storageRecords, nil, 128, 787)

	opts := &TransferOptions{
		DestCaps: map[uint32]uint32{0xB0000300: 3}, // dest cap = 3, partial transfer of 3 units
	}
	res, err := MoveItemsBetweenContainers(slot, []uint32{0xB0000300}, TransferToInventory, opts)
	if err != nil {
		t.Fatalf("MoveItemsBetweenContainers: %v", err)
	}
	if res.Moved != 1 {
		t.Fatalf("Moved %d, want 1", res.Moved)
	}

	storageStart := slot.StorageBoxOffset + StorageHeaderSkip
	h0 := binary.LittleEndian.Uint32(slot.Data[storageStart:])
	qty0 := binary.LittleEndian.Uint32(slot.Data[storageStart+4:])
	acq0 := binary.LittleEndian.Uint32(slot.Data[storageStart+8:])

	if h0 != 0xB0000300 || qty0 != 7 {
		t.Fatalf("Record 0: got handle=0x%08X qty=%d, want qty=7", h0, qty0)
	}
	if acq0 != 1572 {
		t.Errorf("Acquisition index changed after qty decrease: got %d, want 1572", acq0)
	}
	if slot.Storage.NextAcquisitionSortId != 787 {
		t.Errorf("Storage.NextAcquisitionSortId changed after qty decrease: got %d, want 787", slot.Storage.NextAcquisitionSortId)
	}
}

// Regression Test 6:
// Usuniecie rekordu zostawia dziure i nie rusza zadnego innego rekordu ani zadnego z dwoch licznikow.
func TestStorageContract_RemoveRecord_LeavesHoleAndPreservesCounters(t *testing.T) {
	storageRecords := []InventoryItem{
		{GaItemHandle: 0x80000001, Quantity: 1, Index: 2},
		{GaItemHandle: 0x80000002, Quantity: 1, Index: 4},
		{GaItemHandle: 0x80000003, Quantity: 1, Index: 6},
	}
	slot := buildStorageContractFixture(t, storageRecords, nil, 130, 4)

	// Usuniecie rekordu ze srodka (handle 0x80000002 na slocie 1)
	fp := fingerprintInventoryItem(storageRecords[1])
	if err := RemoveInventoryRecordAt(slot, repairScopeStorageCommon, 1, fp); err != nil {
		t.Fatalf("RemoveInventoryRecordAt: %v", err)
	}

	storageStart := slot.StorageBoxOffset + StorageHeaderSkip
	// Slot 0 nienaruszony
	h0 := binary.LittleEndian.Uint32(slot.Data[storageStart:])
	acq0 := binary.LittleEndian.Uint32(slot.Data[storageStart+8:])
	if h0 != 0x80000001 || acq0 != 2 {
		t.Errorf("Slot 0 mutated: handle=0x%08X acq=%d", h0, acq0)
	}

	// Slot 1 wyzerowany w miejscu (dziura)
	h1 := binary.LittleEndian.Uint32(slot.Data[storageStart+1*InvRecordLen:])
	qty1 := binary.LittleEndian.Uint32(slot.Data[storageStart+1*InvRecordLen+4:])
	idx1 := binary.LittleEndian.Uint32(slot.Data[storageStart+1*InvRecordLen+8:])
	if h1 != 0 || qty1 != 0 || idx1 != 0 {
		t.Errorf("Slot 1 not cleared: handle=0x%08X qty=%d idx=%d", h1, qty1, idx1)
	}

	// Slot 2 nienaruszony (dziura nie przesunela rekordu!)
	h2 := binary.LittleEndian.Uint32(slot.Data[storageStart+2*InvRecordLen:])
	acq2 := binary.LittleEndian.Uint32(slot.Data[storageStart+2*InvRecordLen+8:])
	if h2 != 0x80000003 || acq2 != 6 {
		t.Errorf("Slot 2 mutated/shifted: handle=0x%08X acq=%d", h2, acq2)
	}

	// Naglowek = 2 (niepuste rekordy)
	count := binary.LittleEndian.Uint32(slot.Data[slot.StorageBoxOffset:])
	if count != 2 {
		t.Errorf("Storage header count: got %d, want 2", count)
	}

	// Liczniki nietkniete
	if slot.Storage.NextEquipIndex != 130 {
		t.Errorf("NextEquipIndex changed on removal: got %d, want 130", slot.Storage.NextEquipIndex)
	}
	if slot.Storage.NextAcquisitionSortId != 4 {
		t.Errorf("NextAcquisitionSortId changed on removal: got %d, want 4", slot.Storage.NextAcquisitionSortId)
	}
}
