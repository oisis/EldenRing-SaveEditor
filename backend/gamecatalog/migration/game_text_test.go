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

	base, exists := data.lookupName(10)
	if !exists || base.text != "No Skill" || base.source != sourceGameTextArtsNameBase {
		t.Fatalf("base name = %+v, %t", base, exists)
	}
	dlc, exists := data.lookupName(2000)
	if !exists || dlc.text != "Dryleaf Whirlwind" || dlc.source != sourceGameTextArtsNameDLC {
		t.Fatalf("DLC name = %+v, %t", dlc, exists)
	}
	if _, exists := data.lookupName(11); exists {
		t.Fatal("blank FMG entry became a known name")
	}

	sources := data.manifestSources()
	if len(sources) != 2 {
		t.Fatalf("manifest source count = %d, want 2", len(sources))
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
	if name, exists := after.lookupName(10); !exists || name.text != "Changed Extracted Name" {
		t.Fatalf("changed extracted name = %+v, %t", name, exists)
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
		{
			name: "duplicate ID across sources",
			mutate: func(t *testing.T, source fstest.MapFS) {
				document := readTestGameTextDocument(t, source, gameTextSpecs[1])
				document.Entries[0].ID = 10
				writeTestGameTextDocument(t, source, gameTextSpecs[1], document)
			},
			want: "duplicates game text source",
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
	entries := [][]extractedFMGEntry{
		{
			{ID: 10, Text: "No Skill"},
			{ID: 11, Blank: true, Note: "null_offset"},
		},
		{
			{ID: 2000, Text: "Dryleaf Whirlwind"},
		},
	}
	for index, spec := range gameTextSpecs {
		rawPath := spec.directory + "/" + spec.rawFilename
		jsonPath := spec.directory + "/" + spec.jsonFilename
		raw := []byte{byte(index), 1, 2, 3}
		source[rawPath] = &fstest.MapFile{Data: raw}
		document := extractedFMG{
			SourceMSGBND: spec.sourceMSGBND,
			SourceFMG:    spec.sourceFMG,
			FileSize:     int64(len(raw)),
			GroupCount:   1,
			StringCount:  len(entries[index]),
			Entries:      entries[index],
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
