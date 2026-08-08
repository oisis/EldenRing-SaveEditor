package dbviewer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestSourceOriginClass(t *testing.T) {
	cases := map[schema.SourceID]string{
		"legacy_db_data":                   "legacy",
		"regulation_equip_param_weapon":    "regulation",
		"regulation_equip_param_protector": "regulation",
		"community_notes":                  "",
		"legacy_db_data_extra":             "",
		"":                                 "",
	}
	for source, want := range cases {
		if got := sourceOriginClass(source); got != want {
			t.Fatalf("source %q origin class = %q, want %q", source, got, want)
		}
	}
}

// Origin colouring belongs on the source text only. Field values keep their
// plain appearance.
func TestFactsTemplateColoursSourceTextOnly(t *testing.T) {
	templates, err := parseTemplates()
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	facts := []factView{
		{Label: "Legacy field", Value: "1", Known: true,
			Source: "legacy_db_data", SourceLocation: "backend/db/data"},
		{Label: "Regulation field", Value: "2", Known: true,
			Source: "regulation_equip_param_protector", SourceLocation: "regulation.bin/csv/EquipParamProtector.csv"},
		{Label: "Other field", Value: "3", Known: true,
			Source: "community_notes", SourceLocation: "notes"},
	}
	var rendered bytes.Buffer
	if err := templates.templates.ExecuteTemplate(&rendered, "facts", facts); err != nil {
		t.Fatalf("render facts: %v", err)
	}
	html := rendered.String()
	for _, want := range []string{
		`<span class="source-origin legacy"><code>legacy_db_data</code> · <code>backend/db/data</code></span>`,
		`<span class="source-origin regulation"><code>regulation_equip_param_protector</code> · <code>regulation.bin/csv/EquipParamProtector.csv</code></span>`,
		`<span class="source-origin "><code>community_notes</code> · <code>notes</code></span>`,
		`<span class="fact-value">1</span>`,
		`<span class="fact-value">2</span>`,
		`<span class="fact-value">3</span>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered facts missing %q in:\n%s", want, html)
		}
	}
	if strings.Contains(html, `fact-value source-origin`) {
		t.Fatalf("fact values must not carry the source-origin class:\n%s", html)
	}
}

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
