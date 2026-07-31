package dbviewer

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestAliasesAndCanonicalAndVariantSourceRecordsRemainVisible(t *testing.T) {
	sourceID := schema.SourceID("regulation")
	provenance := schema.Provenance{Source: sourceID, Method: "exact row"}
	server := &Server{
		sources: map[schema.SourceID]schema.DataSource{
			sourceID: {
				ID:       sourceID,
				Location: "regulation.bin/csv/EquipParamGoods.csv",
			},
		},
	}
	item := &schema.ItemDocument{
		Aliases: []schema.ItemAlias{{
			GameID: schema.Fact[uint32]{
				Known:      true,
				Value:      0x40000001,
				Provenance: provenance,
			},
			SourceRecords: []schema.ParameterRecord{{
				Table:      "EquipParamGoods",
				RowID:      3,
				Fields:     []schema.ParameterField{{Name: "refId"}},
				Provenance: provenance,
			}},
		}},
		SourceRecords: []schema.ParameterRecord{{
			Table:      "EquipParamGoods",
			RowID:      1,
			Fields:     []schema.ParameterField{{Name: "goodsType"}},
			Provenance: provenance,
		}},
		Variants: []schema.ItemVariant{{
			GameID: schema.Fact[uint32]{Known: true, Value: 0x40000002},
			SourceRecords: []schema.ParameterRecord{{
				Table:      "EquipParamGoods",
				RowID:      2,
				Fields:     []schema.ParameterField{{Name: "goodsType"}},
				Provenance: provenance,
			}},
		}},
	}

	aliases := server.aliasViews(item)
	if len(aliases) != 1 || aliases[0].GameID != "0x40000001" {
		t.Fatalf("alias views = %+v", aliases)
	}
	if aliases[0].SourceLocation != "regulation.bin/csv/EquipParamGoods.csv" {
		t.Fatalf("alias source location = %q", aliases[0].SourceLocation)
	}

	records := server.parameterRecordViews(item)
	if len(records) != 3 {
		t.Fatalf("source record views = %+v", records)
	}
	if records[0].Scope != "Item" || records[0].Fields[0].Name != "goodsType" {
		t.Fatalf("canonical source record = %+v", records[0])
	}
	if records[1].Scope != "Item / Variants 1" || records[1].Fields[0].Name != "goodsType" {
		t.Fatalf("variant source record = %+v", records[1])
	}
	if records[2].Scope != "Item / Aliases 1" || records[2].Fields[0].Name != "refId" {
		t.Fatalf("alias source record = %+v", records[2])
	}
}
