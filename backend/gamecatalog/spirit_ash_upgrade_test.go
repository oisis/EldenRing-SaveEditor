package gamecatalog_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
)

func TestSpiritAshUpgradeTargetUsesStoredGraveAndGhostVariants(t *testing.T) {
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("loader.LoadFS: %v", err)
	}
	catalog, err := gamecatalog.New(data.Manifest, data.Resources())
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}

	for _, testCase := range []struct {
		name string
		base uint32
	}{
		{"grave glovewort", 0x40038A40},
		{"ghost glovewort", 0x400324B0},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			target, err := catalog.SpiritAshUpgradeTarget(testCase.base+4, 10)
			if err != nil {
				t.Fatalf("SpiritAshUpgradeTarget: %v", err)
			}
			if target != testCase.base+10 {
				t.Fatalf("target = 0x%08X, want 0x%08X", target, testCase.base+10)
			}
			if _, err := catalog.SpiritAshUpgradeTarget(testCase.base, 11); err == nil {
				t.Fatal("upgrade level 11 succeeded")
			}
		})
	}
	if _, err := catalog.SpiritAshUpgradeTarget(prototype.DaggerGameID, 1); err == nil {
		t.Fatal("weapon succeeded as a Spirit Ash")
	}
}
