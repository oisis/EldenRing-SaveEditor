package migration

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const gameTextProvenanceRoot = "regulation.bin/msg/engus/"

const (
	sourceGameTextArtsNameBase schema.SourceID = "game_text_arts_name_base"
	sourceGameTextArtsNameDLC  schema.SourceID = "game_text_arts_name_dlc"
)

type gameTextSpec struct {
	sourceID     schema.SourceID
	directory    string
	jsonFilename string
	rawFilename  string
	sourceMSGBND string
	sourceFMG    string
	location     string
}

var gameTextSpecs = []gameTextSpec{
	{
		sourceID:     sourceGameTextArtsNameBase,
		directory:    "item",
		jsonFilename: "ArtsName.fmg.json",
		rawFilename:  "ArtsName.fmg",
		sourceMSGBND: "item.msgbnd",
		sourceFMG:    "ArtsName.fmg",
		location:     gameTextProvenanceRoot + "item.msgbnd/ArtsName.fmg",
	},
	{
		sourceID:     sourceGameTextArtsNameDLC,
		directory:    "item_dlc02",
		jsonFilename: "ArtsName_dlc01.fmg.json",
		rawFilename:  "ArtsName_dlc01.fmg",
		sourceMSGBND: "item_dlc02.msgbnd",
		sourceFMG:    "ArtsName_dlc01.fmg",
		location:     gameTextProvenanceRoot + "item_dlc02.msgbnd/ArtsName_dlc01.fmg",
	},
}

type gameTextName struct {
	text   string
	source schema.SourceID
}

// GameTextData contains the immutable English game text required by migration.
type GameTextData struct {
	names   map[int32]gameTextName
	sources []schema.DataSource
}

type extractedFMG struct {
	SourceMSGBND string              `json:"source_msgbnd"`
	SourceFMG    string              `json:"source_fmg"`
	FileSize     int64               `json:"file_size"`
	GroupCount   int                 `json:"group_count"`
	StringCount  int                 `json:"string_count"`
	Entries      []extractedFMGEntry `json:"entries"`
}

type extractedFMGEntry struct {
	ID    int64  `json:"id"`
	Text  string `json:"text"`
	Blank bool   `json:"blank"`
	Note  string `json:"note"`
}

// ReadGameTextDirectory loads the two supported English ArtsName FMG extracts.
func ReadGameTextDirectory(root string) (*GameTextData, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("game text directory is required")
	}
	return readGameTextFS(os.DirFS(root))
}

func readGameTextFS(source fs.FS) (*GameTextData, error) {
	if source == nil {
		return nil, errors.New("game text filesystem is required")
	}

	data := &GameTextData{
		names:   make(map[int32]gameTextName),
		sources: make([]schema.DataSource, 0, len(gameTextSpecs)),
	}
	for _, spec := range gameTextSpecs {
		if err := data.readSource(source, spec); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (data *GameTextData) readSource(source fs.FS, spec gameTextSpec) error {
	jsonPath := spec.directory + "/" + spec.jsonFilename
	rawPath := spec.directory + "/" + spec.rawFilename
	jsonRaw, err := fs.ReadFile(source, jsonPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", jsonPath, err)
	}
	fmgRaw, err := fs.ReadFile(source, rawPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", rawPath, err)
	}

	var extracted extractedFMG
	decoder := json.NewDecoder(bytes.NewReader(jsonRaw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&extracted); err != nil {
		return fmt.Errorf("decode %s: %w", jsonPath, err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return fmt.Errorf("decode %s: %w", jsonPath, err)
	}
	if extracted.SourceMSGBND != spec.sourceMSGBND {
		return fmt.Errorf(
			"%s source_msgbnd = %q, want %q",
			jsonPath,
			extracted.SourceMSGBND,
			spec.sourceMSGBND,
		)
	}
	if extracted.SourceFMG != spec.sourceFMG {
		return fmt.Errorf(
			"%s source_fmg = %q, want %q",
			jsonPath,
			extracted.SourceFMG,
			spec.sourceFMG,
		)
	}
	if extracted.FileSize != int64(len(fmgRaw)) {
		return fmt.Errorf(
			"%s file_size = %d, matching %s size = %d",
			jsonPath,
			extracted.FileSize,
			rawPath,
			len(fmgRaw),
		)
	}
	if extracted.GroupCount <= 0 {
		return fmt.Errorf("%s group_count must be greater than zero", jsonPath)
	}
	if extracted.StringCount <= 0 {
		return fmt.Errorf("%s string_count must be greater than zero", jsonPath)
	}
	if extracted.StringCount != len(extracted.Entries) {
		return fmt.Errorf(
			"%s string_count = %d, entries = %d",
			jsonPath,
			extracted.StringCount,
			len(extracted.Entries),
		)
	}

	localIDs := make(map[int32]struct{}, len(extracted.Entries))
	for index, entry := range extracted.Entries {
		if entry.ID < 0 || entry.ID > math.MaxInt32 {
			return fmt.Errorf("%s entry %d has invalid ID %d", jsonPath, index, entry.ID)
		}
		id := int32(entry.ID)
		if _, duplicate := localIDs[id]; duplicate {
			return fmt.Errorf("%s has duplicate entry ID %d", jsonPath, id)
		}
		localIDs[id] = struct{}{}
		if entry.Blank != (entry.Text == "") {
			return fmt.Errorf(
				"%s entry ID %d has inconsistent blank marker",
				jsonPath,
				id,
			)
		}
		if entry.Text == "" {
			continue
		}
		if existing, duplicate := data.names[id]; duplicate {
			return fmt.Errorf(
				"%s entry ID %d duplicates game text source %q",
				jsonPath,
				id,
				existing.source,
			)
		}
		data.names[id] = gameTextName{text: entry.Text, source: spec.sourceID}
	}

	data.sources = append(data.sources, schema.DataSource{
		ID:       spec.sourceID,
		Kind:     "game_text_fmg_extract",
		Location: spec.location,
		Version:  gameTextSourceVersion(spec, jsonRaw, fmgRaw),
		Evidence: schema.EvidenceGameData,
		Reviewed: true,
	})
	return nil
}

func gameTextSourceVersion(spec gameTextSpec, jsonRaw []byte, fmgRaw []byte) string {
	sum := sha256.New()
	for _, input := range []struct {
		name string
		raw  []byte
	}{
		{name: spec.rawFilename, raw: fmgRaw},
		{name: spec.jsonFilename, raw: jsonRaw},
	} {
		sum.Write([]byte(input.name))
		sum.Write([]byte{0})
		sum.Write(input.raw)
		sum.Write([]byte{0})
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func requireJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.New("unexpected trailing JSON value")
}

func (data *GameTextData) lookupName(textID int32) (gameTextName, bool) {
	if data == nil {
		return gameTextName{}, false
	}
	name, exists := data.names[textID]
	return name, exists
}

func (data *GameTextData) manifestSources() []schema.DataSource {
	if data == nil {
		return nil
	}
	return append([]schema.DataSource(nil), data.sources...)
}
