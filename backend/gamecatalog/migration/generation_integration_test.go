package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type migratedItemRecord struct {
	document *schema.ItemDocument
	variant  *schema.ItemVariant
}

func TestGenerateFullLegacyCatalogParity(t *testing.T) {
	options := localGenerateOptions(t)
	catalog, err := Generate(options)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if catalog.Manifest.SchemaVersion != schema.CurrentSchemaVersion {
		t.Fatalf(
			"schema version = %d, want %d",
			catalog.Manifest.SchemaVersion,
			schema.CurrentSchemaVersion,
		)
	}
	if decoded, err := hex.DecodeString(catalog.Manifest.DataVersion); err != nil || len(decoded) != sha256.Size {
		t.Fatalf("data version = %q, want SHA-256: %v", catalog.Manifest.DataVersion, err)
	}
	if len(catalog.Resources) != 2810 {
		t.Fatalf("resources = %d, want 2810", len(catalog.Resources))
	}
	if len(catalog.IconSources) != 2681 {
		t.Fatalf("shared icon files = %d, want 2681", len(catalog.IconSources))
	}

	snapshot := collectLegacySnapshot()
	groups, err := groupLegacyItems(snapshot.Items, options.Regulation)
	if err != nil {
		t.Fatalf("groupLegacyItems: %v", err)
	}
	expectedVariants := make(map[uint32]legacyVariantSeed, 3624)
	for _, group := range groups {
		for _, variant := range group.Variants {
			expectedVariants[variant.Item.ID] = variant
		}
	}
	context, err := newGenerationContext(
		options,
		migrationSourceVersions{},
		snapshot,
	)
	if err != nil {
		t.Fatalf("newGenerationContext: %v", err)
	}

	records := make(map[uint32]migratedItemRecord, 6434)
	aliasCount := 0
	gestureSlotCount := 0
	for resourceIndex := range catalog.Resources {
		resource := &catalog.Resources[resourceIndex]
		if resource.Item == nil {
			t.Fatalf("resource %q has no item document", resource.Key)
		}
		addMigratedRecord(t, records, resource.Item.GameID.Value, migratedItemRecord{
			document: resource.Item,
		})
		aliasCount += len(resource.Item.Aliases)
		if resource.Item.Gesture != nil {
			gestureSlotCount += len(resource.Item.Gesture.Slots)
		}
		for variantIndex := range resource.Item.Variants {
			variant := &resource.Item.Variants[variantIndex]
			addMigratedRecord(t, records, variant.GameID.Value, migratedItemRecord{
				variant: variant,
			})
		}
	}
	if len(records) != 6434 {
		t.Fatalf("canonical plus variant IDs = %d, want 6434", len(records))
	}
	if len(expectedVariants) != 3624 {
		t.Fatalf("expected variants = %d, want 3624", len(expectedVariants))
	}
	if aliasCount != 37 {
		t.Fatalf("aliases = %d, want 37", aliasCount)
	}
	if gestureSlotCount != 57 {
		t.Fatalf("gesture slots = %d, want 57", gestureSlotCount)
	}

	for _, item := range snapshot.Items {
		record, exists := records[item.ID]
		if !exists {
			t.Fatalf("legacy item 0x%08X is absent from generated catalog", item.ID)
		}
		family, _, err := itemFamily(item)
		if err != nil {
			t.Fatalf("item 0x%08X family: %v", item.ID, err)
		}
		identity, err := primaryRegulationForLegacyItem(item)
		if err != nil {
			t.Fatalf("item 0x%08X identity: %v", item.ID, err)
		}
		primary, primaryExists, err := options.Regulation.LookupFamilyRow(
			identity.Family,
			RegulationTableRolePrimary,
			identity.RowID,
		)
		if err != nil {
			t.Fatalf("item 0x%08X primary lookup: %v", item.ID, err)
		}
		expectedVariant, isVariant := expectedVariants[item.ID]
		var capabilitiesOverride *schema.ItemCapabilities
		if isVariant && family == schema.ItemFamilySpiritAsh {
			canonicalRecord := records[expectedVariant.CanonicalID]
			capabilitiesOverride = &canonicalRecord.document.Capabilities
		}
		expectedData, err := context.buildDocumentDataWithCapabilities(
			item,
			family,
			primary.Row,
			primaryExists,
			capabilitiesOverride,
		)
		if err != nil {
			t.Fatalf("item 0x%08X expected data: %v", item.ID, err)
		}
		if isVariant {
			canonicalRecord := records[expectedVariant.CanonicalID]
			if canonicalRecord.document == nil {
				t.Fatalf(
					"variant 0x%08X canonical item 0x%08X is absent",
					item.ID,
					expectedVariant.CanonicalID,
				)
			}
			expectedFull, err := fullVariantData(family, expectedData)
			if err != nil {
				t.Fatalf("item 0x%08X full variant data: %v", item.ID, err)
			}
			if !reflect.DeepEqual(record.variant.Data, expectedFull) {
				t.Fatalf("item 0x%08X full variant data differs", item.ID)
			}
		} else if !reflect.DeepEqual(
			builtDataFromDocument(record.document),
			expectedData,
		) {
			t.Fatalf("item 0x%08X common/family data lost during migration", item.ID)
		}
		expectedSourceRecords, err := context.sourceRecordsForItem(item)
		if err != nil {
			t.Fatalf("item 0x%08X expected source records: %v", item.ID, err)
		}
		if family == schema.ItemFamilySpiritAsh {
			rootRow := primary.Row
			if isVariant {
				canonicalRecord := records[expectedVariant.CanonicalID]
				rootRowID := canonicalRecord.document.SpiritAsh.SourceRowID.Value
				goodsTable, tableExists := options.Regulation.Table(RegulationTableGoods)
				if !tableExists {
					t.Fatalf("Regulation table %q is absent", RegulationTableGoods)
				}
				var rowExists bool
				rootRow, rowExists = goodsTable.Row(rootRowID)
				if !rowExists {
					t.Fatalf("canonical Spirit Ash row %d is absent", rootRowID)
				}
			}
			upgradeRecords, upgradeErr := context.spiritAshUpgradeSourceRecords(rootRow)
			if upgradeErr != nil {
				t.Fatalf("item 0x%08X upgrade chain: %v", item.ID, upgradeErr)
			}
			expectedSourceRecords, upgradeErr = mergeParameterRecords(
				expectedSourceRecords,
				upgradeRecords,
			)
			if upgradeErr != nil {
				t.Fatalf("item 0x%08X source record merge: %v", item.ID, upgradeErr)
			}
		}
		if isVariant {
			expectedSourceRecords = enrichParameterRecordFields(
				expectedSourceRecords,
				record.variant,
			)
		} else {
			expectedSourceRecords = enrichParameterRecordFields(
				expectedSourceRecords,
				record.document,
			)
		}
		if !reflect.DeepEqual(migratedSourceRecords(record), expectedSourceRecords) {
			t.Fatalf("item 0x%08X source records differ from exact Regulation rows", item.ID)
		}
		assertSourceRecordsMatchRegulation(
			t,
			options.Regulation,
			migratedSourceRecords(record),
		)

		if isVariant {
			assertVariantIdentity(t, record.variant, expectedVariant)
			continue
		}
		if record.document == nil {
			t.Fatalf("item 0x%08X should be canonical, got variant", item.ID)
		}
		expectedAliases, err := context.buildAliases(item.ID)
		if err != nil {
			t.Fatalf("item 0x%08X aliases: %v", item.ID, err)
		}
		if !reflect.DeepEqual(record.document.Aliases, expectedAliases) {
			t.Fatalf("item 0x%08X aliases lost during migration", item.ID)
		}
		assertCanonicalFamilyPayload(t, context, item, family, record.document)
	}

	for _, slotOnlyID := range []uint32{0x40002354, 0x40002359} {
		if _, exists := records[slotOnlyID]; !exists {
			t.Fatalf("slot-only gesture 0x%08X is absent", slotOnlyID)
		}
	}
	assertSlotOnlyGestureContracts(t, records)
	assertGestureIdentityProvenance(t, records)
	assertRepresentativePayloads(t, records)
	assertIcons(t, options, catalog, snapshot, records)
	assertLogicalProvenanceOnly(t, options, catalog)
	assertAllFactProvenance(t, catalog.Resources)
	assertDataVersionCoversOutput(t, catalog)
	assertGestureParameterCoverage(t, options.Regulation, catalog)
	assertSaveForgeValueCoverage(t, catalog.Resources)
}

