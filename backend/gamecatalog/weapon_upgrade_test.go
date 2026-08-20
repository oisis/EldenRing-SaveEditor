package gamecatalog_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestWeaponUpgradeTargetPreservesTheWeaponAnchor(t *testing.T) {
	catalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}

	for _, testCase := range []struct {
		name    string
		current uint32
		level   uint8
		want    uint32
	}{
		{"standard", prototype.DaggerGameID + 5, 25, prototype.DaggerGameID + 25},
		{"heavy affinity", heavyDaggerVariantID + 5, 9, heavyDaggerVariantID + 9},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			target, err := catalog.WeaponUpgradeTarget(testCase.current, testCase.level)
			if err != nil {
				t.Fatalf("WeaponUpgradeTarget: %v", err)
			}
			if target != testCase.want {
				t.Fatalf("target = 0x%08X, want 0x%08X", target, testCase.want)
			}
			resolved, exists := catalog.ItemByGameID(testCase.current)
			if !exists || resolved.Key != daggerKey || resolved.Item.GameID.Value != testCase.current {
				t.Fatalf("ItemByGameID(0x%08X) = (%+v, %t)", testCase.current, resolved, exists)
			}
		})
	}
}

func TestWeaponInfusionTargetPreservesUpgradeLevel(t *testing.T) {
	catalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}

	for _, testCase := range []struct {
		name     string
		current  uint32
		affinity schema.Affinity
		want     uint32
		level    uint8
	}{
		{"standard to heavy", prototype.DaggerGameID + 5, schema.AffinityHeavy,
			heavyDaggerVariantID + 5, 5},
		{"heavy to standard", heavyDaggerVariantID + 9, schema.AffinityStandard,
			prototype.DaggerGameID + 9, 9},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			target, level, err := catalog.WeaponInfusionTarget(
				testCase.current, testCase.affinity)
			if err != nil {
				t.Fatalf("WeaponInfusionTarget: %v", err)
			}
			if target != testCase.want || level != testCase.level {
				t.Fatalf("target/level = 0x%08X/%d, want 0x%08X/%d",
					target, level, testCase.want, testCase.level)
			}
		})
	}
}

func TestWeaponInfusionTargetRejectsUnsupportedAffinity(t *testing.T) {
	catalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}
	if _, _, err := catalog.WeaponInfusionTarget(
		prototype.DaggerGameID, schema.Affinity("unknown")); err == nil {
		t.Fatal("unknown affinity succeeded")
	}

	manifest, resources := prototype.Data()
	resources[0].Item.Capabilities.Infusion.Enabled = false
	resources[0].Item.Capabilities.Infusion.Rules = nil
	resources[0].Item.Capabilities.Infusion.RulesEvidence = nil
	disabled, err := gamecatalog.New(manifest, resources)
	if err != nil {
		t.Fatalf("New disabled catalog: %v", err)
	}
	if _, _, err := disabled.WeaponInfusionTarget(
		prototype.DaggerGameID, schema.AffinityStandard); err == nil {
		t.Fatal("disabled infusion capability succeeded")
	}
}

func TestWeaponUpgradeTargetRejectsInvalidLevelsAndCapabilities(t *testing.T) {
	catalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}
	if _, err := catalog.WeaponUpgradeTarget(prototype.DaggerGameID, 26); err == nil {
		t.Fatal("upgrade level 26 succeeded, want catalog-limit error")
	}

	manifest, resources := prototype.Data()
	resources[0].Item.Capabilities.Upgrade.Enabled = false
	resources[0].Item.Capabilities.Upgrade.Rules = nil
	resources[0].Item.Capabilities.Upgrade.RulesEvidence = nil
	disabled, err := gamecatalog.New(manifest, resources)
	if err != nil {
		t.Fatalf("New disabled catalog: %v", err)
	}
	if _, err := disabled.WeaponUpgradeTarget(prototype.DaggerGameID, 1); err == nil {
		t.Fatal("disabled upgrade capability succeeded")
	}
}

func TestWeaponMatchmakingLevelStandardAndSomber(t *testing.T) {
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("loader.LoadFS: %v", err)
	}
	catalog, err := gamecatalog.New(data.Manifest, data.Resources())
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}

	const (
		daggerID     = uint32(1000000) // standard weapon (max 25)
		blackKnifeID = uint32(1010000) // somber weapon (max 10)
		spiritAshID  = uint32(0x40038A40)
	)

	// Standard mapping: +N -> N
	for lvl := uint8(0); lvl <= 25; lvl++ {
		got, err := catalog.WeaponMatchmakingLevel(daggerID, lvl)
		if err != nil {
			t.Fatalf("standard dagger level %d: %v", lvl, err)
		}
		if got != lvl {
			t.Errorf("standard dagger level %d: got %d, want %d", lvl, got, lvl)
		}
	}

	// Somber mapping: [0, 0, 5, 7, 10, 12, 15, 17, 20, 24, 25]
	somberExpected := []uint8{0, 0, 5, 7, 10, 12, 15, 17, 20, 24, 25}
	for lvl := uint8(0); lvl <= 10; lvl++ {
		got, err := catalog.WeaponMatchmakingLevel(blackKnifeID, lvl)
		if err != nil {
			t.Fatalf("somber black knife level %d: %v", lvl, err)
		}
		if got != somberExpected[lvl] {
			t.Errorf("somber black knife level %d: got %d, want %d", lvl, got, somberExpected[lvl])
		}
	}

	// Invalid levels (> maxLevel)
	if _, err := catalog.WeaponMatchmakingLevel(daggerID, 26); err == nil {
		t.Error("standard dagger level 26 succeeded, want error")
	}
	if _, err := catalog.WeaponMatchmakingLevel(blackKnifeID, 11); err == nil {
		t.Error("somber black knife level 11 succeeded, want error")
	}

	// Spirit Ash exclusion
	if _, err := catalog.WeaponMatchmakingLevel(spiritAshID, 10); err == nil {
		t.Error("spirit ash succeeded in WeaponMatchmakingLevel, want error")
	}
}
