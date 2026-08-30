package desktop_test

import (
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/application"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/character"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/equipment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/inventory"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/savesession"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
	"github.com/oisis/EldenRing-SaveForge/internal/desktop"
)

var (
	bridgeCatalogOnce sync.Once
	bridgeCatalogData loader.Data
	bridgeCatalogErr  error
)

// testCatalog builds the catalog the way the composition root does, from the
// embedded catalog data. The bridge tests need a real catalog because a nil one
// is what the endpoints reject, so it must not be the default in every case.
func testCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()
	bridgeCatalogOnce.Do(func() {
		bridgeCatalogData, bridgeCatalogErr = loader.LoadFS(catalogdata.Files())
	})
	if bridgeCatalogErr != nil {
		t.Fatalf("loader.LoadFS: %v", bridgeCatalogErr)
	}
	catalog, err := gamecatalog.New(bridgeCatalogData.Manifest, bridgeCatalogData.Resources())
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}
	return catalog
}

type endpointCall func() (any, error)

func newTestBridge(version string) *desktop.Bridge {
	return desktop.NewBridge(version, saveengine.New(), nil)
}

func assertCallsMatch(t *testing.T, bridged endpointCall, direct endpointCall) {
	t.Helper()

	bridgedResult, bridgedErr := bridged()
	directResult, directErr := direct()
	if !reflect.DeepEqual(bridgedResult, directResult) {
		t.Fatalf("bridge result = %#v, want endpoint result %#v", bridgedResult, directResult)
	}
	if (bridgedErr == nil) != (directErr == nil) {
		t.Fatalf("bridge error = %v, endpoint error = %v", bridgedErr, directErr)
	}
	if bridgedErr == nil {
		return
	}
	if reflect.TypeOf(bridgedErr) != reflect.TypeOf(directErr) {
		t.Fatalf("bridge error type = %T, want endpoint error type %T", bridgedErr, directErr)
	}
	if bridgedErr.Error() != directErr.Error() {
		t.Fatalf("bridge error = %q, want endpoint error %q", bridgedErr, directErr)
	}
}

func TestGetApplicationInfoPassesTheWiredVersionToTheEndpoint(t *testing.T) {
	// A value no fallback would produce proves the bridge forwards exactly what
	// the composition root wired, without trimming or substituting a default.
	const version = "  2.0.0-rc.1+local  "

	result, err := newTestBridge(version).GetApplicationInfo()
	if err != nil {
		t.Fatalf("GetApplicationInfo: %v", err)
	}
	if result.ApplicationVersion != version {
		t.Fatalf("applicationVersion = %q, want %q", result.ApplicationVersion, version)
	}
}

// The bridge must return the endpoint result unchanged. Comparing against a
// direct endpoint call proves the bridge adds, drops and reorders nothing.
func TestGetApplicationInfoReturnsTheEndpointResultUnchanged(t *testing.T) {
	const version = "2.0.0"

	bridged, err := newTestBridge(version).GetApplicationInfo()
	if err != nil {
		t.Fatalf("bridge GetApplicationInfo: %v", err)
	}
	direct, err := application.GetApplicationInfo(version)
	if err != nil {
		t.Fatalf("endpoint GetApplicationInfo: %v", err)
	}

	if !reflect.DeepEqual(bridged, direct) {
		t.Fatalf("bridge result = %#v, want the endpoint result %#v", bridged, direct)
	}
}

// Capabilities and schema versions are backend contract. The bridge must not
// declare, filter or extend them, so it can only report what the endpoint does.
func TestGetApplicationInfoDeclaresNoCapabilitiesOrSchemasOfItsOwn(t *testing.T) {
	result, err := newTestBridge("dev").GetApplicationInfo()
	if err != nil {
		t.Fatalf("GetApplicationInfo: %v", err)
	}

	direct, err := application.GetApplicationInfo("dev")
	if err != nil {
		t.Fatalf("endpoint GetApplicationInfo: %v", err)
	}
	if !reflect.DeepEqual(result.Capabilities, direct.Capabilities) {
		t.Fatalf("capabilities = %#v, want the endpoint capabilities %#v",
			result.Capabilities, direct.Capabilities)
	}
	if !reflect.DeepEqual(result.SupportedSchemas, direct.SupportedSchemas) {
		t.Fatalf("supportedSchemas = %#v, want the endpoint schemas %#v",
			result.SupportedSchemas, direct.SupportedSchemas)
	}
}