func assertGestureIdentityProvenance(
	t *testing.T,
	records map[uint32]migratedItemRecord,
) {
	t.Helper()
	for _, itemID := range []uint32{0x40002341, 0x4000234E} {
		document := records[itemID].document
		if document == nil {
			t.Fatalf("gesture 0x%08X is not canonical", itemID)
		}
		wantSource := sourceIDByRegulationTable[RegulationTableGesture]
		if document.GameID.Provenance.Source != wantSource ||
			document.Family.Provenance.Source != wantSource {
			t.Fatalf(
				"gesture 0x%08X identity sources = %q/%q, want %q",
				itemID,
				document.GameID.Provenance.Source,
				document.Family.Provenance.Source,
				wantSource,
			)
		}
		if len(document.SourceRecords) == 0 ||
			document.SourceRecords[0].Table != string(RegulationTableGesture) {
			t.Fatalf(
				"gesture 0x%08X source records = %#v",
				itemID,
				document.SourceRecords,
			)
		}
	}
	unknown := records[0x40002354].document
	if unknown.GameID.Provenance.Source != sourceLegacyUnknown ||
		unknown.Family.Provenance.Source != sourceLegacyUnknown {
		t.Fatalf(
			"gesture 0x40002354 identity sources = %q/%q, want %q",
			unknown.GameID.Provenance.Source,
			unknown.Family.Provenance.Source,
			sourceLegacyUnknown,
		)
	}
	withGoods := records[0x40002359].document
	wantGoods := sourceIDByRegulationTable[RegulationTableGoods]
	if withGoods.GameID.Provenance.Source != wantGoods ||
		withGoods.Family.Provenance.Source != wantGoods {
		t.Fatalf(
			"gesture 0x40002359 identity sources = %q/%q, want %q",
			withGoods.GameID.Provenance.Source,
			withGoods.Family.Provenance.Source,
			wantGoods,
		)
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	options := localGenerateOptions(t)
	first, err := Generate(options)
	if err != nil {
		t.Fatalf("first Generate: %v", err)
	}
	firstDigest := generatedCatalogDigest(t, first)
	first = GeneratedCatalog{}
	runtime.GC()

	second, err := Generate(options)
	if err != nil {
		t.Fatalf("second Generate: %v", err)
	}
	secondDigest := generatedCatalogDigest(t, second)
	if firstDigest != secondDigest {
		t.Fatalf("generated digest changed: %s != %s", firstDigest, secondDigest)
	}
}

func localGenerateOptions(t *testing.T) GenerateOptions {
	t.Helper()
	regulation := readLocalRegulationFixture(t)
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve integration test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	return GenerateOptions{
		Regulation:          regulation,
		RegulationParams:    readLocalRegulationParameterFixture(t),
		GameText:            readLocalGameTextFixture(t),
		LegacyIconDirectory: filepath.Join(root, "frontend", "public"),
		GameVersion:         "local-regulation-audit",
	}
}

func addMigratedRecord(
	t *testing.T,
	records map[uint32]migratedItemRecord,
	itemID uint32,
	record migratedItemRecord,
) {
	t.Helper()
	if _, duplicate := records[itemID]; duplicate {
		t.Fatalf("item 0x%08X occurs more than once as canonical/variant", itemID)
	}
	records[itemID] = record
}

func builtDataFromDocument(document *schema.ItemDocument) builtDocumentData {
	if document == nil {
		return builtDocumentData{}
	}
	return builtDocumentData{
		Category:                document.Category,
		Subcategory:             document.Subcategory,
		Flags:                   document.Flags,
		Presentation:            document.Presentation,
		Storage:                 document.Storage,
		Capabilities:            document.Capabilities,
		Safety:                  document.Safety,
		Acquisition:             document.Acquisition,
		Modifiers:               document.Modifiers,
		Links:                   document.Links,
		Unlocks:                 document.Unlocks,
		RelatedTechnicalRecords: document.RelatedTechnicalRecords,
		Weapon:                  document.Weapon,
		SpiritAsh:               document.SpiritAsh,
	}
}

func migratedSourceRecords(record migratedItemRecord) []schema.ParameterRecord {
	if record.variant != nil {
		return record.variant.SourceRecords
	}
	return record.document.SourceRecords
}

func assertVariantIdentity(
	t *testing.T,
	variant *schema.ItemVariant,
	expected legacyVariantSeed,
) {
	t.Helper()
	if variant == nil {
		t.Fatalf("item 0x%08X should be a variant, got canonical", expected.Item.ID)
	}
	if !variant.GameID.Known || variant.GameID.Value != expected.Item.ID {
		t.Fatalf("variant game ID = %#v, want 0x%08X", variant.GameID, expected.Item.ID)
	}
	if !variant.SourceRowID.Known ||
		variant.SourceRowID.Value != expected.Item.ID&0x0FFFFFFF {
		t.Fatalf("variant 0x%08X source row = %#v", expected.Item.ID, variant.SourceRowID)
	}
	if !variant.Kind.Known || variant.Kind.Value != schema.ItemVariantKind(expected.Kind) {
		t.Fatalf("variant 0x%08X kind = %#v, want %q", expected.Item.ID, variant.Kind, expected.Kind)
	}
	switch expected.Kind {
	case legacyVariantAffinity:
		if !variant.Affinity.Known ||
			variant.Affinity.Value != schema.Affinity(expected.Affinity) {
			t.Fatalf("variant 0x%08X affinity = %#v", expected.Item.ID, variant.Affinity)
		}
		if !reflect.DeepEqual(variant.UpgradeLevel, schema.Fact[uint8]{}) {
			t.Fatalf("affinity variant 0x%08X has upgrade discriminator", expected.Item.ID)
		}
	case legacyVariantUpgrade:
		if !variant.UpgradeLevel.Known ||
			variant.UpgradeLevel.Value != expected.UpgradeLevel {
			t.Fatalf("variant 0x%08X upgrade = %#v", expected.Item.ID, variant.UpgradeLevel)
		}
		if !reflect.DeepEqual(variant.Affinity, schema.Fact[schema.Affinity]{}) {
			t.Fatalf("upgrade variant 0x%08X has affinity discriminator", expected.Item.ID)
		}
	}
}

func assertCanonicalFamilyPayload(
	t *testing.T,
	context *generationContext,
	item seed,
	family schema.ItemFamily,
	got *schema.ItemDocument,
) {
	t.Helper()
	expected := schema.ItemDocument{}
	if err := context.attachFamilyData(&expected, item, family); err != nil {
		t.Fatalf("item 0x%08X expected family data: %v", item.ID, err)
	}
	if !reflect.DeepEqual(got.Weapon, expected.Weapon) ||
		!reflect.DeepEqual(got.Armor, expected.Armor) ||
		!reflect.DeepEqual(got.Talisman, expected.Talisman) ||
		!reflect.DeepEqual(got.AshOfWar, expected.AshOfWar) ||
		!reflect.DeepEqual(got.Spell, expected.Spell) ||
		!reflect.DeepEqual(got.SpiritAsh, expected.SpiritAsh) ||
		!reflect.DeepEqual(got.Goods, expected.Goods) ||
		!reflect.DeepEqual(got.Gesture, expected.Gesture) {
		t.Fatalf("item 0x%08X family payload differs from source data", item.ID)
	}
}

func assertSourceRecordsMatchRegulation(
	t *testing.T,
	regulation *RegulationData,
	records []schema.ParameterRecord,
) {
	t.Helper()
	for _, record := range records {
		tableName := RegulationTableName(record.Table)
		table, exists := regulation.Table(tableName)
		if !exists {
			t.Fatalf("source record references unknown table %q", record.Table)
		}
		if record.RowID < 0 {
			t.Fatalf("%s row ID = %d", record.Table, record.RowID)
		}
		row, exists := table.Row(uint32(record.RowID))
		if !exists {
			t.Fatalf("%s row %d does not exist", record.Table, record.RowID)
		}
		if len(record.Fields) == 0 {
			t.Fatalf("%s row %d has no used-field references", record.Table, record.RowID)
		}
		if record.Provenance.Source == sourceRegulationEquipParamGemRaw {
			if tableName != RegulationTableGem ||
				len(record.Fields) != 2 ||
				record.Fields[0].Name != "Row ID" ||
				record.Fields[1].Name != "canMountWep[0:44]" {
				t.Fatalf("invalid raw EquipParamGem reference: %#v", record)
			}
			continue
		}
		for _, field := range record.Fields {
			if _, exists := row.Field(field.Name); !exists {
				t.Fatalf(
					"%s row %d references unknown field %q",
					record.Table,
					record.RowID,
					field.Name,
				)
			}
		}
		if record.Provenance.Source != sourceIDByRegulationTable[tableName] {
			t.Fatalf(
				"%s row %d source = %q",
				record.Table,
				record.RowID,
				record.Provenance.Source,
			)
		}
	}
}

func assertSlotOnlyGestureContracts(
	t *testing.T,
	records map[uint32]migratedItemRecord,
) {
	t.Helper()
	unknown := records[0x40002354].document
	if unknown == nil {
		t.Fatal("gesture 0x40002354 is not canonical")
	}
	if unknown.Storage.RecordMode.Known ||
		unknown.Storage.MaxInventory.Known ||
		unknown.Storage.MaxStorage.Known ||
		unknown.Storage.GameMaxInventory.Known ||
		unknown.Storage.GameMaxStorage.Known {
		t.Fatalf("gesture 0x40002354 fabricated storage = %#v", unknown.Storage)
	}
	for _, capability := range []bool{
		unknown.Capabilities.Upgrade.Known,
		unknown.Capabilities.Infusion.Known,
		unknown.Capabilities.AshOfWarMount.Known,
		unknown.Capabilities.Stack.Known,
	} {
		if capability {
			t.Fatal("gesture 0x40002354 fabricated capability")
		}
	}
	if !unknown.Capabilities.Equipment.Known ||
		unknown.Capabilities.Equipment.Enabled ||
		unknown.Capabilities.Equipment.Rules != nil {
		t.Fatalf(
			"gesture 0x40002354 equipment capability = %#v",
			unknown.Capabilities.Equipment,
		)
	}

	withGoods := records[0x40002359].document
	if withGoods == nil {
		t.Fatal("gesture 0x40002359 is not canonical")
	}
	if withGoods.Storage.RecordMode.Known ||
		withGoods.Storage.MaxInventory.Known ||
		withGoods.Storage.MaxStorage.Known {
		t.Fatalf("gesture 0x40002359 fabricated authored storage = %#v", withGoods.Storage)
	}
	if !withGoods.Storage.GameMaxInventory.Known ||
		!withGoods.Storage.GameMaxStorage.Known ||
		withGoods.Storage.GameMaxInventory.Provenance.Source !=
			sourceIDByRegulationTable[RegulationTableGoods] {
		t.Fatalf("gesture 0x40002359 Regulation game limits = %#v", withGoods.Storage)
	}
}

func assertRepresentativePayloads(
	t *testing.T,
	records map[uint32]migratedItemRecord,
) {
	t.Helper()
	canonical := records[0x03D85830].document
	if canonical == nil ||
		canonical.Presentation.DisplayName.Value != "Smithscript Cirque" ||
		canonical.Weapon == nil {
		t.Fatalf("canonical Smithscript Cirque = %#v", canonical)
	}
	affinity := records[0x03D85A24].variant
	var affinityWeapon schema.WeaponData
	if affinity != nil && affinity.Data.Weapon != nil {
		affinityWeapon = *affinity.Data.Weapon
	}
	if affinity == nil ||
		affinity.Kind.Value != schema.ItemVariantAffinity ||
		affinity.Affinity.Value != schema.AffinityFlameArt ||
		affinity.Data.Weapon == nil ||
		affinityWeapon.SourceRowID.Value != 64510500 {
		t.Fatalf("Smithscript affinity variant = %#v", affinity)
	}
	upgrade := records[0x40038A41].variant
	if upgrade == nil ||
		upgrade.Kind.Value != schema.ItemVariantUpgrade ||
		upgrade.UpgradeLevel.Value != 1 ||
		upgrade.Data.SpiritAsh == nil {
		t.Fatalf("Lone Wolf Ashes +1 variant = %#v", upgrade)
	}
}

func assertDataVersionCoversOutput(t *testing.T, catalog GeneratedCatalog) {
	t.Helper()
	recomputed, err := computeCatalogDataVersion(
		catalog.Manifest,
		catalog.Resources,
		catalog.IconSources,
	)
	if err != nil {
		t.Fatalf("recompute data version: %v", err)
	}
	if recomputed != catalog.Manifest.DataVersion {
		t.Fatalf(
			"manifest data version = %s, recomputed %s",
			catalog.Manifest.DataVersion,
			recomputed,
		)
	}
	mutatedResources := append([]schema.Resource(nil), catalog.Resources...)
	mutatedResources[0] = catalog.Resources[0]
	mutatedResources[0].Label.Value += " changed"
	mutated, err := computeCatalogDataVersion(
		catalog.Manifest,
		mutatedResources,
		catalog.IconSources,
	)
	if err != nil {
		t.Fatalf("compute mutated data version: %v", err)
	}
	if mutated == catalog.Manifest.DataVersion {
		t.Fatal("data version did not change after output fact mutation")
	}
}

func assertGestureParameterCoverage(
	t *testing.T,
	regulation *RegulationData,
	catalog GeneratedCatalog,
) {
	t.Helper()
	bound := make(map[uint32]struct{})
	for _, resource := range catalog.Resources {
		if resource.Item == nil {
			continue
		}
		for _, record := range resource.Item.SourceRecords {
			if record.Table == string(RegulationTableGesture) {
				bound[uint32(record.RowID)] = struct{}{}
			}
		}
	}
	table, exists := regulation.Table(RegulationTableGesture)
	if !exists {
		t.Fatal("GestureParam table is absent")
	}
	var orphanRows []uint32
	for _, row := range table.Rows() {
		if _, exists := bound[row.RowID]; !exists {
			orphanRows = append(orphanRows, row.RowID)
		}
	}
	if want := []uint32{110}; !reflect.DeepEqual(orphanRows, want) {
		t.Fatalf("unbound GestureParam rows = %#v, want explicit exception %#v", orphanRows, want)
	}
	row, exists := table.Row(110)
	if !exists {
		t.Fatal("explicit GestureParam row 110 exception is absent")
	}
	itemID, exists := row.Field("itemId")
	if !exists || itemID != "9051" {
		t.Fatalf("GestureParam row 110 itemId = %q, %t, want 9051", itemID, exists)
	}
}

func assertIcons(
	t *testing.T,
	options GenerateOptions,
	catalog GeneratedCatalog,
	snapshot legacySnapshot,
	records map[uint32]migratedItemRecord,
) {
	t.Helper()
	expected := collectIconSources(snapshot.Items)
	if !reflect.DeepEqual(catalog.IconSources, expected) {
		t.Fatal("generated shared icon map differs from legacy icon references")
	}
	for _, item := range snapshot.Items {
		icon := migratedIconFact(catalog, records, item.ID)
		if item.IconPath == "" {
			if icon.Known {
				t.Fatalf("item 0x%08X fabricated icon %q", item.ID, icon.Value)
			}
			continue
		}
		destination := "assets/icons/" + filepath.ToSlash(filepath.Clean(item.IconPath))
		source, exists := catalog.IconSources[destination]
		if !exists || source != filepath.ToSlash(filepath.Clean(item.IconPath)) {
			t.Fatalf("item 0x%08X icon map = %q, %t", item.ID, source, exists)
		}
		if !icon.Known || icon.Value != destination {
			t.Fatalf("item 0x%08X icon fact = %#v, want %q", item.ID, icon, destination)
		}
		if _, err := os.Stat(filepath.Join(
			options.LegacyIconDirectory,
			filepath.FromSlash(source),
		)); err != nil {
			t.Fatalf("item 0x%08X icon source %q: %v", item.ID, source, err)
		}
	}
}

func migratedIconFact(
	catalog GeneratedCatalog,
	records map[uint32]migratedItemRecord,
	itemID uint32,
) schema.Fact[string] {
	record := records[itemID]
	if record.document != nil {
		return record.document.Presentation.IconPath
	}
	for index := range catalog.Resources {
		item := catalog.Resources[index].Item
		if item == nil {
			continue
		}
		for _, variant := range item.Variants {
			if variant.GameID.Value == itemID {
				return variant.Data.Presentation.IconPath
			}
		}
	}
	return schema.Fact[string]{}
}

func assertLogicalProvenanceOnly(
	t *testing.T,
	options GenerateOptions,
	catalog GeneratedCatalog,
) {
	t.Helper()
	for _, source := range catalog.Manifest.Sources {
		if strings.Contains(source.Location, "tmp/") ||
			filepath.IsAbs(source.Location) {
			t.Fatalf("manifest source %q exposes local path %q", source.ID, source.Location)
		}
		if strings.HasPrefix(string(source.ID), "regulation_") {
			validLocation := strings.HasPrefix(
				source.Location,
				"regulation.bin/csv/",
			) || strings.HasPrefix(
				source.Location,
				"regulation.bin/params/",
			)
			if !validLocation {
				t.Fatalf(
					"Regulation source %q location = %q",
					source.ID,
					source.Location,
				)
			}
		}
	}
	for _, resource := range catalog.Resources {
		raw, err := json.Marshal(resource)
		if err != nil {
			t.Fatalf("marshal %q: %v", resource.Key, err)
		}
		for _, forbidden := range []string{
			"tmp/regulation-bin-dump",
			options.LegacyIconDirectory,
		} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatalf("resource %q exposes local path %q", resource.Key, forbidden)
			}
		}
	}
}

