package application

import (
	"encoding/binary"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
)

// TestApplyAndFavorites_TypeAFaceBone is the public regression for the shared
// Face / Bone Structure mapping on Type A presets: the reported presets used to
// land at raw Face PartsId UI-1 (5) instead of the real 50. Both user-visible
// outputs — direct Apply and the Mirror Favorites slot — must now carry the raw
// PartsId from the shared 1-6 table, for the reported UI 6 presets and the
// adjacent UI 2 case.
func TestApplyAndFavorites_TypeAFaceBone(t *testing.T) {
	cases := []struct {
		name     string
		faceUI   uint8
		wantFace uint32
	}{
		{"Sekiro, the Wolf Shinobi", 6, 50},
		{"Lord Voldemort, the Dark Wizard", 6, 50},
		{"Red Skull, a Mutated Humanoid", 2, 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := findPresetByName(c.name)
			if p == nil || p.BodyType != 1 || p.FaceModel != c.faceUI {
				t.Fatalf("fixture assumption broken: %q must be a Type A preset with FaceModel %d", c.name, c.faceUI)
			}

			// --- Direct Apply ---
			app := &App{save: &core.SaveFile{}}
			app.save.Slots[0] = core.SaveSlot{
				Data:           make([]byte, core.FaceDataBlobSize),
				FaceDataOffset: core.FaceDataBlobSize, // → FaceDataStart() == 0
			}
			if err := app.ApplyPresetToCharacter(0, c.name); err != nil {
				t.Fatalf("ApplyPresetToCharacter: %v", err)
			}
			fd := app.save.Slots[0].FaceDataStart()
			if got := binary.LittleEndian.Uint32(app.save.Slots[0].Data[fd+core.FDOffFaceModel:]); got != c.wantFace {
				t.Errorf("Apply raw Face PartsId = %d, want %d", got, c.wantFace)
			}

			// --- Mirror Favorites ---
			fav := &App{save: &core.SaveFile{}, favSlotNames: make(map[int]string)}
			fav.save.UserData10.Data = make([]byte, 0x60000)
			fav.save.Slots[0] = core.SaveSlot{
				Data:           make([]byte, core.FaceDataBlobSize),
				FaceDataOffset: core.FaceDataBlobSize,
			}
			written, err := fav.WriteSelectedToFavorites(0, []string{c.name})
			if err != nil || written != 1 {
				t.Fatalf("WriteSelectedToFavorites = (%d, %v), want (1, nil)", written, err)
			}
			slotOff := core.FavBaseOffset
			if got := binary.LittleEndian.Uint32(fav.save.UserData10.Data[slotOff+core.FavOffModelIDs:]); got != c.wantFace {
				t.Errorf("Mirror raw Face PartsId = %d, want %d", got, c.wantFace)
			}
		})
	}
}
