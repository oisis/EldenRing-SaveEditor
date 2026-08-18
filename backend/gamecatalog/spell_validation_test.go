package gamecatalog_test

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestValidateSpellResource(t *testing.T) {
	t.Parallel()

	validSpellResource := func() schema.Resource {
		return schema.Resource{
			Kind: schema.ResourceKindItem,
			Key:  "glintstone_pebble",
			Item: &schema.ItemDocument{
				Family: schema.Fact[schema.ItemFamily]{
					Known: true,
					Value: schema.ItemFamilySpell,
				},
				GameID: schema.Fact[uint32]{
					Known: true,
					Value: 0x40000FA0,
				},
				Spell: &schema.SpellData{
					MemorySlots: schema.Fact[uint8]{
						Known: true,
						Value: 1,
					},
				},
				Capabilities: schema.ItemCapabilities{
					Equipment: schema.Capability[schema.EquipmentRules]{
						Known:   true,
						Enabled: true,
						Rules: &schema.EquipmentRules{
							AllowedSlots: []schema.EquipmentSlot{
								schema.EquipmentSlotSpellMemory,
							},
						},
					},
				},
			},
		}
	}

	tests := []struct {
		name       string
		mutate     func(r *schema.Resource)
		wantRawID  uint32
		wantCost   int
		wantErrMsg string
	}{
		{
			name:      "valid spell resource",
			mutate:    func(r *schema.Resource) {},
			wantRawID: 0x0FA0,
			wantCost:  1,
		},
		{
			name: "no item document",
			mutate: func(r *schema.Resource) {
				r.Item = nil
			},
			wantErrMsg: "has no item document",
		},
		{
			name: "wrong item family",
			mutate: func(r *schema.Resource) {
				r.Item.Family.Value = schema.ItemFamilyWeapon
			},
			wantErrMsg: "has item family",
		},
		{
			name: "unknown game ID",
			mutate: func(r *schema.Resource) {
				r.Item.GameID.Known = false
			},
			wantErrMsg: "has no known game ID",
		},
		{
			name: "unsupported spell game ID prefix",
			mutate: func(r *schema.Resource) {
				r.Item.GameID.Value = 0x10000FA0
			},
			wantErrMsg: "has unsupported spell game ID",
		},
		{
			name: "nil spell data",
			mutate: func(r *schema.Resource) {
				r.Item.Spell = nil
			},
			wantErrMsg: "has invalid memory slots",
		},
		{
			name: "zero memory slots",
			mutate: func(r *schema.Resource) {
				r.Item.Spell.MemorySlots.Value = 0
			},
			wantErrMsg: "has invalid memory slots",
		},
		{
			name: "unconfirmed equipment capability",
			mutate: func(r *schema.Resource) {
				r.Item.Capabilities.Equipment.Known = false
			},
			wantErrMsg: "has no confirmed equipment capability",
		},
		{
			name: "disabled equipment capability",
			mutate: func(r *schema.Resource) {
				r.Item.Capabilities.Equipment.Enabled = false
			},
			wantErrMsg: "has no confirmed equipment capability",
		},
		{
			name: "disallowed equipment slot",
			mutate: func(r *schema.Resource) {
				r.Item.Capabilities.Equipment.Rules.AllowedSlots = []schema.EquipmentSlot{
					schema.EquipmentSlotLeftHand,
				}
			},
			wantErrMsg: "cannot be equipped in the spell memory slot",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := validSpellResource()
			tc.mutate(&res)

			rawID, cost, err := gamecatalog.ValidateSpellResource(res)
			if tc.wantErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Fatalf("error = %v, want one containing %q", err, tc.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rawID != tc.wantRawID {
				t.Errorf("rawID = 0x%08X, want 0x%08X", rawID, tc.wantRawID)
			}
			if cost != tc.wantCost {
				t.Errorf("cost = %d, want %d", cost, tc.wantCost)
			}
		})
	}
}
