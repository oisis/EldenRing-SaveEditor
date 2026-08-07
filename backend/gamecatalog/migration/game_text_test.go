package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestGameTextReaderPreservesNamesAndLogicalProvenance(t *testing.T) {
	source := newGameTextMapFS(t)
	data, err := readGameTextFS(source)
	if err != nil {
		t.Fatalf("readGameTextFS: %v", err)
	}

	// Every catalog is loaded independently, so an entry ID resolves per
	// catalog and never leaks across catalogs.
	for _, entry := range gameTextCatalogs {
		base, exists := data.lookupName(entry.catalog, 10)
		wantBase := string(entry.catalog) + " base"
		if !exists || base.text != wantBase ||
			base.source != schema.SourceID("game_text_"+entry.sourceStem+"_base") ||
			base.fmgFile != string(entry.catalog)+".fmg" || base.entryID != 10 {
			t.Fatalf("%s base name = %+v, %t", entry.catalog, base, exists)
		}
		dlc, exists := data.lookupName(entry.catalog, 2000)
		wantDLC := string(entry.catalog) + " dlc"
		if !exists || dlc.text != wantDLC ||
			dlc.source != schema.SourceID("game_text_"+entry.sourceStem+"_dlc") ||
			dlc.fmgFile != string(entry.catalog)+"_dlc01.fmg" || dlc.entryID != 2000 {
			t.Fatalf("%s DLC name = %+v, %t", entry.catalog, dlc, exists)
		}
		if _, exists := data.lookupName(entry.catalog, 11); exists {
			t.Fatalf("%s blank FMG entry became a known name", entry.catalog)
		}
	}

	sources := data.manifestSources()
	if len(sources) != 2*len(gameTextCatalogs) {
		t.Fatalf("manifest source count = %d, want %d", len(sources), 2*len(gameTextCatalogs))
	}
	for index, spec := range gameTextSpecs {
		got := sources[index]
		decodedVersion, err := hex.DecodeString(got.Version)
		if got.ID != spec.sourceID ||
			got.Kind != "game_text_fmg_extract" ||
			got.Location != spec.location ||
			err != nil ||
			len(decodedVersion) != sha256.Size ||
			got.Evidence != schema.EvidenceGameData ||
			!got.Reviewed {
			t.Fatalf("manifest source %d = %+v", index, got)
		}
		if strings.Contains(got.Location, "tmp/") {
			t.Fatalf("manifest source exposes temporary path %q", got.Location)
		}
	}

	sources[0].Location = "mutated"
	if data.manifestSources()[0].Location == "mutated" {
		t.Fatal("manifest sources mutated through returned slice")
	}
}

func TestGameTextSourceVersionCoversUsedJSONExtract(t *testing.T) {
	source := newGameTextMapFS(t)
	before, err := readGameTextFS(source)
	if err != nil {
		t.Fatalf("read before: %v", err)
	}
	beforeVersion := before.manifestSources()[0].Version
	beforeRaw := append([]byte(nil), source["item/ArtsName.fmg"].Data...)

	document := readTestGameTextDocument(t, source, gameTextSpecs[0])
	document.Entries[0].Text = "Changed Extracted Name"
	writeTestGameTextDocument(t, source, gameTextSpecs[0], document)
	after, err := readGameTextFS(source)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if string(source["item/ArtsName.fmg"].Data) != string(beforeRaw) {
		t.Fatal("raw FMG changed with the JSON fixture")
	}
	if after.manifestSources()[0].Version == beforeVersion {
		t.Fatal("source version did not change with used JSON extract")
	}
	if name, exists := after.lookupName(gameTextSpecs[0].catalog, 10); !exists ||
		name.text != "Changed Extracted Name" {
		t.Fatalf("changed extracted name = %+v, %t", name, exists)
	}
}

// TestGameTextDLCEntryOverridesBaseEntry proves the confirmed DLC-over-base
// precedence: when the DLC extract carries the same entry ID as its base file,
// the DLC text and DLC source win, and other catalogs are unaffected.
func TestGameTextDLCEntryOverridesBaseEntry(t *testing.T) {
	source := newGameTextMapFS(t)
	dlcSpec := gameTextSpecs[1]
	if !dlcSpec.dlc || dlcSpec.catalog != gameTextSpecs[0].catalog {
		t.Fatalf("fixture spec order changed: %+v", dlcSpec)
	}
	document := readTestGameTextDocument(t, source, dlcSpec)
	document.Entries[0].ID = 10
	document.Entries[0].Text = "DLC Override"
	writeTestGameTextDocument(t, source, dlcSpec, document)

	data, err := readGameTextFS(source)
	if err != nil {
		t.Fatalf("readGameTextFS: %v", err)
	}
	overridden, exists := data.lookupName(dlcSpec.catalog, 10)
	if !exists || overridden.text != "DLC Override" ||
		overridden.source != dlcSpec.sourceID ||
		overridden.fmgFile != dlcSpec.sourceFMG {
		t.Fatalf("overridden name = %+v, %t", overridden, exists)
	}
	untouched, exists := data.lookupName(gameTextCatalogGoods, 10)
	if !exists || untouched.text != string(gameTextCatalogGoods)+" base" {
		t.Fatalf("unrelated catalog changed = %+v, %t", untouched, exists)
	}
}

