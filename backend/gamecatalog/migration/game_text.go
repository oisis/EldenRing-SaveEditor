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

// gameTextErrorPrefix marks an FMG entry the game itself treats as a
// placeholder rather than a shipped name.
const gameTextErrorPrefix = "[ERROR]"

// gameTextCatalog identifies one official English FMG name catalog. Each
// catalog is loaded from its base file plus the DLC file that extends it; the
// DLC entry wins whenever both files carry the same entry ID.
type gameTextCatalog string

const (
	gameTextCatalogArts      gameTextCatalog = "ArtsName"
	gameTextCatalogWeapon    gameTextCatalog = "WeaponName"
	gameTextCatalogProtector gameTextCatalog = "ProtectorName"
	gameTextCatalogAccessory gameTextCatalog = "AccessoryName"
	gameTextCatalogGoods     gameTextCatalog = "GoodsName"
	gameTextCatalogGem       gameTextCatalog = "GemName"
)

type gameTextSpec struct {
	catalog      gameTextCatalog
	sourceID     schema.SourceID
	dlc          bool
	directory    string
	jsonFilename string
	rawFilename  string
	sourceMSGBND string
	sourceFMG    string
	location     string
}

// gameTextCatalogs lists every official FMG name catalog the migration reads,
// in base-then-DLC order per catalog.
var gameTextCatalogs = []struct {
	catalog    gameTextCatalog
	sourceStem string
}{
	{catalog: gameTextCatalogArts, sourceStem: "arts_name"},
	{catalog: gameTextCatalogWeapon, sourceStem: "weapon_name"},
	{catalog: gameTextCatalogProtector, sourceStem: "protector_name"},
	{catalog: gameTextCatalogAccessory, sourceStem: "accessory_name"},
	{catalog: gameTextCatalogGoods, sourceStem: "goods_name"},
	{catalog: gameTextCatalogGem, sourceStem: "gem_name"},
}

const (
	sourceGameTextArtsNameBase schema.SourceID = "game_text_arts_name_base"
	sourceGameTextArtsNameDLC  schema.SourceID = "game_text_arts_name_dlc"
)

var gameTextSpecs = buildGameTextSpecs()

func buildGameTextSpecs() []gameTextSpec {
	specs := make([]gameTextSpec, 0, 2*len(gameTextCatalogs))
	for _, entry := range gameTextCatalogs {
		base := string(entry.catalog) + ".fmg"
		dlc := string(entry.catalog) + "_dlc01.fmg"
		specs = append(specs,
			gameTextSpec{
				catalog:      entry.catalog,
				sourceID:     schema.SourceID("game_text_" + entry.sourceStem + "_base"),
				directory:    "item",
				jsonFilename: base + ".json",
				rawFilename:  base,
				sourceMSGBND: "item.msgbnd",
				sourceFMG:    base,
				location:     gameTextProvenanceRoot + "item.msgbnd/" + base,
			},
			gameTextSpec{
				catalog:      entry.catalog,
				sourceID:     schema.SourceID("game_text_" + entry.sourceStem + "_dlc"),
				dlc:          true,
				directory:    "item_dlc02",
				jsonFilename: dlc + ".json",
				rawFilename:  dlc,
				sourceMSGBND: "item_dlc02.msgbnd",
				sourceFMG:    dlc,
				location:     gameTextProvenanceRoot + "item_dlc02.msgbnd/" + dlc,
			},
		)
	}
	return specs
}

type gameTextName struct {
	text    string
	source  schema.SourceID
	fmgFile string
	entryID int32
}

// GameTextData contains the immutable English game text required by migration.
type GameTextData struct {
	names   map[gameTextCatalog]map[int32]gameTextName
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

// ReadGameTextDirectory loads every supported English FMG name extract.
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
		names:   make(map[gameTextCatalog]map[int32]gameTextName, len(gameTextCatalogs)),
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

	catalogNames, exists := data.names[spec.catalog]
	if !exists {
		catalogNames = make(map[int32]gameTextName, len(extracted.Entries))
		data.names[spec.catalog] = catalogNames
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
		// The DLC extract extends its base catalog and overrides colliding
		// entry IDs; two base files may never collide.
		if existing, duplicate := catalogNames[id]; duplicate && !spec.dlc {
			return fmt.Errorf(
				"%s entry ID %d duplicates game text source %q",
				jsonPath,
				id,
				existing.source,
			)
		}
		catalogNames[id] = gameTextName{
			text:    entry.Text,
			source:  spec.sourceID,
			fmgFile: spec.sourceFMG,
			entryID: id,
		}
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

func (data *GameTextData) lookupName(
	catalog gameTextCatalog,
	textID int32,
) (gameTextName, bool) {
	if data == nil {
		return gameTextName{}, false
	}
	name, exists := data.names[catalog][textID]
	return name, exists
}

// itemNameCatalogByFamily is the single mapping from item family to the
// official FMG name catalog that owns its names.
var itemNameCatalogByFamily = map[schema.ItemFamily]gameTextCatalog{
	schema.ItemFamilyWeapon:    gameTextCatalogWeapon,
	schema.ItemFamilyArmor:     gameTextCatalogProtector,
	schema.ItemFamilyTalisman:  gameTextCatalogAccessory,
	schema.ItemFamilyGoods:     gameTextCatalogGoods,
	schema.ItemFamilyGesture:   gameTextCatalogGoods,
	schema.ItemFamilySpell:     gameTextCatalogGoods,
	schema.ItemFamilySpiritAsh: gameTextCatalogGoods,
	schema.ItemFamilyAshOfWar:  gameTextCatalogGem,
}

// gameTextEntryID strips the item-family prefix nibble from a game ID, leaving
// the parameter Row ID that doubles as the FMG entry ID.
func gameTextEntryID(gameID uint32) int32 {
	return int32(gameID & 0x0FFFFFFF)
}

// lookupItemName resolves the official FMG name entry for one game ID.
func (data *GameTextData) lookupItemName(
	family schema.ItemFamily,
	gameID uint32,
) (gameTextName, bool, error) {
	catalog, supported := itemNameCatalogByFamily[family]
	if !supported {
		return gameTextName{}, false, fmt.Errorf(
			"item family %q has no official FMG name catalog",
			family,
		)
	}
	name, exists := data.lookupName(catalog, gameTextEntryID(gameID))
	return name, exists, nil
}

func (data *GameTextData) manifestSources() []schema.DataSource {
	if data == nil {
		return nil
	}
	return append([]schema.DataSource(nil), data.sources...)
}
