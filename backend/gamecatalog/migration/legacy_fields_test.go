package migration

import "testing"

func TestLegacySeedCopiesRepresentativeFields(t *testing.T) {
	snapshot := collectLegacySnapshot()

	dagger := findSeed(t, snapshot.Items, 0x000F4240)
	if dagger.Category != "melee_armaments" || dagger.Name != "Dagger" {
		t.Fatalf("dagger identity = %q/%q", dagger.Category, dagger.Name)
	}
	if dagger.MaxInventory != 1 || dagger.MaxStorage != 1 || dagger.MaxUpgrade != 25 {
		t.Fatalf("dagger authored caps = %d/%d, upgrade %d", dagger.MaxInventory, dagger.MaxStorage, dagger.MaxUpgrade)
	}
	if dagger.IconPath != "items/melee_armaments/dagger.png" {
		t.Fatalf("dagger icon = %q", dagger.IconPath)
	}
	if dagger.Text == nil || dagger.Text.CanonicalName != "Dagger" || dagger.Text.CanonicalSource != "fmg" {
		t.Fatalf("dagger text = %#v", dagger.Text)
	}
	if dagger.Description == nil || dagger.Description.Weapon == nil || dagger.Description.Weapon.PhysDamage != 74 {
		t.Fatalf("dagger description stats = %#v", dagger.Description)
	}
	if dagger.WeaponStats == nil || dagger.WeaponStats.SourceRowID != 1000000 || !dagger.WeaponStats.IsInfusable {
		t.Fatalf("dagger generated stats = %#v", dagger.WeaponStats)
	}
	if dagger.Weight == nil || *dagger.Weight != 1.5 {
		t.Fatalf("dagger weight = %v", dagger.Weight)
	}
	if dagger.WeaponEdit == nil || dagger.WeaponEdit.WepType != 1 ||
		dagger.WeaponEdit.GemMountType != 2 || !dagger.WeaponEdit.CanChangeAffinity {
		t.Fatalf("dagger edit metadata = %#v", dagger.WeaponEdit)
	}
	if dagger.SortKey == nil || dagger.SortKey.SortID == 0 {
		t.Fatalf("dagger sort key = %#v", dagger.SortKey)
	}
	if dagger.GameMaxInventoryKnown || dagger.GameMaxStorageKnown {
		t.Fatalf("dagger direct ItemData game-limit known bits = %v/%v, want false/false",
			dagger.GameMaxInventoryKnown, dagger.GameMaxStorageKnown)
	}

	spell := findSeed(t, snapshot.Items, 0x40000FA0)
	if spell.SpellMemory == nil || *spell.SpellMemory != 1 {
		t.Fatalf("Glintstone Pebble memory = %v", spell.SpellMemory)
	}
	if spell.GameLimits == nil || !spell.GameLimits.InventoryKnown ||
		spell.GameLimits.MaxInventory != 99 || spell.GameLimits.MaxStorage != 600 {
		t.Fatalf("Glintstone Pebble limits = %#v", spell.GameLimits)
	}
	if spell.Description == nil || spell.Description.Spell == nil ||
		spell.Description.Spell.FPCost != 7 {
		t.Fatalf("Glintstone Pebble description = %#v", spell.Description)
	}

	aow := findSeed(t, snapshot.Items, 0x8000EA60)
	if aow.AoWCompatMask == nil || *aow.AoWCompatMask != 0xF4000FEFFFF {
		t.Fatalf("Determination compatibility mask = %v", aow.AoWCompatMask)
	}

	firePot := findSeed(t, snapshot.Items, 0x4000012C)
	if firePot.Acquisition.RequiredContainerID == nil ||
		*firePot.Acquisition.RequiredContainerID != 0x4000251C {
		t.Fatalf("Fire Pot required container = %#v", firePot.Acquisition.RequiredContainerID)
	}

	crackedPot := findSeed(t, snapshot.Items, 0x4000251C)
	if !crackedPot.Acquisition.IsContainer ||
		len(crackedPot.Acquisition.ContainerPickupFlags) != 20 ||
		len(crackedPot.Acquisition.ContainerVendorFlags) != 1 {
		t.Fatalf("Cracked Pot acquisition = %#v", crackedPot.Acquisition)
	}

	arsenalCharm := findSeed(t, snapshot.Items, 0x20000406)
	if arsenalCharm.EquipLoad == nil || arsenalCharm.EquipLoad.EquipLoadRate != 0.15 {
		t.Fatalf("Arsenal Charm equip-load modifier = %#v", arsenalCharm.EquipLoad)
	}
}

func TestLegacySeedCollectsUnlockMapsIndependently(t *testing.T) {
	snapshot := collectLegacySnapshot()
	assertUnlock(t, findSeed(t, snapshot.Items, 0x8000C418), "ash_of_war", 65841)
	assertUnlock(t, findSeed(t, snapshot.Items, 0x4000218E), "whetblade", 60130)
	assertUnlock(t, findSeed(t, snapshot.Items, 0x40002198), "map", 62010)
	bellBearing := assertUnlock(t, findSeed(t, snapshot.Items, 0x400022CE), "bell_bearing", 11109710)
	if bellBearing.Name != "Pidia's Bell Bearing" || bellBearing.Category != "npc" {
		t.Fatalf("bell-bearing metadata = %#v", bellBearing)
	}
	cookbook := assertUnlock(t, findSeed(t, snapshot.Items, 0x40002454), "cookbook", 67000)
	if cookbook.Name != "Nomadic Warrior's Cookbook [1]" ||
		cookbook.Category != "Nomadic Warrior's Cookbook" {
		t.Fatalf("cookbook metadata = %#v", cookbook)
	}
}

func TestLegacySnapshotCopiesAliasAndGestureFields(t *testing.T) {
	snapshot := collectLegacySnapshot()

	var flaskAlias *aliasSeed
	for index := range snapshot.Aliases {
		if snapshot.Aliases[index].AliasID == 0x400003E8 {
			flaskAlias = &snapshot.Aliases[index]
			break
		}
	}
	if flaskAlias == nil || flaskAlias.CanonicalID != 0x400003E9 {
		t.Fatalf("Crimson Flask alias = %#v", flaskAlias)
	}

	var carianOath *gestureSlotSeed
	for index := range snapshot.GestureSlots {
		if snapshot.GestureSlots[index].SlotID == 111 {
			carianOath = &snapshot.GestureSlots[index]
			break
		}
	}
	if carianOath == nil || carianOath.ItemID != 0x40002341 ||
		carianOath.Name != "The Carian Oath" || len(carianOath.Flags) != 2 ||
		carianOath.Flags[0] != "cut_content" || carianOath.Flags[1] != "ban_risk" {
		t.Fatalf("Carian Oath slot = %#v", carianOath)
	}
}

func findSeed(t *testing.T, items []seed, id uint32) *seed {
	t.Helper()
	for index := range items {
		if items[index].ID == id {
			return &items[index]
		}
	}
	t.Fatalf("item 0x%08X not found", id)
	return nil
}

func assertUnlock(t *testing.T, item *seed, kind string, flagID uint32) unlockSeed {
	t.Helper()
	for _, unlock := range item.Unlocks {
		if unlock.Kind == kind && unlock.FlagID == flagID {
			return unlock
		}
	}
	t.Fatalf("item 0x%08X unlocks = %#v, want %s/%d", item.ID, item.Unlocks, kind, flagID)
	return unlockSeed{}
}