// An empty version is a wiring error owned by the endpoint. The bridge must
// propagate it instead of hiding it behind a fallback version.
func TestGetApplicationInfoPropagatesTheEmptyVersionWiringError(t *testing.T) {
	result, err := newTestBridge("").GetApplicationInfo()
	if err == nil {
		t.Fatal("GetApplicationInfo with an empty wired version = nil error, want a rejection")
	}
	if err.Error() != "application version is required" {
		t.Fatalf("error = %q, want %q", err.Error(), "application version is required")
	}
	if !reflect.DeepEqual(result, application.GetApplicationInfoResult{}) {
		t.Fatalf("result = %#v, want the empty result", result)
	}
}

func TestReadOnlySaveMethodsReturnEndpointResultsAndErrorsUnchanged(t *testing.T) {
	engine := saveengine.New()
	catalog := testCatalog(t)
	bridge := desktop.NewBridge("dev", engine, catalog)
	missingSource := filepath.Join(t.TempDir(), "missing.sl2")
	const unknownSessionID = "unknown-session"

	tests := []struct {
		name    string
		bridged endpointCall
		direct  endpointCall
	}{
		{
			name: "LoadSave",
			bridged: func() (any, error) {
				return bridge.LoadSave(missingSource, "pc")
			},
			direct: func() (any, error) {
				return savesession.LoadSave(engine, missingSource, "pc")
			},
		},
		{
			name: "GetLoadedSave",
			bridged: func() (any, error) {
				return bridge.GetLoadedSave(unknownSessionID)
			},
			direct: func() (any, error) {
				return savesession.GetLoadedSave(engine, unknownSessionID)
			},
		},
		{
			name: "CloseSave",
			bridged: func() (any, error) {
				return nil, bridge.CloseSave(unknownSessionID)
			},
			direct: func() (any, error) {
				return nil, savesession.CloseSave(engine, unknownSessionID)
			},
		},
		{
			name: "GetSaveCharacters",
			bridged: func() (any, error) {
				return bridge.GetSaveCharacters(unknownSessionID)
			},
			direct: func() (any, error) {
				return character.GetSaveCharacters(engine, unknownSessionID)
			},
		},
		{
			name: "GetCharacterProfile",
			bridged: func() (any, error) {
				return bridge.GetCharacterProfile(unknownSessionID, 0)
			},
			direct: func() (any, error) {
				return character.GetCharacterProfile(engine, unknownSessionID, 0)
			},
		},
		{
			name: "GetCharacterStats",
			bridged: func() (any, error) {
				return bridge.GetCharacterStats(unknownSessionID, 0)
			},
			direct: func() (any, error) {
				return character.GetCharacterStats(engine, unknownSessionID, 0)
			},
		},
		{
			name: "GetInventory",
			bridged: func() (any, error) {
				return bridge.GetInventory(unknownSessionID, 0, "common", 1, 30)
			},
			direct: func() (any, error) {
				return inventory.GetInventory(engine, catalog, unknownSessionID, 0, "common", 1, 30)
			},
		},
		{
			name: "GetStorage",
			bridged: func() (any, error) {
				return bridge.GetStorage(unknownSessionID, 0, "common", 1, 30)
			},
			direct: func() (any, error) {
				return inventory.GetStorage(engine, catalog, unknownSessionID, 0, "common", 1, 30)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertCallsMatch(t, test.bridged, test.direct)
		})
	}
}