func assertAllFactProvenance(t *testing.T, resources []schema.Resource) {
	t.Helper()
	for index := range resources {
		assertValueProvenance(
			t,
			reflect.ValueOf(resources[index]),
			resources[index].Key,
		)
	}
}

func assertValueProvenance(t *testing.T, value reflect.Value, path string) {
	t.Helper()
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Interface {
		assertValueProvenance(t, value.Elem(), path)
		return
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		assertValueProvenance(t, value.Elem(), path)
		return
	}
	if value.Kind() == reflect.Struct {
		provenanceField := value.FieldByName("Provenance")
		knownField := value.FieldByName("Known")
		if provenanceField.IsValid() &&
			provenanceField.Type() == reflect.TypeOf(schema.Provenance{}) &&
			knownField.IsValid() &&
			knownField.Kind() == reflect.Bool {
			provenance := provenanceField.Interface().(schema.Provenance)
			if provenance.Source == "" || provenance.Method == "" {
				if strings.HasSuffix(path, ".Affinity") ||
					strings.HasSuffix(path, ".UpgradeLevel") {
					return
				}
				t.Fatalf("%s has empty provenance", path)
			}
			if strings.HasPrefix(string(provenance.Source), "regulation_") &&
				(provenance.Table == "" ||
					provenance.Row == "" ||
					provenance.Field == "") {
				t.Fatalf("%s has incomplete Regulation provenance: %#v", path, provenance)
			}
		}
		for index := 0; index < value.NumField(); index++ {
			assertValueProvenance(
				t,
				value.Field(index),
				path+"."+value.Type().Field(index).Name,
			)
		}
		return
	}
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		for index := 0; index < value.Len(); index++ {
			assertValueProvenance(
				t,
				value.Index(index),
				path+"["+decimalIndex(index)+"]",
			)
		}
	}
}

