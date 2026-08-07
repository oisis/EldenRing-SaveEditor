package migration

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

func TestLegacySnapshotCopiesMutableSourceData(t *testing.T) {
	const weaponID = uint32(0x0016E360)
	const gestureSlotID = uint32(111)
	const containerID = uint32(0x4000251C)

	first := collectLegacySnapshot()
	weapon := findSeed(t, first.Items, weaponID)
	if len(weapon.Flags) == 0 || weapon.WeaponStats == nil || len(weapon.WeaponStats.Warnings) == 0 ||
		weapon.Text == nil || weapon.Text.Caption == "" {
		t.Fatalf("weapon copy fixture is incomplete: %#v", weapon)
	}
	originalCaption := weapon.Text.Caption
	weapon.Flags[0] = "mutated"
	weapon.WeaponStats.Warnings[0] = "mutated"
	weapon.Text.Caption = "mutated"
	first.Aliases[0].CanonicalID = 0
	container := findSeed(t, first.Items, containerID)
	if len(container.Acquisition.ContainerPickupFlags) == 0 {
		t.Fatalf("container acquisition fixture is incomplete: %#v", container.Acquisition)
	}
	container.Acquisition.ContainerPickupFlags[0] = 0
	for index := range first.GestureSlots {
		if first.GestureSlots[index].SlotID == gestureSlotID {
			first.GestureSlots[index].Flags[0] = "mutated"
		}
	}

	sourceWeapon := data.Weapons[weaponID]
	if sourceWeapon.Flags[0] != "dlc" {
		t.Fatalf("legacy source flags mutated: %#v", sourceWeapon.Flags)
	}
	if data.WeaponStatsV1ByID[weaponID].Warnings[0] != "status-deferred" {
		t.Fatalf("legacy source warnings mutated: %#v", data.WeaponStatsV1ByID[weaponID].Warnings)
	}

	second := collectLegacySnapshot()
	weapon = findSeed(t, second.Items, weaponID)
	if weapon.Flags[0] != "dlc" || weapon.WeaponStats.Warnings[0] != "status-deferred" ||
		weapon.Text.Caption != originalCaption {
		t.Fatalf("second snapshot retained mutations: %#v", weapon)
	}
	if second.Aliases[0].CanonicalID == 0 {
		t.Fatal("second snapshot retained alias mutation")
	}
	container = findSeed(t, second.Items, containerID)
	if container.Acquisition.ContainerPickupFlags[0] != 66000 {
		t.Fatalf("second snapshot retained acquisition mutation: %#v", container.Acquisition)
	}
	for _, slot := range second.GestureSlots {
		if slot.SlotID == gestureSlotID && slot.Flags[0] != "cut_content" {
			t.Fatalf("second snapshot retained gesture mutation: %#v", slot)
		}
	}
}