func TestReadOnlySaveMethodsPropagateNilEngineErrorsWithoutFallback(t *testing.T) {
	catalog := testCatalog(t)
	bridge := desktop.NewBridge("dev", nil, catalog)

	tests := []struct {
		name    string
		bridged endpointCall
		direct  endpointCall
	}{
		{
			name: "LoadSave",
			bridged: func() (any, error) {
				return bridge.LoadSave("source.sl2", "")
			},
			direct: func() (any, error) {
				return savesession.LoadSave(nil, "source.sl2", "")
			},
		},
		{
			name: "GetLoadedSave",
			bridged: func() (any, error) {
				return bridge.GetLoadedSave("session")
			},
			direct: func() (any, error) {
				return savesession.GetLoadedSave(nil, "session")
			},
		},
		{
			name: "CloseSave",
			bridged: func() (any, error) {
				return nil, bridge.CloseSave("session")
			},
			direct: func() (any, error) {
				return nil, savesession.CloseSave(nil, "session")
			},
		},
		{
			name: "GetSaveCharacters",
			bridged: func() (any, error) {
				return bridge.GetSaveCharacters("session")
			},
			direct: func() (any, error) {
				return character.GetSaveCharacters(nil, "session")
			},
		},
		{
			name: "GetCharacterProfile",
			bridged: func() (any, error) {
				return bridge.GetCharacterProfile("session", 0)
			},
			direct: func() (any, error) {
				return character.GetCharacterProfile(nil, "session", 0)
			},
		},
		{
			name: "GetCharacterStats",
			bridged: func() (any, error) {
				return bridge.GetCharacterStats("session", 0)
			},
			direct: func() (any, error) {
				return character.GetCharacterStats(nil, "session", 0)
			},
		},
		{
			name: "GetInventory",
			bridged: func() (any, error) {
				return bridge.GetInventory("session", 0, "common", 1, 30)
			},
			direct: func() (any, error) {
				return inventory.GetInventory(nil, catalog, "session", 0, "common", 1, 30)
			},
		},
		{
			name: "GetStorage",
			bridged: func() (any, error) {
				return bridge.GetStorage("session", 0, "common", 1, 30)
			},
			direct: func() (any, error) {
				return inventory.GetStorage(nil, catalog, "session", 0, "common", 1, 30)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertCallsMatch(t, test.bridged, test.direct)
		})
	}
}

// A nil catalog is a wiring error owned by the endpoints. The bridge must
// propagate their rejection instead of building a catalog of its own.
func TestItemMethodsPropagateTheNilCatalogErrorWithoutFallback(t *testing.T) {
	engine := saveengine.New()
	bridge := desktop.NewBridge("dev", engine, nil)

	tests := []struct {
		name    string
		bridged endpointCall
		direct  endpointCall
	}{
		{
			name: "GetInventory",
			bridged: func() (any, error) {
				return bridge.GetInventory("session", 0, "common", 1, 30)
			},
			direct: func() (any, error) {
				return inventory.GetInventory(engine, nil, "session", 0, "common", 1, 30)
			},
		},
		{
			name: "GetStorage",
			bridged: func() (any, error) {
				return bridge.GetStorage("session", 0, "common", 1, 30)
			},
			direct: func() (any, error) {
				return inventory.GetStorage(engine, nil, "session", 0, "common", 1, 30)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertCallsMatch(t, test.bridged, test.direct)
			result, err := test.bridged()
			if err == nil {
				t.Fatal("call with a nil catalog = nil error, want the endpoint rejection")
			}
			if err.Error() != "game catalog is not available" {
				t.Fatalf("error = %q, want %q", err.Error(), "game catalog is not available")
			}
			if !reflect.ValueOf(result).IsZero() {
				t.Fatalf("result = %#v, want the empty result", result)
			}
		})
	}
}

// The section, page and page size are backend contract. The bridge must not
// normalise, default or reorder them, so an unusual value has to reach the
// endpoint exactly as given and produce the endpoint's own outcome.
func TestItemMethodsForwardSectionAndPagingUnchanged(t *testing.T) {
	engine := saveengine.New()
	catalog := testCatalog(t)
	bridge := desktop.NewBridge("dev", engine, catalog)

	arguments := []struct {
		name             string
		saveSessionID    string
		characterID      int
		containerSection string
		page             int
		pageSize         int
	}{
		{"untrimmed section", "  Session ID  ", 0, "  Common  ", 1, 30},
		{"empty section", "session", 9, "", 1, 30},
		{"unknown section", "session", 0, "future_section", 1, 30},
		{"zero paging", "session", 0, "common", 0, 0},
		{"negative paging", "session", -1, "common", -1, -1},
		{"large paging", "session", 42, "common", 999999, 999999},
	}

	for _, argument := range arguments {
		t.Run(argument.name, func(t *testing.T) {
			assertCallsMatch(t,
				func() (any, error) {
					return bridge.GetInventory(argument.saveSessionID, argument.characterID,
						argument.containerSection, argument.page, argument.pageSize)
				},
				func() (any, error) {
					return inventory.GetInventory(engine, catalog, argument.saveSessionID,
						argument.characterID, argument.containerSection, argument.page, argument.pageSize)
				})
			assertCallsMatch(t,
				func() (any, error) {
					return bridge.GetStorage(argument.saveSessionID, argument.characterID,
						argument.containerSection, argument.page, argument.pageSize)
				},
				func() (any, error) {
					return inventory.GetStorage(engine, catalog, argument.saveSessionID,
						argument.characterID, argument.containerSection, argument.page, argument.pageSize)
				})
		})
	}
}