func decimalIndex(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var reversed [20]byte
	index := len(reversed)
	for value > 0 {
		index--
		reversed[index] = digits[value%10]
		value /= 10
	}
	return string(reversed[index:])
}

func generatedCatalogDigest(t *testing.T, catalog GeneratedCatalog) string {
	t.Helper()
	sum := sha256.New()
	writeJSONDigest(t, sum, catalog.Manifest)
	for _, resource := range catalog.Resources {
		writeJSONDigest(t, sum, resource)
	}
	destinations := make([]string, 0, len(catalog.IconSources))
	for destination := range catalog.IconSources {
		destinations = append(destinations, destination)
	}
	sort.Strings(destinations)
	for _, destination := range destinations {
		sum.Write([]byte(destination))
		sum.Write([]byte{0})
		sum.Write([]byte(catalog.IconSources[destination]))
		sum.Write([]byte{0})
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func writeJSONDigest(t *testing.T, sum interface{ Write([]byte) (int, error) }, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal digest input: %v", err)
	}
	if _, err := sum.Write(raw); err != nil {
		t.Fatalf("hash digest input: %v", err)
	}
	if _, err := sum.Write([]byte{0}); err != nil {
		t.Fatalf("hash digest separator: %v", err)
	}
}

func assertSaveForgeValueCoverage(
	t *testing.T,
	resources []schema.Resource,
) {
	t.Helper()
	actual := make(map[string]int)
	for index := range resources {
		countSaveForgeValues(reflect.ValueOf(resources[index]), actual)
	}
	expected := map[string]int{
		"maxInventory-sfv": 16,
		"maxStorage-sfv":   361,
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("SaveForge value coverage = %#v, want %#v", actual, expected)
	}
}

func countSaveForgeValues(value reflect.Value, counts map[string]int) {
	if !value.IsValid() {
		return
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Struct:
		valueType := value.Type()
		for index := 0; index < value.NumField(); index++ {
			fieldType := valueType.Field(index)
			if fieldType.PkgPath != "" {
				continue
			}
			jsonName := strings.Split(fieldType.Tag.Get("json"), ",")[0]
			fieldValue := value.Field(index)
			if strings.HasSuffix(jsonName, "-sfv") {
				if fieldValue.Kind() == reflect.Pointer && !fieldValue.IsNil() {
					counts[jsonName]++
				}
				continue
			}
			countSaveForgeValues(fieldValue, counts)
		}
	case reflect.Array, reflect.Slice:
		for index := 0; index < value.Len(); index++ {
			countSaveForgeValues(value.Index(index), counts)
		}
	}
}
