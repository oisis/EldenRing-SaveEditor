package saveengine

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestReadGaItemMapUsesLegacyRecordCount(t *testing.T) {
	const characterID = 0
	base := ps4SlotDataOffset + characterID*ps4SlotSize
	markerAt := base + gaItemTableOffset + gaItemOldRecordCount*gaItemRecordSize
	data := make([]byte, base+ps4SlotSize)
	binary.LittleEndian.PutUint32(data[base:], gaItemVersionBreak)
	copy(data[markerAt:], gaItemAnchor)

	entries, err := readGaItemMap(&codec{data: data}, PlatformPS4, characterID)
	if err != nil {
		t.Fatalf("readGaItemMap: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("entries = %v, want an empty map", entries)
	}
}

func TestResolveGaItemHandleRejectsMissingInstanceRecord(t *testing.T) {
	_, err := resolveGaItemHandle(nil, gaItemArmorHandle|1)
	if err == nil {
		t.Fatal("resolveGaItemHandle accepted an armor handle without a GaItem record")
	}
	if !strings.Contains(err.Error(), "has no record") {
		t.Errorf("error = %q, want a missing-record error", err)
	}
}