// GetResources reads GameCatalog only, so it is proven against the endpoint on
// its own, over the argument shapes the endpoint gives meaning to: an empty
// filter, an untrimmed one, an unknown value, the rejected endpoint filter and
// every paging boundary. The bridge has to produce the endpoint outcome in each
// of them, including the rejections.
func TestGetResourcesForwardsEveryArgumentToTheEndpointUnchanged(t *testing.T) {
	gameCatalog := testCatalog(t)
	bridge := desktop.NewBridge("dev", saveengine.New(), gameCatalog)

	arguments := []struct {
		name         string
		resourceType string
		family       string
		capability   string
		endpointID   string
		search       string
		page         int
		pageSize     int
	}{
		{"empty filters", "", "", "", "", "", 0, 0},
		{"every filter set", "item", "weapon", "upgrade", "", "sword", 1, 5},
		{"untrimmed and recased filters", " Item ", " Weapon ", " Upgrade ", "", "  Sword  ", 1, 5},
		{"mixed case search", "item", "", "", "", "UCHIGATANA", 1, 5},
		{"unknown resource type", "future_kind", "", "", "", "", 1, 5},
		{"unknown family", "item", "future_family", "", "", "", 1, 5},
		{"unknown capability", "item", "", "future_capability", "", "", 1, 5},
		{"rejected endpoint filter", "item", "", "", "get_resources", "", 1, 5},
		{"zero paging", "item", "", "", "", "", 0, 0},
		{"negative page", "item", "", "", "", "", -1, 5},
		{"negative page size", "item", "", "", "", "", 1, -1},
		{"page past the last one", "item", "", "", "", "", 999999, 5},
		{"non-item kind", "class", "", "", "", "", 1, 5},
	}

	for _, argument := range arguments {
		t.Run(argument.name, func(t *testing.T) {
			assertCallsMatch(t,
				func() (any, error) {
					return bridge.GetResources(argument.resourceType, argument.family,
						argument.capability, argument.endpointID, argument.search,
						argument.page, argument.pageSize)
				},
				func() (any, error) {
					return catalog.GetResources(gameCatalog, argument.resourceType, argument.family,
						argument.capability, argument.endpointID, argument.search,
						argument.page, argument.pageSize)
				})
		})
	}
}

// The bridge owns no default of its own: an empty resource type must stay the
// unfiltered catalog rather than becoming an implicit item filter, and zero
// paging must produce the endpoint page size rather than a bridge constant.
func TestGetResourcesAppliesNoFiltersOrDefaultsOfItsOwn(t *testing.T) {
	gameCatalog := testCatalog(t)
	bridge := desktop.NewBridge("dev", saveengine.New(), gameCatalog)

	result, err := bridge.GetResources("", "", "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetResources: %v", err)
	}

	if result.Page != 1 {
		t.Fatalf("page = %d, want the endpoint default 1", result.Page)
	}
	if result.PageSize != catalog.GetResourcesDefaultPageSize {
		t.Fatalf("pageSize = %d, want the endpoint default %d",
			result.PageSize, catalog.GetResourcesDefaultPageSize)
	}

	items, err := bridge.GetResources("item", "", "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetResources for items: %v", err)
	}
	// The catalog holds more than items, so an unfiltered total has to exceed the
	// item total. This fails if the bridge ever substitutes an item filter for an
	// empty resource type.
	if result.Total <= items.Total {
		t.Fatalf("unfiltered total = %d, want more than the item total %d", result.Total, items.Total)
	}
}

// A nil catalog is a wiring error owned by the endpoint. The bridge must
// propagate its rejection instead of loading a catalog of its own.
func TestGetResourcesPropagatesTheNilCatalogErrorWithoutFallback(t *testing.T) {
	bridge := desktop.NewBridge("dev", saveengine.New(), nil)

	assertCallsMatch(t,
		func() (any, error) {
			return bridge.GetResources("item", "", "", "", "", 1, 5)
		},
		func() (any, error) {
			return catalog.GetResources(nil, "item", "", "", "", "", 1, 5)
		})

	result, err := bridge.GetResources("item", "", "", "", "", 1, 5)
	if err == nil {
		t.Fatal("GetResources with a nil catalog = nil error, want the endpoint rejection")
	}
	if err.Error() != "game catalog is not loaded" {
		t.Fatalf("error = %q, want %q", err.Error(), "game catalog is not loaded")
	}
	if !reflect.DeepEqual(result, catalog.GetResourcesResult{}) {
		t.Fatalf("result = %#v, want the empty result", result)
	}
}