func TestGameTextReaderRejectsMalformedInput(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, source fstest.MapFS)
		want   string
	}{
		{
			name: "missing raw FMG",
			mutate: func(_ *testing.T, source fstest.MapFS) {
				delete(source, "item/ArtsName.fmg")
			},
			want: "read item/ArtsName.fmg",
		},
		{
			name: "unknown JSON field",
			mutate: func(_ *testing.T, source fstest.MapFS) {
				source["item/ArtsName.fmg.json"] = &fstest.MapFile{
					Data: []byte(`{"source_msgbnd":"item.msgbnd","source_fmg":"ArtsName.fmg","file_size":4,"group_count":1,"string_count":0,"entries":[],"unexpected":true}`),
				}
			},
			want: `unknown field "unexpected"`,
		},
		{
			name: "wrong source metadata",
			mutate: func(t *testing.T, source fstest.MapFS) {
				document := readTestGameTextDocument(t, source, gameTextSpecs[0])
				document.SourceMSGBND = "other.msgbnd"
				writeTestGameTextDocument(t, source, gameTextSpecs[0], document)
			},
			want: `source_msgbnd = "other.msgbnd"`,
		},
		{
			name: "raw size mismatch",
			mutate: func(_ *testing.T, source fstest.MapFS) {
				source["item/ArtsName.fmg"] = &fstest.MapFile{Data: []byte("changed")}
			},
			want: "matching item/ArtsName.fmg size",
		},
		{
			name: "string count mismatch",
			mutate: func(t *testing.T, source fstest.MapFS) {
				document := readTestGameTextDocument(t, source, gameTextSpecs[0])
				document.StringCount++
				writeTestGameTextDocument(t, source, gameTextSpecs[0], document)
			},
			want: "string_count",
		},
		{
			name: "duplicate ID in source",
			mutate: func(t *testing.T, source fstest.MapFS) {
				document := readTestGameTextDocument(t, source, gameTextSpecs[0])
				document.Entries = append(document.Entries, document.Entries[0])
				document.StringCount++
				writeTestGameTextDocument(t, source, gameTextSpecs[0], document)
			},
			want: "duplicate entry ID 10",
		},
		{
			name: "inconsistent blank marker",
			mutate: func(t *testing.T, source fstest.MapFS) {
				document := readTestGameTextDocument(t, source, gameTextSpecs[0])
				document.Entries[0].Blank = true
				writeTestGameTextDocument(t, source, gameTextSpecs[0], document)
			},
			want: "inconsistent blank marker",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := newGameTextMapFS(t)
			test.mutate(t, source)
			_, err := readGameTextFS(source)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestReadGameTextDirectoryRequiresPath(t *testing.T) {
	_, err := ReadGameTextDirectory("")
	if err == nil || !strings.Contains(err.Error(), "game text directory is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateRequiresGameText(t *testing.T) {
	_, err := Generate(GenerateOptions{
		Regulation:  &RegulationData{},
		GameVersion: "test",
	})
	if err == nil || !strings.Contains(err.Error(), "game text data is required") {
		t.Fatalf("error = %v", err)
	}
}

func newGameTextMapFS(t *testing.T) fstest.MapFS {
	t.Helper()
	source := make(fstest.MapFS, len(gameTextSpecs)*2)
	for index, spec := range gameTextSpecs {
		entries := []extractedFMGEntry{
			{ID: 10, Text: string(spec.catalog) + " base"},
			{ID: 11, Blank: true, Note: "null_offset"},
		}
		if spec.dlc {
			entries = []extractedFMGEntry{{ID: 2000, Text: string(spec.catalog) + " dlc"}}
		}
		rawPath := spec.directory + "/" + spec.rawFilename
		jsonPath := spec.directory + "/" + spec.jsonFilename
		raw := []byte{byte(index), 1, 2, 3}
		source[rawPath] = &fstest.MapFile{Data: raw}
		document := extractedFMG{
			SourceMSGBND: spec.sourceMSGBND,
			SourceFMG:    spec.sourceFMG,
			FileSize:     int64(len(raw)),
			GroupCount:   1,
			StringCount:  len(entries),
			Entries:      entries,
		}
		encoded, err := json.Marshal(document)
		if err != nil {
			t.Fatalf("marshal %s: %v", jsonPath, err)
		}
		source[jsonPath] = &fstest.MapFile{Data: encoded}
	}
	return source
}

func readTestGameTextDocument(
	t *testing.T,
	source fstest.MapFS,
	spec gameTextSpec,
) extractedFMG {
	t.Helper()
	path := spec.directory + "/" + spec.jsonFilename
	var document extractedFMG
	if err := json.Unmarshal(source[path].Data, &document); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return document
}

func writeTestGameTextDocument(
	t *testing.T,
	source fstest.MapFS,
	spec gameTextSpec,
	document extractedFMG,
) {
	t.Helper()
	path := spec.directory + "/" + spec.jsonFilename
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	source[path] = &fstest.MapFile{Data: encoded}
}
