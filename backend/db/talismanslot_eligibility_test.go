package db

import "testing"

// Talisman slot (equipment row 4, slots 1–4) — all four slots share the exact
// same "talismans"-only filter. crimsonAmberMedallion is the canonical present
// item; erdtreesFavor is a second talisman of a different shape/name. The four
// exclusions each belong to a distinct non-talisman category (weapon, armor,
// arrow, flask) that must never leak into a talisman slot.
const (
	crimsonAmberMedallion = uint32(0x200003E8) // talisman (present)
	erdtreesFavor         = uint32(0x20000410) // talisman (present, different shape/name)
	claymore              = uint32(0x003085E0) // melee armament (excluded)
	arrow                 = uint32(0x02FAF080) // arrow (excluded)
	flaskOfCrimsonTears   = uint32(0x400003E9) // tool/flask (excluded)
)

func TestGetTalismanSlotEligibleItems(t *testing.T) {
	items := GetTalismanSlotEligibleItems("PS4")

	assertSlotFilter(t, items, crimsonAmberMedallion, "talismans",
		[]uint32{claymore, knightHelm, arrow, flaskOfCrimsonTears})

	byID := make(map[uint32]bool, len(items))
	for _, it := range items {
		byID[it.ID] = true
	}
	if !byID[erdtreesFavor] {
		t.Errorf("expected talisman 0x%08X (Erdtree's Favor) to be present, missing from result", erdtreesFavor)
	}
}