func TestGetResourcePresentationSummariesForwardsTheOrderedBatchUnchanged(t *testing.T) {
	gameCatalog := testCatalog(t)
	bridge := desktop.NewBridge("dev", saveengine.New(), gameCatalog)
	identities := []catalog.ResourcePresentationIdentity{
		{Kind: "item", Key: "000F4240"},
		{Kind: "class", Key: "0"},
		{Kind: "item", Key: "000F4240"},
	}

	assertCallsMatch(t,
		func() (any, error) {
			return bridge.GetResourcePresentationSummaries(identities)
		},
		func() (any, error) {
			return catalog.GetResourcePresentationSummaries(gameCatalog, identities)
		})

	empty, err := bridge.GetResourcePresentationSummaries(nil)
	if err != nil {
		t.Fatalf("GetResourcePresentationSummaries(nil): %v", err)
	}
	if empty.Resources == nil || len(empty.Resources) != 0 {
		t.Fatalf("empty Resources = %#v, want non-nil empty slice", empty.Resources)
	}
}

func TestGetResourcePresentationSummariesPropagatesEndpointFailureWithoutFallback(t *testing.T) {
	bridge := desktop.NewBridge("dev", saveengine.New(), nil)
	identities := []catalog.ResourcePresentationIdentity{{Kind: "item", Key: "000F4240"}}

	assertCallsMatch(t,
		func() (any, error) {
			return bridge.GetResourcePresentationSummaries(identities)
		},
		func() (any, error) {
			return catalog.GetResourcePresentationSummaries(nil, identities)
		})
}

// GetResource reads GameCatalog only, so it is proven against the endpoint on
// its own, over the identity shapes the endpoint gives meaning to: a real pair,
// an empty, unknown, untrimmed and recased kind or key, the pre-migration
// prefixed key and a GameID passed as a key. The bridge has to produce the
// endpoint outcome in each of them, including the four distinguishable
// rejections.
func TestGetResourceForwardsKindAndKeyToTheEndpointUnchanged(t *testing.T) {
	gameCatalog := testCatalog(t)
	bridge := desktop.NewBridge("dev", saveengine.New(), gameCatalog)

	arguments := []struct {
		name string
		kind string
		key  string
	}{
		{"known item", "item", "000F4240"},
		{"known non-item kind", "class", "0"},
		{"empty kind and key", "", ""},
		{"empty kind with a known key", "", "000F4240"},
		{"empty key in a known kind", "item", ""},
		{"untrimmed kind", " item ", "000F4240"},
		{"untrimmed key", "item", " 000F4240"},
		{"recased kind", "Item", "000F4240"},
		{"recased key", "item", "000f4240"},
		{"unknown kind", "future_kind", "000F4240"},
		{"unknown key in a known kind", "item", "FFFFFFFF"},
		{"pre-migration prefixed key", "item", "item:000F4240"},
		{"game ID passed as a key", "item", "1000000"},
	}

	for _, argument := range arguments {
		t.Run(argument.name, func(t *testing.T) {
			assertCallsMatch(t,
				func() (any, error) {
					return bridge.GetResource(argument.kind, argument.key)
				},
				func() (any, error) {
					return catalog.GetResource(gameCatalog, argument.kind, argument.key)
				})
		})
	}
}

// The bridge must not retry an unresolved identity under another kind and must
// not fall back to a different resource: a key that exists under kind item is
// unknown under every other kind, and the item document of the resolved
// resource is the only document the result carries.
func TestGetResourceResolvesOneKindOnlyWithoutFallback(t *testing.T) {
	gameCatalog := testCatalog(t)
	bridge := desktop.NewBridge("dev", saveengine.New(), gameCatalog)

	result, err := bridge.GetResource("item", "000F4240")
	if err != nil {
		t.Fatalf("GetResource: %v", err)
	}
	if result.Resource.Kind != schema.ResourceKindItem || result.Resource.Key != "000F4240" {
		t.Fatalf("resource identity = (%q, %q), want (item, 000F4240)",
			result.Resource.Kind, result.Resource.Key)
	}
	if result.Resource.Item == nil {
		t.Fatal("item document = nil, want the resolved item document")
	}
	if result.Resource.Class != nil {
		t.Fatal("class document is set on an item resource, want nil")
	}

	// The same key under a different kind is unknown; no second lookup rescues it.
	other, err := bridge.GetResource("class", "000F4240")
	if err == nil {
		t.Fatal("GetResource with a foreign kind = nil error, want the endpoint rejection")
	}
	if !reflect.DeepEqual(other, catalog.GetResourceResult{}) {
		t.Fatalf("result = %#v, want the empty result", other)
	}
}

