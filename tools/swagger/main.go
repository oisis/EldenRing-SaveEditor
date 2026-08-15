// Command swagger serves a local OpenAPI explorer for the currently implemented
// GameCatalog getters and, in loopback mode only, for the save-session lifecycle,
// character getters and explicitly exposed mutations. It is a
// standalone developer tool: the Wails application neither imports nor starts
// it, and every route only calls an existing endpoint and serialises its result.
package main

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/appearance"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/application"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/character"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/equipment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/inventory"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/network"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/savesession"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/world"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

//go:embed openapi.json
var assets embed.FS

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	address := flag.String("addr", "127.0.0.1:8788", "local address used by the explorer")
	dataDirectory := flag.String("data", "./backend/gamecatalog/data", "catalog data directory")
	allowExternal := flag.Bool("allow-external-bind", false, "allow binding a non-loopback address")
	// The explorer is a developer tool with no build-time version of its own, so
	// it supplies the application version GetApplicationInfo requires.
	applicationVersion := flag.String("app-version", "dev", "application version reported by /api/v1/application/info")
	flag.Parse()

	if err := validateAddress(*address, *allowExternal); err != nil {
		return err
	}

	gameCatalog, err := loadCatalog(*dataDirectory)
	if err != nil {
		return err
	}

	// The save-session routes read local save files and hold sessions in memory,
	// so they exist only while the explorer is a loopback-only developer tool.
	// An external bind withholds the engine, and every save-session route is then
	// absent from the mux and answers 404.
	var saveEngine *saveengine.Engine
	if !*allowExternal {
		saveEngine = saveengine.New()
	}

	server := &http.Server{
		Addr:              *address,
		Handler:           newHandler(gameCatalog, *applicationVersion, saveEngine),
		ReadHeaderTimeout: 5 * time.Second,
	}
	// The browser UI is Scalar Docs, served by a separate process; this one is
	// only the local API host it calls, so the log names the API surface rather
	// than advertising a page this process no longer serves.
	log.Printf("SaveForge local API host on http://%s (OpenAPI document at /openapi.json)", *address)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve explorer: %w", err)
	}
	return nil
}

func loadCatalog(directory string) (*gamecatalog.Catalog, error) {
	data, err := loader.LoadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("load catalog directory: %w", err)
	}
	// Missing or invalid network parameters stop the explorer here, instead of
	// serving a catalog whose network route could not answer.
	networkPresets, err := gamecatalog.LoadNetworkParams(os.DirFS(directory))
	if err != nil {
		return nil, fmt.Errorf("load network parameters: %w", err)
	}
	// The same rule applies to the appearance presets and their assets.
	appearancePresets, err := gamecatalog.LoadAppearancePresets(os.DirFS(directory))
	if err != nil {
		return nil, fmt.Errorf("load appearance presets: %w", err)
	}
	gameCatalog, err := gamecatalog.NewWithData(gamecatalog.CatalogData{
		Manifest:          data.Manifest,
		Resources:         data.Resources(),
		NetworkPresets:    networkPresets,
		AppearancePresets: appearancePresets,
	})
	if err != nil {
		return nil, fmt.Errorf("build catalog: %w", err)
	}
	return gameCatalog, nil
}

// validateAddress keeps the explorer on loopback unless the operator asks for
// an external bind explicitly. The catalog is development data, but the server
// is unauthenticated, so an accidental LAN bind must not be possible by default.
func validateAddress(address string, allowExternal bool) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid address %q: %w", address, err)
	}
	if allowExternal {
		return nil
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf(
			"address %q is not a loopback address; pass -allow-external-bind to serve it anyway",
			address,
		)
	}
	return nil
}

