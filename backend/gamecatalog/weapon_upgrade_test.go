package gamecatalog_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
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