// A nil catalog is a wiring error owned by the endpoint. The bridge must
// propagate its rejection instead of loading a catalog of its own.
func TestGetResourcePropagatesTheNilCatalogErrorWithoutFallback(t *testing.T) {
	bridge := desktop.NewBridge("dev", saveengine.New(), nil)

	assertCallsMatch(t,
		func() (any, error) {
			return bridge.GetResource("item", "000F4240")
		},
		func() (any, error) {
			return catalog.GetResource(nil, "item", "000F4240")
		})

	result, err := bridge.GetResource("item", "000F4240")
	if err == nil {
		t.Fatal("GetResource with a nil catalog = nil error, want the endpoint rejection")
	}
	if err.Error() != "game catalog is not loaded" {
		t.Fatalf("error = %q, want %q", err.Error(), "game catalog is not loaded")
	}
	if !reflect.DeepEqual(result, catalog.GetResourceResult{}) {
		t.Fatalf("result = %#v, want the empty result", result)
	}
}

// GetItemVariants reads GameCatalog only, so it is proven against the endpoint
// on its own, over the identity shapes the endpoint gives meaning to: a real
// item, an item without variants, an empty, unknown, untrimmed and recased kind
// or key, a non-item kind and the pre-migration prefixed key. The bridge has to
// produce the endpoint outcome in each of them.
func TestGetItemVariantsForwardsKindAndKeyToTheEndpointUnchanged(t *testing.T) {
	gameCatalog := testCatalog(t)
	bridge := desktop.NewBridge("dev", saveengine.New(), gameCatalog)

	arguments := []struct {
		name string
		kind string
		key  string
	}{
		{"item with variants", "item", "000F4240"},
		{"item without variants", "item", "10009C40"},
		{"non-item kind", "class", "0"},
		{"empty kind and key", "", ""},
		{"empty kind with a known key", "", "000F4240"},
		{"empty key in the item kind", "item", ""},
		{"untrimmed kind", " item ", "000F4240"},
		{"untrimmed key", "item", " 000F4240"},
		{"recased kind", "Item", "000F4240"},
		{"recased key", "item", "000f4240"},
		{"unknown kind", "future_kind", "000F4240"},
		{"unknown key in the item kind", "item", "FFFFFFFF"},
		{"pre-migration prefixed key", "item", "item:000F4240"},
		{"game ID passed as a key", "item", "1000000"},
	}

	for _, argument := range arguments {
		t.Run(argument.name, func(t *testing.T) {
			assertCallsMatch(t,
				func() (any, error) {
					return bridge.GetItemVariants(argument.kind, argument.key)
				},
				func() (any, error) {
					return catalog.GetItemVariants(gameCatalog, argument.kind, argument.key)
				})
		})
	}
}

// The variants reach the caller in catalog order, and an item that carries none
// is a valid empty result rather than a rejection or a nil slice.
func TestGetItemVariantsKeepsCatalogOrderAndTheEmptyResult(t *testing.T) {
	gameCatalog := testCatalog(t)
	bridge := desktop.NewBridge("dev", saveengine.New(), gameCatalog)

	result, err := bridge.GetItemVariants("item", "000F4240")
	if err != nil {
		t.Fatalf("GetItemVariants: %v", err)
	}
	direct, err := catalog.GetItemVariants(gameCatalog, "item", "000F4240")
	if err != nil {
		t.Fatalf("catalog.GetItemVariants: %v", err)
	}
	if len(result.Variants) == 0 {
		t.Fatal("variants = 0, want the variants of the item document")
	}
	// Order is the catalog's, so the sequence of GameIDs has to match exactly.
	bridged := make([]uint32, 0, len(result.Variants))
	endpoint := make([]uint32, 0, len(direct.Variants))
	for _, variant := range result.Variants {
		bridged = append(bridged, variant.GameID.Value)
	}
	for _, variant := range direct.Variants {
		endpoint = append(endpoint, variant.GameID.Value)
	}
	if !reflect.DeepEqual(bridged, endpoint) {
		t.Fatalf("variant order = %v, want %v", bridged, endpoint)
	}

	empty, err := bridge.GetItemVariants("item", "10009C40")
	if err != nil {
		t.Fatalf("GetItemVariants without variants: %v", err)
	}
	if empty.Variants == nil {
		t.Fatal("variants = nil, want an empty slice")
	}
	if len(empty.Variants) != 0 {
		t.Fatalf("variants = %d, want 0", len(empty.Variants))
	}
}