// newHandler builds the explorer mux. A nil saveEngine registers no
// save-session route at all, which is how the caller disables the
// save-session lifecycle for an external bind.
func newHandler(gameCatalog *gamecatalog.Catalog, applicationVersion string, saveEngine *saveengine.Engine) http.Handler {
	mux := http.NewServeMux()

	if saveEngine != nil {
		registerSaveSessionRoutes(mux, saveEngine, gameCatalog)
	}

	mux.HandleFunc("GET /healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /openapi.json", func(writer http.ResponseWriter, _ *http.Request) {
		serveAsset(writer, "openapi.json", "application/json")
	})

	mux.HandleFunc("GET /api/v1/application/info", func(writer http.ResponseWriter, _ *http.Request) {
		result, err := application.GetApplicationInfo(applicationVersion)
		if err != nil {
			// The version comes from the backend, not from the client, so a
			// rejected version is a server configuration error.
			writeError(writer, http.StatusInternalServerError, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	mux.HandleFunc("GET /api/v1/catalog/info", func(writer http.ResponseWriter, _ *http.Request) {
		result, err := catalog.GetCatalogInfo(gameCatalog)
		if err != nil {
			// GetCatalogInfo takes no input, so a failure is always server side.
			writeError(writer, http.StatusInternalServerError, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	mux.HandleFunc("GET /api/v1/catalog/resource", func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		result, err := catalog.GetResource(gameCatalog, query.Get("kind"), query.Get("key"))
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	mux.HandleFunc("GET /api/v1/catalog/resource-relations", func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		result, err := catalog.GetResourceRelations(
			gameCatalog,
			query.Get("kind"),
			query.Get("key"),
			query.Get("relationType"),
			query.Get("direction"),
		)
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	mux.HandleFunc("GET /api/v1/catalog/resources", func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		page, err := parsePagingValue(query.Get("page"), "page")
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		pageSize, err := parsePagingValue(query.Get("pageSize"), "pageSize")
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		result, err := catalog.GetResources(
			gameCatalog,
			query.Get("resourceType"),
			query.Get("family"),
			query.Get("capability"),
			query.Get("endpointId"),
			query.Get("search"),
			page,
			pageSize,
		)
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	mux.HandleFunc("GET /api/v1/catalog/item-variants", func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		result, err := catalog.GetItemVariants(gameCatalog, query.Get("kind"), query.Get("key"))
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	// The presets live in the catalog data: the route needs the catalog, but not
	// the application version.
	mux.HandleFunc("GET /api/v1/network/presets", func(writer http.ResponseWriter, request *http.Request) {
		result, err := network.GetNetworkPresets(gameCatalog, request.URL.Query().Get("presetID"))
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	// The presets and their assets live in the catalog data, so the route needs
	// the catalog only. It serves preset metadata; the images themselves are not
	// exposed by any route yet.
	mux.HandleFunc("GET /api/v1/appearance/presets", func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		result, err := appearance.GetAppearancePresets(gameCatalog, query.Get("search"), parseTags(query.Get("tags")))
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	return withPortalCORS(mux)
}

// portalOrigin is the single origin the Scalar documentation portal is served
// from by `npx @scalar/cli project preview` in tools/swagger/docs-portal. The
// portal reads openapi.json from disk, so this is the one browser context that
// has to reach the explorer over the network to send a request.
const portalOrigin = "http://localhost:7970"

// withPortalCORS grants that one portal origin, and nothing else, permission to
// read the explorer's responses from a browser. It never echoes an arbitrary
// Origin, never answers "*" and never allows credentials, so the set of callers
// it admits is a single fixed loopback page. The bind address stays the only
// network contract: an explorer started with -allow-external-bind still serves
// no save-session route, and this header grants no reader that could not already
// call the explorer directly.
func withPortalCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// The answer differs per Origin even when permission is refused, so a
		// cache must not reuse one origin's response for another.
		writer.Header().Add("Vary", "Origin")
		allowed := request.Header.Get("Origin") == portalOrigin
		if allowed {
			writer.Header().Set("Access-Control-Allow-Origin", portalOrigin)
		}

		// A preflight carries a method the mux does not route, so it is answered
		// here; the mux only knows the real methods and would reject OPTIONS.
		// A refused origin still gets 204, but without the permission headers,
		// which is what makes the browser block the request it was asking about.
		if request.Method == http.MethodOptions && request.Header.Get("Access-Control-Request-Method") != "" {
			if allowed {
				writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				// The documented request bodies are application/json; no other
				// request header is needed.
				writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			}
			writer.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(writer, request)
	})
}

// requireJSONBody refuses a mutating body that does not declare application/json.
//
// A POST carrying text/plain, a form media type or no Content-Type at all is a
// CORS simple request: the browser sends it without a preflight, so a foreign
// page could reach a session-creating or file-writing route even though the
// refused origin never gets to read the answer. The check runs before the body
// is decoded and before any endpoint or SaveEngine call, so a rejected request
// performs no operation. Parameters are allowed, because
// "application/json; charset=utf-8" is the same media type.
func requireJSONBody(request *http.Request) error {
	declared := request.Header.Get("Content-Type")
	if declared == "" {
		return errors.New("Content-Type must be application/json; the request declared none")
	}
	mediaType, _, err := mime.ParseMediaType(declared)
	if err != nil {
		return fmt.Errorf("Content-Type %q is not a valid media type; it must be application/json", declared)
	}
	if mediaType != "application/json" {
		return fmt.Errorf("Content-Type must be application/json; the request declared %q", mediaType)
	}
	return nil
}

// loadSaveRequest is the JSON body of POST /api/v1/save-sessions. Both fields
// reach LoadSave exactly as sent: they are never trimmed, normalised or given a
// default here.
type loadSaveRequest struct {
	Source           string `json:"source"`
	ExpectedPlatform string `json:"expectedPlatform"`
}

// writeSaveRequest is the JSON body of POST
// /api/v1/save-sessions/{saveSessionID}/write. Both values reach WriteSave
// exactly as sent; the route owns no revision or path rule.
type writeSaveRequest struct {
	ExpectedRevision string `json:"expectedRevision"`
	Target           string `json:"target"`
}

// setSaveAccountIDRequest is the strict JSON body of the account-identifier
// route. accountID stays a string so a large identifier survives JSON, and both
// values reach the endpoint exactly as sent.
type setSaveAccountIDRequest struct {
	AccountID        string `json:"accountID"`
	ExpectedRevision string `json:"expectedRevision"`
}

// deleteCharacterRequest is the strict JSON body of the permanent slot
// deletion route. SaveEngine owns the revision and deletion rules.
type deleteCharacterRequest struct {
	ExpectedRevision string `json:"expectedRevision"`
}

// cloneCharacterRequest is the strict JSON body of the slot-cloning route. A
// pointer distinguishes an omitted targetSlotID from the valid slot zero.
type cloneCharacterRequest struct {
	TargetSlotID     *int   `json:"targetSlotID"`
	ExpectedRevision string `json:"expectedRevision"`
}

// setCharacterActiveRequest is the strict JSON body of the slot-activity
// route. A pointer distinguishes an omitted active field from explicit false.
type setCharacterActiveRequest struct {
	Active           *bool  `json:"active"`
	ExpectedRevision string `json:"expectedRevision"`
}

// setCharacterNameRequest is the strict JSON body of the character-name route.
// Both values reach the endpoint unchanged; the transport owns no Unicode,
// length or revision rule.
type setCharacterNameRequest struct {
	Name             string `json:"name"`
	ExpectedRevision string `json:"expectedRevision"`
}

// setCharacterGenderRequest is the strict JSON body of the body-type route. A
// pointer distinguishes an omitted gender from the valid Type B value zero.
type setCharacterGenderRequest struct {
	Gender           *uint8 `json:"gender"`
	ExpectedRevision string `json:"expectedRevision"`
}

// setCharacterRunesRequest is the strict JSON body of the held-runes route. A
// pointer distinguishes an omitted runes field from an explicit zero.
type setCharacterRunesRequest struct {
	Runes            *uint32 `json:"runes"`
	ExpectedRevision string  `json:"expectedRevision"`
}

// undoCharacterChangesRequest is the strict JSON body of the character undo
// route. Both values reach SaveEngine exactly as sent; the transport owns no
// token or revision rule.
type undoCharacterChangesRequest struct {
	UndoToken        string `json:"undoToken"`
	ExpectedRevision string `json:"expectedRevision"`
}

// setCharacterAttributesRequest keeps every one of the eight mandatory
// attributes distinguishable from an omitted field, so a missing attribute is
// rejected instead of reaching SaveEngine as the illegal value zero.
type setCharacterAttributesRequest struct {
	Vigor        *uint32 `json:"vigor"`
	Mind         *uint32 `json:"mind"`
	Endurance    *uint32 `json:"endurance"`
	Strength     *uint32 `json:"strength"`
	Dexterity    *uint32 `json:"dexterity"`
	Intelligence *uint32 `json:"intelligence"`
	Faith        *uint32 `json:"faith"`
	Arcane       *uint32 `json:"arcane"`
}

// setCharacterStatsRequest is the strict JSON body of the statistics route. The
// level is not an input: SaveEngine recalculates it from the attributes.
type setCharacterStatsRequest struct {
	Attributes       *setCharacterAttributesRequest `json:"attributes"`
	LevelPolicy      string                         `json:"levelPolicy"`
	ExpectedRevision string                         `json:"expectedRevision"`
}

func (request setCharacterAttributesRequest) values() (character.CharacterAttributes, error) {
	values := character.CharacterAttributes{}
	for _, field := range []struct {
		name   string
		source *uint32
		target *uint32
	}{
		{"vigor", request.Vigor, &values.Vigor},
		{"mind", request.Mind, &values.Mind},
		{"endurance", request.Endurance, &values.Endurance},
		{"strength", request.Strength, &values.Strength},
		{"dexterity", request.Dexterity, &values.Dexterity},
		{"intelligence", request.Intelligence, &values.Intelligence},
		{"faith", request.Faith, &values.Faith},
		{"arcane", request.Arcane, &values.Arcane},
	} {
		if field.source == nil {
			return character.CharacterAttributes{},
				fmt.Errorf("attributes.%s is required", field.name)
		}
		*field.target = *field.source
	}
	return values, nil
}

// setCharacterAppearanceValuesRequest keeps required zero-valued scalar fields
// distinguishable from omitted ones and validates the four physical array
// lengths before converting them to SaveEngine's fixed-size model.
type setCharacterAppearanceValuesRequest struct {
	Gender    *uint8   `json:"gender"`
	VoiceType *uint8   `json:"voiceType"`
	ModelIDs  []uint32 `json:"modelIDs"`
	FaceShape []uint8  `json:"faceShape"`
	Body      []uint8  `json:"body"`
	Skin      []uint8  `json:"skin"`
}

type setCharacterAppearanceRequest struct {
	Appearance       *setCharacterAppearanceValuesRequest `json:"appearance"`
	ExpectedRevision string                               `json:"expectedRevision"`
}

// applyAppearancePresetRequest is the strict JSON body of the appearance
// preset route. The endpoint resolves presetID exactly.
type applyAppearancePresetRequest struct {
	PresetID         string `json:"presetID"`
	ExpectedRevision string `json:"expectedRevision"`
}

func (request setCharacterAppearanceValuesRequest) values() (character.CharacterAppearanceValues, error) {
	if request.Gender == nil {
		return character.CharacterAppearanceValues{}, errors.New("appearance.gender is required")
	}
	if request.VoiceType == nil {
		return character.CharacterAppearanceValues{}, errors.New("appearance.voiceType is required")
	}
	for _, field := range []struct {
		name string
		got  int
		want int
	}{
		{"modelIDs", len(request.ModelIDs), 8},
		{"faceShape", len(request.FaceShape), 64},
		{"body", len(request.Body), 7},
		{"skin", len(request.Skin), 91},
	} {
		if field.got != field.want {
			return character.CharacterAppearanceValues{}, fmt.Errorf(
				"appearance.%s has %d values, want exactly %d", field.name, field.got, field.want)
		}
	}

	values := character.CharacterAppearanceValues{
		Gender:    *request.Gender,
		VoiceType: *request.VoiceType,
	}
	copy(values.ModelIDs[:], request.ModelIDs)
	copy(values.FaceShape[:], request.FaceShape)
	copy(values.Body[:], request.Body)
	copy(values.Skin[:], request.Skin)
	return values, nil
}

// setPhysickMixtureRequest is the strict JSON body of the complete two-position
// Physick assignment. A null entry clears that exact position; the endpoint
// validates the required length and every catalog reference.
type setPhysickMixtureRequest struct {
	CrystalTearResources []*schema.ResourceRef `json:"crystalTearResources"`
	ExpectedRevision     string                `json:"expectedRevision"`
}

// setPouchItemsRequest is the strict JSON body of the complete six-position
// Pouch assignment.
type setPouchItemsRequest struct {
	SlotAssignments  []*string `json:"slotAssignments"`
	ExpectedRevision string    `json:"expectedRevision"`
}

// setQuickItemsRequest is the strict JSON body of the complete ten-position
// Quick Items assignment.
type setQuickItemsRequest struct {
	SlotAssignments  []*string `json:"slotAssignments"`
	ExpectedRevision string    `json:"expectedRevision"`
}

// setEquippedArmorRequest is the strict JSON body of the complete head,
// chest, arms and legs assignment.
type setEquippedArmorRequest struct {
	SlotAssignments  []*string `json:"slotAssignments"`
	ExpectedRevision string    `json:"expectedRevision"`
}

// setEquippedArmamentsRequest is the strict JSON body of the complete six-slot
// hand-armament assignment.
type setEquippedArmamentsRequest struct {
	SlotAssignments  []*string `json:"slotAssignments"`
	ExpectedRevision string    `json:"expectedRevision"`
}

// setEquippedSpellsRequest is the strict JSON body of the equipped spells route.
type setEquippedSpellsRequest struct {
	OrderedResources []*schema.ResourceRef `json:"orderedResources"`
	ExpectedRevision string                `json:"expectedRevision"`
}

// setEquippedTalismansRequest is the strict JSON body of the compact talisman loadout route.
type setEquippedTalismansRequest struct {
	OrderedOwnedItemIDs []string `json:"orderedOwnedItemIDs"`
	ExpectedRevision    string   `json:"expectedRevision"`
}

// setOwnedItemQuantityRequest is the strict JSON body of the quantity route.
// The transport owns no quantity or revision rule; both values reach the
// endpoint unchanged.
type setOwnedItemQuantityRequest struct {
	Quantity         uint32 `json:"quantity"`
	ExpectedRevision string `json:"expectedRevision"`
}

// setInventoryOrderRequest is the strict JSON body of the complete supported
// Inventory common order. SaveEngine and GameCatalog own every identity,
// category, completeness, revision and acquisition-index rule.
type setInventoryOrderRequest struct {
	OrderedOwnedItemIDs []string `json:"orderedOwnedItemIDs"`
	ExpectedRevision    string   `json:"expectedRevision"`
}

// setStorageOrderRequest is the strict JSON body of the complete supported
// Storage common order. SaveEngine and GameCatalog own every identity,
// category, completeness, revision and acquisition-index rule.
type setStorageOrderRequest struct {
	OrderedOwnedItemIDs []string `json:"orderedOwnedItemIDs"`
	ExpectedRevision    string   `json:"expectedRevision"`
}

// moveOwnedItemToInventoryRequest is the strict JSON body of the move route.
type moveOwnedItemToInventoryRequest struct {
	TargetPosition   *int   `json:"targetPosition"`
	ExpectedRevision string `json:"expectedRevision"`
}

// moveOwnedItemToStorageRequest is the strict JSON body of the move route.
// The endpoint owns the position, revision and destination-limit rules.
type moveOwnedItemToStorageRequest struct {
	TargetPosition   *int   `json:"targetPosition"`
	ExpectedRevision string `json:"expectedRevision"`
}

// setUpgradeLevelRequest is the strict JSON body shared by the two owned-item
// upgrade routes. SaveEngine and GameCatalog own every level and mutation rule.
type setUpgradeLevelRequest struct {
	UpgradeLevel     *uint8 `json:"upgradeLevel"`
	ExpectedRevision string `json:"expectedRevision"`
}

// setWeaponInfusionRequest is the strict JSON body of the weapon infusion
// route. GameCatalog owns the affinity vocabulary and compatibility rules.
type setWeaponInfusionRequest struct {
	Affinity         schema.Affinity `json:"affinity"`
	ExpectedRevision string          `json:"expectedRevision"`
}

// setWeaponAshOfWarRequest keeps the two resource selectors as raw JSON so the
// transport can distinguish an explicit null (remove) from an omitted field.
type setWeaponAshOfWarRequest struct {
	AshOfWarKind     json.RawMessage `json:"ashOfWarKind"`
	AshOfWarKey      json.RawMessage `json:"ashOfWarKey"`
	ExpectedRevision string          `json:"expectedRevision"`
}

func requiredNullableString(raw json.RawMessage, name string) (*string, error) {
	if raw == nil {
		return nil, fmt.Errorf("%s is required", name)
	}
	if string(raw) == "null" {
		return nil, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s must be a string or null: %w", name, err)
	}
	return &value, nil
}

// addItemToInventoryRequest is the strict JSON body of the add route. The
// transport owns no catalog, family, routing, quantity or revision rule; every
// value reaches the endpoint unchanged, and an absent variantID stays absent
// rather than becoming a zero.
type addItemToInventoryRequest struct {
	Kind             string  `json:"kind"`
	Key              string  `json:"key"`
	VariantID        *uint32 `json:"variantID"`
	Quantity         uint32  `json:"quantity"`
	ExpectedRevision string  `json:"expectedRevision"`
}

// addItemToStorageRequest is the strict JSON body of the common-Storage add
// route. Every value reaches the endpoint unchanged.
type addItemToStorageRequest struct {
	Kind             string  `json:"kind"`
	Key              string  `json:"key"`
	VariantID        *uint32 `json:"variantID"`
	Quantity         uint32  `json:"quantity"`
	ExpectedRevision string  `json:"expectedRevision"`
}

// removeOwnedItemRequest is the strict JSON body of the removal route. The
// revision travels in the body like it does for every other mutation, so the
// transport keeps one convention; it reaches the endpoint unchanged.
type removeOwnedItemRequest struct {
	ExpectedRevision string `json:"expectedRevision"`
}

// setCookbookUnlockedRequest is the strict JSON body of the set cookbook unlocked route.
type setCookbookUnlockedRequest struct {
	CookbookKind     string `json:"cookbookKind"`
	CookbookKey      string `json:"cookbookKey"`
	Unlocked         *bool  `json:"unlocked"`
	ExpectedRevision string `json:"expectedRevision"`
}

// setBellBearingUnlockedRequest is the strict JSON body of the Bell Bearing mutation.
type setBellBearingUnlockedRequest struct {
	BellBearingKind  string `json:"bellBearingKind"`
	BellBearingKey   string `json:"bellBearingKey"`
	Unlocked         *bool  `json:"unlocked"`
	ExpectedRevision string `json:"expectedRevision"`
}

// setWhetbladeUnlockedRequest is the strict JSON body of the Whetblade mutation.
type setWhetbladeUnlockedRequest struct {
	WhetbladeKind    string `json:"whetbladeKind"`
	WhetbladeKey     string `json:"whetbladeKey"`
	Unlocked         *bool  `json:"unlocked"`
	ExpectedRevision string `json:"expectedRevision"`
}

// setGestureUnlockedRequest is the strict JSON body of the gesture mutation.
type setGestureUnlockedRequest struct {
	GestureKind      string `json:"gestureKind"`
	GestureKey       string `json:"gestureKey"`
	Unlocked         *bool  `json:"unlocked"`
	ExpectedRevision string `json:"expectedRevision"`
}

// setNetworkSettingsRequest is the strict JSON body of the complete network
// settings replacement. SaveEngine owns every field and cross-field rule.
type setNetworkSettingsRequest struct {
	NetworkSettings  gamecatalog.NetworkParamValues `json:"networkSettings"`
	ExpectedRevision string                         `json:"expectedRevision"`
}

// applyNetworkPresetRequest is the strict JSON body of the preset application
// route. The endpoint resolves presetID; the transport does not normalise it.
type applyNetworkPresetRequest struct {
	PresetID         string `json:"presetID"`
	ExpectedRevision string `json:"expectedRevision"`
}

// closeSaveResponse is the confirmation of DELETE /api/v1/save-sessions/{id}.
// CloseSave returns no value, so the route states the closed session itself.
type closeSaveResponse struct {
	SaveSessionID string `json:"saveSessionID"`
	Closed        bool   `json:"closed"`
}

// registerSaveSessionRoutes exposes the implemented save-session lifecycle and
// character getters. Every route calls exactly one endpoint and
// serialises its typed result; the source path, the save content and the
// character data are never logged. The catalog is passed on to the one getter
// that resolves save values into catalog documents.
func registerSaveSessionRoutes(
	mux *http.ServeMux,
	saveEngine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
) {
	mux.HandleFunc("POST /api/v1/save-sessions", func(writer http.ResponseWriter, request *http.Request) {
		if err := requireJSONBody(request); err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		var body loadSaveRequest
		decoder := json.NewDecoder(request.Body)
		// An unknown field is a client mistake, not something to ignore silently.
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		result, err := savesession.LoadSave(saveEngine, body.Source, body.ExpectedPlatform)
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	mux.HandleFunc("GET /api/v1/save-sessions/{saveSessionID}", func(writer http.ResponseWriter, request *http.Request) {
		result, err := savesession.GetLoadedSave(saveEngine, request.PathValue("saveSessionID"))
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	mux.HandleFunc("POST /api/v1/save-sessions/{saveSessionID}/write", func(writer http.ResponseWriter, request *http.Request) {
		if err := requireJSONBody(request); err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		var body writeSaveRequest
		decoder := json.NewDecoder(request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		result, err := savesession.WriteSave(
			saveEngine,
			request.PathValue("saveSessionID"),
			body.ExpectedRevision,
			body.Target,
		)
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	mux.HandleFunc(
		"PATCH /api/v1/save-sessions/{saveSessionID}/account-id",
		func(writer http.ResponseWriter, request *http.Request) {
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setSaveAccountIDRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := savesession.SetSaveAccountID(
				saveEngine,
				request.PathValue("saveSessionID"),
				body.AccountID,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc("DELETE /api/v1/save-sessions/{saveSessionID}", func(writer http.ResponseWriter, request *http.Request) {
		saveSessionID := request.PathValue("saveSessionID")
		if err := savesession.CloseSave(saveEngine, saveSessionID); err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, closeSaveResponse{SaveSessionID: saveSessionID, Closed: true})
	})

	mux.HandleFunc("GET /api/v1/save-sessions/{saveSessionID}/characters", func(writer http.ResponseWriter, request *http.Request) {
		result, err := character.GetSaveCharacters(saveEngine, request.PathValue("saveSessionID"))
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/profile",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := character.GetCharacterProfile(saveEngine, request.PathValue("saveSessionID"), characterID)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"POST /api/v1/save-sessions/{saveSessionID}/characters/{sourceCharacterID}/clone",
		func(writer http.ResponseWriter, request *http.Request) {
			sourceCharacterID, err := parseCharacterID(
				request.PathValue("sourceCharacterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body cloneCharacterRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.TargetSlotID == nil {
				writeError(writer, http.StatusBadRequest, errors.New("targetSlotID is required"))
				return
			}
			result, err := character.CloneCharacter(
				saveEngine,
				request.PathValue("saveSessionID"),
				sourceCharacterID,
				*body.TargetSlotID,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"DELETE /api/v1/save-sessions/{saveSessionID}/characters/{characterID}",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body deleteCharacterRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := character.DeleteCharacter(
				saveEngine,
				request.PathValue("saveSessionID"),
				characterID,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/active",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setCharacterActiveRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.Active == nil {
				writeError(writer, http.StatusBadRequest, errors.New("active is required"))
				return
			}
			result, err := character.SetCharacterActive(
				saveEngine,
				request.PathValue("saveSessionID"),
				characterID,
				*body.Active,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/name",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setCharacterNameRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := character.SetCharacterName(
				saveEngine,
				request.PathValue("saveSessionID"),
				characterID,
				body.Name,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/gender",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setCharacterGenderRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.Gender == nil {
				writeError(writer, http.StatusBadRequest, errors.New("gender is required"))
				return
			}
			result, err := character.SetCharacterGender(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				*body.Gender,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/runes",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setCharacterRunesRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.Runes == nil {
				writeError(writer, http.StatusBadRequest, errors.New("runes is required"))
				return
			}
			result, err := character.SetCharacterRunes(
				saveEngine,
				request.PathValue("saveSessionID"),
				characterID,
				*body.Runes,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/stats",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setCharacterStatsRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.Attributes == nil {
				writeError(writer, http.StatusBadRequest, errors.New("attributes is required"))
				return
			}
			attributes, err := body.Attributes.values()
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := character.SetCharacterStats(
				saveEngine,
				request.PathValue("saveSessionID"),
				characterID,
				attributes,
				body.LevelPolicy,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/stats",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := character.GetCharacterStats(saveEngine, request.PathValue("saveSessionID"), characterID)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/undo",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := character.GetUndoState(saveEngine, request.PathValue("saveSessionID"), characterID)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"POST /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/undo",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body undoCharacterChangesRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := character.UndoCharacterChanges(
				saveEngine,
				request.PathValue("saveSessionID"),
				characterID,
				body.UndoToken,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := character.GetCharacterAppearance(saveEngine, request.PathValue("saveSessionID"), characterID)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setCharacterAppearanceRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.Appearance == nil {
				writeError(writer, http.StatusBadRequest, errors.New("appearance is required"))
				return
			}
			values, err := body.Appearance.values()
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := character.SetCharacterAppearance(
				saveEngine,
				request.PathValue("saveSessionID"),
				characterID,
				values,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance/preset",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body applyAppearancePresetRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := appearance.ApplyAppearancePreset(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.PresetID,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipment",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := equipment.GetEquipment(saveEngine, request.PathValue("saveSessionID"), characterID)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-armaments",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setEquippedArmamentsRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := equipment.SetEquippedArmaments(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.SlotAssignments,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-armor",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setEquippedArmorRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := equipment.SetEquippedArmor(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.SlotAssignments,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/quick-items",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := equipment.GetQuickItems(saveEngine, request.PathValue("saveSessionID"), characterID)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/quick-items",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setQuickItemsRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := equipment.SetQuickItems(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.SlotAssignments,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/pouch-items",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := equipment.GetPouchItems(saveEngine, request.PathValue("saveSessionID"), characterID)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/pouch-items",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setPouchItemsRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := equipment.SetPouchItems(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.SlotAssignments,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/physick-mixture",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := equipment.GetPhysickMixture(saveEngine, request.PathValue("saveSessionID"), characterID)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/physick-mixture",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setPhysickMixtureRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := equipment.SetPhysickMixture(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.CrystalTearResources,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-spells",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := equipment.GetEquippedSpells(
				saveEngine, gameCatalog, request.PathValue("saveSessionID"), characterID)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-spells",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setEquippedSpellsRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := equipment.SetEquippedSpells(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.OrderedResources,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-talismans",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setEquippedTalismansRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.OrderedOwnedItemIDs == nil {
				writeError(writer, http.StatusBadRequest, errors.New("orderedOwnedItemIDs is required"))
				return
			}
			result, err := equipment.SetEquippedTalismans(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.OrderedOwnedItemIDs,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	// The inventory route is the one save-session route with paging, so it uses
	// the same query parsing as the catalog list routes. It calls
	// inventory.GetInventory, which resolves every record through the catalog.
	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			query := request.URL.Query()
			page, err := parsePagingValue(query.Get("page"), "page")
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			pageSize, err := parsePagingValue(query.Get("pageSize"), "pageSize")
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := inventory.GetInventory(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				query.Get("containerSection"),
				page,
				pageSize,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/item-capacity",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			query := request.URL.Query()
			variantID, err := parseOptionalUint32(query.Get("variantID"), "variantID")
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			quantity, err := parseRequiredUint32(query.Get("quantity"), "quantity")
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := inventory.GetItemCapacity(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				query.Get("destination"),
				query.Get("kind"),
				query.Get("key"),
				variantID,
				quantity,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory/order",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setInventoryOrderRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.OrderedOwnedItemIDs == nil {
				writeError(writer, http.StatusBadRequest, errors.New("orderedOwnedItemIDs is required"))
				return
			}
			result, err := inventory.SetInventoryOrder(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.OrderedOwnedItemIDs,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	// Adding an item creates or tops up a record of the same collection the
	// inventory route lists, so it posts to that collection. The route parses
	// only the typed path/body envelope and delegates every catalog, family,
	// routing, limit, revision and mutation rule to AddItemToInventory.
	mux.HandleFunc(
		"POST /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory/items",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			// This is a session-mutating POST, so it carries the same
			// simple-request guard the two other POST routes do.
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body addItemToInventoryRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := inventory.AddItemToInventory(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.Kind,
				body.Key,
				body.VariantID,
				body.Quantity,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	// The storage route mirrors the inventory route: it is the second
	// save-session route with paging and resolves its records through the catalog
	// the same way. It calls GetStorage and nothing else, so the two containers
	// stay independent.
	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/storage",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			query := request.URL.Query()
			page, err := parsePagingValue(query.Get("page"), "page")
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			pageSize, err := parsePagingValue(query.Get("pageSize"), "pageSize")
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := inventory.GetStorage(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				query.Get("containerSection"),
				page,
				pageSize,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"POST /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/storage/items",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body addItemToStorageRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := inventory.AddItemToStorage(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.Kind,
				body.Key,
				body.VariantID,
				body.Quantity,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/storage/order",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setStorageOrderRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.OrderedOwnedItemIDs == nil {
				writeError(writer, http.StatusBadRequest, errors.New("orderedOwnedItemIDs is required"))
				return
			}
			result, err := inventory.SetStorageOrder(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.OrderedOwnedItemIDs,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	// The owned-item route addresses one instance instead of a page, so the
	// opaque identifier is the last path segment. It is handed over exactly as it
	// arrived: the transport never parses, trims, normalises or reconstructs it,
	// and it owns no identity rule of its own.
	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := inventory.GetOwnedItem(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				request.PathValue("ownedItemID"),
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	// Removing one instance addresses the same path as reading it, with the
	// method carrying the intent. The route parses only the typed path/body
	// envelope and delegates every identity, revision and mutation rule to
	// RemoveOwnedItem, which needs no catalog.
	mux.HandleFunc(
		"DELETE /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body removeOwnedItemRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := inventory.RemoveOwnedItem(
				saveEngine,
				request.PathValue("saveSessionID"),
				characterID,
				request.PathValue("ownedItemID"),
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	// Quantity is the only owned-item field mutation currently exposed. The route
	// parses only the typed path/body envelope and delegates every identity,
	// catalog-limit, revision and mutation rule to SetOwnedItemQuantity.
	mux.HandleFunc(
		"PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/quantity",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setOwnedItemQuantityRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := inventory.SetOwnedItemQuantity(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				request.PathValue("ownedItemID"),
				body.Quantity,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"POST /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/move-to-inventory",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body moveOwnedItemToInventoryRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.TargetPosition == nil {
				writeError(writer, http.StatusBadRequest, errors.New("targetPosition is required"))
				return
			}
			result, err := inventory.MoveOwnedItemToInventory(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				request.PathValue("ownedItemID"),
				*body.TargetPosition,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"POST /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/move-to-storage",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body moveOwnedItemToStorageRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.TargetPosition == nil {
				writeError(writer, http.StatusBadRequest, errors.New("targetPosition is required"))
				return
			}
			result, err := inventory.MoveOwnedItemToStorage(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				request.PathValue("ownedItemID"),
				*body.TargetPosition,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/upgrade-level",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setUpgradeLevelRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.UpgradeLevel == nil {
				writeError(writer, http.StatusBadRequest, errors.New("upgradeLevel is required"))
				return
			}
			result, err := inventory.SetWeaponUpgradeLevel(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				request.PathValue("ownedItemID"),
				*body.UpgradeLevel,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/spirit-ash-upgrade-level",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setUpgradeLevelRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.UpgradeLevel == nil {
				writeError(writer, http.StatusBadRequest, errors.New("upgradeLevel is required"))
				return
			}
			result, err := inventory.SetSpiritAshUpgradeLevel(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				request.PathValue("ownedItemID"),
				*body.UpgradeLevel,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/infusion",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setWeaponInfusionRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.Affinity == "" {
				writeError(writer, http.StatusBadRequest, errors.New("affinity is required"))
				return
			}
			result, err := inventory.SetWeaponInfusion(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				request.PathValue("ownedItemID"),
				body.Affinity,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/ash-of-war",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setWeaponAshOfWarRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			ashOfWarKind, err := requiredNullableString(body.AshOfWarKind, "ashOfWarKind")
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			ashOfWarKey, err := requiredNullableString(body.AshOfWarKey, "ashOfWarKey")
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := inventory.SetWeaponAshOfWar(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				request.PathValue("ownedItemID"),
				ashOfWarKind,
				ashOfWarKey,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	// The gestures route is the one save-session route that joins two sources:
	// the raw GestureGameData block of the slot and the gesture definitions of
	// the catalog. availabilityFilter is handed over exactly as it arrived,
	// because the getter owns the rule and the wording of its own rejection.
	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/gestures",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := world.GetGestures(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				request.URL.Query().Get("availabilityFilter"),
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/gestures/unlock",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setGestureUnlockedRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.Unlocked == nil {
				writeError(writer, http.StatusBadRequest, errors.New("unlocked is required"))
				return
			}
			result, err := world.SetGestureUnlocked(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.GestureKind,
				body.GestureKey,
				*body.Unlocked,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	// The cookbooks route joins the cookbook definitions of the catalog with the
	// event flags of the slot. availabilityFilter is handed over exactly as it
	// arrived, because the getter owns the rule and the wording of its own
	// rejection.
	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/cookbooks",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := world.GetCookbooks(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				request.URL.Query().Get("availabilityFilter"),
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/bell-bearings",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := world.GetBellBearings(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				request.URL.Query().Get("availabilityFilter"),
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/bell-bearings/unlock",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setBellBearingUnlockedRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.Unlocked == nil {
				writeError(writer, http.StatusBadRequest, errors.New("unlocked is required"))
				return
			}
			result, err := world.SetBellBearingUnlocked(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.BellBearingKind,
				body.BellBearingKey,
				*body.Unlocked,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/whetblades",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := world.GetWhetblades(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				request.URL.Query().Get("availabilityFilter"),
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/colosseums",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := world.GetColosseums(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/regions",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := world.GetRegions(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/summoning-pools",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := world.GetSummoningPools(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/graces",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := world.GetGraces(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/bosses",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := world.GetBosses(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/whetblades/unlock",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setWhetbladeUnlockedRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.Unlocked == nil {
				writeError(writer, http.StatusBadRequest, errors.New("unlocked is required"))
				return
			}
			result, err := world.SetWhetbladeUnlocked(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.WhetbladeKind,
				body.WhetbladeKey,
				*body.Unlocked,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/cookbooks/unlock",
		func(writer http.ResponseWriter, request *http.Request) {
			characterID, err := parseCharacterID(request.PathValue("characterID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setCookbookUnlockedRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			if body.Unlocked == nil {
				writeError(writer, http.StatusBadRequest, errors.New("unlocked is required"))
				return
			}
			result, err := world.SetCookbookUnlocked(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				characterID,
				body.CookbookKind,
				body.CookbookKey,
				*body.Unlocked,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	// The network settings belong to the session, not to a character slot: they
	// are the regulation values stored once per save. The route passes the
	// identifier on unchanged and reads no catalog.
	mux.HandleFunc(
		"GET /api/v1/save-sessions/{saveSessionID}/network-settings",
		func(writer http.ResponseWriter, request *http.Request) {
			result, err := network.GetNetworkSettings(saveEngine, request.PathValue("saveSessionID"))
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/network-settings",
		func(writer http.ResponseWriter, request *http.Request) {
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body setNetworkSettingsRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := network.SetNetworkSettings(
				saveEngine,
				request.PathValue("saveSessionID"),
				body.NetworkSettings,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)

	mux.HandleFunc(
		"PUT /api/v1/save-sessions/{saveSessionID}/network-settings/preset",
		func(writer http.ResponseWriter, request *http.Request) {
			if err := requireJSONBody(request); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			var body applyNetworkPresetRequest
			decoder := json.NewDecoder(request.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			result, err := network.ApplyNetworkPreset(
				saveEngine,
				gameCatalog,
				request.PathValue("saveSessionID"),
				body.PresetID,
				body.ExpectedRevision,
			)
			if err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		},
	)
}

// parseCharacterID turns the path segment into the integer the character getters
// expect. It is decimal only and is never trimmed or defaulted, so text the
// getter could not report is rejected here; the allowed slot range stays a
// SaveEngine rule.
func parseCharacterID(raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("characterID must be an integer; got %q", raw)
	}
	return value, nil
}

// parseTags splits the single comma-separated tags parameter into the list the
// getter expects. An absent or empty parameter means "do not filter" and becomes
// nil; the individual tags are never trimmed, so an empty element stays empty
// and the getter rejects it.
func parseTags(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
}

// parsePagingValue turns a query string into the integer GetResources expects.
// An absent parameter stays 0, which is the getter's "use the default" value, so
// the HTTP layer never invents a page or a page size of its own. Anything that
// is not an integer is rejected here, because the getter only sees numbers and
// could not report the malformed text.
func parsePagingValue(raw string, name string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer; got %q", name, raw)
	}
	return value, nil
}

func parseOptionalUint32(raw string, name string) (*uint32, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("%s must be a decimal uint32; got %q", name, raw)
	}
	parsed := uint32(value)
	return &parsed, nil
}

func parseRequiredUint32(raw string, name string) (uint32, error) {
	if raw == "" {
		return 0, fmt.Errorf("%s is required", name)
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be a decimal uint32; got %q", name, raw)
	}
	return uint32(value), nil
}

func serveAsset(writer http.ResponseWriter, name string, contentType string) {
	content, err := assets.ReadFile(name)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err)
		return
	}
	writer.Header().Set("Content-Type", contentType)
	if _, err := writer.Write(content); err != nil {
		log.Printf("write %s: %v", name, err)
	}
}

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		log.Printf("encode response: %v", err)
	}
}

// writeError keeps the message the getter returned, because the getter owns the
// wording of its own validation rules.
// ponytail: no shared EndpointError type yet; endpoints.md defers that model
// until several endpoint contracts exist.
func writeError(writer http.ResponseWriter, status int, err error) {
	writeJSON(writer, status, map[string]string{"error": err.Error()})
}
