package migration

import "testing"

func TestLegacyMetadataExactCoverage(t *testing.T) {
	snapshot := collectLegacySnapshot()
	counts := struct {
		sortKeys              int
		requiredContainers    int
		containers            int
		containerPickupFlags  int
		containerVendorFlags  int
		bolsteringItemKeys    int
		bolsteringPickupFlags int
		worldPickupFlags      int
		companionItemKeys     int
		companionEventFlags   int
		equipLoadModifiers    int
		gameInventoryKnown    int
		gameStorageKnown      int
		unlocksByKind         map[string]int
	}{
		unlocksByKind: make(map[string]int),
	}

	for _, item := range snapshot.Items {
		if item.SortKey != nil {
			counts.sortKeys++
		}
		if item.Acquisition.RequiredContainerID != nil {
			counts.requiredContainers++
		}
		if item.Acquisition.IsContainer {
			counts.containers++
		}
		counts.containerPickupFlags += len(item.Acquisition.ContainerPickupFlags)
		counts.containerVendorFlags += len(item.Acquisition.ContainerVendorFlags)
		if len(item.Acquisition.BolsteringPickupFlags) > 0 {
			counts.bolsteringItemKeys++
		}
		counts.bolsteringPickupFlags += len(item.Acquisition.BolsteringPickupFlags)
		if item.Acquisition.WorldPickupFlagID != nil {
			counts.worldPickupFlags++
		}
		if len(item.Acquisition.CompanionEventFlagIDs) > 0 {
			counts.companionItemKeys++
		}
		counts.companionEventFlags += len(item.Acquisition.CompanionEventFlagIDs)
		if item.EquipLoad != nil {
			counts.equipLoadModifiers++
		}
		if item.GameMaxInventoryKnown {
			counts.gameInventoryKnown++
		}
		if item.GameMaxStorageKnown {
			counts.gameStorageKnown++
		}
		for _, unlock := range item.Unlocks {
			counts.unlocksByKind[unlock.Kind]++
		}
	}

	assertCount(t, "sort keys", counts.sortKeys, 1614)
	assertCount(t, "required containers", counts.requiredContainers, 59)
	assertCount(t, "container items", counts.containers, 4)
	assertCount(t, "container pickup flags", counts.containerPickupFlags, 50)
	assertCount(t, "container vendor flags", counts.containerVendorFlags, 1)
	assertCount(t, "bolstering item keys", counts.bolsteringItemKeys, 5)
	assertCount(t, "bolstering pickup flags", counts.bolsteringPickupFlags, 125)
	assertCount(t, "world pickup flags", counts.worldPickupFlags, 318)
	assertCount(t, "companion item keys", counts.companionItemKeys, 6)
	assertCount(t, "companion event flags", counts.companionEventFlags, 9)
	assertCount(t, "equip-load modifiers", counts.equipLoadModifiers, 11)
	assertCount(t, "direct ItemData inventory limits", counts.gameInventoryKnown, 0)
	assertCount(t, "direct ItemData storage limits", counts.gameStorageKnown, 0)

	wantUnlocks := map[string]int{
		"ash_of_war":   116,
		"bell_bearing": 62,
		"cookbook":     104,
		"map":          24,
		"whetblade":    6,
	}
	if len(counts.unlocksByKind) != len(wantUnlocks) {
		t.Fatalf("unlock kind count = %d, want %d: %#v", len(counts.unlocksByKind), len(wantUnlocks), counts.unlocksByKind)
	}
	for kind, want := range wantUnlocks {
		assertCount(t, "unlock "+kind, counts.unlocksByKind[kind], want)
	}
}

func assertCount(t *testing.T, label string, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %d, want %d", label, got, want)
	}
}