// A nil catalog is a wiring error owned by the endpoint. The bridge must
// propagate its rejection instead of loading a catalog of its own.
func TestGetItemVariantsPropagatesTheNilCatalogErrorWithoutFallback(t *testing.T) {
	bridge := desktop.NewBridge("dev", saveengine.New(), nil)

	assertCallsMatch(t,
		func() (any, error) {
			return bridge.GetItemVariants("item", "000F4240")
		},
		func() (any, error) {
			return catalog.GetItemVariants(nil, "item", "000F4240")
		})

	result, err := bridge.GetItemVariants("item", "000F4240")
	if err == nil {
		t.Fatal("GetItemVariants with a nil catalog = nil error, want the endpoint rejection")
	}
	if err.Error() != "game catalog is not loaded" {
		t.Fatalf("error = %q, want %q", err.Error(), "game catalog is not loaded")
	}
	if !reflect.DeepEqual(result, catalog.GetItemVariantsResult{}) {
		t.Fatalf("result = %#v, want the empty result", result)
	}
}

// The five equipment getters are read-only and share one argument pair, so they
// are proven together over the argument shapes the endpoints give meaning to:
// an untrimmed identifier, an empty one, a negative slot and a slot far past the
// last one. The bridge has to produce the endpoint outcome in each of them,
// which is what proves it neither trims the identifier nor clamps the slot.
func TestEquipmentGettersForwardEveryArgumentToTheEndpointUnchanged(t *testing.T) {
	engine := saveengine.New()
	gameCatalog := testCatalog(t)
	bridge := desktop.NewBridge("dev", engine, gameCatalog)

	arguments := []struct {
		name          string
		saveSessionID string
		characterID   int
	}{
		{"untrimmed session identifier", "  Session ID  ", 0},
		{"empty session identifier", "", 0},
		{"unknown session identifier", "unknown-session", 0},
		{"negative slot", "session", -1},
		{"last slot", "session", 9},
		{"slot past the last one", "session", 42},
	}

	for _, argument := range arguments {
		t.Run(argument.name, func(t *testing.T) {
			assertCallsMatch(t,
				func() (any, error) {
					return bridge.GetEquipment(argument.saveSessionID, argument.characterID)
				},
				func() (any, error) {
					return equipment.GetEquipment(engine, argument.saveSessionID, argument.characterID)
				})
			assertCallsMatch(t,
				func() (any, error) {
					return bridge.GetQuickItems(argument.saveSessionID, argument.characterID)
				},
				func() (any, error) {
					return equipment.GetQuickItems(engine, argument.saveSessionID, argument.characterID)
				})
			assertCallsMatch(t,
				func() (any, error) {
					return bridge.GetPouchItems(argument.saveSessionID, argument.characterID)
				},
				func() (any, error) {
					return equipment.GetPouchItems(engine, argument.saveSessionID, argument.characterID)
				})
			assertCallsMatch(t,
				func() (any, error) {
					return bridge.GetPhysickMixture(argument.saveSessionID, argument.characterID)
				},
				func() (any, error) {
					return equipment.GetPhysickMixture(engine, argument.saveSessionID, argument.characterID)
				})
			assertCallsMatch(t,
				func() (any, error) {
					return bridge.GetEquippedSpells(argument.saveSessionID, argument.characterID)
				},
				func() (any, error) {
					return equipment.GetEquippedSpells(
						engine, gameCatalog, argument.saveSessionID, argument.characterID)
				})
		})
	}
}

// A nil engine is a wiring error owned by the endpoints. The bridge must
// propagate their rejection unchanged instead of building an engine of its own,
// so the result stays empty and the message stays the endpoint message.
func TestEquipmentGettersPropagateTheNilEngineErrorWithoutFallback(t *testing.T) {
	gameCatalog := testCatalog(t)
	bridge := desktop.NewBridge("dev", nil, gameCatalog)

	tests := []struct {
		name    string
		bridged endpointCall
		direct  endpointCall
	}{
		{
			name: "GetEquipment",
			bridged: func() (any, error) {
				return bridge.GetEquipment("session", 0)
			},
			direct: func() (any, error) {
				return equipment.GetEquipment(nil, "session", 0)
			},
		},
		{
			name: "GetQuickItems",
			bridged: func() (any, error) {
				return bridge.GetQuickItems("session", 0)
			},
			direct: func() (any, error) {
				return equipment.GetQuickItems(nil, "session", 0)
			},
		},
		{
			name: "GetPouchItems",
			bridged: func() (any, error) {
				return bridge.GetPouchItems("session", 0)
			},
			direct: func() (any, error) {
				return equipment.GetPouchItems(nil, "session", 0)
			},
		},
		{
			name: "GetPhysickMixture",
			bridged: func() (any, error) {
				return bridge.GetPhysickMixture("session", 0)
			},
			direct: func() (any, error) {
				return equipment.GetPhysickMixture(nil, "session", 0)
			},
		},
		{
			name: "GetEquippedSpells",
			bridged: func() (any, error) {
				return bridge.GetEquippedSpells("session", 0)
			},
			direct: func() (any, error) {
				return equipment.GetEquippedSpells(nil, gameCatalog, "session", 0)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertCallsMatch(t, test.bridged, test.direct)
			result, err := test.bridged()
			if err == nil {
				t.Fatal("call with a nil engine = nil error, want the endpoint rejection")
			}
			if err.Error() != "save engine is not available" {
				t.Fatalf("error = %q, want %q", err.Error(), "save engine is not available")
			}
			if !reflect.ValueOf(result).IsZero() {
				t.Fatalf("result = %#v, want the empty result", result)
			}
		})
	}
}

// GetEquippedSpells is the only equipment getter that reads GameCatalog. A nil
// catalog has to reach the endpoint and produce its own rejection: the bridge
// must neither build nor load a catalog to cover for the composition root.
func TestGetEquippedSpellsPropagatesTheNilCatalogErrorWithoutFallback(t *testing.T) {
	engine := saveengine.New()
	bridge := desktop.NewBridge("dev", engine, nil)

	assertCallsMatch(t,
		func() (any, error) {
			return bridge.GetEquippedSpells("session", 0)
		},
		func() (any, error) {
			return equipment.GetEquippedSpells(engine, nil, "session", 0)
		})

	result, err := bridge.GetEquippedSpells("session", 0)
	if err == nil {
		t.Fatal("GetEquippedSpells with a nil catalog = nil error, want the endpoint rejection")
	}
	if err.Error() != "game catalog is not available" {
		t.Fatalf("error = %q, want %q", err.Error(), "game catalog is not available")
	}
	if !reflect.DeepEqual(result, equipment.GetEquippedSpellsResult{}) {
		t.Fatalf("result = %#v, want the empty result", result)
	}
}

// The other four equipment getters read no catalog at all. Wiring a nil one
// must therefore change nothing about them, which is what proves the bridge
// passes the catalog only where the endpoint contract asks for it.
func TestEquipmentGettersWithoutACatalogAreUnaffectedByANilCatalog(t *testing.T) {
	engine := saveengine.New()
	withCatalog := desktop.NewBridge("dev", engine, testCatalog(t))
	withoutCatalog := desktop.NewBridge("dev", engine, nil)

	tests := []struct {
		name   string
		call   func(*desktop.Bridge) (any, error)
		direct endpointCall
	}{
		{
			name: "GetEquipment",
			call: func(b *desktop.Bridge) (any, error) { return b.GetEquipment("session", 0) },
			direct: func() (any, error) {
				return equipment.GetEquipment(engine, "session", 0)
			},
		},
		{
			name: "GetQuickItems",
			call: func(b *desktop.Bridge) (any, error) { return b.GetQuickItems("session", 0) },
			direct: func() (any, error) {
				return equipment.GetQuickItems(engine, "session", 0)
			},
		},
		{
			name: "GetPouchItems",
			call: func(b *desktop.Bridge) (any, error) { return b.GetPouchItems("session", 0) },
			direct: func() (any, error) {
				return equipment.GetPouchItems(engine, "session", 0)
			},
		},
		{
			name: "GetPhysickMixture",
			call: func(b *desktop.Bridge) (any, error) { return b.GetPhysickMixture("session", 0) },
			direct: func() (any, error) {
				return equipment.GetPhysickMixture(engine, "session", 0)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertCallsMatch(t, func() (any, error) { return test.call(withoutCatalog) }, test.direct)
			assertCallsMatch(t, func() (any, error) { return test.call(withCatalog) }, test.direct)
		})
	}
}
