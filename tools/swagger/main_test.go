package main

import (
	"bytes"
	"compress/flate"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"unicode/utf16"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/appearance"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/application"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/character"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/equipment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/favorites"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/inventory"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/network"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/savesession"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/templates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/world"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// The prototype catalog holds real, schema-valid resources, so every route is
// exercised against the same data the getter tests use instead of a mock.
const (
	resourceKindItem  = "item"
	daggerResourceKey = "000F4240"
)

// testApplicationVersion stands in for the -app-version flag the explorer passes
// to newHandler.
const testApplicationVersion = "test-version"

var (
	prototypeCatalogOnce sync.Once
	prototypeCatalog     *gamecatalog.Catalog
	prototypeCatalogErr  error
)

// newPrototypeCatalog returns the prototype catalog shared by every route test.
// It is built once per test binary: parsing the embedded catalog on each of the
// calls in this package cost minutes of test time and exceeded the default
// go test timeout. GameCatalog is never mutated at runtime and every query
// returns an independent copy, so one instance is safe to share.
func newPrototypeCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	prototypeCatalogOnce.Do(func() {
		prototypeCatalog, prototypeCatalogErr = gamecatalog.NewPrototype()
	})
	if prototypeCatalogErr != nil {
		t.Fatalf("gamecatalog.NewPrototype: %v", prototypeCatalogErr)
	}
	return prototypeCatalog
}

func do(t *testing.T, gameCatalog *gamecatalog.Catalog, target string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	// The catalog routes need no save engine, so they are served by a handler
	// with the save-session routes disabled.
	newHandler(gameCatalog, testApplicationVersion, nil).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	return recorder
}

// decode returns the response body as generic JSON so a route can be compared
// with the getter result semantically instead of byte by byte.
func decode(t *testing.T, payload []byte) any {
	t.Helper()

	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode JSON %q: %v", string(payload), err)
	}
	return decoded
}

func marshalled(t *testing.T, value any) any {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal getter result: %v", err)
	}
	return decode(t, raw)
}

func assertJSONContentType(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}
}

func assertErrorMessage(t *testing.T, recorder *httptest.ResponseRecorder, status int, want error) {
	t.Helper()

	if recorder.Code != status {
		t.Fatalf("status = %d, want %d (body %q)", recorder.Code, status, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error body %q: %v", recorder.Body.String(), err)
	}
	if want == nil {
		t.Fatal("the getter must fail for this input, otherwise the route is not comparable")
	}
	if payload.Error != want.Error() {
		t.Fatalf("error = %q, want the getter message %q", payload.Error, want.Error())
	}
}

func TestApplicationInfoRouteMatchesGetter(t *testing.T) {
	want, err := application.GetApplicationInfo(testApplicationVersion)
	if err != nil {
		t.Fatalf("application.GetApplicationInfo: %v", err)
	}

	// The route never touches the catalog, so building one would only slow the test.
	recorder := do(t, nil, "/api/v1/application/info")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	got := decode(t, recorder.Body.Bytes())
	if !reflect.DeepEqual(got, marshalled(t, want)) {
		t.Fatalf("body = %#v, want the GetApplicationInfo result %#v", got, marshalled(t, want))
	}
}

// An empty version is a backend configuration error, not a client error.
func TestApplicationInfoRouteRejectsAnEmptyVersion(t *testing.T) {
	recorder := httptest.NewRecorder()
	newHandler(nil, "", nil).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/application/info", nil),
	)

	_, want := application.GetApplicationInfo("")
	assertErrorMessage(t, recorder, http.StatusInternalServerError, want)
}

func TestCatalogInfoRouteMatchesGetter(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	want, err := catalog.GetCatalogInfo(gameCatalog)
	if err != nil {
		t.Fatalf("catalog.GetCatalogInfo: %v", err)
	}

	recorder := do(t, gameCatalog, "/api/v1/catalog/info")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	got := decode(t, recorder.Body.Bytes())
	if !reflect.DeepEqual(got, marshalled(t, want)) {
		t.Fatalf("body = %#v, want the GetCatalogInfo result %#v", got, marshalled(t, want))
	}
}

func TestResourceRouteMatchesGetter(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	want, err := catalog.GetResource(gameCatalog, resourceKindItem, daggerResourceKey)
	if err != nil {
		t.Fatalf("catalog.GetResource: %v", err)
	}

	recorder := do(t, gameCatalog, "/api/v1/catalog/resource?kind="+resourceKindItem+"&key="+daggerResourceKey)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	got := decode(t, recorder.Body.Bytes())
	if !reflect.DeepEqual(got, marshalled(t, want)) {
		t.Fatal("resource route body differs from the GetResource result")
	}
}

func TestResourceRouteRejectsMissingKindAndKey(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	_, wantMissingKind := catalog.GetResource(gameCatalog, "", "")
	assertErrorMessage(t, do(t, gameCatalog, "/api/v1/catalog/resource"), http.StatusBadRequest, wantMissingKind)

	_, wantMissingKey := catalog.GetResource(gameCatalog, resourceKindItem, "")
	assertErrorMessage(
		t,
		do(t, gameCatalog, "/api/v1/catalog/resource?kind="+resourceKindItem),
		http.StatusBadRequest,
		wantMissingKey,
	)
}

func TestResourceRouteReportsUnknownKindAndKey(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	const unknownKey = "FFFFFFFF"
	_, wantUnknownKey := catalog.GetResource(gameCatalog, resourceKindItem, unknownKey)
	assertErrorMessage(
		t,
		do(t, gameCatalog, "/api/v1/catalog/resource?kind="+resourceKindItem+"&key="+unknownKey),
		http.StatusBadRequest,
		wantUnknownKey,
	)

	_, wantUnknownKind := catalog.GetResource(gameCatalog, "gesture", daggerResourceKey)
	assertErrorMessage(
		t,
		do(t, gameCatalog, "/api/v1/catalog/resource?kind=gesture&key="+daggerResourceKey),
		http.StatusBadRequest,
		wantUnknownKind,
	)
}

func TestResourceRelationsRouteMatchesGetter(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	want, err := catalog.GetResourceRelations(gameCatalog, resourceKindItem, daggerResourceKey, "", "")
	if err != nil {
		t.Fatalf("catalog.GetResourceRelations: %v", err)
	}

	recorder := do(t, gameCatalog, "/api/v1/catalog/resource-relations?kind="+resourceKindItem+"&key="+daggerResourceKey)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	got := decode(t, recorder.Body.Bytes())
	if !reflect.DeepEqual(got, marshalled(t, want)) {
		t.Fatal("resource relations route body differs from the GetResourceRelations result")
	}
}

// The route owns no filtering of its own, so all four query parameters have to
// reach the getter unchanged.
func TestResourceRelationsRoutePassesEveryParameter(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	want, err := catalog.GetResourceRelations(
		gameCatalog,
		resourceKindItem,
		daggerResourceKey,
		"compatible_with_aow",
		"outgoing",
	)
	if err != nil {
		t.Fatalf("catalog.GetResourceRelations: %v", err)
	}
	if len(want.Outgoing) == 0 {
		t.Fatal("the filtered getter result is empty, so the route comparison would not prove anything")
	}

	recorder := do(
		t,
		gameCatalog,
		"/api/v1/catalog/resource-relations?kind="+resourceKindItem+"&key="+daggerResourceKey+
			"&relationType=compatible_with_aow&direction=outgoing",
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}

	got := decode(t, recorder.Body.Bytes())
	if !reflect.DeepEqual(got, marshalled(t, want)) {
		t.Fatal("filtered resource relations route body differs from the GetResourceRelations result")
	}
}

func TestResourceRelationsRouteRejectsInvalidInput(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	_, wantMissingKind := catalog.GetResourceRelations(gameCatalog, "", "", "", "")
	assertErrorMessage(
		t,
		do(t, gameCatalog, "/api/v1/catalog/resource-relations"),
		http.StatusBadRequest,
		wantMissingKind,
	)

	_, wantMissingKey := catalog.GetResourceRelations(gameCatalog, resourceKindItem, "", "", "")
	assertErrorMessage(
		t,
		do(t, gameCatalog, "/api/v1/catalog/resource-relations?kind="+resourceKindItem),
		http.StatusBadRequest,
		wantMissingKey,
	)

	const unknownKey = "FFFFFFFF"
	_, wantUnknownKey := catalog.GetResourceRelations(gameCatalog, resourceKindItem, unknownKey, "", "")
	assertErrorMessage(
		t,
		do(t, gameCatalog, "/api/v1/catalog/resource-relations?kind="+resourceKindItem+"&key="+unknownKey),
		http.StatusBadRequest,
		wantUnknownKey,
	)

	_, wantUnknownDirection := catalog.GetResourceRelations(gameCatalog, resourceKindItem, daggerResourceKey, "", "both")
	assertErrorMessage(
		t,
		do(
			t,
			gameCatalog,
			"/api/v1/catalog/resource-relations?kind="+resourceKindItem+"&key="+daggerResourceKey+"&direction=both",
		),
		http.StatusBadRequest,
		wantUnknownDirection,
	)
}

func TestItemVariantsRouteMatchesGetter(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	want, err := catalog.GetItemVariants(gameCatalog, resourceKindItem, daggerResourceKey)
	if err != nil {
		t.Fatalf("catalog.GetItemVariants: %v", err)
	}

	recorder := do(t, gameCatalog, "/api/v1/catalog/item-variants?kind="+resourceKindItem+"&key="+daggerResourceKey)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	got := decode(t, recorder.Body.Bytes())
	if !reflect.DeepEqual(got, marshalled(t, want)) {
		t.Fatal("item variants route body differs from the GetItemVariants result")
	}
}

func TestItemVariantsRouteRejectsMissingKindAndKey(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	_, wantMissingKind := catalog.GetItemVariants(gameCatalog, "", "")
	assertErrorMessage(t, do(t, gameCatalog, "/api/v1/catalog/item-variants"), http.StatusBadRequest, wantMissingKind)

	_, wantMissingKey := catalog.GetItemVariants(gameCatalog, resourceKindItem, "")
	assertErrorMessage(
		t,
		do(t, gameCatalog, "/api/v1/catalog/item-variants?kind="+resourceKindItem),
		http.StatusBadRequest,
		wantMissingKey,
	)
}

func TestItemVariantsRouteReportsUnknownKindAndKey(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	const unknownKey = "FFFFFFFF"
	_, wantUnknownKey := catalog.GetItemVariants(gameCatalog, resourceKindItem, unknownKey)
	assertErrorMessage(
		t,
		do(t, gameCatalog, "/api/v1/catalog/item-variants?kind="+resourceKindItem+"&key="+unknownKey),
		http.StatusBadRequest,
		wantUnknownKey,
	)

	_, wantUnsupportedKind := catalog.GetItemVariants(gameCatalog, "gesture", daggerResourceKey)
	assertErrorMessage(
		t,
		do(t, gameCatalog, "/api/v1/catalog/item-variants?kind=gesture&key="+daggerResourceKey),
		http.StatusBadRequest,
		wantUnsupportedKind,
	)
}

func TestResourcesRouteMatchesGetter(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	want, err := catalog.GetResources(gameCatalog, resourceKindItem, "", "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("catalog.GetResources: %v", err)
	}

	recorder := do(t, gameCatalog, "/api/v1/catalog/resources?resourceType="+resourceKindItem)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	got := decode(t, recorder.Body.Bytes())
	if !reflect.DeepEqual(got, marshalled(t, want)) {
		t.Fatal("resources route body differs from the GetResources result")
	}
}

func TestResourcesRouteRejectsInvalidPaging(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	// A non-integer page never reaches the getter, so the route owns the message.
	for _, target := range []string{
		"/api/v1/catalog/resources?page=first",
		"/api/v1/catalog/resources?pageSize=all",
	} {
		if recorder := do(t, gameCatalog, target); recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %q)", target, recorder.Code, recorder.Body.String())
		}
	}

	// A negative page is a getter rule, so the route must carry its wording.
	_, wantNegativePage := catalog.GetResources(gameCatalog, "", "", "", "", "", -1, 0)
	assertErrorMessage(
		t,
		do(t, gameCatalog, "/api/v1/catalog/resources?page=-1"),
		http.StatusBadRequest,
		wantNegativePage,
	)

	_, wantEndpointID := catalog.GetResources(gameCatalog, "", "", "", "get_resource", "", 0, 0)
	assertErrorMessage(
		t,
		do(t, gameCatalog, "/api/v1/catalog/resources?endpointId=get_resource"),
		http.StatusBadRequest,
		wantEndpointID,
	)
}

func TestNetworkPresetsRouteMatchesGetter(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)
	want, err := network.GetNetworkPresets(gameCatalog, "")
	if err != nil {
		t.Fatalf("network.GetNetworkPresets: %v", err)
	}

	recorder := do(t, gameCatalog, "/api/v1/network/presets")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	got := decode(t, recorder.Body.Bytes())
	if !reflect.DeepEqual(got, marshalled(t, want)) {
		t.Fatal("network presets route body differs from the GetNetworkPresets result")
	}
}

func TestNetworkPresetsRouteFiltersByPresetID(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)
	want, err := network.GetNetworkPresets(gameCatalog, "faster-reds")
	if err != nil {
		t.Fatalf("network.GetNetworkPresets: %v", err)
	}

	recorder := do(t, gameCatalog, "/api/v1/network/presets?presetID=faster-reds")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}

	got := decode(t, recorder.Body.Bytes())
	if !reflect.DeepEqual(got, marshalled(t, want)) {
		t.Fatal("filtered network presets route body differs from the GetNetworkPresets result")
	}
}

// An unknown preset is a client error, so the route must carry the getter wording.
func TestNetworkPresetsRouteReportsAnUnknownPresetID(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)
	_, want := network.GetNetworkPresets(gameCatalog, "fast-invasions")
	assertErrorMessage(
		t,
		do(t, gameCatalog, "/api/v1/network/presets?presetID=fast-invasions"),
		http.StatusBadRequest,
		want,
	)
}

func TestAppearancePresetsRouteMatchesGetter(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)
	want, err := appearance.GetAppearancePresets(gameCatalog, "", nil)
	if err != nil {
		t.Fatalf("appearance.GetAppearancePresets: %v", err)
	}

	recorder := do(t, gameCatalog, "/api/v1/appearance/presets")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	got := decode(t, recorder.Body.Bytes())
	if !reflect.DeepEqual(got, marshalled(t, want)) {
		t.Fatal("appearance presets route body differs from the GetAppearancePresets result")
	}
}

func TestAppearancePresetsRouteFiltersBySearch(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)
	want, err := appearance.GetAppearancePresets(gameCatalog, "witcher", nil)
	if err != nil {
		t.Fatalf("appearance.GetAppearancePresets: %v", err)
	}
	if len(want.Presets) == 0 {
		t.Fatal("the filtered getter result is empty, so the route comparison would not prove anything")
	}

	recorder := do(t, gameCatalog, "/api/v1/appearance/presets?search=witcher")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}

	got := decode(t, recorder.Body.Bytes())
	if !reflect.DeepEqual(got, marshalled(t, want)) {
		t.Fatal("searched appearance presets route body differs from the GetAppearancePresets result")
	}
}

// The single tags parameter is comma separated, so both tags have to reach the
// getter as separate values.
func TestAppearancePresetsRouteSplitsCommaSeparatedTags(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)
	want, err := appearance.GetAppearancePresets(gameCatalog, "", []string{"witcher", "male"})
	if err != nil {
		t.Fatalf("appearance.GetAppearancePresets: %v", err)
	}

	recorder := do(t, gameCatalog, "/api/v1/appearance/presets?tags=witcher,male")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}

	got := decode(t, recorder.Body.Bytes())
	if !reflect.DeepEqual(got, marshalled(t, want)) {
		t.Fatal("tagged appearance presets route body differs from the GetAppearancePresets result")
	}
	if !reflect.DeepEqual(parseTags("witcher,male"), []string{"witcher", "male"}) {
		t.Fatalf("parseTags = %#v, want two separate tags", parseTags("witcher,male"))
	}
	if parseTags("") != nil {
		t.Fatalf("parseTags(\"\") = %#v, want nil", parseTags(""))
	}
}

// An empty element is client input, so the getter rejection must reach the
// client as a 400 with the getter wording.
func TestAppearancePresetsRouteRejectsAnEmptyTagElement(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)
	_, want := appearance.GetAppearancePresets(gameCatalog, "", []string{"foo", "", "bar"})
	assertErrorMessage(
		t,
		do(t, gameCatalog, "/api/v1/appearance/presets?tags=foo,,bar"),
		http.StatusBadRequest,
		want,
	)
}

// Synthetic PC container layout. The explorer owns none of these values; they
// are duplicated here only to build a local fixture SaveEngine accepts, so the
// test needs no real save, no repository fixture and no endpoint test helper.
const (
	pcHeaderSize       = 0x300
	pcEntryCountOffset = 0x0C
	pcEntryCount       = 12
	pcFixtureSize      = int64(pcHeaderSize) + 10*0x280010 + 0x60010
)

func writePCFixture(t *testing.T) string {
	t.Helper()

	header := make([]byte, pcHeaderSize)
	copy(header, []byte("BND4"))
	binary.LittleEndian.PutUint32(header[pcEntryCountOffset:], pcEntryCount)

	path := filepath.Join(t.TempDir(), "save.sl2")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	defer file.Close()
	if _, err := file.Write(header); err != nil {
		t.Fatalf("write fixture header: %v", err)
	}
	if err := file.Truncate(pcFixtureSize); err != nil {
		t.Fatalf("size fixture: %v", err)
	}
	return path
}

// doSave serves one save-session request against a handler that shares the given
// engine. A nil engine is the -allow-external-bind mode, in which no
// save-session route is registered. The request mirrors what a documented client
// sends: a body is declared as application/json, and a bodyless one declares
// nothing.
func doSave(
	t *testing.T,
	saveEngine *saveengine.Engine,
	method string,
	target string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	contentType := ""
	if body != "" {
		contentType = "application/json"
	}
	return doSaveTyped(t, saveEngine, method, target, body, contentType)
}

// doSaveTyped is doSave with a caller-chosen Content-Type. An empty contentType
// sends no header at all.
func doSaveTyped(
	t *testing.T,
	saveEngine *saveengine.Engine,
	method string,
	target string,
	body string,
	contentType string,
) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, target, reader)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	recorder := httptest.NewRecorder()
	newHandler(newPrototypeCatalog(t), testApplicationVersion, saveEngine).
		ServeHTTP(recorder, request)
	return recorder
}

func assertOK(t *testing.T, recorder *httptest.ResponseRecorder, target string) {
	t.Helper()

	if recorder.Code != http.StatusOK {
		t.Fatalf("%s: status = %d, want 200 (body %q)", target, recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)
}

// The full local lifecycle has to work through the routes alone: load a session,
// read it back, list its characters, close it, and then fail to read the closed
// session.
func TestSaveSessionLifecycleRoutes(t *testing.T) {
	saveEngine := saveengine.New()
	source := writePCFixture(t)

	created := doSave(t, saveEngine, http.MethodPost, "/api/v1/save-sessions",
		`{"source":`+strconv.Quote(source)+`,"expectedPlatform":"pc"}`)
	assertOK(t, created, "POST /api/v1/save-sessions")

	var session saveengine.SessionInfo
	if err := json.Unmarshal(created.Body.Bytes(), &session); err != nil {
		t.Fatalf("decode LoadSave body %q: %v", created.Body.String(), err)
	}
	if session.SaveSessionID == "" {
		t.Fatal("the route returned an empty saveSessionID")
	}
	if session.Platform != "pc" {
		t.Fatalf("platform = %q, want pc", session.Platform)
	}

	loaded := doSave(t, saveEngine, http.MethodGet, "/api/v1/save-sessions/"+session.SaveSessionID, "")
	assertOK(t, loaded, "GET /api/v1/save-sessions/{id}")
	if !reflect.DeepEqual(decode(t, loaded.Body.Bytes()), marshalled(t, session)) {
		t.Fatal("GetLoadedSave route body differs from the LoadSave result")
	}

	wantCharacters, err := character.GetSaveCharacters(saveEngine, session.SaveSessionID)
	if err != nil {
		t.Fatalf("character.GetSaveCharacters: %v", err)
	}
	characters := doSave(t, saveEngine, http.MethodGet,
		"/api/v1/save-sessions/"+session.SaveSessionID+"/characters", "")
	assertOK(t, characters, "GET /api/v1/save-sessions/{id}/characters")
	if !reflect.DeepEqual(decode(t, characters.Body.Bytes()), marshalled(t, wantCharacters)) {
		t.Fatal("characters route body differs from the GetSaveCharacters result")
	}

	writtenTarget := filepath.Join(t.TempDir(), "written.sl2")
	written := doSave(t, saveEngine, http.MethodPost,
		"/api/v1/save-sessions/"+session.SaveSessionID+"/write",
		`{"expectedRevision":"0","target":`+strconv.Quote(writtenTarget)+`}`)
	assertOK(t, written, "POST /api/v1/save-sessions/{id}/write")
	var writeResult saveengine.WriteSaveResult
	if err := json.Unmarshal(written.Body.Bytes(), &writeResult); err != nil {
		t.Fatalf("decode WriteSave body %q: %v", written.Body.String(), err)
	}
	if writeResult.SaveSessionID != session.SaveSessionID || writeResult.SaveRevision != "1" {
		t.Fatalf("WriteSave result = %+v, want the session at revision 1", writeResult)
	}
	if _, err := saveengine.New().LoadSave(writtenTarget, "pc"); err != nil {
		t.Fatalf("reload WriteSave target: %v", err)
	}

	closed := doSave(t, saveEngine, http.MethodDelete, "/api/v1/save-sessions/"+session.SaveSessionID, "")
	assertOK(t, closed, "DELETE /api/v1/save-sessions/{id}")
	var confirmation closeSaveResponse
	if err := json.Unmarshal(closed.Body.Bytes(), &confirmation); err != nil {
		t.Fatalf("decode CloseSave body %q: %v", closed.Body.String(), err)
	}
	if confirmation.SaveSessionID != session.SaveSessionID || !confirmation.Closed {
		t.Fatalf("close confirmation = %#v, want the closed session", confirmation)
	}
	// The source file is evidence: closing a session must not touch it.
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("the source save is gone after closing the session: %v", err)
	}

	_, wantUnknown := savesession.GetLoadedSave(saveEngine, session.SaveSessionID)
	assertErrorMessage(
		t,
		doSave(t, saveEngine, http.MethodGet, "/api/v1/save-sessions/"+session.SaveSessionID, ""),
		http.StatusBadRequest,
		wantUnknown,
	)
}

// A POST declaring text/plain, or declaring nothing at all, is a CORS simple
// request, so a foreign page can send it without a preflight. Both bodied POST
// routes must refuse it before they create a session or write a file, and must
// still accept the media type with a charset parameter.
func TestBodiedPostRoutesRequireJSONContentType(t *testing.T) {
	saveEngine := saveengine.New()
	source := writePCFixture(t)
	loadBody := `{"source":` + strconv.Quote(source) + `,"expectedPlatform":"pc"}`

	for _, contentType := range []string{"text/plain", ""} {
		rejected := doSaveTyped(t, saveEngine, http.MethodPost, "/api/v1/save-sessions", loadBody, contentType)
		if rejected.Code != http.StatusBadRequest {
			t.Fatalf("LoadSave with Content-Type %q: status = %d, want 400 (body %q)",
				contentType, rejected.Code, rejected.Body.String())
		}
		var envelope map[string]string
		if err := json.Unmarshal(rejected.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("decode rejection body %q: %v", rejected.Body.String(), err)
		}
		if envelope["error"] == "" {
			t.Fatalf("the rejection carries no error message: %q", rejected.Body.String())
		}
	}

	created := doSaveTyped(t, saveEngine, http.MethodPost, "/api/v1/save-sessions", loadBody,
		"application/json; charset=utf-8")
	assertOK(t, created, "POST /api/v1/save-sessions with a charset parameter")
	var session saveengine.SessionInfo
	if err := json.Unmarshal(created.Body.Bytes(), &session); err != nil {
		t.Fatalf("decode LoadSave body %q: %v", created.Body.String(), err)
	}

	target := filepath.Join(t.TempDir(), "written.sl2")
	writeBody := `{"expectedRevision":"0","target":` + strconv.Quote(target) + `}`
	writeTarget := "/api/v1/save-sessions/" + session.SaveSessionID + "/write"

	for _, contentType := range []string{"text/plain", ""} {
		refused := doSaveTyped(t, saveEngine, http.MethodPost, writeTarget, writeBody, contentType)
		if refused.Code != http.StatusBadRequest {
			t.Fatalf("WriteSave with Content-Type %q: status = %d, want 400 (body %q)",
				contentType, refused.Code, refused.Body.String())
		}
		if _, err := os.Stat(target); !os.IsNotExist(err) {
			t.Fatalf("the rejected WriteSave request still touched the target: %v", err)
		}
	}

	written := doSaveTyped(t, saveEngine, http.MethodPost, writeTarget, writeBody,
		"application/json; charset=utf-8")
	assertOK(t, written, "POST /api/v1/save-sessions/{id}/write with a charset parameter")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("the accepted WriteSave request wrote no target: %v", err)
	}
}

func TestSaveCharacterRoutesMatchGetters(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writePCFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0"

	wantProfile, err := character.GetCharacterProfile(saveEngine, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("character.GetCharacterProfile: %v", err)
	}
	profile := doSave(t, saveEngine, http.MethodGet, base+"/profile", "")
	assertOK(t, profile, base+"/profile")
	if !reflect.DeepEqual(decode(t, profile.Body.Bytes()), marshalled(t, wantProfile)) {
		t.Fatal("profile route body differs from the GetCharacterProfile result")
	}

	wantStats, err := character.GetCharacterStats(saveEngine, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("character.GetCharacterStats: %v", err)
	}
	stats := doSave(t, saveEngine, http.MethodGet, base+"/stats", "")
	assertOK(t, stats, base+"/stats")
	if !reflect.DeepEqual(decode(t, stats.Body.Bytes()), marshalled(t, wantStats)) {
		t.Fatal("stats route body differs from the GetCharacterStats result")
	}

	wantAppearance, err := character.GetCharacterAppearance(saveEngine, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("character.GetCharacterAppearance: %v", err)
	}
	appearanceResponse := doSave(t, saveEngine, http.MethodGet, base+"/appearance", "")
	assertOK(t, appearanceResponse, base+"/appearance")
	if !reflect.DeepEqual(decode(t, appearanceResponse.Body.Bytes()), marshalled(t, wantAppearance)) {
		t.Fatal("appearance route body differs from the GetCharacterAppearance result")
	}

	wantEquipment, err := equipment.GetEquipment(saveEngine, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("equipment.GetEquipment: %v", err)
	}
	equipmentResponse := doSave(t, saveEngine, http.MethodGet, base+"/equipment", "")
	assertOK(t, equipmentResponse, base+"/equipment")
	if !reflect.DeepEqual(decode(t, equipmentResponse.Body.Bytes()), marshalled(t, wantEquipment)) {
		t.Fatal("equipment route body differs from the GetEquipment result")
	}

	wantQuickItems, err := equipment.GetQuickItems(saveEngine, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("equipment.GetQuickItems: %v", err)
	}
	quickItemsResponse := doSave(t, saveEngine, http.MethodGet, base+"/quick-items", "")
	assertOK(t, quickItemsResponse, base+"/quick-items")
	if !reflect.DeepEqual(decode(t, quickItemsResponse.Body.Bytes()), marshalled(t, wantQuickItems)) {
		t.Fatal("quick items route body differs from the GetQuickItems result")
	}

	wantPouchItems, err := equipment.GetPouchItems(saveEngine, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("equipment.GetPouchItems: %v", err)
	}
	pouchItemsResponse := doSave(t, saveEngine, http.MethodGet, base+"/pouch-items", "")
	assertOK(t, pouchItemsResponse, base+"/pouch-items")
	if !reflect.DeepEqual(decode(t, pouchItemsResponse.Body.Bytes()), marshalled(t, wantPouchItems)) {
		t.Fatal("pouch items route body differs from the GetPouchItems result")
	}

	wantPhysickMixture, err := equipment.GetPhysickMixture(saveEngine, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("equipment.GetPhysickMixture: %v", err)
	}
	physickMixtureResponse := doSave(t, saveEngine, http.MethodGet, base+"/physick-mixture", "")
	assertOK(t, physickMixtureResponse, base+"/physick-mixture")
	if !reflect.DeepEqual(decode(t, physickMixtureResponse.Body.Bytes()), marshalled(t, wantPhysickMixture)) {
		t.Fatal("physick mixture route body differs from the GetPhysickMixture result")
	}

	// A non-decimal characterID never reaches the getter, so the route owns it.
	for _, raw := range []string{"one", " 0", "0x1"} {
		target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/" + url.PathEscape(raw) + "/profile"
		if recorder := doSave(t, saveEngine, http.MethodGet, target, ""); recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %q)", target, recorder.Code, recorder.Body.String())
		}
	}
}

// With -allow-external-bind the explorer withholds the engine, so no
// save-session route exists while the catalog routes keep working.
func TestSaveSessionRoutesAreAbsentWithoutAnEngine(t *testing.T) {
	for _, request := range []struct {
		method string
		target string
		body   string
	}{
		{http.MethodPost, "/api/v1/save-sessions", `{"source":"unused-by-a-missing-route","expectedPlatform":""}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session", ""},
		{http.MethodPost, "/api/v1/save-sessions/any-session/write", `{"expectedRevision":"0","target":"unused"}`},
		{http.MethodPatch, "/api/v1/save-sessions/any-session/account-id", `{"accountID":"1","expectedRevision":"0"}`},
		{http.MethodDelete, "/api/v1/save-sessions/any-session", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/profile", ""},
		{http.MethodPatch, "/api/v1/save-sessions/any-session/characters/0/gender", `{"gender":0,"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/stats", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/undo", ""},
		{http.MethodPost, "/api/v1/save-sessions/any-session/characters/0/undo", `{"undoToken":"any-token","expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/appearance", ""},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/appearance", `{"appearance":{},"expectedRevision":"0"}`},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/appearance/preset", `{"presetID":"geralt-of-rivia-the-witcher","expectedRevision":"0"}`},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/appearance/favorite-preset", `{"favoriteSlotID":0,"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/equipment", ""},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/equipped-armaments", `{"slotAssignments":[null,null,null,null,null,null],"expectedRevision":"0"}`},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/equipped-armor", `{"slotAssignments":[null,null,null,null],"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/quick-items", ""},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/quick-items", `{"slotAssignments":[null,null,null,null,null,null,null,null,null,null],"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/pouch-items", ""},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/pouch-items", `{"slotAssignments":[null,null,null,null,null,null],"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/physick-mixture", ""},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/physick-mixture", `{"crystalTearResources":[null,null],"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/equipped-spells", ""},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/equipped-spells", `{"orderedResources":[],"expectedRevision":"0"}`},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/equipped-talismans", `{"orderedOwnedItemIDs":[],"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/storage", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/item-capacity?destination=inventory&kind=item&key=400006A4&quantity=1", ""},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/inventory/order", `{"orderedOwnedItemIDs":["any-token"],"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/owned-items/any-token", ""},
		{http.MethodDelete, "/api/v1/save-sessions/any-session/characters/0/owned-items/any-token", `{"expectedRevision":"0"}`},
		{http.MethodPatch, "/api/v1/save-sessions/any-session/characters/0/owned-items/any-token/quantity", `{"quantity":1,"expectedRevision":"0"}`},
		{http.MethodPost, "/api/v1/save-sessions/any-session/characters/0/owned-items/any-token/move-to-inventory", `{"targetPosition":0,"expectedRevision":"0"}`},
		{http.MethodPost, "/api/v1/save-sessions/any-session/characters/0/owned-items/any-token/move-to-storage", `{"targetPosition":0,"expectedRevision":"0"}`},
		{http.MethodPatch, "/api/v1/save-sessions/any-session/characters/0/owned-items/any-token/upgrade-level", `{"upgradeLevel":1,"expectedRevision":"0"}`},
		{http.MethodPatch, "/api/v1/save-sessions/any-session/characters/0/owned-items/any-token/ash-of-war", `{"ashOfWarKind":null,"ashOfWarKey":null,"expectedRevision":"0"}`},
		{http.MethodPost, "/api/v1/save-sessions/any-session/characters/0/inventory/items", `{"kind":"item","key":"400006A4","quantity":1,"expectedRevision":"0"}`},
		{http.MethodPost, "/api/v1/save-sessions/any-session/characters/0/storage/items", `{"kind":"item","key":"400006A4","quantity":1,"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/gestures", ""},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/gestures/unlock", `{"gestureKind":"item","gestureKey":"401EA7AB","unlocked":true,"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/cookbooks", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/bell-bearings", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/whetblades", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/colosseums", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/regions", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/summoning-pools", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/graces", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/bosses", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/map-regions", ""},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/map-regions/reveal", `{"mapRegionKind":"map_region","mapRegionKey":"limgrave_weeping_peninsula","revealed":true,"expectedRevision":"0"}`},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/fog-of-war", `{"removed":true,"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/tutorials", ""},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/tutorials/unlock", `{"tutorialKind":"tutorial","tutorialKey":"2010","unlocked":true,"expectedRevision":"0"}`},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/cookbooks/unlock", `{"cookbookKind":"item","cookbookKey":"40002455","unlocked":true,"expectedRevision":"0"}`},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/summoning-pools/activate", `{"summoningPoolKind":"summoning_pool","summoningPoolKey":"stormveil_castle_liftside_chamber","activated":true,"expectedRevision":"0"}`},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/bosses/defeat", `{"bossKind":"boss","bossKey":"stormveil_castle_godrick_the_grafted","defeated":true,"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/quests?questKind=quest", ""},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/quests/step", `{"questKind":"quest","questKey":"brother_corhyn","stepKind":"quest_step","stepKey":"legacy_000","expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/favorite-presets", ""},
		{http.MethodDelete, "/api/v1/save-sessions/any-session/favorite-presets/0", `{"expectedRevision":"0"}`},
	} {
		recorder := doSave(t, nil, request.method, request.target, request.body)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s %s: status = %d, want 404 (body %q)",
				request.method, request.target, recorder.Code, recorder.Body.String())
		}
	}

	if recorder := do(t, newPrototypeCatalog(t), "/api/v1/catalog/info"); recorder.Code != http.StatusOK {
		t.Fatalf("catalog route status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
}

func TestHealthz(t *testing.T) {
	recorder := do(t, newPrototypeCatalog(t), "/healthz")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	assertJSONContentType(t, recorder)

	var payload map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body %q: %v", recorder.Body.String(), err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("status field = %q, want ok", payload["status"])
	}
}

func TestOpenAPIDocumentDescribesEveryRoute(t *testing.T) {
	recorder := do(t, newPrototypeCatalog(t), "/openapi.json")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	assertJSONContentType(t, recorder)

	var document struct {
		OpenAPI string                    `json:"openapi"`
		Paths   map[string]map[string]any `json:"paths"`
		Comps   struct {
			Parameters map[string]struct {
				Schema struct {
					Enum []string `json:"enum"`
				} `json:"schema"`
			} `json:"parameters"`
			Schemas map[string]any `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode openapi.json: %v", err)
	}
	if !strings.HasPrefix(document.OpenAPI, "3.") {
		t.Fatalf("openapi = %q, want a 3.x version", document.OpenAPI)
	}

	for _, path := range []string{
		"/api/v1/application/info",
		"/api/v1/catalog/info",
		"/api/v1/catalog/resource",
		"/api/v1/catalog/resource-relations",
		"/api/v1/catalog/item-variants",
		"/api/v1/catalog/resources",
		"/api/v1/network/presets",
		"/api/v1/appearance/presets",
		"/healthz",
		"/openapi.json",
	} {
		operation, exists := document.Paths[path]
		if !exists {
			t.Fatalf("openapi.json does not describe %s", path)
		}
		if _, hasGet := operation["get"]; !hasGet {
			t.Fatalf("openapi.json describes %s without a GET operation", path)
		}
	}
	// The save-session routes exist only in the local loopback mode, so the
	// document has to describe them with their own methods.
	for path, method := range map[string]string{
		"/api/v1/save-sessions":                                                                             "post",
		"/api/v1/save-sessions/{saveSessionID}":                                                             "get",
		"/api/v1/save-sessions/{saveSessionID}/write":                                                       "post",
		"/api/v1/save-sessions/{saveSessionID}/account-id":                                                  "patch",
		"/api/v1/save-sessions/{saveSessionID}/characters":                                                  "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{sourceCharacterID}/clone":                        "post",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}":                                    "delete",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/profile":                            "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/active":                             "patch",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/name":                               "patch",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/gender":                             "patch",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/runes":                              "patch",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/stats":                              "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/undo":                               "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance":                         "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance/preset":                  "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance/favorite-preset":         "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/validation-report":                  "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipment":                          "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-armaments":                 "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-armor":                     "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/quick-items":                        "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/pouch-items":                        "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/physick-mixture":                    "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory":                          "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/item-capacity":                      "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory/order":                    "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory/items":                    "post",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/storage":                            "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/storage/items":                      "post",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}":          "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/quantity": "patch",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/gestures":                           "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/gestures/unlock":                    "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/cookbooks":                          "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/bell-bearings":                      "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/bell-bearings/unlock":               "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/whetblades":                         "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/colosseums":                         "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/regions":                            "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/regions/unlock":                     "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/summoning-pools":                    "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/summoning-pools/activate":           "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/graces":                             "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/bosses":                             "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/bosses/defeat":                      "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/map-regions":                        "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/map-regions/reveal":                 "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/fog-of-war":                         "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/tutorials":                          "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/tutorials/unlock":                   "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/whetblades/unlock":                  "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/cookbooks/unlock":                   "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/quests":                             "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/quests/step":                        "put",
		"/api/v1/save-sessions/{saveSessionID}/network-settings":                                            "get",
		"/api/v1/save-sessions/{saveSessionID}/network-settings/preset":                                     "put",
		"/api/v1/save-sessions/{saveSessionID}/favorite-presets":                                            "get",
		"/api/v1/build-templates":              "get",
		"/api/v1/build-templates/{templateID}": "get",
	} {
		operation, exists := document.Paths[path]
		if !exists {
			t.Fatalf("openapi.json does not describe %s", path)
		}
		if _, hasMethod := operation[method]; !hasMethod {
			t.Fatalf("openapi.json describes %s without a %s operation", path, strings.ToUpper(method))
		}
	}
	if _, hasDelete := document.Paths["/api/v1/save-sessions/{saveSessionID}"]["delete"]; !hasDelete {
		t.Fatal("openapi.json describes no DELETE for /api/v1/save-sessions/{saveSessionID}")
	}
	// The build-template path carries the getter, DeleteBuildTemplate and UpdateBuildTemplate,
	// so the map above can only state one of them.
	buildTemplate := "/api/v1/build-templates/{templateID}"
	if _, hasDelete := document.Paths[buildTemplate]["delete"]; !hasDelete {
		t.Fatalf("openapi.json describes no DELETE for %s", buildTemplate)
	}
	if _, hasPut := document.Paths[buildTemplate]["put"]; !hasPut {
		t.Fatalf("openapi.json describes no PUT for %s", buildTemplate)
	}
	buildTemplatesList := "/api/v1/build-templates"
	if _, hasPost := document.Paths[buildTemplatesList]["post"]; !hasPost {
		t.Fatalf("openapi.json describes no POST for %s", buildTemplatesList)
	}
	buildTemplatePreview := "/api/v1/build-templates/{templateID}/preview"
	if _, hasPost := document.Paths[buildTemplatePreview]["post"]; !hasPost {
		t.Fatalf("openapi.json describes no POST for %s", buildTemplatePreview)
	}
	buildTemplateApply := "/api/v1/build-templates/{templateID}/apply"
	if _, hasPost := document.Paths[buildTemplateApply]["post"]; !hasPost {
		t.Fatalf("openapi.json describes no POST for %s", buildTemplateApply)
	}
	// The favorite-presets slot path carries both PUT and DELETE operations.
	favSlot := "/api/v1/save-sessions/{saveSessionID}/favorite-presets/{favoriteSlotID}"
	if _, hasPut := document.Paths[favSlot]["put"]; !hasPut {
		t.Fatalf("openapi.json describes no PUT for %s", favSlot)
	}
	if _, hasDelete := document.Paths[favSlot]["delete"]; !hasDelete {
		t.Fatalf("openapi.json describes no DELETE for %s", favSlot)
	}
	// The undo path carries the getter and the mutation, so the map above can
	// only state one of them.
	undo := "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/undo"
	if _, hasPost := document.Paths[undo]["post"]; !hasPost {
		t.Fatalf("openapi.json describes no POST for %s", undo)
	}

	// The owned-item path carries two operations: reading one instance and
	// removing it, so the map above can only state one of them.
	ownedItem := "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}"
	if _, hasDelete := document.Paths[ownedItem]["delete"]; !hasDelete {
		t.Fatalf("openapi.json describes no DELETE for %s", ownedItem)
	}
	moveToStorage := ownedItem + "/move-to-storage"
	if _, hasPost := document.Paths[moveToStorage]["post"]; !hasPost {
		t.Fatalf("openapi.json describes no POST for %s", moveToStorage)
	}
	moveToInventory := ownedItem + "/move-to-inventory"
	if _, hasPost := document.Paths[moveToInventory]["post"]; !hasPost {
		t.Fatalf("openapi.json describes no POST for %s", moveToInventory)
	}
	weaponUpgrade := ownedItem + "/upgrade-level"
	if _, hasPatch := document.Paths[weaponUpgrade]["patch"]; !hasPatch {
		t.Fatalf("openapi.json describes no PATCH for %s", weaponUpgrade)
	}
	weaponAshOfWar := ownedItem + "/ash-of-war"
	if _, hasPatch := document.Paths[weaponAshOfWar]["patch"]; !hasPatch {
		t.Fatalf("openapi.json describes no PATCH for %s", weaponAshOfWar)
	}
	spiritAshUpgrade := ownedItem + "/spirit-ash-upgrade-level"
	if _, hasPatch := document.Paths[spiritAshUpgrade]["patch"]; !hasPatch {
		t.Fatalf("openapi.json describes no PATCH for %s", spiritAshUpgrade)
	}
	characterAppearance := "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance"
	if _, hasPut := document.Paths[characterAppearance]["put"]; !hasPut {
		t.Fatalf("openapi.json describes no PUT for %s", characterAppearance)
	}
	characterStats := "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/stats"
	if _, hasPatch := document.Paths[characterStats]["patch"]; !hasPatch {
		t.Fatalf("openapi.json describes no PATCH for %s", characterStats)
	}
	physickMixture := "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/physick-mixture"
	if _, hasPut := document.Paths[physickMixture]["put"]; !hasPut {
		t.Fatalf("openapi.json describes no PUT for %s", physickMixture)
	}
	equippedSpells := "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-spells"
	if _, hasPut := document.Paths[equippedSpells]["put"]; !hasPut {
		t.Fatalf("openapi.json describes no PUT for %s", equippedSpells)
	}
	equippedTalismans := "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-talismans"
	if _, hasPut := document.Paths[equippedTalismans]["put"]; !hasPut {
		t.Fatalf("openapi.json describes no PUT for %s", equippedTalismans)
	}
	quickItems := "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/quick-items"
	if _, hasPut := document.Paths[quickItems]["put"]; !hasPut {
		t.Fatalf("openapi.json describes no PUT for %s", quickItems)
	}
	pouchItems := "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/pouch-items"
	if _, hasPut := document.Paths[pouchItems]["put"]; !hasPut {
		t.Fatalf("openapi.json describes no PUT for %s", pouchItems)
	}
	networkSettings := "/api/v1/save-sessions/{saveSessionID}/network-settings"
	if _, hasPut := document.Paths[networkSettings]["put"]; !hasPut {
		t.Fatalf("openapi.json describes no PUT for %s", networkSettings)
	}
	assertLoopbackOnlySaveSessionRoutes(t, document.Paths)

	for _, name := range []string{
		"ResourceKind", "ResourceKey", "ResourceTypeFilter", "RelationType",
		"RelationDirection", "AvailabilityFilter",
	} {
		if _, exists := document.Comps.Parameters[name]; !exists {
			t.Fatalf("openapi.json is missing the shared %s parameter", name)
		}
	}
	if _, exists := document.Comps.Parameters["ResourceID"]; exists {
		t.Fatal("openapi.json still declares the removed ResourceID parameter")
	}
	resourceTypes := document.Comps.Parameters["ResourceTypeFilter"].Schema.Enum
	if !reflect.DeepEqual(resourceTypes, []string{
		"", "item", "colosseum", "region", "summoning_pool", "grace", "boss",
		"map_region", "tutorial", "quest",
	}) {
		t.Fatalf("ResourceTypeFilter enum = %v", resourceTypes)
	}
	for _, name := range []string{
		"Error",
		"SupportedSchema",
		"GetApplicationInfoResult",
		"GetCatalogInfoResult",
		"Resource",
		"ColosseumDocument",
		"RegionDocument",
		"SummoningPoolDocument",
		"GraceDocument",
		"BossDocument",
		"MapRegionDocument",
		"TutorialDocument",
		"QuestDocument",
		"QuestStepDocument",
		"QuestFlag",
		"GetResourceResult",
		"GetItemVariantsResult",
		"GetResourceRelationsResult",
		"GetResourcesResult",
		"Relation",
		"ResourceRef",
		"NetworkParamValues",
		"NetworkPreset",
		"GetNetworkPresetsResult",
		"GetNetworkSettingsResult",
		"SetNetworkSettingsRequest",
		"SetNetworkSettingsResult",
		"ApplyNetworkPresetRequest",
		"ApplyNetworkPresetResult",
		"AppearancePresetSummary",
		"GetAppearancePresetsResult",
		"LoadSaveRequest",
		"WriteSaveRequest",
		"WriteSaveResult",
		"CloseSaveResult",
		"SessionInfo",
		"SaveCharacters",
		"CharacterProfile",
		"CloneCharacterRequest",
		"CloneCharacterResult",
		"DeleteCharacterRequest",
		"DeleteCharacterResult",
		"SetCharacterActiveRequest",
		"SetCharacterActiveResult",
		"SetCharacterNameRequest",
		"SetCharacterNameResult",
		"SetCharacterGenderRequest",
		"SetCharacterGenderResult",
		"SetCharacterRunesRequest",
		"SetCharacterRunesResult",
		"CharacterStats",
		"CharacterAttributes",
		"SetCharacterStatsRequest",
		"SetCharacterStatsResult",
		"CharacterAppearance",
		"CharacterAppearanceValues",
		"SetCharacterAppearanceRequest",
		"SetCharacterAppearanceResult",
		"ApplyAppearancePresetRequest",
		"ApplyAppearancePresetResult",
		"CharacterEquipment",
		"EquippedArmamentAssignment",
		"SetEquippedArmamentsRequest",
		"SetEquippedArmamentsResult",
		"SetEquippedArmorRequest",
		"SetEquippedArmorResult",
		"QuickItemSlot",
		"CharacterQuickItems",
		"SetQuickItemsRequest",
		"SetQuickItemsResult",
		"SetEquippedTalismansRequest",
		"SetEquippedTalismansResult",
		"PouchItemSlot",
		"CharacterPouchItems",
		"SetPouchItemsRequest",
		"SetPouchItemsResult",
		"CharacterPhysickMixture",
		"SetPhysickMixtureRequest",
		"SetPhysickMixtureResult",
		"InventoryRecord",
		"CharacterInventory",
		"StorageRecord",
		"CharacterStorage",
		"OwnedItem",
		"AddItemToInventoryRequest",
		"AddItemToInventoryResult",
		"AddItemToStorageRequest",
		"AddItemToStorageResult",
		"SetOwnedItemQuantityRequest",
		"SetOwnedItemQuantityResult",
		"SetInventoryOrderRequest",
		"SetInventoryOrderResult",
		"SetStorageOrderRequest",
		"SetStorageOrderResult",
		"MoveOwnedItemToInventoryRequest",
		"MoveOwnedItemToInventoryResult",
		"MoveOwnedItemToStorageRequest",
		"MoveOwnedItemToStorageResult",
		"ItemCapacity",
		"SetWeaponUpgradeLevelRequest",
		"SetWeaponUpgradeLevelResult",
		"SetSpiritAshUpgradeLevelRequest",
		"SetSpiritAshUpgradeLevelResult",
		"SetWeaponInfusionRequest",
		"SetWeaponInfusionResult",
		"SetWeaponAshOfWarRequest",
		"SetWeaponAshOfWarResult",
		"GestureEntry",
		"GetGesturesResult",
		"SetGestureUnlockedRequest",
		"SetGestureUnlockedResult",
		"CookbookEntry",
		"GetCookbooksResult",
		"BellBearingEntry",
		"GetBellBearingsResult",
		"SetBellBearingUnlockedRequest",
		"SetBellBearingUnlockedResult",
		"WhetbladeEntry",
		"GetWhetbladesResult",
		"ColosseumEntry",
		"GetColosseumsResult",
		"RegionEntry",
		"GetRegionsResult",
		"SummoningPoolEntry",
		"GetSummoningPoolsResult",
		"GraceEntry",
		"GetGracesResult",
		"BossEntry",
		"GetBossesResult",
		"MapRegionEntry",
		"GetMapRegionsResult",
		"TutorialEntry",
		"GetTutorialsResult",
		"QuestStepEntry",
		"QuestEntry",
		"GetQuestsResult",
		"SetWhetbladeUnlockedRequest",
		"SetWhetbladeUnlockedResult",
		"SetCookbookUnlockedRequest",
		"SetCookbookUnlockedResult",
		"SetSummoningPoolActivatedRequest",
		"SetSummoningPoolActivatedResult",
		"SetBossDefeatedRequest",
		"SetBossDefeatedResult",
		"SetGraceVisitedRequest",
		"SetGraceVisitedResult",
		"SetColosseumUnlockedRequest",
		"SetColosseumUnlockedResult",
		"SetRegionUnlockedRequest",
		"SetRegionUnlockedResult",
		"SetMapRegionRevealedRequest",
		"SetMapRegionRevealedResult",
		"SetFogOfWarRemovedRequest",
		"SetFogOfWarRemovedResult",
		"SetTutorialUnlockedRequest",
		"SetTutorialUnlockedResult",
		"FavoritePreset",
		"GetFavoritePresetsResult",
		"SetFavoritePresetRequest",
		"SetFavoritePresetResult",
		"DeleteFavoritePresetRequest",
		"DeleteFavoritePresetResult",
		"DeleteBuildTemplateRequest",
		"DeleteBuildTemplateResult",
	} {
		if _, exists := document.Comps.Schemas[name]; !exists {
			t.Fatalf("openapi.json is missing the %s schema", name)
		}
	}
	if _, exists := document.Comps.Schemas["ResourceID"]; exists {
		t.Fatal("openapi.json still declares the removed ResourceID schema")
	}
	assertQuantityFitsTheRecord(t, document.Comps.Schemas)
	assertRelationEndpointsAreResourceRefs(t, document.Comps.Schemas)
}

// The document declares OpenAPI 3.0.3, whose Schema Object has no const
// keyword, so the Resource union has to discriminate kind with a one-element
// enum instead.
// resourceUnionDocuments maps one resource kind onto the Resource union property
// that carries its document. They are equal for every kind but summoning_pool,
// whose document property is camelCase.
var resourceUnionDocuments = map[string]string{
	"item":           "item",
	"colosseum":      "colosseum",
	"region":         "region",
	"summoning_pool": "summoningPool",
	"grace":          "grace",
	"boss":           "boss",
	"map_region":     "mapRegion",
	"tutorial":       "tutorial",
	"quest":          "quest",
}

func TestResourceUnionStaysWithinOpenAPI303(t *testing.T) {
	recorder := do(t, newPrototypeCatalog(t), "/openapi.json")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), `"const"`) {
		t.Fatal("openapi.json uses the const keyword, which OpenAPI 3.0.3 does not define")
	}

	var document struct {
		OpenAPI string `json:"openapi"`
		Comps   struct {
			Schemas map[string]struct {
				OneOf []struct {
					Properties map[string]struct {
						Enum []string `json:"enum"`
					} `json:"properties"`
					Required []string `json:"required"`
					Not      struct {
						AnyOf []struct {
							Required []string `json:"required"`
						} `json:"anyOf"`
					} `json:"not"`
				} `json:"oneOf"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode openapi.json: %v", err)
	}
	if document.OpenAPI != "3.0.3" {
		t.Fatalf("openapi = %q, want 3.0.3", document.OpenAPI)
	}

	branches := document.Comps.Schemas["Resource"].OneOf
	if len(branches) != 9 {
		t.Fatalf("Resource has %d oneOf branches, want 9", len(branches))
	}
	for index, kind := range []string{
		"item", "colosseum", "region", "summoning_pool", "grace", "boss", "map_region", "tutorial", "quest",
	} {
		branch := branches[index]
		if got := branch.Properties["kind"].Enum; len(got) != 1 || got[0] != kind {
			t.Fatalf("Resource oneOf[%d].kind enum = %v, want [%s]", index, got, kind)
		}
		documentProperty := resourceUnionDocuments[kind]
		if len(branch.Required) != 1 || branch.Required[0] != documentProperty {
			t.Fatalf("Resource oneOf[%d] required = %v, want [%s]", index, branch.Required, documentProperty)
		}
		// Counting the exclusions is not enough: three copies of the same foreign
		// document would pass it while leaving two documents unexcluded.
		missing := map[string]struct{}{}
		for _, other := range resourceUnionDocuments {
			if other != documentProperty {
				missing[other] = struct{}{}
			}
		}
		if len(branch.Not.AnyOf) != len(missing) {
			t.Fatalf("Resource oneOf[%d] excludes %d documents, want %d",
				index, len(branch.Not.AnyOf), len(missing))
		}
		for _, excluded := range branch.Not.AnyOf {
			if len(excluded.Required) != 1 {
				t.Fatalf("Resource oneOf[%d] invalid exclusion %v", index, excluded.Required)
			}
			if _, expected := missing[excluded.Required[0]]; !expected {
				t.Fatalf("Resource oneOf[%d] excludes %q, which is not one of the remaining documents",
					index, excluded.Required[0])
			}
			// Deleting is what rejects the same document excluded twice: the
			// second copy no longer belongs to the remaining set.
			delete(missing, excluded.Required[0])
		}
	}
}

// The curated Graces table uses the blocks 71 to 74 and 76, so the document must
// describe two separate ranges. One continuous 71000-76999 range would advertise
// the block 75 values that resolveEventFlag rejects and no grace declares.
func TestGraceVisitEventFlagOpenAPIExcludesBlock75(t *testing.T) {
	recorder := do(t, newPrototypeCatalog(t), "/openapi.json")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var document struct {
		Comps struct {
			Schemas map[string]struct {
				Properties struct {
					VisitEventFlagID struct {
						Properties struct {
							Value struct {
								Minimum *uint32 `json:"minimum"`
								Maximum *uint32 `json:"maximum"`
								OneOf   []struct {
									Type    string `json:"type"`
									Format  string `json:"format"`
									Minimum uint32 `json:"minimum"`
									Maximum uint32 `json:"maximum"`
								} `json:"oneOf"`
							} `json:"value"`
						} `json:"properties"`
					} `json:"visitEventFlagID"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode openapi.json: %v", err)
	}

	value := document.Comps.Schemas["GraceDocument"].Properties.VisitEventFlagID.Properties.Value
	if value.Minimum != nil || value.Maximum != nil {
		t.Fatal("GraceDocument.visitEventFlagID.value still carries a continuous range")
	}
	if len(value.OneOf) != 2 {
		t.Fatalf("visitEventFlagID.value has %d oneOf branches, want 2", len(value.OneOf))
	}
	for index, want := range []struct{ minimum, maximum uint32 }{
		{71000, 74999},
		{76000, 76999},
	} {
		branch := value.OneOf[index]
		if branch.Type != "integer" || branch.Format != "int64" {
			t.Fatalf("branch %d = %q/%q, want integer/int64", index, branch.Type, branch.Format)
		}
		if branch.Minimum != want.minimum || branch.Maximum != want.maximum {
			t.Fatalf("branch %d = %d-%d, want %d-%d",
				index, branch.Minimum, branch.Maximum, want.minimum, want.maximum)
		}
	}

	// oneOf accepts a value when exactly one branch does, which for two disjoint
	// ranges is the same as any branch accepting it.
	accepts := func(id uint32) bool {
		matched := 0
		for _, branch := range value.OneOf {
			if id >= branch.Minimum && id <= branch.Maximum {
				matched++
			}
		}
		return matched == 1
	}
	for _, boundary := range []struct {
		id      uint32
		allowed bool
	}{
		{71000, true},
		{74999, true},
		{75000, false},
		{75999, false},
		{76000, true},
		{76999, true},
		{77000, false},
	} {
		if accepts(boundary.id) != boundary.allowed {
			t.Errorf("visit event flag %d accepted = %t, want %t",
				boundary.id, accepts(boundary.id), boundary.allowed)
		}
	}
}

// SaveEngine stores a quantity in 31 bits because 0x80000000 is a preserved
// record flag, so a document promising the full uint32 range would advertise
// values SetOwnedItemQuantity rejects.
func assertQuantityFitsTheRecord(t *testing.T, schemas map[string]any) {
	t.Helper()

	for _, name := range []string{
		"SetOwnedItemQuantityRequest", "SetOwnedItemQuantityResult",
		"AddItemToInventoryRequest", "AddItemToInventoryResult",
		"AddItemToStorageRequest", "AddItemToStorageResult",
		"MoveOwnedItemToInventoryResult",
		"MoveOwnedItemToStorageResult",
	} {
		schema, _ := schemas[name].(map[string]any)
		properties, _ := schema["properties"].(map[string]any)
		quantity, _ := properties["quantity"].(map[string]any)
		if quantity["maximum"] != float64(2147483647) {
			t.Fatalf("%s.quantity maximum = %v, want 2147483647", name, quantity["maximum"])
		}
	}
}

// A save-session route is registered only when the explorer runs without
// -allow-external-bind, so its description must say so; otherwise the document
// would advertise a route an externally bound explorer answers with 404.
func assertLoopbackOnlySaveSessionRoutes(t *testing.T, paths map[string]map[string]any) {
	t.Helper()

	found := 0
	for path, operations := range paths {
		if !strings.HasPrefix(path, "/api/v1/save-sessions") {
			continue
		}
		for method, raw := range operations {
			operation, ok := raw.(map[string]any)
			if !ok {
				t.Fatalf("%s %s = %#v, want an operation object", method, path, raw)
			}
			description, _ := operation["description"].(string)
			if !strings.Contains(description, "-allow-external-bind") {
				t.Fatalf("%s %s does not state that it needs the local loopback mode", method, path)
			}
			found++
		}
	}
	if found != 82 {
		t.Fatalf("openapi.json describes %d save-session operations, want 82", found)
	}
}

// The relation endpoints carry the migrated (kind, key) identity; a numeric
// from/to would silently reintroduce the removed ResourceID contract.
func assertRelationEndpointsAreResourceRefs(t *testing.T, schemas map[string]any) {
	t.Helper()

	ref, ok := schemas["ResourceRef"].(map[string]any)
	if !ok {
		t.Fatalf("ResourceRef schema = %#v, want an object", schemas["ResourceRef"])
	}
	properties, ok := ref["properties"].(map[string]any)
	if !ok {
		t.Fatalf("ResourceRef properties = %#v, want an object", ref["properties"])
	}
	for _, field := range []string{"kind", "key"} {
		if _, exists := properties[field]; !exists {
			t.Fatalf("ResourceRef is missing the %q property", field)
		}
	}

	relation, ok := schemas["Relation"].(map[string]any)
	if !ok {
		t.Fatalf("Relation schema = %#v, want an object", schemas["Relation"])
	}
	relationProperties, ok := relation["properties"].(map[string]any)
	if !ok {
		t.Fatalf("Relation properties = %#v, want an object", relation["properties"])
	}
	for _, endpoint := range []string{"from", "to"} {
		field, ok := relationProperties[endpoint].(map[string]any)
		if !ok {
			t.Fatalf("Relation.%s = %#v, want an object", endpoint, relationProperties[endpoint])
		}
		if field["$ref"] != "#/components/schemas/ResourceRef" {
			t.Fatalf("Relation.%s = %#v, want a ResourceRef reference", endpoint, field)
		}
	}
}

// The built-in browser explorer was removed: Scalar Docs is the only browser UI
// and this process is the API host it calls. The route must be gone rather than
// answering an empty or broken page, and the document must not advertise it.
func TestDocsRouteIsNoLongerServed(t *testing.T) {
	for _, target := range []string{"/docs", "/docs/", "/docs.html"} {
		recorder := do(t, newPrototypeCatalog(t), target)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want %d", target, recorder.Code, http.StatusNotFound)
		}
	}

	// The asset itself is no longer embedded, so nothing can serve it again by
	// name even if a route were reintroduced by mistake.
	if _, err := assets.ReadFile("docs.html"); err == nil {
		t.Fatal("docs.html is still embedded in the binary")
	}

	recorder := do(t, newPrototypeCatalog(t), "/openapi.json")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var document struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode openapi.json: %v", err)
	}
	if _, exists := document.Paths["/docs"]; exists {
		t.Fatal("openapi.json still describes /docs")
	}
}

// Scalar renders the API Reference from another origin, so a relative server
// would make "Try it" call the documentation portal instead of this process.
// The document therefore names exactly one absolute loopback server.
func TestOpenAPIDocumentServesOneAbsoluteLoopbackServer(t *testing.T) {
	recorder := do(t, newPrototypeCatalog(t), "/openapi.json")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var document struct {
		Servers []struct {
			URL string `json:"url"`
		} `json:"servers"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode openapi.json: %v", err)
	}
	if len(document.Servers) != 1 {
		t.Fatalf("servers = %#v, want exactly one", document.Servers)
	}
	if document.Servers[0].URL != "http://127.0.0.1:8788" {
		t.Fatalf("server url = %q, want http://127.0.0.1:8788", document.Servers[0].URL)
	}
}

func TestValidateAddressRequiresLoopbackByDefault(t *testing.T) {
	for _, address := range []string{"127.0.0.1:8788", "localhost:8788", "[::1]:8788"} {
		if err := validateAddress(address, false); err != nil {
			t.Fatalf("validateAddress(%q, false) = %v, want nil", address, err)
		}
	}
	for _, address := range []string{"0.0.0.0:8788", "192.168.1.10:8788", ":8788", "[::]:8788"} {
		if err := validateAddress(address, false); err == nil {
			t.Fatalf("validateAddress(%q, false) = nil, want a rejection", address)
		}
	}
	if err := validateAddress("0.0.0.0:8788", true); err != nil {
		t.Fatalf("validateAddress with -allow-external-bind = %v, want nil", err)
	}
	if err := validateAddress("127.0.0.1", false); err == nil {
		t.Fatal("validateAddress accepted an address without a port")
	}
}

// doCORS sends one request with the given method, Origin and preflight headers
// through the full handler, so the CORS answer is observed at the HTTP layer the
// browser sees rather than on the middleware in isolation.
func doCORS(t *testing.T, method string, target string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, target, nil)
	for name, values := range header {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	recorder := httptest.NewRecorder()
	newHandler(newPrototypeCatalog(t), testApplicationVersion, nil).ServeHTTP(recorder, request)
	return recorder
}

// The documentation portal is served from another origin than the explorer, so
// a plain GET from it has to carry the permission header or the browser hides
// the body the route already produced.
func TestPortalOriginReceivesCORSPermission(t *testing.T) {
	recorder := doCORS(t, http.MethodGet, "/api/v1/catalog/info", http.Header{"Origin": []string{portalOrigin}})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != portalOrigin {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, portalOrigin)
	}
	// The permission must never be the wildcard: it admits one fixed page only.
	if recorder.Header().Get("Access-Control-Allow-Origin") == "*" {
		t.Fatal("the explorer answers a wildcard CORS origin")
	}
	if recorder.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("the explorer allows credentials over CORS")
	}
	if got := recorder.Header().Values("Vary"); !slices.Contains(got, "Origin") {
		t.Fatalf("Vary = %v, want it to contain Origin", got)
	}
	// The route itself still answers: the middleware must not swallow the body.
	if recorder.Body.Len() == 0 {
		t.Fatal("the route returned no body")
	}
}

// JSON requests make the browser preflight the mutation routes. The mux routes
// no OPTIONS, so the preflight has to be answered before it reaches a route.
func TestPortalPreflightAllowsTheDocumentedMethodsAndHeaders(t *testing.T) {
	recorder := doCORS(t, http.MethodOptions, "/api/v1/save-sessions", http.Header{
		"Origin":                         []string{portalOrigin},
		"Access-Control-Request-Method":  []string{http.MethodPost},
		"Access-Control-Request-Headers": []string{"content-type"},
	})

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != portalOrigin {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, portalOrigin)
	}
	methods := recorder.Header().Get("Access-Control-Allow-Methods")
	for _, method := range []string{
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions,
	} {
		if !strings.Contains(methods, method) {
			t.Fatalf("Access-Control-Allow-Methods = %q, want it to allow %s", methods, method)
		}
	}
	// Content-Type is what turns the JSON body into a preflighted request.
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Content-Type") {
		t.Fatalf("Access-Control-Allow-Headers = %q, want it to allow Content-Type", got)
	}
}

// Any other origin gets no permission at all, for a simple request and for a
// preflight alike, so the single allowed page stays the only browser caller.
func TestForeignOriginReceivesNoCORSPermission(t *testing.T) {
	foreignOrigins := []string{
		"http://evil.example",
		"https://localhost:7970", // the scheme is part of the origin
		"http://localhost:7971",  // so is the port
		"http://127.0.0.1:7970",  // and the host, even another loopback spelling
		"http://localhost:7970.example",
	}
	for _, origin := range foreignOrigins {
		simple := doCORS(t, http.MethodGet, "/api/v1/catalog/info", http.Header{"Origin": []string{origin}})
		if got := simple.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("origin %q received Access-Control-Allow-Origin %q, want none", origin, got)
		}

		preflight := doCORS(t, http.MethodOptions, "/api/v1/save-sessions", http.Header{
			"Origin":                        []string{origin},
			"Access-Control-Request-Method": []string{http.MethodPost},
		})
		if got := preflight.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("origin %q received a preflight permission %q, want none", origin, got)
		}
		if got := preflight.Header().Get("Access-Control-Allow-Methods"); got != "" {
			t.Fatalf("origin %q received Access-Control-Allow-Methods %q, want none", origin, got)
		}
	}
}

// The equipped-spells route is the one save-session route that also needs the
// catalog, so it gets its own checks: the successful body, the characterID the
// route itself rejects, and its absence without an engine.
// Layout of the local active-slot fixture this one test needs. The generic
// fixture leaves every slot inactive, which would prove nothing about the
// catalog the route passes on, so slot 0 is filled in here instead.
const (
	equippedSpellsUserData10Offset = int64(pcHeaderSize) + 10*0x280010 + 0x10
	equippedSpellsFlagsOffset      = 0x1954
	equippedSpellsSlotDataBase     = int64(pcHeaderSize) + 0x10
	equippedSpellsAnchorAt         = 0x0640
	equippedSpellsSectionAt        = 0x9205
	equippedSpellsRecordCount      = 14
	equippedSpellsRecordSize       = 8
	rawGlintstonePebble            = 0x0FA0
)

// equippedSpellsFixtureAnchor is the confirmed 65-byte SaveEngine anchor: one
// leading 0x00 byte, then four full repetitions of a 16-byte block made of
// 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes.
var equippedSpellsFixtureAnchor = []byte{
	0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

// writeActiveSpellsFixture writes a synthetic PC save into t.TempDir() whose
// slot 0 is active and carries one occupied spell record followed by thirteen
// native empty pairs.
func writeActiveSpellsFixture(t *testing.T) string {
	t.Helper()

	data := make([]byte, pcFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[pcEntryCountOffset:], pcEntryCount)

	data[equippedSpellsUserData10Offset+equippedSpellsFlagsOffset] = 1

	anchorBase := equippedSpellsSlotDataBase + equippedSpellsAnchorAt
	copy(data[anchorBase:], equippedSpellsFixtureAnchor)
	for index := 0; index < equippedSpellsRecordCount; index++ {
		at := anchorBase + equippedSpellsSectionAt + int64(index)*equippedSpellsRecordSize
		spellID, follower := uint32(0xFFFFFFFF), uint32(0x00000000)
		if index == 0 {
			spellID, follower = rawGlintstonePebble, 0xFFFFFFFF
		}
		binary.LittleEndian.PutUint32(data[at:], spellID)
		binary.LittleEndian.PutUint32(data[at+4:], follower)
	}

	path := filepath.Join(t.TempDir(), "active-spells.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// Both undo routes share one path, so the getter and the mutation have to be
// reachable under their own methods, and the mutation body has to stay strict.
func TestCharacterUndoRoutes(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writePCFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/undo"

	want, err := character.GetUndoState(saveEngine, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("character.GetUndoState: %v", err)
	}
	state := doSave(t, saveEngine, http.MethodGet, target, "")
	assertOK(t, state, target)
	if !reflect.DeepEqual(decode(t, state.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("undo route body differs from the GetUndoState result")
	}

	unknownField := doSave(t, saveEngine, http.MethodPost, target,
		`{"undoToken":"any-token","expectedRevision":"0","extra":true}`)
	if unknownField.Code != http.StatusBadRequest {
		t.Fatalf("unknown field: status = %d, body = %q",
			unknownField.Code, unknownField.Body.String())
	}
	if !strings.Contains(unknownField.Body.String(), "unknown field") {
		t.Fatalf("unknown field: body = %q, want the strict-decoder rejection",
			unknownField.Body.String())
	}

	// A well-formed request reaches SaveEngine, which owns the absent-point rule.
	delegated := doSave(t, saveEngine, http.MethodPost, target,
		`{"undoToken":"any-token","expectedRevision":"0"}`)
	if delegated.Code != http.StatusBadRequest {
		t.Fatalf("absent undo point: status = %d, body = %q",
			delegated.Code, delegated.Body.String())
	}
	if !strings.Contains(delegated.Body.String(), "no undo point is available") {
		t.Fatalf("absent undo point: body = %q, want the SaveEngine rejection",
			delegated.Body.String())
	}
}

func TestDeleteCharacterRoute(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0"

	for name, body := range map[string]string{
		"missing revision": `{}`,
		"unknown field":    `{"expectedRevision":"0","extra":true}`,
	} {
		t.Run(name, func(t *testing.T) {
			rejected := doSave(t, saveEngine, http.MethodDelete, target, body)
			if rejected.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %q", rejected.Code, rejected.Body.String())
			}
		})
	}

	recorder := doSave(t, saveEngine, http.MethodDelete, target,
		`{"expectedRevision":"0"}`)
	assertOK(t, recorder, target)
	var got character.DeleteCharacterResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	want := character.DeleteCharacterResult{
		SaveSessionID: session.SaveSessionID,
		SaveRevision:  "1",
		CharacterID:   0,
	}
	if got != want {
		t.Errorf("result = %+v, want %+v", got, want)
	}
	profile, err := saveEngine.GetCharacterProfile(session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("GetCharacterProfile: %v", err)
	}
	if profile.Active {
		t.Errorf("profile = %+v, want deleted slot", profile)
	}
}

func TestCloneCharacterRoute(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if _, err := saveEngine.SetCharacterName(
		session.SaveSessionID, 0, "Ranni", "0"); err != nil {
		t.Fatalf("SetCharacterName fixture setup: %v", err)
	}
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/clone"

	for name, body := range map[string]string{
		"missing target": `{ "expectedRevision":"1" }`,
		"unknown field":  `{ "targetSlotID":1,"expectedRevision":"1","extra":true }`,
	} {
		t.Run(name, func(t *testing.T) {
			rejected := doSave(t, saveEngine, http.MethodPost, target, body)
			if rejected.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %q", rejected.Code, rejected.Body.String())
			}
		})
	}

	recorder := doSave(t, saveEngine, http.MethodPost, target,
		`{"targetSlotID":1,"expectedRevision":"1"}`)
	assertOK(t, recorder, target)
	var got character.CloneCharacterResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	want := character.CloneCharacterResult{
		SaveSessionID:     session.SaveSessionID,
		SaveRevision:      "2",
		SourceCharacterID: 0,
		TargetSlotID:      1,
		Name:              "Ranni 2",
	}
	if got != want {
		t.Errorf("result = %+v, want %+v", got, want)
	}
	profile, err := saveEngine.GetCharacterProfile(session.SaveSessionID, 1)
	if err != nil {
		t.Fatalf("GetCharacterProfile: %v", err)
	}
	if !profile.Active || profile.Name != "Ranni 2" {
		t.Errorf("profile = %+v, want active Ranni 2", profile)
	}
}

func TestSetCharacterActiveRoute(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/active"

	recorder := doSave(t, saveEngine, http.MethodPatch, target,
		`{"active":false,"expectedRevision":"0"}`)
	assertOK(t, recorder, target)
	var got character.SetCharacterActiveResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.SaveSessionID != session.SaveSessionID || got.SaveRevision != "1" ||
		got.CharacterID != 0 || got.Active {
		t.Errorf("result = %+v, want inactive character 0 at revision 1", got)
	}

	for name, body := range map[string]string{
		"missing active": `{"expectedRevision":"1"}`,
		"unknown field":  `{"active":true,"expectedRevision":"1","extra":1}`,
	} {
		t.Run(name, func(t *testing.T) {
			rejected := doSave(t, saveEngine, http.MethodPatch, target, body)
			if rejected.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %q", rejected.Code, rejected.Body.String())
			}
		})
	}
}

func TestSetSaveAccountIDRoute(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writePCFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/account-id"

	recorder := doSave(t, saveEngine, http.MethodPatch, target,
		`{"accountID":"1311768467463790320","expectedRevision":"0"}`)
	assertOK(t, recorder, target)
	var got savesession.SetSaveAccountIDResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.SaveSessionID != session.SaveSessionID || got.SaveRevision != "1" {
		t.Errorf("result = %+v, want the session at revision 1", got)
	}
	// The identifier is private account data and must not travel back.
	if strings.Contains(recorder.Body.String(), "1311768467463790320") {
		t.Errorf("the response repeats the identifier: %q", recorder.Body.String())
	}

	for name, body := range map[string]string{
		"missing accountID":        `{"expectedRevision":"1"}`,
		"missing expectedRevision": `{"accountID":"1"}`,
		"unknown field":            `{"accountID":"1","expectedRevision":"1","extra":1}`,
	} {
		t.Run(name, func(t *testing.T) {
			rejected := doSave(t, saveEngine, http.MethodPatch, target, body)
			if rejected.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %q", rejected.Code, rejected.Body.String())
			}
		})
	}
}

func TestSetCharacterNameRoute(t *testing.T) {
	saveEngine := saveengine.New()

	t.Run("conforms to endpoint mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		body, err := json.Marshal(setCharacterNameRequest{
			Name: "Ranni 🌙", ExpectedRevision: "0",
		})
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		target := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/0/name"
		request := httptest.NewRequest(http.MethodPatch, target, bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(newPrototypeCatalog(t), testApplicationVersion, saveEngine).
			ServeHTTP(recorder, request)
		assertOK(t, recorder, target)

		var got character.SetCharacterNameResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if got.SaveRevision != "1" || got.CharacterID != 0 || got.Name != "Ranni 🌙" {
			t.Errorf("result = %+v, want revision 1 and exact name", got)
		}

		directSession, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave direct session: %v", err)
		}
		want, err := character.SetCharacterName(
			saveEngine, directSession.SaveSessionID, 0, "Ranni 🌙", "0")
		if err != nil {
			t.Fatalf("character.SetCharacterName: %v", err)
		}
		want.SaveSessionID = got.SaveSessionID
		if got != want {
			t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
		}
	})

	t.Run("rejects invalid bodies before mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		base := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/0/name"
		for name, body := range map[string]string{
			"empty name":    `{"name":"","expectedRevision":"0"}`,
			"unknown field": `{"name":"Ranni","expectedRevision":"0","extra":1}`,
		} {
			t.Run(name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPatch, base, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				newHandler(newPrototypeCatalog(t), testApplicationVersion, saveEngine).
					ServeHTTP(recorder, request)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body %q)",
						recorder.Code, recorder.Body.String())
				}
			})
		}

		request := httptest.NewRequest(http.MethodPatch, base, strings.NewReader(
			`{"name":"Ranni","expectedRevision":"0"}`))
		recorder := httptest.NewRecorder()
		newHandler(newPrototypeCatalog(t), testApplicationVersion, saveEngine).
			ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("missing Content-Type: status = %d, want 400 (body %q)",
				recorder.Code, recorder.Body.String())
		}
		info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
		if err != nil {
			t.Fatalf("GetSessionInfo: %v", err)
		}
		if info.UnsavedChanges {
			t.Errorf("session after rejected bodies = %+v, want clean", info)
		}
	})
}

func TestSetCharacterAppearanceRoute(t *testing.T) {
	saveEngine := saveengine.New()
	fixture := writeActiveSpellsFixture(t)
	fixtureData, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	face := fixtureData[equippedSpellsSlotDataBase+0x3000:]
	copy(face, []byte{0xFF, 0xFF, 0xFF, 0xFF, 'F', 'A', 'C', 'E'})
	binary.LittleEndian.PutUint32(face[0x08:], 4)
	binary.LittleEndian.PutUint32(face[0x0C:], 0x120)
	if err := os.WriteFile(fixture, fixtureData, 0o600); err != nil {
		t.Fatalf("rewrite fixture: %v", err)
	}
	session, err := savesession.LoadSave(saveEngine, fixture, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	current, err := character.GetCharacterAppearance(
		saveEngine, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("GetCharacterAppearance: %v", err)
	}
	gender, voiceType := uint8(0), uint8(5)
	requestBody := setCharacterAppearanceRequest{
		Appearance: &setCharacterAppearanceValuesRequest{
			Gender:    &gender,
			VoiceType: &voiceType,
			ModelIDs:  append([]uint32(nil), current.ModelIDs[:]...),
			FaceShape: append([]uint8(nil), current.FaceShape[:]...),
			Body:      append([]uint8(nil), current.Body[:]...),
			Skin:      append([]uint8(nil), current.Skin[:]...),
		},
		ExpectedRevision: "0",
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/appearance"
	recorder := doSave(t, saveEngine, http.MethodPut, target, string(body))
	assertOK(t, recorder, target)

	var got character.SetCharacterAppearanceResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.SaveSessionID != session.SaveSessionID || got.SaveRevision != "1" ||
		got.CharacterID != 0 || got.Appearance.Gender != gender ||
		got.Appearance.VoiceType != voiceType {
		t.Errorf("result = %+v, want committed appearance receipt", got)
	}

	requestBody.Appearance.ModelIDs = requestBody.Appearance.ModelIDs[:7]
	requestBody.ExpectedRevision = "1"
	body, err = json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal invalid request: %v", err)
	}
	rejected := doSave(t, saveEngine, http.MethodPut, target, string(body))
	if rejected.Code != http.StatusBadRequest ||
		!strings.Contains(rejected.Body.String(), "appearance.modelIDs has 7 values, want exactly 8") {
		t.Fatalf("invalid array: status = %d, body = %q", rejected.Code, rejected.Body.String())
	}
	readBack, err := character.GetCharacterAppearance(saveEngine, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("GetCharacterAppearance after rejection: %v", err)
	}
	if readBack.ModelIDs != got.Appearance.ModelIDs ||
		readBack.FaceShape != got.Appearance.FaceShape ||
		readBack.Body != got.Appearance.Body || readBack.Skin != got.Appearance.Skin {
		t.Error("rejected transport request changed the appearance")
	}
}

func TestSetCharacterGenderRoute(t *testing.T) {
	saveEngine := saveengine.New()
	fixture := writeActiveSpellsFixture(t)
	fixtureData, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	face := fixtureData[equippedSpellsSlotDataBase+0x3000:]
	copy(face, []byte{0xFF, 0xFF, 0xFF, 0xFF, 'F', 'A', 'C', 'E'})
	binary.LittleEndian.PutUint32(face[0x08:], 4)
	binary.LittleEndian.PutUint32(face[0x0C:], 0x120)
	if err := os.WriteFile(fixture, fixtureData, 0o600); err != nil {
		t.Fatalf("rewrite fixture: %v", err)
	}
	session, err := savesession.LoadSave(saveEngine, fixture, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	gender := uint8(0)
	body, err := json.Marshal(setCharacterGenderRequest{
		Gender: &gender, ExpectedRevision: "0",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/gender"
	recorder := doSave(t, saveEngine, http.MethodPatch, target, string(body))
	assertOK(t, recorder, target)

	var got character.SetCharacterGenderResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	wantModelIDs := [8]uint32{40, 0, 0, 0, 0, 0, 0, 0}
	if got.SaveSessionID != session.SaveSessionID || got.SaveRevision != "1" ||
		got.CharacterID != 0 ||
		got.PresetID != "ciri-the-princess-of-cintra-from-witcher" ||
		got.Appearance.Gender != gender || got.Appearance.ModelIDs != wantModelIDs {
		t.Fatalf("result = %+v, want committed Type B default", got)
	}

	rejected := doSave(t, saveEngine, http.MethodPatch, target,
		`{"expectedRevision":"1"}`)
	if rejected.Code != http.StatusBadRequest ||
		!strings.Contains(rejected.Body.String(), "gender is required") {
		t.Fatalf("missing gender: status = %d, body = %q",
			rejected.Code, rejected.Body.String())
	}
}

func TestApplyAppearancePresetRoute(t *testing.T) {
	saveEngine := saveengine.New()
	fixture := writeActiveSpellsFixture(t)
	fixtureData, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	face := fixtureData[equippedSpellsSlotDataBase+0x3000:]
	copy(face, []byte{0xFF, 0xFF, 0xFF, 0xFF, 'F', 'A', 'C', 'E'})
	binary.LittleEndian.PutUint32(face[0x08:], 4)
	binary.LittleEndian.PutUint32(face[0x0C:], 0x120)
	if err := os.WriteFile(fixture, fixtureData, 0o600); err != nil {
		t.Fatalf("rewrite fixture: %v", err)
	}
	session, err := savesession.LoadSave(saveEngine, fixture, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	requestBody := applyAppearancePresetRequest{
		PresetID:         "yennefer-sorceress-from-the-witcher",
		ExpectedRevision: "0",
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/appearance/preset"
	recorder := doSave(t, saveEngine, http.MethodPut, target, string(body))
	assertOK(t, recorder, target)

	var got appearance.ApplyAppearancePresetResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	wantModelIDs := [8]uint32{50, 106, 0, 14, 0, 0, 8, 3}
	if got.SaveSessionID != session.SaveSessionID || got.SaveRevision != "1" ||
		got.CharacterID != 0 || got.PresetID != requestBody.PresetID ||
		got.Appearance.ModelIDs != wantModelIDs {
		t.Fatalf("result = %+v, want committed Yennefer preset", got)
	}

	rejected := doSave(t, saveEngine, http.MethodPut, target,
		`{"presetID":"geralt-of-rivia-the-witcher","expectedRevision":"1","extra":true}`)
	if rejected.Code != http.StatusBadRequest ||
		!strings.Contains(rejected.Body.String(), "unknown field") {
		t.Fatalf("unknown field: status = %d, body = %q",
			rejected.Code, rejected.Body.String())
	}
}

func TestSetCharacterNameRouteIsAbsentWithoutAnEngine(t *testing.T) {
	target := "/api/v1/save-sessions/any-session/characters/0/name"
	recorder := doSave(t, nil, http.MethodPatch, target,
		`{"name":"Ranni","expectedRevision":"0"}`)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("%s: status = %d, want 404 (body %q)",
			target, recorder.Code, recorder.Body.String())
	}
}

func TestSetCharacterActiveRouteIsAbsentWithoutAnEngine(t *testing.T) {
	target := "/api/v1/save-sessions/any-session/characters/0/active"
	recorder := doSave(t, nil, http.MethodPatch, target,
		`{"active":false,"expectedRevision":"0"}`)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("%s: status = %d, want 404 (body %q)",
			target, recorder.Code, recorder.Body.String())
	}
}

func TestDeleteCharacterRouteIsAbsentWithoutAnEngine(t *testing.T) {
	target := "/api/v1/save-sessions/any-session/characters/0"
	recorder := doSave(t, nil, http.MethodDelete, target,
		`{"expectedRevision":"0"}`)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("%s: status = %d, want 404 (body %q)",
			target, recorder.Code, recorder.Body.String())
	}
}

func TestCloneCharacterRouteIsAbsentWithoutAnEngine(t *testing.T) {
	target := "/api/v1/save-sessions/any-session/characters/0/clone"
	recorder := doSave(t, nil, http.MethodPost, target,
		`{"targetSlotID":1,"expectedRevision":"0"}`)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("%s: status = %d, want 404 (body %q)",
			target, recorder.Code, recorder.Body.String())
	}
}

func TestSetCharacterRunesRoute(t *testing.T) {
	saveEngine := saveengine.New()

	t.Run("conforms to endpoint mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		runes := uint32(999_999_999)
		body, err := json.Marshal(setCharacterRunesRequest{
			Runes: &runes, ExpectedRevision: "0",
		})
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		target := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/0/runes"
		request := httptest.NewRequest(http.MethodPatch, target, bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(newPrototypeCatalog(t), testApplicationVersion, saveEngine).
			ServeHTTP(recorder, request)
		assertOK(t, recorder, target)

		var got character.SetCharacterRunesResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if got.SaveRevision != "1" || got.CharacterID != 0 || got.Runes != runes {
			t.Errorf("result = %+v, want revision 1 and runes %d", got, runes)
		}

		directSession, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave direct session: %v", err)
		}
		want, err := character.SetCharacterRunes(
			saveEngine, directSession.SaveSessionID, 0, runes, "0")
		if err != nil {
			t.Fatalf("character.SetCharacterRunes: %v", err)
		}
		want.SaveSessionID = got.SaveSessionID
		if got != want {
			t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
		}
	})

	t.Run("accepts explicit zero", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		target := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/0/runes"
		request := httptest.NewRequest(http.MethodPatch, target, strings.NewReader(
			`{"runes":0,"expectedRevision":"0"}`))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(newPrototypeCatalog(t), testApplicationVersion, saveEngine).
			ServeHTTP(recorder, request)
		assertOK(t, recorder, target)
	})

	t.Run("rejects invalid bodies before mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		base := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/0/runes"
		for name, body := range map[string]string{
			"missing runes":  `{"expectedRevision":"0"}`,
			"above maximum":  `{"runes":1000000000,"expectedRevision":"0"}`,
			"negative runes": `{"runes":-1,"expectedRevision":"0"}`,
			"unknown field":  `{"runes":1,"expectedRevision":"0","extra":1}`,
		} {
			t.Run(name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPatch, base, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				newHandler(newPrototypeCatalog(t), testApplicationVersion, saveEngine).
					ServeHTTP(recorder, request)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body %q)",
						recorder.Code, recorder.Body.String())
				}
			})
		}
		info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
		if err != nil {
			t.Fatalf("GetSessionInfo: %v", err)
		}
		if info.UnsavedChanges {
			t.Errorf("session after rejected bodies = %+v, want clean", info)
		}
	})
}

func TestSetCharacterRunesRouteIsAbsentWithoutAnEngine(t *testing.T) {
	target := "/api/v1/save-sessions/any-session/characters/0/runes"
	recorder := doSave(t, nil, http.MethodPatch, target,
		`{"runes":1,"expectedRevision":"0"}`)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("%s: status = %d, want 404 (body %q)",
			target, recorder.Code, recorder.Body.String())
	}
}

// setCharacterStatsRouteBody is a legal Vagabond assignment for the synthetic
// route fixture: every value is at or above the Vagabond base and the sum
// recalculates to level 44.
const setCharacterStatsRouteBody = `{"attributes":{"vigor":20,"mind":15,"endurance":16,` +
	`"strength":20,"dexterity":18,"intelligence":12,"faith":12,"arcane":10},` +
	`"levelPolicy":"recalculate","expectedRevision":"0"}`

func TestSetCharacterStatsRoute(t *testing.T) {
	saveEngine := saveengine.New()

	t.Run("conforms to endpoint mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/stats"
		request := httptest.NewRequest(http.MethodPatch, target,
			strings.NewReader(setCharacterStatsRouteBody))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(newPrototypeCatalog(t), testApplicationVersion, saveEngine).
			ServeHTTP(recorder, request)
		assertOK(t, recorder, target)

		var got character.SetCharacterStatsResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		directSession, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave direct session: %v", err)
		}
		want, err := character.SetCharacterStats(
			saveEngine, directSession.SaveSessionID, 0,
			character.CharacterAttributes{
				Vigor: 20, Mind: 15, Endurance: 16, Strength: 20,
				Dexterity: 18, Intelligence: 12, Faith: 12, Arcane: 10,
			}, "recalculate", "0")
		if err != nil {
			t.Fatalf("character.SetCharacterStats: %v", err)
		}
		want.SaveSessionID = got.SaveSessionID
		if got != want {
			t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
		}
	})

	t.Run("rejects invalid bodies before mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/stats"
		for name, body := range map[string]string{
			"missing attributes": `{"levelPolicy":"recalculate","expectedRevision":"0"}`,
			"missing one attribute": `{"attributes":{"vigor":20,"mind":15,"endurance":16,` +
				`"strength":20,"dexterity":18,"intelligence":12,"faith":12},` +
				`"levelPolicy":"recalculate","expectedRevision":"0"}`,
			"unknown attribute field": `{"attributes":{"vigor":20,"mind":15,"endurance":16,` +
				`"strength":20,"dexterity":18,"intelligence":12,"faith":12,"arcane":10,"luck":5},` +
				`"levelPolicy":"recalculate","expectedRevision":"0"}`,
			"unknown top-level field": `{"attributes":{"vigor":20,"mind":15,"endurance":16,` +
				`"strength":20,"dexterity":18,"intelligence":12,"faith":12,"arcane":10},` +
				`"levelPolicy":"recalculate","expectedRevision":"0","level":44}`,
			"negative attribute": `{"attributes":{"vigor":-1,"mind":15,"endurance":16,` +
				`"strength":20,"dexterity":18,"intelligence":12,"faith":12,"arcane":10},` +
				`"levelPolicy":"recalculate","expectedRevision":"0"}`,
			"unknown level policy": `{"attributes":{"vigor":20,"mind":15,"endurance":16,` +
				`"strength":20,"dexterity":18,"intelligence":12,"faith":12,"arcane":10},` +
				`"levelPolicy":"preserve","expectedRevision":"0"}`,
		} {
			t.Run(name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPatch, base, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				newHandler(newPrototypeCatalog(t), testApplicationVersion, saveEngine).
					ServeHTTP(recorder, request)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body %q)",
						recorder.Code, recorder.Body.String())
				}
			})
		}
		info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
		if err != nil {
			t.Fatalf("GetSessionInfo: %v", err)
		}
		if info.UnsavedChanges {
			t.Errorf("session after rejected bodies = %+v, want clean", info)
		}
	})
}

func TestSetCharacterStatsRouteIsAbsentWithoutAnEngine(t *testing.T) {
	target := "/api/v1/save-sessions/any-session/characters/0/stats"
	recorder := doSave(t, nil, http.MethodPatch, target, setCharacterStatsRouteBody)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("%s: status = %d, want 404 (body %q)",
			target, recorder.Code, recorder.Body.String())
	}
}

func writeSetPhysickMixtureRouteFixture(t *testing.T) string {
	t.Helper()

	path := writeActiveSpellsFixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	anchorBase := equippedSpellsSlotDataBase + equippedSpellsAnchorAt
	inventoryAt := anchorBase + 505
	binary.LittleEndian.PutUint32(data[inventoryAt-4:], 1)
	binary.LittleEndian.PutUint32(data[inventoryAt:], 0xB00000FA)
	binary.LittleEndian.PutUint32(data[inventoryAt+4:], 1)
	keyAt := inventoryAt + 0xA80*12 + 4
	binary.LittleEndian.PutUint32(data[keyAt-4:], 1)
	binary.LittleEndian.PutUint32(data[keyAt:], 0xB0002AF9)
	binary.LittleEndian.PutUint32(data[keyAt+4:], 1)

	physickAt := anchorBase + 0x931D + 4 + 0x9C
	binary.LittleEndian.PutUint32(data[physickAt:], saveengine.PhysickEmptyTearID)
	binary.LittleEndian.PutUint32(data[physickAt+4:], saveengine.PhysickEmptyTearID)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}
	return path
}

func TestSetPhysickMixtureRoute(t *testing.T) {
	saveEngine := saveengine.New()
	gameCatalog := newFullCatalog(t)
	session, err := savesession.LoadSave(saveEngine, writeSetPhysickMixtureRouteFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	body, err := json.Marshal(setPhysickMixtureRequest{
		CrystalTearResources: []*schema.ResourceRef{
			{Kind: schema.ResourceKindItem, Key: "40002AF9"}, nil,
		},
		ExpectedRevision: "0",
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/physick-mixture"
	request := httptest.NewRequest(http.MethodPut, target, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
	assertOK(t, recorder, target)

	var got equipment.SetPhysickMixtureResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.SaveRevision != "1" || got.CharacterID != 0 ||
		got.CrystalTearResources[0] == nil || got.CrystalTearResources[1] != nil {
		t.Fatalf("result = %+v, want revision 1 and one occupied position", got)
	}

	mixture, err := equipment.GetPhysickMixture(saveEngine, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("GetPhysickMixture: %v", err)
	}
	if want := [2]uint32{0x40002AF9, saveengine.PhysickEmptyTearID}; mixture.Tears != want {
		t.Errorf("stored tears = %08X/%08X, want %08X/%08X",
			mixture.Tears[0], mixture.Tears[1], want[0], want[1])
	}
}

func writeSetPouchItemsRouteFixture(t *testing.T) string {
	t.Helper()

	path := writeActiveSpellsFixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	anchorBase := equippedSpellsSlotDataBase + equippedSpellsAnchorAt

	pairAt := anchorBase + 0x92CD
	for i := 0; i < 6; i++ {
		binary.LittleEndian.PutUint32(data[pairAt+int64(i)*8:], 0)
		binary.LittleEndian.PutUint32(data[pairAt+int64(i)*8+4:], 0xFFFFFFFF)
	}

	countAt := anchorBase + 0x931D
	binary.LittleEndian.PutUint32(data[countAt:], 17)

	tailAt := countAt + 4 + 17*8 + 0x80
	for i := 0; i < 6; i++ {
		binary.LittleEndian.PutUint32(data[tailAt+int64(i)*4:], 0xFFFFFFFF)
	}

	inventoryAt := anchorBase + 505
	binary.LittleEndian.PutUint32(data[inventoryAt-4:], 1)
	binary.LittleEndian.PutUint32(data[inventoryAt:], 0xB00006A4)
	binary.LittleEndian.PutUint32(data[inventoryAt+4:], 10)
	binary.LittleEndian.PutUint32(data[inventoryAt+8:], 1)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}
	return path
}

func writeSetQuickItemsRouteFixture(t *testing.T) string {
	t.Helper()

	path := writeSetPouchItemsRouteFixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	anchorBase := equippedSpellsSlotDataBase + equippedSpellsAnchorAt
	pairAt := anchorBase + 0x9279
	for i := 0; i < 10; i++ {
		binary.LittleEndian.PutUint32(data[pairAt+int64(i)*8:], 0)
		binary.LittleEndian.PutUint32(data[pairAt+int64(i)*8+4:], 0xFFFFFFFF)
	}
	countAt := anchorBase + 0x931D
	tailAt := countAt + 4 + 17*8 + 0x58
	for i := 0; i < 10; i++ {
		binary.LittleEndian.PutUint32(data[tailAt+int64(i)*4:], 0xFFFFFFFF)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}
	return path
}

func TestSetQuickItemsRoute(t *testing.T) {
	saveEngine := saveengine.New()
	gameCatalog := newFullCatalog(t)
	session, err := savesession.LoadSave(saveEngine, writeSetQuickItemsRouteFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := saveEngine.GetInventory(session.SaveSessionID, 0, "common", 1, 50)
	if err != nil || len(inventory.Records) == 0 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
	}
	token := inventory.Records[0].OwnedItemID
	assignments := make([]*string, 10)
	assignments[0] = &token
	body, err := json.Marshal(setQuickItemsRequest{
		SlotAssignments: assignments, ExpectedRevision: "0",
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	target := fmt.Sprintf(
		"/api/v1/save-sessions/%s/characters/0/quick-items", session.SaveSessionID)
	request := httptest.NewRequest(http.MethodPut, target, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}

	var got equipment.SetQuickItemsResult
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if got.SaveRevision != "1" || got.CharacterID != 0 ||
		got.SlotAssignments[0] == nil || got.SlotAssignments[0].Key != "400006A4" {
		t.Fatalf("result = %+v", got)
	}
	quick, err := equipment.GetQuickItems(saveEngine, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("GetQuickItems: %v", err)
	}
	if quick.Items[0].ItemID == 0 || quick.Items[0].EquipIndex != 0x180 {
		t.Errorf("quick item 0 = %+v, want valid inventory reference", quick.Items[0])
	}
}

func writeSetEquippedArmamentsRouteFixture(t *testing.T) string {
	t.Helper()
	const (
		anchorAt    = 0xA07B
		inventoryAt = anchorAt + 505
		unarmedID   = uint32(0x0001ADB0)
		weaponID    = uint32(0x000F4240)
	)

	data := make([]byte, pcFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[pcEntryCountOffset:], pcEntryCount)
	data[equippedSpellsUserData10Offset+equippedSpellsFlagsOffset] = 1
	binary.LittleEndian.PutUint32(data[equippedSpellsSlotDataBase:], 83)

	for index := 0; index < 7; index++ {
		handle := uint32(0x80000100 + index)
		gameID := weaponID
		if index == 0 {
			gameID = unarmedID
		}
		position := equippedSpellsSlotDataBase + 0x20 + int64(index*21)
		binary.LittleEndian.PutUint32(data[position:], handle)
		binary.LittleEndian.PutUint32(data[position+4:], gameID)
		rowAt := equippedSpellsSlotDataBase + inventoryAt + int64(index*12)
		binary.LittleEndian.PutUint32(data[rowAt:], handle)
		binary.LittleEndian.PutUint32(data[rowAt+4:], 1)
		binary.LittleEndian.PutUint32(data[rowAt+8:], uint32(index+1))
	}
	anchor := equippedSpellsSlotDataBase + anchorAt
	copy(data[anchor:], equippedSpellsFixtureAnchor)
	armamentsAt := anchor + 0x931D + 4
	for slot := 0; slot < 6; slot++ {
		binary.LittleEndian.PutUint32(data[anchor+0xD1+int64(slot*4):], 0x180)
		binary.LittleEndian.PutUint32(data[anchor+0x145+int64(slot*4):], unarmedID)
		binary.LittleEndian.PutUint32(data[anchor+0x19D+int64(slot*4):], 0x80000100)
		binary.LittleEndian.PutUint32(data[armamentsAt+int64(slot*4):], unarmedID)
	}

	path := filepath.Join(t.TempDir(), "set-equipped-armaments-route.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestSetEquippedArmamentsRoute(t *testing.T) {
	saveEngine := saveengine.New()
	gameCatalog := newFullCatalog(t)
	session, err := savesession.LoadSave(
		saveEngine, writeSetEquippedArmamentsRouteFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := saveEngine.GetInventory(session.SaveSessionID, 0, "common", 1, 50)
	if err != nil || len(inventory.Records) != 7 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
	}
	assignments := make([]*string, 6)
	for slot := range assignments {
		token := inventory.Records[slot+1].OwnedItemID
		assignments[slot] = &token
	}
	body, err := json.Marshal(setEquippedArmamentsRequest{
		SlotAssignments:  assignments,
		ExpectedRevision: "0",
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	target := fmt.Sprintf(
		"/api/v1/save-sessions/%s/characters/0/equipped-armaments", session.SaveSessionID)
	request := httptest.NewRequest(http.MethodPut, target, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}

	var got equipment.SetEquippedArmamentsResult
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if got.SaveRevision != "1" || got.SlotAssignments[0] == nil ||
		got.SlotAssignments[0].Key != "000F4240" {
		t.Fatalf("result = %+v", got)
	}
	equipped, err := saveEngine.GetEquipment(session.SaveSessionID, 0)
	if err != nil || equipped.Slots[0] != 0x000F4240 || equipped.Slots[5] != 0x000F4240 {
		t.Fatalf("GetEquipment: %v, slots=%#v", err, equipped.Slots)
	}
}

func writeSetEquippedArmorRouteFixture(t *testing.T) string {
	t.Helper()
	const (
		anchorAt    = 0xA060
		inventoryAt = anchorAt + 505
	)
	emptyGameIDs := [4]uint32{0x10002710, 0x10002774, 0x100027D8, 0x1000283C}
	actualGameIDs := [4]uint32{0x10009C40, 0x10009CA4, 0x10009D08, 0x10009D6C}

	data := make([]byte, pcFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[pcEntryCountOffset:], pcEntryCount)
	data[equippedSpellsUserData10Offset+equippedSpellsFlagsOffset] = 1
	binary.LittleEndian.PutUint32(data[equippedSpellsSlotDataBase:], 83)

	gameIDs := append(emptyGameIDs[:], actualGameIDs[:]...)
	position := equippedSpellsSlotDataBase + 0x20
	for index, gameID := range gameIDs {
		handle := uint32(0x90000100 + index)
		binary.LittleEndian.PutUint32(data[position:], handle)
		binary.LittleEndian.PutUint32(data[position+4:], gameID)
		position += 16
		rowAt := equippedSpellsSlotDataBase + inventoryAt + int64(index*12)
		binary.LittleEndian.PutUint32(data[rowAt:], handle)
		binary.LittleEndian.PutUint32(data[rowAt+4:], 1)
		binary.LittleEndian.PutUint32(data[rowAt+8:], uint32(index+1))
	}
	anchor := equippedSpellsSlotDataBase + anchorAt
	copy(data[anchor:], equippedSpellsFixtureAnchor)
	armamentsAt := anchor + 0x931D + 4
	for slot, gameID := range emptyGameIDs {
		handle := uint32(0x90000100 + slot)
		binary.LittleEndian.PutUint32(data[anchor+0x101+int64(slot*4):], 0x180+uint32(slot))
		binary.LittleEndian.PutUint32(data[anchor+0x175+int64(slot*4):], gameID&0x0FFFFFFF)
		binary.LittleEndian.PutUint32(data[anchor+0x1CD+int64(slot*4):], handle)
		binary.LittleEndian.PutUint32(data[armamentsAt+int64((12+slot)*4):], gameID)
	}

	path := filepath.Join(t.TempDir(), "set-equipped-armor-route.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestSetEquippedArmorRoute(t *testing.T) {
	saveEngine := saveengine.New()
	gameCatalog := newFullCatalog(t)
	session, err := savesession.LoadSave(saveEngine, writeSetEquippedArmorRouteFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := saveEngine.GetInventory(session.SaveSessionID, 0, "common", 1, 50)
	if err != nil || len(inventory.Records) != 8 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
	}
	assignments := make([]*string, 4)
	for slot := range assignments {
		token := inventory.Records[slot+4].OwnedItemID
		assignments[slot] = &token
	}
	body, err := json.Marshal(setEquippedArmorRequest{
		SlotAssignments:  assignments,
		ExpectedRevision: "0",
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	target := fmt.Sprintf(
		"/api/v1/save-sessions/%s/characters/0/equipped-armor", session.SaveSessionID)
	request := httptest.NewRequest(http.MethodPut, target, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}

	var got equipment.SetEquippedArmorResult
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if got.SaveRevision != "1" || got.SlotAssignments[0] == nil ||
		got.SlotAssignments[0].Key != "10009C40" {
		t.Fatalf("result = %+v", got)
	}
	equipped, err := saveEngine.GetEquipment(session.SaveSessionID, 0)
	if err != nil || equipped.Slots[12] != 0x10009C40 || equipped.Slots[15] != 0x10009D6C {
		t.Fatalf("GetEquipment: %v, slots=%#v", err, equipped.Slots)
	}
}

func writeSetEquippedTalismansRouteFixture(t *testing.T) string {
	t.Helper()
	path := writeSetPouchItemsRouteFixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	anchor := equippedSpellsSlotDataBase + equippedSpellsAnchorAt
	for _, blockAt := range []int64{0x115, 0x189, 0x1E1} {
		for index := 0; index < 4; index++ {
			value := uint32(0xFFFFFFFF)
			if blockAt == 0x1E1 {
				value = 0
			}
			binary.LittleEndian.PutUint32(data[anchor+blockAt+int64(index*4):], value)
		}
	}
	countAt := anchor + 0x931D
	armamentsAt := countAt + 4 + 17*8
	for index := 17; index <= 20; index++ {
		binary.LittleEndian.PutUint32(data[armamentsAt+int64(index*4):], 0xFFFFFFFF)
	}
	inventoryAt := anchor + 505
	binary.LittleEndian.PutUint32(data[inventoryAt:], 0xA0000474)
	binary.LittleEndian.PutUint32(data[inventoryAt+4:], 1)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}
	return path
}

func TestSetEquippedTalismansRoute(t *testing.T) {
	saveEngine := saveengine.New()
	gameCatalog := newFullCatalog(t)
	session, err := savesession.LoadSave(
		saveEngine, writeSetEquippedTalismansRouteFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := saveEngine.GetInventory(session.SaveSessionID, 0, "common", 1, 50)
	if err != nil || len(inventory.Records) != 1 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
	}
	body, err := json.Marshal(setEquippedTalismansRequest{
		OrderedOwnedItemIDs: []string{inventory.Records[0].OwnedItemID},
		ExpectedRevision:    "0",
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	target := fmt.Sprintf(
		"/api/v1/save-sessions/%s/characters/0/equipped-talismans", session.SaveSessionID)
	request := httptest.NewRequest(http.MethodPut, target, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}

	var got equipment.SetEquippedTalismansResult
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if got.SaveRevision != "1" || got.UnlockedSlots != 1 ||
		len(got.OrderedResources) != 1 || got.OrderedResources[0].Key != "20000474" {
		t.Fatalf("result = %+v", got)
	}
	equipped, err := saveEngine.GetEquipment(session.SaveSessionID, 0)
	if err != nil || equipped.Slots[17] != 0x20000474 {
		t.Fatalf("GetEquipment: %v, talisman1=0x%08X", err, equipped.Slots[17])
	}
}

func TestSetPouchItemsRoute(t *testing.T) {
	saveEngine := saveengine.New()
	gameCatalog := newFullCatalog(t)
	session, err := savesession.LoadSave(saveEngine, writeSetPouchItemsRouteFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	inv, err := saveEngine.GetInventory(session.SaveSessionID, 0, "common", 1, 50)
	if err != nil || len(inv.Records) == 0 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(inv.Records))
	}
	tok := inv.Records[0].OwnedItemID

	assignments := make([]*string, 6)
	assignments[0] = &tok

	body, err := json.Marshal(setPouchItemsRequest{
		SlotAssignments:  assignments,
		ExpectedRevision: "0",
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	url := fmt.Sprintf("/api/v1/save-sessions/%s/characters/0/pouch-items", session.SaveSessionID)
	request := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}

	var got equipment.SetPouchItemsResult
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if got.SaveSessionID != session.SaveSessionID || got.SaveRevision != "1" || got.CharacterID != 0 {
		t.Fatalf("result header = %+v", got)
	}
	if got.SlotAssignments[0] == nil || got.SlotAssignments[0].Key != "400006A4" {
		t.Errorf("slot 0 = %+v, want key 400006A4", got.SlotAssignments[0])
	}
	for i := 1; i < 6; i++ {
		if got.SlotAssignments[i] != nil {
			t.Errorf("slot %d = %+v, want nil", i, got.SlotAssignments[i])
		}
	}

	pouchItems, err := equipment.GetPouchItems(saveEngine, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("GetPouchItems: %v", err)
	}
	if pouchState := pouchItems.Items[0]; pouchState.ItemID == 0 || pouchState.EquipIndex < 0x180 {
		t.Errorf("GetPouchItems slot 0 = %+v, want valid item and index", pouchState)
	}
}

// newFullCatalog builds a catalog from the embedded catalog data. The prototype
// catalog carries no spell document, so it could not resolve the equipped spell.
func newFullCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("loader.LoadFS: %v", err)
	}
	gameCatalog, err := gamecatalog.New(data.Manifest, data.Resources())
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}
	return gameCatalog
}

func TestEquippedSpellsRouteMatchesTheGetter(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0"
	gameCatalog := newFullCatalog(t)

	want, err := equipment.GetEquippedSpells(saveEngine, gameCatalog, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("equipment.GetEquippedSpells: %v", err)
	}
	if !want.Active {
		t.Fatal("the fixture slot is inactive, so the route would prove no catalog resolution")
	}
	wantFirst := equipment.EquippedSpellSlot{
		RawMagicParamID: rawGlintstonePebble,
		ResourceKey:     "40000FA0",
		Name:            "Glintstone Pebble",
		MemorySlots:     1,
	}
	if want.Spells[0] != wantFirst {
		t.Fatalf("first record = %+v, want %+v", want.Spells[0], wantFirst)
	}

	// The route has to run against the same catalog, so it is served by a
	// handler built here instead of by the shared prototype-catalog helper.
	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, base+"/equipped-spells", nil))
	assertOK(t, recorder, base+"/equipped-spells")
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("equipped spells route body differs from the GetEquippedSpells result")
	}
}

func TestSetEquippedSpellsRoute(t *testing.T) {
	saveEngine := saveengine.New()
	gameCatalog := newFullCatalog(t)
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	body, err := json.Marshal(setEquippedSpellsRequest{
		OrderedResources: []*schema.ResourceRef{
			{Kind: schema.ResourceKindItem, Key: "40000FA0"},
			{Kind: schema.ResourceKindItem, Key: "40000FA1"},
		},
		ExpectedRevision: "0",
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/equipped-spells"
	request := httptest.NewRequest(http.MethodPut, target, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
	assertOK(t, recorder, target)

	var got equipment.SetEquippedSpellsResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.SaveRevision != "1" || got.CharacterID != 0 {
		t.Fatalf("result = %+v, want revision 1 and character 0", got)
	}
	if len(got.OrderedResources) != 2 ||
		got.OrderedResources[0].Key != "40000FA0" ||
		got.OrderedResources[1].Key != "40000FA1" {
		t.Fatalf("orderedResources = %+v, want 40000FA0 and 40000FA1", got.OrderedResources)
	}

	spells, err := equipment.GetEquippedSpells(saveEngine, gameCatalog, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("GetEquippedSpells: %v", err)
	}
	if len(spells.Spells) != 12 ||
		spells.Spells[0].ResourceKey != "40000FA0" ||
		spells.Spells[1].ResourceKey != "40000FA1" ||
		spells.Spells[2].ResourceKey != "" {
		t.Errorf("stored spells = %+v, want compact loadout with pebble and swift shard", spells.Spells[:3])
	}
}

func TestEquippedSpellsRouteRejectsAMalformedCharacterID(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writePCFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}

	for _, raw := range []string{"one", " 0", "0x1"} {
		target := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/" + url.PathEscape(raw) + "/equipped-spells"
		if recorder := doSave(t, saveEngine, http.MethodGet, target, ""); recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %q)", target, recorder.Code, recorder.Body.String())
		}
	}
}

func TestEquippedSpellsRouteIsAbsentWithoutAnEngine(t *testing.T) {
	target := "/api/v1/save-sessions/any-session/characters/0/equipped-spells"
	recorder := doSave(t, nil, http.MethodGet, target, "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("%s: status = %d, want 404 (body %q)", target, recorder.Code, recorder.Body.String())
	}
}

func TestEquippedSpellsRouteIsDescribedInTheOpenAPIDocument(t *testing.T) {
	recorder := do(t, newPrototypeCatalog(t), "/openapi.json")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var document struct {
		Paths map[string]map[string]any `json:"paths"`
		Comps struct {
			Schemas map[string]any `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode openapi.json: %v", err)
	}

	const path = "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-spells"
	operation, exists := document.Paths[path]
	if !exists {
		t.Fatalf("openapi.json does not describe %s", path)
	}
	if _, hasGet := operation["get"]; !hasGet {
		t.Fatalf("openapi.json describes %s without a GET operation", path)
	}
	if _, hasPut := operation["put"]; !hasPut {
		t.Fatalf("openapi.json describes %s without a PUT operation", path)
	}
	for _, name := range []string{"EquippedSpellSlot", "CharacterEquippedSpells", "SetEquippedSpellsRequest", "SetEquippedSpellsResult"} {
		if _, exists := document.Comps.Schemas[name]; !exists {
			t.Fatalf("openapi.json is missing the %s schema", name)
		}
	}
}

// The network-settings route reads the regulation of one session, so it needs a
// fixture that actually carries UserData11. It is built here from the documented
// format rules and written into t.TempDir(); no real save is involved.
const (
	networkSettingsUserData11Offset = int64(pcHeaderSize) + 10*0x280010 + 0x60010
	networkSettingsRowSize          = 0x24C
)

var networkSettingsRegulationKey = []byte{
	0x99, 0xBF, 0xFC, 0x36, 0x6A, 0x6B, 0xC8, 0xC6,
	0xF5, 0x82, 0x7D, 0x09, 0x36, 0x02, 0xD6, 0x76,
	0xC4, 0x28, 0x92, 0xA0, 0x1C, 0x20, 0x7F, 0xB0,
	0x24, 0xD3, 0xAF, 0x4E, 0x49, 0x3F, 0xEF, 0x99,
}

func writeNetworkSettingsFixture(t *testing.T) string {
	t.Helper()

	data := make([]byte, networkSettingsUserData11Offset)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[pcEntryCountOffset:], pcEntryCount)

	userData11 := make([]byte, 0x20)
	copy(userData11[0x10:], []byte{0x20, 0x47, 0x45, 0x52})
	data = append(data, append(userData11, networkSettingsRegulation(t)...)...)

	path := filepath.Join(t.TempDir(), "network-settings.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// networkSettingsRegulation builds the encrypted DFLT regulation blob holding one
// NetworkParam.param whose row 0 carries the values the route must report.
func networkSettingsRegulation(t *testing.T) []byte {
	t.Helper()

	bnd4 := networkSettingsBND4(t)

	var payload bytes.Buffer
	payload.Write([]byte{0x78, 0x01})
	writer, err := flate.NewWriter(&payload, flate.BestSpeed)
	if err != nil {
		t.Fatalf("flate.NewWriter: %v", err)
	}
	if _, err := writer.Write(bnd4); err != nil {
		t.Fatalf("deflate: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close deflate: %v", err)
	}

	archive := make([]byte, 76, 76+payload.Len())
	copy(archive, []byte("DCX\x00"))
	binary.BigEndian.PutUint32(archive[28:], uint32(len(bnd4)))
	binary.BigEndian.PutUint32(archive[32:], uint32(payload.Len()))
	copy(archive[40:], []byte("DFLT"))
	archive = append(archive, payload.Bytes()...)

	plaintext := make([]byte, (len(archive)+aes.BlockSize-1)/aes.BlockSize*aes.BlockSize)
	copy(plaintext, archive)

	iv := bytes.Repeat([]byte{0x5A}, aes.BlockSize)
	block, err := aes.NewCipher(networkSettingsRegulationKey)
	if err != nil {
		t.Fatalf("aes.NewCipher: %v", err)
	}
	ciphertext := make([]byte, len(plaintext))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, plaintext)
	return append(append([]byte{}, iv...), ciphertext...)
}

func networkSettingsBND4(t *testing.T) []byte {
	t.Helper()

	param := networkSettingsParamFile()

	units := utf16.Encode([]rune(`N:\GR\data\Param\param\GameParam\NetworkParam.param`))
	name := make([]byte, len(units)*2)
	for index, unit := range units {
		binary.LittleEndian.PutUint16(name[index*2:], unit)
	}

	const nameAt = 0x64
	dataAt := (nameAt + len(name) + 2 + 0x0F) &^ 0x0F

	archive := make([]byte, dataAt+len(param))
	copy(archive, []byte("BND4"))
	binary.LittleEndian.PutUint32(archive[0x0C:], 1)
	binary.LittleEndian.PutUint64(archive[0x40+8:], uint64(len(param)))
	binary.LittleEndian.PutUint32(archive[0x40+24:], uint32(dataAt))
	binary.LittleEndian.PutUint32(archive[0x40+32:], uint32(nameAt))
	copy(archive[nameAt:], name)
	copy(archive[dataAt:], param)
	return archive
}

// networkSettingsParamFile writes all 22 distinct values at their confirmed row
// offsets, so a shifted or defaulted parameter set cannot pass the route test.
func networkSettingsParamFile() []byte {
	const rowAt = 0x60
	param := make([]byte, rowAt+networkSettingsRowSize)
	param[0x2D] = 0x04
	binary.LittleEndian.PutUint64(param[0x48:], rowAt)

	row := param[rowAt:]
	putInt := func(offset int, value int32) {
		binary.LittleEndian.PutUint32(row[offset:], uint32(value))
	}
	putFloat := func(offset int, value float32) {
		binary.LittleEndian.PutUint32(row[offset:], math.Float32bits(value))
	}

	putFloat(0x08, 15.75)
	putFloat(0x1C, 16.5)
	putInt(0x20, 17)
	putInt(0x24, 18)
	putFloat(0x28, 19.25)
	putInt(0x60, 20)
	putFloat(0x64, 21.5)
	putFloat(0x68, 22.75)
	putInt(0x70, 11)
	putFloat(0x74, 12.5)
	putFloat(0x78, 13.25)
	putInt(0x7C, 14)
	putFloat(0x180, 23.5)
	putInt(0x184, 24)
	putInt(0x18C, 25)
	putFloat(0x190, 26.25)
	putFloat(0x194, 27.75)
	row[0x1D8] = 28
	row[0x1D9] = 29
	putInt(0x240, 30)
	putFloat(0x244, 31.5)
	putFloat(0x248, 32.25)
	return param
}

// The route is the loopback-only transport of GetNetworkSettings: with an engine
// it must answer 200 with exactly the stored values, and without one — the
// -allow-external-bind mode — it must not exist at all.
func TestNetworkSettingsRoute(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeNetworkSettingsFixture(t), "pc")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/network-settings"

	want := network.GetNetworkSettingsResult{
		SaveSessionID: session.SaveSessionID,
		Parameters: gamecatalog.NetworkParamValues{
			MaxBreakInTargetListCount:     11,
			BreakInRequestIntervalTimeSec: 12.5,
			BreakInRequestTimeOutSec:      13.25,
			BreakInRequestAreaCount:       14,

			SummonTimeoutTime: 15.75,

			ReloadSignIntervalTime2: 16.5,
			ReloadSignTotalCount:    17,
			ReloadSignCellCount:     18,
			UpdateSignIntervalTime:  19.25,
			SingGetMax:              20,
			SignDownloadSpan:        21.5,
			SignUpdateSpan:          22.75,

			ReloadVisitListCoolTime:   23.5,
			MaxCoopBlueSummonCount:    24,
			MaxVisitListCount:         25,
			ReloadSearchCoopBlueMin:   26.25,
			ReloadSearchCoopBlueMax:   27.75,
			AllAreaSearchRateCoopBlue: 28,
			AllAreaSearchRateVsBlue:   29,

			VisitorListMax:      30,
			VisitorTimeOutTime:  31.5,
			VisitorDownloadSpan: 32.25,
		},
	}

	recorder := doSave(t, saveEngine, http.MethodGet, target, "")
	assertOK(t, recorder, target)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatalf("network settings body %q differs from the expected values", recorder.Body.String())
	}

	settings := gamecatalog.NetworkParamValues{
		MaxBreakInTargetListCount:     8,
		BreakInRequestIntervalTimeSec: 12,
		BreakInRequestTimeOutSec:      8,
		BreakInRequestAreaCount:       8,
		SummonTimeoutTime:             45,
		ReloadSignIntervalTime2:       20,
		ReloadSignTotalCount:          40,
		ReloadSignCellCount:           20,
		UpdateSignIntervalTime:        15,
		SingGetMax:                    64,
		SignDownloadSpan:              15,
		SignUpdateSpan:                20,
		ReloadVisitListCoolTime:       8,
		MaxCoopBlueSummonCount:        2,
		MaxVisitListCount:             10,
		ReloadSearchCoopBlueMin:       10,
		ReloadSearchCoopBlueMax:       40,
		AllAreaSearchRateCoopBlue:     60,
		AllAreaSearchRateVsBlue:       30,
		VisitorListMax:                10,
		VisitorTimeOutTime:            60,
		VisitorDownloadSpan:           60,
	}
	body, err := json.Marshal(setNetworkSettingsRequest{
		NetworkSettings:  settings,
		ExpectedRevision: "0",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	updated := doSave(t, saveEngine, http.MethodPut, target, string(body))
	assertOK(t, updated, target)
	wantSet := network.SetNetworkSettingsResult{
		SaveSessionID:   session.SaveSessionID,
		SaveRevision:    "1",
		NetworkSettings: settings,
	}
	if !reflect.DeepEqual(decode(t, updated.Body.Bytes()), marshalled(t, wantSet)) {
		t.Fatalf("set network settings body %q differs from the expected result", updated.Body.String())
	}

	stored := doSave(t, saveEngine, http.MethodGet, target, "")
	assertOK(t, stored, target)
	if !reflect.DeepEqual(decode(t, stored.Body.Bytes()), marshalled(t, network.GetNetworkSettingsResult{
		SaveSessionID: session.SaveSessionID,
		Parameters:    settings,
	})) {
		t.Fatalf("stored network settings body %q differs from the committed values", stored.Body.String())
	}

	presets, err := network.GetNetworkPresets(newPrototypeCatalog(t), "faster-reds")
	if err != nil {
		t.Fatalf("GetNetworkPresets: %v", err)
	}
	presetBody, err := json.Marshal(applyNetworkPresetRequest{
		PresetID:         "faster-reds",
		ExpectedRevision: "1",
	})
	if err != nil {
		t.Fatalf("marshal preset request: %v", err)
	}
	presetTarget := target + "/preset"
	applied := doSave(t, saveEngine, http.MethodPut, presetTarget, string(presetBody))
	assertOK(t, applied, presetTarget)
	preset := presets.Presets[0]
	wantApplied := network.ApplyNetworkPresetResult{
		SaveSessionID:   session.SaveSessionID,
		SaveRevision:    "2",
		PresetID:        preset.ID,
		NetworkSettings: preset.Parameters,
	}
	if !reflect.DeepEqual(decode(t, applied.Body.Bytes()), marshalled(t, wantApplied)) {
		t.Fatalf("apply network preset body %q differs from the expected result", applied.Body.String())
	}

	stored = doSave(t, saveEngine, http.MethodGet, target, "")
	assertOK(t, stored, target)
	if !reflect.DeepEqual(decode(t, stored.Body.Bytes()), marshalled(t, network.GetNetworkSettingsResult{
		SaveSessionID: session.SaveSessionID,
		Parameters:    preset.Parameters,
	})) {
		t.Fatalf("stored network settings body %q differs from the applied preset", stored.Body.String())
	}

	absent := doSave(t, nil, http.MethodGet, target, "")
	if absent.Code != http.StatusNotFound {
		t.Fatalf("%s without an engine: status = %d, want 404 (body %q)",
			target, absent.Code, absent.Body.String())
	}
	absent = doSave(t, nil, http.MethodPut, target, string(body))
	if absent.Code != http.StatusNotFound {
		t.Fatalf("PUT %s without an engine: status = %d, want 404 (body %q)",
			target, absent.Code, absent.Body.String())
	}
	absent = doSave(t, nil, http.MethodPut, presetTarget, string(presetBody))
	if absent.Code != http.StatusNotFound {
		t.Fatalf("PUT %s without an engine: status = %d, want 404 (body %q)",
			presetTarget, absent.Code, absent.Body.String())
	}
}

// The inventory route is the one save-session route with a section filter and
// paging, so its fixture carries real records: two common and one key. The
// offsets are restated here instead of reused from SaveEngine.
const (
	inventoryRouteSlotDataBase = int64(pcHeaderSize) + 0x10
	inventoryRouteUserData10   = int64(pcHeaderSize) + 10*0x280010 + 0x10
	inventoryRouteFlagsOffset  = 0x1954
	inventoryRouteAnchorAt     = 0xA03A
	inventoryRouteCommonAt     = 505
	inventoryRouteKeyAt        = inventoryRouteCommonAt + 0xA80*12 + 4
)

// writeInventoryFixture writes a synthetic PC save into t.TempDir() whose slot 0
// is active and holds three non-empty InventoryHeld records.
func writeInventoryFixture(t *testing.T) string {
	t.Helper()

	data := make([]byte, pcFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[pcEntryCountOffset:], pcEntryCount)
	data[inventoryRouteUserData10+inventoryRouteFlagsOffset] = 1

	anchorBase := inventoryRouteSlotDataBase + inventoryRouteAnchorAt
	copy(data[anchorBase:], equippedSpellsFixtureAnchor)
	binary.LittleEndian.PutUint32(data[inventoryRouteSlotDataBase:], 82)
	binary.LittleEndian.PutUint32(data[inventoryRouteSlotDataBase+0x20:], 0xB000272E)
	binary.LittleEndian.PutUint32(data[inventoryRouteSlotDataBase+0x24:], 0x000F4240)
	binary.LittleEndian.PutUint32(data[inventoryRouteSlotDataBase+0x35:], 0x90001111)
	binary.LittleEndian.PutUint32(data[inventoryRouteSlotDataBase+0x39:], 0x000F4240)
	binary.LittleEndian.PutUint32(data[inventoryRouteSlotDataBase+0x4A:], 0xC0000001)
	binary.LittleEndian.PutUint32(data[inventoryRouteSlotDataBase+0x4E:], 0x8000EA60)

	record := func(at int64, handle, quantity, acquisition uint32) {
		binary.LittleEndian.PutUint32(data[anchorBase+at:], handle)
		binary.LittleEndian.PutUint32(data[anchorBase+at+4:], quantity)
		binary.LittleEndian.PutUint32(data[anchorBase+at+8:], acquisition)
	}
	record(inventoryRouteCommonAt, 0xB000272E, 0x80000003, 7)
	record(inventoryRouteCommonAt+5*12, 0x90001111, 1, 9)
	record(inventoryRouteKeyAt+2*12, 0xC0000001, 0x80000001, 12)

	path := filepath.Join(t.TempDir(), "inventory.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestInventoryRoute(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeInventoryFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/inventory"
	gameCatalog := newPrototypeCatalog(t)

	want, err := inventory.GetInventory(saveEngine, gameCatalog, session.SaveSessionID, 0, "", 0, 0)
	if err != nil {
		t.Fatalf("inventory.GetInventory: %v", err)
	}
	if !want.Active || want.Total != 3 {
		t.Fatalf("fixture result = %+v, want an active slot with three records", want)
	}
	recorder := doSave(t, saveEngine, http.MethodGet, base, "")
	assertOK(t, recorder, base)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("inventory route body differs from the GetInventory result")
	}

	// The section filter and both paging values have to reach the getter.
	target := base + "?containerSection=key&page=1&pageSize=1"
	wantKey, err := inventory.GetInventory(saveEngine, gameCatalog, session.SaveSessionID, 0, "key", 1, 1)
	if err != nil {
		t.Fatalf("inventory.GetInventory(key): %v", err)
	}
	if len(wantKey.Records) != 1 || wantKey.Records[0].ContainerSection != "key" {
		t.Fatalf("key page = %+v, want the single key record", wantKey)
	}
	filtered := doSave(t, saveEngine, http.MethodGet, target, "")
	assertOK(t, filtered, target)
	if !reflect.DeepEqual(decode(t, filtered.Body.Bytes()), marshalled(t, wantKey)) {
		t.Fatal("filtered inventory route body differs from the GetInventory result")
	}

	for _, query := range []string{"?containerSection=Common", "?containerSection=%20key", "?page=-1", "?page=x"} {
		rejected := doSave(t, saveEngine, http.MethodGet, base+query, "")
		if rejected.Code != http.StatusBadRequest {
			t.Fatalf("%s%s: status = %d, want 400 (body %q)", base, query, rejected.Code, rejected.Body.String())
		}
	}

	absent := doSave(t, nil, http.MethodGet,
		"/api/v1/save-sessions/any-session/characters/0/inventory", "")
	if absent.Code != http.StatusNotFound {
		t.Fatalf("inventory route without an engine: status = %d, want 404 (body %q)",
			absent.Code, absent.Body.String())
	}
}

func TestSetInventoryOrderRoute(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeInventoryFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	listed, err := saveEngine.GetInventory(
		session.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	ownedItemIDs := make(map[int]string)
	for _, record := range listed.Records {
		if record.PhysicalIndex == 0 || record.PhysicalIndex == 5 {
			ownedItemIDs[record.PhysicalIndex] = record.OwnedItemID
		}
	}
	if len(ownedItemIDs) != 2 {
		t.Fatalf("fixture has %d supported Dagger records, want two", len(ownedItemIDs))
	}

	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/inventory/order"
	serve := func(body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPut, target, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(newFullCatalog(t), testApplicationVersion, saveEngine).
			ServeHTTP(recorder, request)
		return recorder
	}

	for name, body := range map[string]string{
		"missing order": `{"expectedRevision":"0"}`,
		"unknown field": `{"orderedOwnedItemIDs":[],"expectedRevision":"0","unknown":true}`,
	} {
		recorder := serve(body)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %q)",
				name, recorder.Code, recorder.Body.String())
		}
	}

	body, err := json.Marshal(setInventoryOrderRequest{
		OrderedOwnedItemIDs: []string{ownedItemIDs[5], ownedItemIDs[0]},
		ExpectedRevision:    "0",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	recorder := serve(string(body))
	assertOK(t, recorder, target)
	var result inventory.SetInventoryOrderResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode SetInventoryOrder body %q: %v", recorder.Body.String(), err)
	}
	if result.SaveSessionID != session.SaveSessionID || result.SaveRevision != "1" ||
		result.CharacterID != 0 || len(result.OrderedResources) != 2 ||
		result.OrderedResources[0].Key != daggerResourceKey ||
		result.OrderedResources[1].Key != daggerResourceKey ||
		!reflect.DeepEqual(result.AcquisitionIndices, []uint32{434, 436}) {
		t.Fatalf("SetInventoryOrder result = %+v", result)
	}

	updated, err := saveEngine.GetInventory(
		session.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory after reorder: %v", err)
	}
	wantIndices := map[int]uint32{5: 434, 0: 436}
	for _, record := range updated.Records {
		if want, exists := wantIndices[record.PhysicalIndex]; exists &&
			record.AcquisitionIndex != want {
			t.Fatalf("Dagger at physical index %d has acquisition index %d, want %d",
				record.PhysicalIndex, record.AcquisitionIndex, want)
		}
	}
}

// The owned-item route addresses one instance instead of a page, so it is
// verified on the same fixture the inventory route uses: the identifier can only
// come from a container read of that very session.
func TestOwnedItemRoute(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeInventoryFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newPrototypeCatalog(t)

	listed, err := inventory.GetInventory(saveEngine, gameCatalog, session.SaveSessionID, 0, "", 0, 0)
	if err != nil {
		t.Fatalf("inventory.GetInventory: %v", err)
	}
	if len(listed.Records) != 3 {
		t.Fatalf("fixture listed %d records, want 3", len(listed.Records))
	}

	// Every path parameter has to reach the getter, so each of the three records
	// is fetched through the route by its own identifier.
	for _, record := range listed.Records {
		target := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/0/owned-items/" + url.PathEscape(record.OwnedItemID)
		want, err := inventory.GetOwnedItem(
			saveEngine, gameCatalog, session.SaveSessionID, 0, record.OwnedItemID)
		if err != nil {
			t.Fatalf("inventory.GetOwnedItem: %v", err)
		}
		recorder := doSave(t, saveEngine, http.MethodGet, target, "")
		assertOK(t, recorder, target)
		if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
			t.Fatalf("owned-item route body differs from the GetOwnedItem result for %s", target)
		}
	}

	// The identifier is never trimmed or reconstructed by the transport, and a
	// slot the token was not minted for is rejected rather than resolved.
	base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/"
	for _, target := range []string{
		base + "0/owned-items/" + url.PathEscape(" "+listed.Records[0].OwnedItemID),
		base + "0/owned-items/" + url.PathEscape(listed.Records[0].OwnedItemID+"0"),
		base + "1/owned-items/" + url.PathEscape(listed.Records[0].OwnedItemID),
		base + "one/owned-items/" + url.PathEscape(listed.Records[0].OwnedItemID),
	} {
		rejected := doSave(t, saveEngine, http.MethodGet, target, "")
		if rejected.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %q)", target, rejected.Code, rejected.Body.String())
		}
	}

	absent := doSave(t, nil, http.MethodGet,
		"/api/v1/save-sessions/any-session/characters/0/owned-items/any-token", "")
	if absent.Code != http.StatusNotFound {
		t.Fatalf("owned-item route without an engine: status = %d, want 404 (body %q)",
			absent.Code, absent.Body.String())
	}
}

// The removal route shares its path with the owned-item getter, so the method
// is what separates reading an instance from deleting it. The route owns the
// typed envelope only: the identity, the revision and the plan belong to the
// endpoint below it.
func TestRemoveOwnedItemRoute(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeInventoryFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newPrototypeCatalog(t)

	listed, err := inventory.GetInventory(saveEngine, gameCatalog, session.SaveSessionID, 0, "", 0, 0)
	if err != nil {
		t.Fatalf("inventory.GetInventory: %v", err)
	}
	if len(listed.Records) != 3 {
		t.Fatalf("fixture listed %d records, want 3", len(listed.Records))
	}
	ownedItemID := listed.Records[0].OwnedItemID
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/owned-items/" + url.PathEscape(ownedItemID)

	// A body with an unknown field, a non-decimal characterID and a stale
	// revision are all refused before anything changes.
	for _, rejected := range []struct {
		target string
		body   string
	}{
		{target, `{"expectedRevision":"0","unknown":true}`},
		{target, `{"expectedRevision":"7"}`},
		{"/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/one/owned-items/" + url.PathEscape(ownedItemID), `{"expectedRevision":"0"}`},
	} {
		recorder := doSave(t, saveEngine, http.MethodDelete, rejected.target, rejected.body)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s %q: status = %d, want 400 (body %q)",
				rejected.target, rejected.body, recorder.Code, recorder.Body.String())
		}
	}
	info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo after the rejected requests: %v", err)
	}
	if info.UnsavedChanges {
		t.Fatal("a rejected request changed the session")
	}

	recorder := doSave(t, saveEngine, http.MethodDelete, target, `{"expectedRevision":"0"}`)
	assertOK(t, recorder, target)
	var result inventory.RemoveOwnedItemResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode RemoveOwnedItem body %q: %v", recorder.Body.String(), err)
	}
	want := inventory.RemoveOwnedItemResult{
		SaveSessionID: session.SaveSessionID,
		SaveRevision:  "1",
		OwnedItemID:   ownedItemID,
		CharacterID:   0,
		GameID:        listed.Records[0].GameID,
	}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("RemoveOwnedItem result = %+v, want %+v", result, want)
	}

	updated, err := inventory.GetInventory(saveEngine, gameCatalog, session.SaveSessionID, 0, "", 0, 0)
	if err != nil {
		t.Fatalf("inventory.GetInventory after the removal: %v", err)
	}
	if updated.SaveRevision != "1" || len(updated.Records) != 2 {
		t.Fatalf("updated inventory = %+v, want two records at revision 1", updated)
	}
	// The identity is retired with the revision, so repeating the same request
	// removes nothing a second time.
	repeated := doSave(t, saveEngine, http.MethodDelete, target, `{"expectedRevision":"1"}`)
	if repeated.Code != http.StatusBadRequest {
		t.Fatalf("repeated removal: status = %d, want 400 (body %q)",
			repeated.Code, repeated.Body.String())
	}
}

func TestSetOwnedItemQuantityRoute(t *testing.T) {
	fixture := writeInventoryFixture(t)
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read quantity fixture: %v", err)
	}
	// This route needs one stackable catalog item. Keep the shared fixture
	// unchanged: in this private copy, remove the instance mappings and the two
	// unrelated rows. The remaining goods handle then resolves by its documented
	// type prefix to the stackable Memory Stone 0x4000272E.
	clear(data[inventoryRouteSlotDataBase+0x20 : inventoryRouteSlotDataBase+inventoryRouteAnchorAt])
	clear(data[inventoryRouteSlotDataBase+inventoryRouteAnchorAt+inventoryRouteCommonAt+5*12 : inventoryRouteSlotDataBase+inventoryRouteAnchorAt+inventoryRouteCommonAt+6*12])
	clear(data[inventoryRouteSlotDataBase+inventoryRouteAnchorAt+inventoryRouteKeyAt+2*12 : inventoryRouteSlotDataBase+inventoryRouteAnchorAt+inventoryRouteKeyAt+3*12])
	if err := os.WriteFile(fixture, data, 0o600); err != nil {
		t.Fatalf("write quantity fixture: %v", err)
	}

	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, fixture, "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	listed, err := inventory.GetInventory(saveEngine, gameCatalog, session.SaveSessionID, 0, "", 0, 0)
	if err != nil {
		t.Fatalf("inventory.GetInventory: %v", err)
	}
	if len(listed.Records) == 0 {
		t.Fatal("fixture returned no owned item")
	}
	ownedItemID := listed.Records[0].OwnedItemID
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/owned-items/" + url.PathEscape(ownedItemID) + "/quantity"
	serve := func(method, target, body string) *httptest.ResponseRecorder {
		var reader io.Reader
		if body != "" {
			reader = strings.NewReader(body)
		}
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).
			ServeHTTP(recorder, httptest.NewRequest(method, target, reader))
		return recorder
	}

	malformed := serve(http.MethodPatch, target,
		`{"quantity":4,"expectedRevision":"0","unknown":true}`)
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("strict body: status = %d, want 400 (body %q)", malformed.Code, malformed.Body.String())
	}
	info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo after malformed body: %v", err)
	}
	if info.UnsavedChanges {
		t.Fatal("malformed body changed the session")
	}

	recorder := serve(http.MethodPatch, target,
		`{"quantity":4,"expectedRevision":"0"}`)
	assertOK(t, recorder, target)
	var result inventory.SetOwnedItemQuantityResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode SetOwnedItemQuantity body %q: %v", recorder.Body.String(), err)
	}
	want := inventory.SetOwnedItemQuantityResult{
		SaveSessionID: session.SaveSessionID,
		SaveRevision:  "1",
		OwnedItemID:   ownedItemID,
		CharacterID:   0,
		Quantity:      4,
	}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("SetOwnedItemQuantity result = %+v, want %+v", result, want)
	}

	updated, err := inventory.GetInventory(saveEngine, gameCatalog, session.SaveSessionID, 0, "", 0, 0)
	if err != nil {
		t.Fatalf("inventory.GetInventory after mutation: %v", err)
	}
	if updated.SaveRevision != "1" || updated.Records[0].Quantity != 4 {
		t.Fatalf("updated inventory = %+v, want quantity 4 at revision 1", updated)
	}
}

func TestMoveOwnedItemToStorageRoute(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeInventoryFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	listed, err := inventory.GetInventory(
		saveEngine, gameCatalog, session.SaveSessionID, 0, "common", 0, 0)
	if err != nil {
		t.Fatalf("inventory.GetInventory: %v", err)
	}
	ownedItemID := ""
	for _, record := range listed.Records {
		if record.PhysicalIndex == 5 {
			ownedItemID = record.OwnedItemID
			break
		}
	}
	if ownedItemID == "" {
		t.Fatal("fixture returned no storable weapon record")
	}
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/owned-items/" + url.PathEscape(ownedItemID) + "/move-to-storage"

	malformed := doSave(t, saveEngine, http.MethodPost, target,
		`{"targetPosition":0,"expectedRevision":"0","unknown":true}`)
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("strict body: status = %d, want 400 (body %q)",
			malformed.Code, malformed.Body.String())
	}
	missing := doSave(t, saveEngine, http.MethodPost, target,
		`{"expectedRevision":"0"}`)
	if missing.Code != http.StatusBadRequest ||
		!strings.Contains(missing.Body.String(), "targetPosition is required") {
		t.Fatalf("missing targetPosition: status = %d, body = %q",
			missing.Code, missing.Body.String())
	}
	recorder := doSave(t, saveEngine, http.MethodPost, target,
		`{"targetPosition":0,"expectedRevision":"0"}`)
	assertOK(t, recorder, target)
	var result inventory.MoveOwnedItemToStorageResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode MoveOwnedItemToStorage body %q: %v", recorder.Body.String(), err)
	}
	if result.SaveRevision != "1" || result.OwnedItemID != ownedItemID ||
		result.GameID != 1000000 || result.TargetPosition != 0 || result.PhysicalIndex != 0 {
		t.Fatalf("result = %+v", result)
	}
	stored, err := inventory.GetStorage(
		saveEngine, gameCatalog, session.SaveSessionID, 0, "common", 0, 0)
	if err != nil {
		t.Fatalf("inventory.GetStorage: %v", err)
	}
	if len(stored.Records) != 1 || stored.Records[0].GameID != 1000000 {
		t.Fatalf("Storage records = %+v", stored.Records)
	}
}

func TestMoveOwnedItemToInventoryRoute(t *testing.T) {
	path := writeStorageFixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Storage fixture: %v", err)
	}
	sectionAt := storageRouteSlotDataBase + storageRouteAnchorAt + storageRouteSectionAt
	binary.LittleEndian.PutUint32(data[sectionAt:], 2)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write Storage move fixture: %v", err)
	}

	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, path, "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	listed, err := inventory.GetStorage(
		saveEngine, gameCatalog, session.SaveSessionID, 0, "common", 0, 0)
	if err != nil {
		t.Fatalf("inventory.GetStorage: %v", err)
	}
	ownedItemID := ""
	for _, record := range listed.Records {
		if record.PhysicalIndex == 5 {
			ownedItemID = record.OwnedItemID
			break
		}
	}
	if ownedItemID == "" {
		t.Fatal("fixture returned no movable Storage weapon record")
	}
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/owned-items/" + url.PathEscape(ownedItemID) + "/move-to-inventory"

	malformed := doSave(t, saveEngine, http.MethodPost, target,
		`{"targetPosition":0,"expectedRevision":"0","unknown":true}`)
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("strict body: status = %d, want 400 (body %q)",
			malformed.Code, malformed.Body.String())
	}
	missing := doSave(t, saveEngine, http.MethodPost, target,
		`{"expectedRevision":"0"}`)
	if missing.Code != http.StatusBadRequest ||
		!strings.Contains(missing.Body.String(), "targetPosition is required") {
		t.Fatalf("missing targetPosition: status = %d, body = %q",
			missing.Code, missing.Body.String())
	}
	recorder := doSave(t, saveEngine, http.MethodPost, target,
		`{"targetPosition":0,"expectedRevision":"0"}`)
	assertOK(t, recorder, target)
	var result inventory.MoveOwnedItemToInventoryResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode MoveOwnedItemToInventory body %q: %v", recorder.Body.String(), err)
	}
	if result.SaveRevision != "1" || result.OwnedItemID != ownedItemID ||
		result.GameID != 1000000 || result.TargetPosition != 0 || result.PhysicalIndex != 0 {
		t.Fatalf("result = %+v", result)
	}
	carried, err := inventory.GetInventory(
		saveEngine, gameCatalog, session.SaveSessionID, 0, "common", 0, 0)
	if err != nil {
		t.Fatalf("inventory.GetInventory: %v", err)
	}
	if len(carried.Records) != 1 || carried.Records[0].GameID != 1000000 {
		t.Fatalf("Inventory records = %+v", carried.Records)
	}
}

func TestSetWeaponUpgradeLevelRoute(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(
		saveEngine, writeSetEquippedArmamentsRouteFixture(t), "pc")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	listed, err := saveEngine.GetInventory(
		session.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
	if err != nil || len(listed.Records) < 2 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(listed.Records))
	}
	ownedItemID := listed.Records[1].OwnedItemID
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/owned-items/" + url.PathEscape(ownedItemID) + "/upgrade-level"

	rejected := doSave(t, saveEngine, http.MethodPatch, target,
		`{"expectedRevision":"0"}`)
	if rejected.Code != http.StatusBadRequest ||
		!strings.Contains(rejected.Body.String(), "upgradeLevel is required") {
		t.Fatalf("missing level: status = %d, body = %q",
			rejected.Code, rejected.Body.String())
	}
	recorder := doSave(t, saveEngine, http.MethodPatch, target,
		`{"upgradeLevel":5,"expectedRevision":"0"}`)
	assertOK(t, recorder, target)
	var result inventory.SetWeaponUpgradeLevelResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode SetWeaponUpgradeLevel body %q: %v", recorder.Body.String(), err)
	}
	if result.SaveSessionID != session.SaveSessionID || result.SaveRevision != "1" ||
		result.OwnedItemID != ownedItemID || result.PreviousGameID != 1000000 ||
		result.GameID != 1000005 || result.UpgradeLevel != 5 {
		t.Fatalf("result = %+v", result)
	}
}

func TestSetSpiritAshUpgradeLevelRoute(t *testing.T) {
	path := writeSetEquippedArmamentsRouteFixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	const spiritAshRowAt = equippedSpellsSlotDataBase + 0xA07B + 505 + 12
	binary.LittleEndian.PutUint32(data[spiritAshRowAt:], 0xB0038A44)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, path, "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	listed, err := saveEngine.GetInventory(
		session.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
	if err != nil || len(listed.Records) < 2 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(listed.Records))
	}
	ownedItemID := listed.Records[1].OwnedItemID
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/owned-items/" + url.PathEscape(ownedItemID) +
		"/spirit-ash-upgrade-level"
	gameCatalog := newFullCatalog(t)

	serve := func(body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPatch, target, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).
			ServeHTTP(recorder, request)
		return recorder
	}
	rejected := serve(`{"expectedRevision":"0"}`)
	if rejected.Code != http.StatusBadRequest ||
		!strings.Contains(rejected.Body.String(), "upgradeLevel is required") {
		t.Fatalf("missing level: status = %d, body = %q",
			rejected.Code, rejected.Body.String())
	}
	recorder := serve(`{"upgradeLevel":10,"expectedRevision":"0"}`)
	assertOK(t, recorder, target)
	var result inventory.SetSpiritAshUpgradeLevelResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode SetSpiritAshUpgradeLevel body %q: %v", recorder.Body.String(), err)
	}
	if result.SaveRevision != "1" || result.OwnedItemID != ownedItemID ||
		result.PreviousGameID != 0x40038A44 || result.GameID != 0x40038A4A ||
		result.UpgradeLevel != 10 {
		t.Fatalf("result = %+v", result)
	}
}

func TestSetWeaponInfusionRoute(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(
		saveEngine, writeSetEquippedArmamentsRouteFixture(t), "pc")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	listed, err := saveEngine.GetInventory(
		session.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
	if err != nil || len(listed.Records) < 2 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(listed.Records))
	}
	ownedItemID := listed.Records[1].OwnedItemID
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/owned-items/" + url.PathEscape(ownedItemID) + "/infusion"

	rejected := doSave(t, saveEngine, http.MethodPatch, target,
		`{"expectedRevision":"0"}`)
	if rejected.Code != http.StatusBadRequest ||
		!strings.Contains(rejected.Body.String(), "affinity is required") {
		t.Fatalf("missing affinity: status = %d, body = %q",
			rejected.Code, rejected.Body.String())
	}
	recorder := doSave(t, saveEngine, http.MethodPatch, target,
		`{"affinity":"heavy","expectedRevision":"0"}`)
	assertOK(t, recorder, target)
	var result inventory.SetWeaponInfusionResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode SetWeaponInfusion body %q: %v", recorder.Body.String(), err)
	}
	if result.SaveSessionID != session.SaveSessionID || result.SaveRevision != "1" ||
		result.OwnedItemID != ownedItemID || result.PreviousGameID != 1000000 ||
		result.GameID != 1000100 || result.Affinity != schema.AffinityHeavy ||
		result.UpgradeLevel != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func writeSetWeaponAshOfWarRouteFixture(t *testing.T) string {
	t.Helper()
	path := writeSetEquippedArmamentsRouteFixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	aoWAt := equippedSpellsSlotDataBase + 0x20 + 7*21
	binary.LittleEndian.PutUint32(data[aoWAt:], 0xC0000200)
	binary.LittleEndian.PutUint32(data[aoWAt+4:], 0x8000EA60)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestSetWeaponAshOfWarRoute(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(
		saveEngine, writeSetWeaponAshOfWarRouteFixture(t), "pc")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	listed, err := saveEngine.GetInventory(
		session.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
	if err != nil || len(listed.Records) < 2 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(listed.Records))
	}
	ownedItemID := listed.Records[1].OwnedItemID
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/owned-items/" + url.PathEscape(ownedItemID) + "/ash-of-war"

	rejected := doSave(t, saveEngine, http.MethodPatch, target,
		`{"ashOfWarKind":"item","expectedRevision":"0"}`)
	if rejected.Code != http.StatusBadRequest ||
		!strings.Contains(rejected.Body.String(), "ashOfWarKey is required") {
		t.Fatalf("missing key: status = %d, body = %q",
			rejected.Code, rejected.Body.String())
	}
	recorder := doSave(t, saveEngine, http.MethodPatch, target,
		`{"ashOfWarKind":"item","ashOfWarKey":"8000EA60","expectedRevision":"0"}`)
	assertOK(t, recorder, target)
	var result inventory.SetWeaponAshOfWarResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode SetWeaponAshOfWar body %q: %v", recorder.Body.String(), err)
	}
	if result.SaveRevision != "1" || result.WeaponOwnedItemID != ownedItemID ||
		result.WeaponGameID != 0x000F4240 || result.PreviousAshOfWarGameID != 0 ||
		result.AshOfWarGameID != 0x8000EA60 {
		t.Fatalf("result = %+v", result)
	}
}

func TestGetItemCapacityRoute(t *testing.T) {
	fixture := writeInventoryFixture(t)
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, fixture, "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	base := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/item-capacity"
	target := base + "?destination=inventory&kind=item&key=400006A4&quantity=3"

	want, err := inventory.GetItemCapacity(
		saveEngine, gameCatalog, session.SaveSessionID, 0, "inventory",
		"item", "400006A4", nil, 3)
	if err != nil {
		t.Fatalf("inventory.GetItemCapacity: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
	assertOK(t, recorder, target)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatalf("item-capacity route body differs from endpoint result: %s", recorder.Body.String())
	}
	info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges || want.SaveRevision != "0" {
		t.Fatalf("capacity route changed session state: result=%+v info=%+v", want, info)
	}

	for _, malformed := range []string{
		base + "?destination=inventory&kind=item&key=400006A4",
		base + "?destination=inventory&kind=item&key=400006A4&quantity=1&variantID=not-a-number",
	} {
		request := httptest.NewRequest(http.MethodGet, malformed, nil)
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %q)",
				malformed, recorder.Code, recorder.Body.String())
		}
	}
}

func TestAddItemToInventoryRoute(t *testing.T) {
	// This route needs one addable catalog item, which the Throwing Dagger is: a
	// goods resource outside category key_items that stacks up to 40 per record
	// and per inventory. The shared fixture stays unchanged; nothing in it holds
	// that item, so the add opens a new record.
	fixture := writeInventoryFixture(t)
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, fixture, "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	before, err := inventory.GetInventory(saveEngine, gameCatalog, session.SaveSessionID, 0, "", 0, 0)
	if err != nil {
		t.Fatalf("inventory.GetInventory: %v", err)
	}

	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/inventory/items"
	serve := func(method, target, body string) *httptest.ResponseRecorder {
		var reader io.Reader
		if body != "" {
			reader = strings.NewReader(body)
		}
		request := httptest.NewRequest(method, target, reader)
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
		return recorder
	}

	// A session-mutating POST that arrives as a CORS simple request performs no
	// operation at all.
	for _, contentType := range []string{"text/plain", ""} {
		request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(
			`{"kind":"item","key":"400006A4","quantity":3,"expectedRevision":"0"}`))
		if contentType != "" {
			request.Header.Set("Content-Type", contentType)
		} else {
			request.Header.Del("Content-Type")
		}
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("add with Content-Type %q: status = %d, want 400 (body %q)",
				contentType, recorder.Code, recorder.Body.String())
		}
	}

	malformed := serve(http.MethodPost, target,
		`{"kind":"item","key":"400006A4","quantity":3,"expectedRevision":"0","unknown":true}`)
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("strict body: status = %d, want 400 (body %q)", malformed.Code, malformed.Body.String())
	}
	info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo after malformed body: %v", err)
	}
	if info.UnsavedChanges {
		t.Fatal("malformed body changed the session")
	}

	recorder := serve(http.MethodPost, target,
		`{"kind":"item","key":"400006A4","quantity":3,"expectedRevision":"0"}`)
	assertOK(t, recorder, target)
	var result inventory.AddItemToInventoryResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode AddItemToInventory body %q: %v", recorder.Body.String(), err)
	}
	want := inventory.AddItemToInventoryResult{
		SaveSessionID:    session.SaveSessionID,
		SaveRevision:     "1",
		CharacterID:      0,
		GameID:           0x400006A4,
		Added:            3,
		Quantity:         3,
		CreatedRecord:    true,
		ContainerSection: "common",
		PhysicalIndex:    result.PhysicalIndex,
	}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("AddItemToInventory result = %+v, want %+v", result, want)
	}

	after, err := inventory.GetInventory(saveEngine, gameCatalog, session.SaveSessionID, 0, "", 0, 0)
	if err != nil {
		t.Fatalf("inventory.GetInventory after the add: %v", err)
	}
	if after.SaveRevision != "1" || after.Total != before.Total+1 {
		t.Fatalf("updated inventory = %+v, want %d records at revision 1", after, before.Total+1)
	}

	// The revision has advanced, so repeating the same request is rejected
	// instead of adding a second record.
	repeated := serve(http.MethodPost, target,
		`{"kind":"item","key":"400006A4","quantity":3,"expectedRevision":"0"}`)
	if repeated.Code != http.StatusBadRequest {
		t.Fatalf("repeated add: status = %d, want 400 (body %q)",
			repeated.Code, repeated.Body.String())
	}
}

func TestAddItemToStorageRoute(t *testing.T) {
	fixture := writeInventoryFixture(t)
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, fixture, "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/storage/items"
	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(
		`{"kind":"item","key":"400006A4","quantity":3,"expectedRevision":"0"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
	assertOK(t, recorder, target)

	var result inventory.AddItemToStorageResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode AddItemToStorage body %q: %v", recorder.Body.String(), err)
	}
	if result.SaveRevision != "1" || result.GameID != 0x400006A4 ||
		result.Added != 3 || result.Quantity != 3 || !result.CreatedRecord ||
		result.ContainerSection != saveengine.StorageSectionCommon {
		t.Fatalf("AddItemToStorage result = %+v", result)
	}
	stored, err := inventory.GetStorage(
		saveEngine, gameCatalog, session.SaveSessionID, 0, saveengine.StorageSectionCommon, 0, 0)
	if err != nil {
		t.Fatalf("inventory.GetStorage: %v", err)
	}
	if stored.SaveRevision != "1" {
		t.Fatalf("Storage revision = %q, want 1", stored.SaveRevision)
	}
	found := false
	for _, record := range stored.Records {
		if record.GameID == 0x400006A4 && record.Quantity == 3 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Storage does not contain the added Throwing Dagger: %+v", stored.Records)
	}
}

// The storage route is the second save-session route with a section filter and
// paging, so its fixture carries real records too: two common and one key. The
// offsets are restated here instead of reused from SaveEngine, including the
// acquired-projectile count the section sits behind.
const (
	storageRouteSlotDataBase = int64(pcHeaderSize) + 0x10
	storageRouteUserData10   = int64(pcHeaderSize) + 10*0x280010 + 0x10
	storageRouteFlagsOffset  = 0x1954
	storageRouteAnchorAt     = 0xA03A
	storageRouteProjCountAt  = 0xD0 + 0x58 + 0x1C + 0x58 + 0x58 + 0x9011 + 0x74 + 0x8C + 0x18
	storageRouteProjectiles  = 4
	storageRouteBlocksBefore = 0x9C + 0x0C + 0x12F
	storageRouteSectionAt    = storageRouteProjCountAt + 4 +
		storageRouteProjectiles*8 + storageRouteBlocksBefore
	storageRouteCommonAt = storageRouteSectionAt + 4
	storageRouteKeyAt    = storageRouteCommonAt + 0x780*12 + 4
)

// writeStorageFixture writes a synthetic PC save into t.TempDir() whose slot 0
// is active and holds three non-empty Storage Box records.
func writeStorageFixture(t *testing.T) string {
	t.Helper()

	data := make([]byte, pcFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[pcEntryCountOffset:], pcEntryCount)
	data[storageRouteUserData10+storageRouteFlagsOffset] = 1

	anchorBase := storageRouteSlotDataBase + storageRouteAnchorAt
	copy(data[anchorBase:], equippedSpellsFixtureAnchor)
	binary.LittleEndian.PutUint32(data[storageRouteSlotDataBase:], 82)
	binary.LittleEndian.PutUint32(data[storageRouteSlotDataBase+0x20:], 0xB000272E)
	binary.LittleEndian.PutUint32(data[storageRouteSlotDataBase+0x24:], 0x000F4240)
	binary.LittleEndian.PutUint32(data[storageRouteSlotDataBase+0x35:], 0x90001111)
	binary.LittleEndian.PutUint32(data[storageRouteSlotDataBase+0x39:], 0x000F4240)
	binary.LittleEndian.PutUint32(data[storageRouteSlotDataBase+0x4A:], 0xC0000001)
	binary.LittleEndian.PutUint32(data[storageRouteSlotDataBase+0x4E:], 0x8000EA60)
	binary.LittleEndian.PutUint32(data[anchorBase+storageRouteProjCountAt:], storageRouteProjectiles)

	record := func(at int64, handle, quantity, acquisition uint32) {
		binary.LittleEndian.PutUint32(data[anchorBase+at:], handle)
		binary.LittleEndian.PutUint32(data[anchorBase+at+4:], quantity)
		binary.LittleEndian.PutUint32(data[anchorBase+at+8:], acquisition)
	}
	record(storageRouteCommonAt, 0xB000272E, 0x80000003, 7)
	record(storageRouteCommonAt+5*12, 0x90001111, 1, 9)
	record(storageRouteKeyAt+2*12, 0xC0000001, 0x80000001, 12)

	path := filepath.Join(t.TempDir(), "storage.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestStorageRoute(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeStorageFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/storage"
	gameCatalog := newPrototypeCatalog(t)

	want, err := inventory.GetStorage(saveEngine, gameCatalog, session.SaveSessionID, 0, "", 0, 0)
	if err != nil {
		t.Fatalf("inventory.GetStorage: %v", err)
	}
	if !want.Active || want.Total != 3 {
		t.Fatalf("fixture result = %+v, want an active slot with three records", want)
	}
	recorder := doSave(t, saveEngine, http.MethodGet, base, "")
	assertOK(t, recorder, base)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("storage route body differs from the GetStorage result")
	}

	// The section filter and both paging values have to reach the getter.
	target := base + "?containerSection=key&page=1&pageSize=1"
	wantKey, err := inventory.GetStorage(saveEngine, gameCatalog, session.SaveSessionID, 0, "key", 1, 1)
	if err != nil {
		t.Fatalf("inventory.GetStorage(key): %v", err)
	}
	if len(wantKey.Records) != 1 || wantKey.Records[0].ContainerSection != "key" {
		t.Fatalf("key page = %+v, want the single key record", wantKey)
	}
	filtered := doSave(t, saveEngine, http.MethodGet, target, "")
	assertOK(t, filtered, target)
	if !reflect.DeepEqual(decode(t, filtered.Body.Bytes()), marshalled(t, wantKey)) {
		t.Fatal("filtered storage route body differs from the GetStorage result")
	}

	for _, query := range []string{"?containerSection=Common", "?containerSection=%20key", "?page=-1", "?page=x"} {
		rejected := doSave(t, saveEngine, http.MethodGet, base+query, "")
		if rejected.Code != http.StatusBadRequest {
			t.Fatalf("%s%s: status = %d, want 400 (body %q)", base, query, rejected.Code, rejected.Body.String())
		}
	}

	// The route must not answer for a malformed characterID either.
	malformed := doSave(t, saveEngine, http.MethodGet,
		"/api/v1/save-sessions/"+session.SaveSessionID+"/characters/one/storage", "")
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("malformed characterID: status = %d, want 400 (body %q)",
			malformed.Code, malformed.Body.String())
	}

	absent := doSave(t, nil, http.MethodGet,
		"/api/v1/save-sessions/any-session/characters/0/storage", "")
	if absent.Code != http.StatusNotFound {
		t.Fatalf("storage route without an engine: status = %d, want 404 (body %q)",
			absent.Code, absent.Body.String())
	}
}

func TestSetStorageOrderRoute(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeStorageFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	listed, err := saveEngine.GetStorage(
		session.SaveSessionID, 0, saveengine.StorageSectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	ownedItemIDs := make(map[int]string)
	for _, record := range listed.Records {
		if record.PhysicalIndex == 0 || record.PhysicalIndex == 5 {
			ownedItemIDs[record.PhysicalIndex] = record.OwnedItemID
		}
	}
	if len(ownedItemIDs) != 2 {
		t.Fatalf("fixture has %d supported Dagger Storage records, want two", len(ownedItemIDs))
	}

	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/storage/order"
	serve := func(body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPut, target, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(newFullCatalog(t), testApplicationVersion, saveEngine).
			ServeHTTP(recorder, request)
		return recorder
	}

	for name, body := range map[string]string{
		"missing order": `{"expectedRevision":"0"}`,
		"unknown field": `{"orderedOwnedItemIDs":[],"expectedRevision":"0","unknown":true}`,
	} {
		recorder := serve(body)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %q)",
				name, recorder.Code, recorder.Body.String())
		}
	}

	body, err := json.Marshal(setStorageOrderRequest{
		OrderedOwnedItemIDs: []string{ownedItemIDs[5], ownedItemIDs[0]},
		ExpectedRevision:    "0",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	recorder := serve(string(body))
	assertOK(t, recorder, target)
	var result inventory.SetStorageOrderResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode SetStorageOrder body %q: %v", recorder.Body.String(), err)
	}
	if result.SaveSessionID != session.SaveSessionID || result.SaveRevision != "1" ||
		result.CharacterID != 0 || len(result.OrderedResources) != 2 ||
		result.OrderedResources[0].Key != daggerResourceKey ||
		result.OrderedResources[1].Key != daggerResourceKey ||
		!reflect.DeepEqual(result.AcquisitionIndices, []uint32{14, 16}) {
		t.Fatalf("SetStorageOrder result = %+v", result)
	}

	updated, err := saveEngine.GetStorage(
		session.SaveSessionID, 0, saveengine.StorageSectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetStorage after reorder: %v", err)
	}
	wantIndices := map[int]uint32{5: 14, 0: 16}
	for _, record := range updated.Records {
		if want, exists := wantIndices[record.PhysicalIndex]; exists &&
			record.AcquisitionIndex != want {
			t.Fatalf("Dagger at physical index %d has acquisition index %d, want %d",
				record.PhysicalIndex, record.AcquisitionIndex, want)
		}
	}
}

// The gestures route is the one save-session route that joins the raw slot data
// with the catalog gesture definitions, so its fixture unlocks real canonical
// slot IDs. The offsets are restated here instead of reused from SaveEngine,
// including the acquired-projectile count and the Storage Box the block sits
// behind.
const (
	gesturesRouteSlotDataBase = int64(pcHeaderSize) + 0x10
	gesturesRouteUserData10   = int64(pcHeaderSize) + 10*0x280010 + 0x10
	gesturesRouteFlagsOffset  = 0x1954
	gesturesRouteAnchorAt     = 0x0640
	gesturesRouteProjCountAt  = 0xD0 + 0x58 + 0x1C + 0x58 + 0x58 + 0x9011 + 0x74 + 0x8C + 0x18
	gesturesRouteProjectiles  = 4
	gesturesRouteBlocksBefore = 0x9C + 0x0C + 0x12F
	gesturesRouteStorageBox   = 0x6010
	gesturesRouteSectionAt    = gesturesRouteProjCountAt + 4 +
		gesturesRouteProjectiles*8 + gesturesRouteBlocksBefore + gesturesRouteStorageBox
	gesturesRouteSlotCount = 64

	// One canonical slot ID the fixture unlocks, and the native empty sentinel
	// every other record carries.
	gesturesRouteUnlockedSlotID = uint32(1)
	gesturesRouteEmptySentinel  = uint32(0xFFFFFFFE)
)

// writeGesturesFixture writes a synthetic PC save into t.TempDir() whose slot 0
// is active and whose GestureGameData unlocks exactly one canonical gesture.
func writeGesturesFixture(t *testing.T) string {
	t.Helper()

	data := make([]byte, pcFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[pcEntryCountOffset:], pcEntryCount)
	data[gesturesRouteUserData10+gesturesRouteFlagsOffset] = 1

	anchorBase := gesturesRouteSlotDataBase + gesturesRouteAnchorAt
	copy(data[anchorBase:], equippedSpellsFixtureAnchor)
	binary.LittleEndian.PutUint32(data[anchorBase+gesturesRouteProjCountAt:], gesturesRouteProjectiles)

	for index := 0; index < gesturesRouteSlotCount; index++ {
		record := gesturesRouteEmptySentinel
		if index == 0 {
			record = gesturesRouteUnlockedSlotID
		}
		binary.LittleEndian.PutUint32(
			data[anchorBase+gesturesRouteSectionAt+int64(index)*4:], record)
	}

	path := filepath.Join(t.TempDir(), "gestures.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestGesturesRouteMatchesTheGetter(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeGesturesFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/gestures"
	// The prototype catalog carries no gesture document, so it could not prove
	// that the route passes the catalog on.
	gameCatalog := newFullCatalog(t)

	serve := func(target string) *httptest.ResponseRecorder {
		t.Helper()
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).
			ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		return recorder
	}

	want, err := world.GetGestures(saveEngine, gameCatalog, session.SaveSessionID, 0, "")
	if err != nil {
		t.Fatalf("world.GetGestures: %v", err)
	}
	if !want.Active || len(want.Gestures) == 0 {
		t.Fatalf("fixture result = active %t with %d gestures, want an active slot with gestures",
			want.Active, len(want.Gestures))
	}
	unlocked := 0
	for _, entry := range want.Gestures {
		if entry.Unlocked {
			unlocked++
		}
	}
	if unlocked != 1 {
		t.Fatalf("fixture unlocked %d gestures, want exactly 1", unlocked)
	}

	recorder := serve(base)
	assertOK(t, recorder, base)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("gestures route body differs from the GetGestures result")
	}

	// Both accepted filter values have to reach the getter unchanged.
	for _, filter := range []string{"unlocked", "locked"} {
		target := base + "?availabilityFilter=" + filter
		wantFiltered, err := world.GetGestures(saveEngine, gameCatalog, session.SaveSessionID, 0, filter)
		if err != nil {
			t.Fatalf("world.GetGestures(%q): %v", filter, err)
		}
		if len(wantFiltered.Gestures) == 0 || len(wantFiltered.Gestures) == len(want.Gestures) {
			t.Fatalf("filter %q kept %d of %d gestures, want a real subset",
				filter, len(wantFiltered.Gestures), len(want.Gestures))
		}
		filtered := serve(target)
		assertOK(t, filtered, target)
		if !reflect.DeepEqual(decode(t, filtered.Body.Bytes()), marshalled(t, wantFiltered)) {
			t.Fatalf("filtered gestures route body differs from the GetGestures(%q) result", filter)
		}
	}

	// The value is never trimmed, normalised or aliased on the way through, so a
	// padded or case-shifted filter is rejected by the getter with a 400.
	for _, query := range []string{
		"?availabilityFilter=Unlocked",
		"?availabilityFilter=%20unlocked",
		"?availabilityFilter=unlocked%20",
		"?availabilityFilter=all",
	} {
		rejected := serve(base + query)
		if rejected.Code != http.StatusBadRequest {
			t.Fatalf("%s%s: status = %d, want 400 (body %q)",
				base, query, rejected.Code, rejected.Body.String())
		}
	}

	// The route must not answer for a malformed characterID either.
	malformed := serve("/api/v1/save-sessions/" + session.SaveSessionID + "/characters/one/gestures")
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("malformed characterID: status = %d, want 400 (body %q)",
			malformed.Code, malformed.Body.String())
	}
}

func TestGesturesRouteIsAbsentWithoutAnEngine(t *testing.T) {
	target := "/api/v1/save-sessions/any-session/characters/0/gestures"
	recorder := doSave(t, nil, http.MethodGet, target, "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("%s: status = %d, want 404 (body %q)", target, recorder.Code, recorder.Body.String())
	}
}

func TestSetGestureUnlockedRoute(t *testing.T) {
	gameCatalog := newFullCatalog(t)
	saveEngine := saveengine.New()
	boolPointer := func(value bool) *bool { return &value }

	t.Run("conforms to endpoint mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeGesturesFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		body, _ := json.Marshal(setGestureUnlockedRequest{
			GestureKind:      "item",
			GestureKey:       "401EA7AB",
			Unlocked:         boolPointer(true),
			ExpectedRevision: "0",
		})
		target := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/0/gestures/unlock"
		request := httptest.NewRequest(http.MethodPut, target, bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
		assertOK(t, recorder, target)

		var got world.SetGestureUnlockedResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if got.SaveRevision != "1" || !got.Unlocked || got.GestureKey != "401EA7AB" {
			t.Errorf("result = %+v, want revision 1 and unlocked gesture 401EA7AB", got)
		}

		directSession, err := savesession.LoadSave(saveEngine, writeGesturesFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave direct session: %v", err)
		}
		want, err := world.SetGestureUnlocked(
			saveEngine, gameCatalog, directSession.SaveSessionID, 0,
			"item", "401EA7AB", true, "0")
		if err != nil {
			t.Fatalf("world.SetGestureUnlocked: %v", err)
		}
		want.SaveSessionID = got.SaveSessionID
		if got != want {
			t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
		}
	})

	t.Run("rejects invalid bodies before mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeGesturesFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		base := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/0/gestures/unlock"
		for name, body := range map[string]string{
			"missing unlocked": `{"gestureKind":"item","gestureKey":"401EA7AB","expectedRevision":"0"}`,
			"unknown field":    `{"gestureKind":"item","gestureKey":"401EA7AB","unlocked":true,"expectedRevision":"0","extra":1}`,
		} {
			t.Run(name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPut, base, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
				}
			})
		}

		request := httptest.NewRequest(http.MethodPut, base, strings.NewReader(
			`{"gestureKind":"item","gestureKey":"401EA7AB","unlocked":true,"expectedRevision":"0"}`))
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("missing Content-Type: status = %d, want 400 (body %q)",
				recorder.Code, recorder.Body.String())
		}
		info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
		if err != nil {
			t.Fatalf("GetSessionInfo: %v", err)
		}
		if info.UnsavedChanges {
			t.Errorf("session after rejected bodies = %+v, want clean", info)
		}
	})
}

// The cookbooks route joins the catalog cookbook definitions with the event flag
// bitfield of the slot, so its fixture walks the whole confirmed chain from the
// anchor to that bitfield and sets one real cookbook flag. The distances are
// restated here instead of reused from SaveEngine.
const (
	cookbooksRouteProjectiles    = 4
	cookbooksRouteBlocksBefore   = 0x9C + 0x0C + 0x12F
	cookbooksRouteStorageBox     = 0x6010
	cookbooksRouteGestureSection = 64 * 4
	cookbooksRouteHorseSize      = 0x28 + 1
	cookbooksRouteBloodStainSize = 0x44 + 8
	cookbooksRouteDynamicHeader  = 2 + 2 + 4
	cookbooksRouteTrophySize     = 0x34
	cookbooksRouteGaItemSize     = 8 + 7000*16
	cookbooksRouteScalarsSize    = 3 + 4 + 4 + 1 + 4 + 4 + 1 + 4 + 4
	cookbooksRouteEventFlagBlock = 125
	cookbooksRouteRegions        = 0
	cookbooksRouteMenuSize       = 0
	cookbooksRouteTutorialSize   = 0

	// Distance from the anchor to the first byte of the event flag bitfield, for
	// the lengths this fixture declares.
	cookbooksRouteSectionAt = gesturesRouteProjCountAt + 4 +
		cookbooksRouteProjectiles*8 +
		cookbooksRouteBlocksBefore + cookbooksRouteStorageBox +
		cookbooksRouteGestureSection + 4 +
		cookbooksRouteRegions*4 + cookbooksRouteHorseSize + cookbooksRouteBloodStainSize +
		cookbooksRouteDynamicHeader + cookbooksRouteMenuSize +
		cookbooksRouteTrophySize + cookbooksRouteGaItemSize +
		cookbooksRouteDynamicHeader + cookbooksRouteTutorialSize +
		cookbooksRouteScalarsSize

	// The one cookbook event flag the fixture sets. The stored catalog maps it to
	// Nomadic Warrior's Cookbook [1].
	cookbooksRouteSetFlag = uint32(67000)
)

// writeCookbooksFixture writes a synthetic PC save into t.TempDir() whose slot 0
// is active and whose event flag bitfield carries exactly one cookbook flag.
func writeCookbooksFixture(t *testing.T) string {
	t.Helper()

	data := make([]byte, pcFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[pcEntryCountOffset:], pcEntryCount)
	data[gesturesRouteUserData10+gesturesRouteFlagsOffset] = 1

	anchorBase := gesturesRouteSlotDataBase + gesturesRouteAnchorAt
	copy(data[anchorBase:], equippedSpellsFixtureAnchor)
	binary.LittleEndian.PutUint32(
		data[anchorBase+gesturesRouteProjCountAt:], cookbooksRouteProjectiles)

	index := int64(cookbooksRouteSetFlag % 1000)
	offset := 17*cookbooksRouteEventFlagBlock + index/8
	data[anchorBase+cookbooksRouteSectionAt+offset] |= 1 << uint8(7-index%8)

	path := filepath.Join(t.TempDir(), "cookbooks.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestCookbooksRouteMatchesTheGetter(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/cookbooks"
	// The prototype catalog declares no cookbook unlock, so it could not prove
	// that the route passes the catalog on.
	gameCatalog := newFullCatalog(t)

	serve := func(target string) *httptest.ResponseRecorder {
		t.Helper()
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).
			ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		return recorder
	}

	want, err := world.GetCookbooks(saveEngine, gameCatalog, session.SaveSessionID, 0, "")
	if err != nil {
		t.Fatalf("world.GetCookbooks: %v", err)
	}
	if !want.Active || len(want.Cookbooks) == 0 {
		t.Fatalf("fixture result = active %t with %d cookbooks, want an active slot with cookbooks",
			want.Active, len(want.Cookbooks))
	}
	unlocked := 0
	for _, entry := range want.Cookbooks {
		if entry.Unlocked {
			unlocked++
		}
	}
	if unlocked != 1 {
		t.Fatalf("fixture unlocked %d cookbooks, want exactly 1", unlocked)
	}

	recorder := serve(base)
	assertOK(t, recorder, base)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("cookbooks route body differs from the GetCookbooks result")
	}

	// Both accepted filter values have to reach the getter unchanged.
	for _, filter := range []string{"unlocked", "locked"} {
		target := base + "?availabilityFilter=" + filter
		wantFiltered, err := world.GetCookbooks(saveEngine, gameCatalog, session.SaveSessionID, 0, filter)
		if err != nil {
			t.Fatalf("world.GetCookbooks(%q): %v", filter, err)
		}
		if len(wantFiltered.Cookbooks) == 0 || len(wantFiltered.Cookbooks) == len(want.Cookbooks) {
			t.Fatalf("filter %q kept %d of %d cookbooks, want a real subset",
				filter, len(wantFiltered.Cookbooks), len(want.Cookbooks))
		}
		filtered := serve(target)
		assertOK(t, filtered, target)
		if !reflect.DeepEqual(decode(t, filtered.Body.Bytes()), marshalled(t, wantFiltered)) {
			t.Fatalf("filtered cookbooks route body differs from the GetCookbooks(%q) result", filter)
		}
	}

	// The value is never trimmed, normalised or aliased on the way through, so a
	// padded or case-shifted filter is rejected by the getter with a 400.
	for _, query := range []string{
		"?availabilityFilter=Unlocked",
		"?availabilityFilter=%20unlocked",
		"?availabilityFilter=unlocked%20",
		"?availabilityFilter=all",
	} {
		rejected := serve(base + query)
		if rejected.Code != http.StatusBadRequest {
			t.Fatalf("%s%s: status = %d, want 400 (body %q)",
				base, query, rejected.Code, rejected.Body.String())
		}
	}

	// The route must not answer for a malformed characterID either.
	malformed := serve("/api/v1/save-sessions/" + session.SaveSessionID + "/characters/one/cookbooks")
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("malformed characterID: status = %d, want 400 (body %q)",
			malformed.Code, malformed.Body.String())
	}
}

func writeBellBearingsRouteFixture(t *testing.T) string {
	t.Helper()
	path := writeCookbooksFixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	const bellBearingFlag = uint32(11109710)
	index := int64(bellBearingFlag % 1000)
	offset := int64(11129)*cookbooksRouteEventFlagBlock + index/8
	anchorBase := gesturesRouteSlotDataBase + gesturesRouteAnchorAt
	data[anchorBase+cookbooksRouteSectionAt+offset] |= 1 << uint8(7-index%8)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestBellBearingsRouteMatchesTheGetter(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeBellBearingsRouteFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	base := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/bell-bearings"
	serve := func(target string) *httptest.ResponseRecorder {
		t.Helper()
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).
			ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		return recorder
	}
	want, err := world.GetBellBearings(
		saveEngine, gameCatalog, session.SaveSessionID, 0, "")
	if err != nil {
		t.Fatalf("world.GetBellBearings: %v", err)
	}
	if !want.Active || len(want.BellBearings) != 62 {
		t.Fatalf("fixture result = active %t with %d bell bearings, want true/62",
			want.Active, len(want.BellBearings))
	}

	recorder := serve(base)
	assertOK(t, recorder, base)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("bell bearings route body differs from the getter result")
	}

	target := base + "?availabilityFilter=unlocked"
	wantFiltered, err := world.GetBellBearings(
		saveEngine, gameCatalog, session.SaveSessionID, 0, "unlocked")
	if err != nil {
		t.Fatalf("world.GetBellBearings(unlocked): %v", err)
	}
	filtered := serve(target)
	assertOK(t, filtered, target)
	if !reflect.DeepEqual(decode(t, filtered.Body.Bytes()), marshalled(t, wantFiltered)) {
		t.Fatal("filtered bell bearings route body differs from the getter result")
	}
}

func TestSetBellBearingUnlockedRoute(t *testing.T) {
	gameCatalog := newFullCatalog(t)
	saveEngine := saveengine.New()
	boolPointer := func(value bool) *bool { return &value }

	t.Run("conforms to endpoint mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeBellBearingsRouteFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		body, _ := json.Marshal(setBellBearingUnlockedRequest{
			BellBearingKind: "item", BellBearingKey: "400022CF",
			Unlocked: boolPointer(true), ExpectedRevision: "0",
		})
		target := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/0/bell-bearings/unlock"
		request := httptest.NewRequest(http.MethodPut, target, bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
		assertOK(t, recorder, target)

		var got world.SetBellBearingUnlockedResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if got.SaveRevision != "1" || !got.Unlocked || got.BellBearingKey != "400022CF" {
			t.Errorf("result = %+v, want revision 1 and unlocked Bell Bearing 400022CF", got)
		}

		directSession, err := savesession.LoadSave(saveEngine, writeBellBearingsRouteFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave direct session: %v", err)
		}
		want, err := world.SetBellBearingUnlocked(
			saveEngine, gameCatalog, directSession.SaveSessionID, 0,
			"item", "400022CF", true, "0")
		if err != nil {
			t.Fatalf("world.SetBellBearingUnlocked: %v", err)
		}
		want.SaveSessionID = got.SaveSessionID
		if got != want {
			t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
		}
	})

	t.Run("rejects invalid body before mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeBellBearingsRouteFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		target := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/0/bell-bearings/unlock"
		request := httptest.NewRequest(http.MethodPut, target, strings.NewReader(
			`{"bellBearingKind":"item","bellBearingKey":"400022CF","expectedRevision":"0"}`))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
		}
		info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
		if err != nil {
			t.Fatalf("GetSessionInfo: %v", err)
		}
		if info.UnsavedChanges {
			t.Errorf("rejected body dirtied the session: %+v", info)
		}
	})

	t.Run("absent without engine", func(t *testing.T) {
		target := "/api/v1/save-sessions/session1/characters/0/bell-bearings/unlock"
		request := httptest.NewRequest(http.MethodPut, target, strings.NewReader(
			`{"bellBearingKind":"item","bellBearingKey":"400022CF","unlocked":true,"expectedRevision":"0"}`))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, nil).ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404 (body %q)", recorder.Code, recorder.Body.String())
		}
	})
}

func writeWhetbladesRouteFixture(t *testing.T) string {
	t.Helper()
	path := writeCookbooksFixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	const whetstoneKnifeFlag = uint32(60130)
	index := int64(whetstoneKnifeFlag % 1000)
	offset := 10*cookbooksRouteEventFlagBlock + index/8
	anchorBase := gesturesRouteSlotDataBase + gesturesRouteAnchorAt
	data[anchorBase+cookbooksRouteSectionAt+offset] |= 1 << uint8(7-index%8)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestWhetbladesRouteMatchesTheGetter(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeWhetbladesRouteFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/whetblades"
	serve := func(target string) *httptest.ResponseRecorder {
		t.Helper()
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).
			ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		return recorder
	}
	want, err := world.GetWhetblades(saveEngine, gameCatalog, session.SaveSessionID, 0, "")
	if err != nil {
		t.Fatalf("world.GetWhetblades: %v", err)
	}
	if !want.Active || len(want.Whetblades) != 6 {
		t.Fatalf("fixture result = active %t with %d whetblades, want true/6",
			want.Active, len(want.Whetblades))
	}

	recorder := serve(base)
	assertOK(t, recorder, base)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("whetblades route body differs from the GetWhetblades result")
	}

	target := base + "?availabilityFilter=unlocked"
	wantFiltered, err := world.GetWhetblades(
		saveEngine, gameCatalog, session.SaveSessionID, 0, "unlocked")
	if err != nil {
		t.Fatalf("world.GetWhetblades(unlocked): %v", err)
	}
	filtered := serve(target)
	assertOK(t, filtered, target)
	if !reflect.DeepEqual(decode(t, filtered.Body.Bytes()), marshalled(t, wantFiltered)) {
		t.Fatal("filtered whetblades route body differs from the getter result")
	}
}

func TestColosseumsRouteMatchesTheGetter(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/colosseums"

	want, err := world.GetColosseums(saveEngine, gameCatalog, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("world.GetColosseums: %v", err)
	}
	if !want.Active || len(want.Colosseums) != 3 {
		t.Fatalf("fixture result = active %t with %d colosseums, want true/3",
			want.Active, len(want.Colosseums))
	}

	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	assertOK(t, recorder, target)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("colosseums route body differs from the GetColosseums result")
	}
}

func TestRegionsRouteMatchesTheGetter(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/regions"

	want, err := world.GetRegions(saveEngine, gameCatalog, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("world.GetRegions: %v", err)
	}
	if !want.Active || len(want.Regions) != 274 {
		t.Fatalf("fixture result = active %t with %d regions, want true/274",
			want.Active, len(want.Regions))
	}

	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	assertOK(t, recorder, target)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("regions route body differs from the GetRegions result")
	}
}

func TestSummoningPoolsRouteMatchesTheGetter(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/summoning-pools"

	want, err := world.GetSummoningPools(saveEngine, gameCatalog, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("world.GetSummoningPools: %v", err)
	}
	if !want.Active || len(want.SummoningPools) != 213 {
		t.Fatalf("fixture result = active %t with %d summoning pools, want true/213",
			want.Active, len(want.SummoningPools))
	}

	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	assertOK(t, recorder, target)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("summoning pools route body differs from the GetSummoningPools result")
	}
}

func TestGracesRouteMatchesTheGetter(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/graces"

	want, err := world.GetGraces(saveEngine, gameCatalog, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("world.GetGraces: %v", err)
	}
	if !want.Active || len(want.Graces) != 419 {
		t.Fatalf("fixture result = active %t with %d graces, want true/419",
			want.Active, len(want.Graces))
	}

	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	assertOK(t, recorder, target)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("graces route body differs from the GetGraces result")
	}
}

// The curated Bosses table uses block 9 alone, so the document must describe
// exactly that block. A wider range would advertise the neighbouring blocks 8
// and 10, which resolveEventFlag rejects and no boss declares.
func TestBossDefeatEventFlagOpenAPICoversBlock9Only(t *testing.T) {
	recorder := do(t, newPrototypeCatalog(t), "/openapi.json")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var document struct {
		Comps struct {
			Schemas map[string]struct {
				Properties struct {
					DefeatEventFlagID struct {
						Properties struct {
							Value struct {
								Type    string  `json:"type"`
								Format  string  `json:"format"`
								Minimum *uint32 `json:"minimum"`
								Maximum *uint32 `json:"maximum"`
							} `json:"value"`
						} `json:"properties"`
					} `json:"defeatEventFlagID"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode openapi.json: %v", err)
	}

	value := document.Comps.Schemas["BossDocument"].Properties.DefeatEventFlagID.Properties.Value
	if value.Type != "integer" || value.Format != "int64" {
		t.Fatalf("defeatEventFlagID.value = %q/%q, want integer/int64", value.Type, value.Format)
	}
	if value.Minimum == nil || *value.Minimum != 9000 ||
		value.Maximum == nil || *value.Maximum != 9999 {
		t.Fatalf("defeatEventFlagID.value range = %v-%v, want 9000-9999",
			value.Minimum, value.Maximum)
	}
}

func TestBossesRouteMatchesTheGetter(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/bosses"

	want, err := world.GetBosses(saveEngine, gameCatalog, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("world.GetBosses: %v", err)
	}
	if !want.Active || len(want.Bosses) != 110 {
		t.Fatalf("fixture result = active %t with %d bosses, want true/110",
			want.Active, len(want.Bosses))
	}

	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	assertOK(t, recorder, target)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("bosses route body differs from the GetBosses result")
	}
}

func TestMapRegionVisibilityEventFlagOpenAPICoversBlock62Only(t *testing.T) {
	recorder := do(t, newPrototypeCatalog(t), "/openapi.json")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var document struct {
		Comps struct {
			Schemas map[string]struct {
				Properties struct {
					VisibleEventFlagID struct {
						Properties struct {
							Value struct {
								Type    string  `json:"type"`
								Format  string  `json:"format"`
								Minimum *uint32 `json:"minimum"`
								Maximum *uint32 `json:"maximum"`
							} `json:"value"`
						} `json:"properties"`
					} `json:"visibleEventFlagID"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode openapi.json: %v", err)
	}

	value := document.Comps.Schemas["MapRegionDocument"].Properties.
		VisibleEventFlagID.Properties.Value
	if value.Type != "integer" || value.Format != "int64" {
		t.Fatalf("visibleEventFlagID.value = %q/%q, want integer/int64",
			value.Type, value.Format)
	}
	if value.Minimum == nil || *value.Minimum != 62000 ||
		value.Maximum == nil || *value.Maximum != 62999 {
		t.Fatalf("visibleEventFlagID.value range = %v-%v, want 62000-62999",
			value.Minimum, value.Maximum)
	}
}

func TestMapRegionsRouteMatchesTheGetter(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/map-regions"

	want, err := world.GetMapRegions(saveEngine, gameCatalog, session.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("world.GetMapRegions: %v", err)
	}
	if !want.Active || len(want.MapRegions) != 263 {
		t.Fatalf("fixture result = active %t with %d map regions, want true/263",
			want.Active, len(want.MapRegions))
	}

	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	assertOK(t, recorder, target)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("map regions route body differs from the GetMapRegions result")
	}
}

func TestTutorialsRouteMatchesTheGetter(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeSetTutorialRouteFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/tutorials?availabilityFilter=locked"

	want, err := world.GetTutorials(
		saveEngine, gameCatalog, session.SaveSessionID, 0, "locked")
	if err != nil {
		t.Fatalf("world.GetTutorials: %v", err)
	}
	if !want.Active || len(want.Tutorials) != 72 {
		t.Fatalf("fixture result = active %t with %d tutorials, want true/72",
			want.Active, len(want.Tutorials))
	}

	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	assertOK(t, recorder, target)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("tutorials route body differs from the GetTutorials result")
	}
}

func TestSetWhetbladeUnlockedRoute(t *testing.T) {
	gameCatalog := newFullCatalog(t)
	saveEngine := saveengine.New()
	unlocked := true
	session, err := savesession.LoadSave(saveEngine, writeWhetbladesRouteFixture(t), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	body, _ := json.Marshal(setWhetbladeUnlockedRequest{
		WhetbladeKind: "item", WhetbladeKey: "4000230C",
		Unlocked: &unlocked, ExpectedRevision: "0",
	})
	target := "/api/v1/save-sessions/" + session.SaveSessionID +
		"/characters/0/whetblades/unlock"
	request := httptest.NewRequest(http.MethodPut, target, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
	assertOK(t, recorder, target)

	var got world.SetWhetbladeUnlockedResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.SaveRevision != "1" || !got.Unlocked || got.WhetbladeKey != "4000230C" {
		t.Errorf("result = %+v, want revision 1 and unlocked Whetblade 4000230C", got)
	}

	directSession, err := savesession.LoadSave(saveEngine, writeWhetbladesRouteFixture(t), "")
	if err != nil {
		t.Fatalf("LoadSave direct session: %v", err)
	}
	want, err := world.SetWhetbladeUnlocked(
		saveEngine, gameCatalog, directSession.SaveSessionID, 0,
		"item", "4000230C", true, "0")
	if err != nil {
		t.Fatalf("world.SetWhetbladeUnlocked: %v", err)
	}
	want.SaveSessionID = got.SaveSessionID
	if got != want {
		t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
	}

	rejectedSession, err := savesession.LoadSave(saveEngine, writeWhetbladesRouteFixture(t), "")
	if err != nil {
		t.Fatalf("LoadSave rejected session: %v", err)
	}
	rejectedTarget := "/api/v1/save-sessions/" + rejectedSession.SaveSessionID +
		"/characters/0/whetblades/unlock"
	rejected := httptest.NewRequest(http.MethodPut, rejectedTarget, strings.NewReader(
		`{"whetbladeKind":"item","whetbladeKey":"4000230C","expectedRevision":"0"}`))
	rejected.Header.Set("Content-Type", "application/json")
	rejectedRecorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).
		ServeHTTP(rejectedRecorder, rejected)
	if rejectedRecorder.Code != http.StatusBadRequest {
		t.Fatalf("missing unlocked status = %d, want 400", rejectedRecorder.Code)
	}
	info, err := saveEngine.GetSessionInfo(rejectedSession.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Errorf("rejected body dirtied the session: %+v", info)
	}
}

func TestCookbooksRouteIsAbsentWithoutAnEngine(t *testing.T) {
	target := "/api/v1/save-sessions/any-session/characters/0/cookbooks"
	recorder := doSave(t, nil, http.MethodGet, target, "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("%s: status = %d, want 404 (body %q)", target, recorder.Code, recorder.Body.String())
	}
}

func TestSetCookbookUnlockedRoute(t *testing.T) {
	gameCatalog := newFullCatalog(t)
	saveEngine := saveengine.New()
	boolPointer := func(value bool) *bool { return &value }

	t.Run("conforms to endpoint mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}

		bodyBytes, _ := json.Marshal(setCookbookUnlockedRequest{
			CookbookKind:     "item",
			CookbookKey:      "40002455",
			Unlocked:         boolPointer(true),
			ExpectedRevision: "0",
		})

		req := httptest.NewRequest(http.MethodPut,
			"/api/v1/save-sessions/"+session.SaveSessionID+"/characters/0/cookbooks/unlock",
			bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}

		assertJSONContentType(t, recorder)

		var got world.SetCookbookUnlockedResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if got.SaveRevision != "1" || !got.Unlocked || got.CookbookKey != "40002455" {
			t.Errorf("got = %+v, want revision 1 and unlocked true", got)
		}

		directSession, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave direct session: %v", err)
		}
		want, err := world.SetCookbookUnlocked(
			saveEngine, gameCatalog, directSession.SaveSessionID, 0, "item", "40002455", true, "0")
		if err != nil {
			t.Fatalf("world.SetCookbookUnlocked: %v", err)
		}
		want.SaveSessionID = got.SaveSessionID
		if got != want {
			t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
		}
	})

	t.Run("rejects invalid bodies", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/cookbooks/unlock"
		for name, body := range map[string]string{
			"missing unlocked": `{"cookbookKind":"item","cookbookKey":"40002455","expectedRevision":"0"}`,
			"unknown field":    `{"cookbookKind":"item","cookbookKey":"40002455","unlocked":true,"expectedRevision":"0","extra":1}`,
		} {
			t.Run(name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPut, base, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
				}
			})
		}
		request := httptest.NewRequest(http.MethodPut, base, strings.NewReader(
			`{"cookbookKind":"item","cookbookKey":"40002455","unlocked":true,"expectedRevision":"0"}`))
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("missing Content-Type: status = %d, want 400 (body %q)",
				recorder.Code, recorder.Body.String())
		}
		info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
		if err != nil {
			t.Fatalf("GetSessionInfo: %v", err)
		}
		if info.UnsavedChanges {
			t.Errorf("session after rejected bodies = %+v, want clean", info)
		}
		result, err := world.SetCookbookUnlocked(
			saveEngine, gameCatalog, session.SaveSessionID, 0, "item", "40002455", true, "0")
		if err != nil {
			t.Fatalf("valid mutation after rejected bodies: %v", err)
		}
		if result.SaveRevision != "1" {
			t.Errorf("revision after rejected bodies = %q, want first commit revision 1", result.SaveRevision)
		}
	})

	t.Run("absent without engine", func(t *testing.T) {
		bodyBytes, _ := json.Marshal(setCookbookUnlockedRequest{
			CookbookKind:     "item",
			CookbookKey:      "40002455",
			Unlocked:         boolPointer(true),
			ExpectedRevision: "0",
		})
		req := httptest.NewRequest(http.MethodPut,
			"/api/v1/save-sessions/session1/characters/0/cookbooks/unlock",
			bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, nil).ServeHTTP(recorder, req)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404 when save engine is nil", recorder.Code)
		}
	})
}

func TestSetBossDefeatedRoute(t *testing.T) {
	gameCatalog := newFullCatalog(t)
	saveEngine := saveengine.New()
	defeated := true
	const bossKey = "stormveil_castle_godrick_the_grafted"

	t.Run("conforms to endpoint mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		bodyBytes, _ := json.Marshal(setBossDefeatedRequest{
			BossKind:         "boss",
			BossKey:          bossKey,
			Defeated:         &defeated,
			ExpectedRevision: "0",
		})

		request := httptest.NewRequest(http.MethodPut,
			"/api/v1/save-sessions/"+session.SaveSessionID+"/characters/0/bosses/defeat",
			bytes.NewReader(bodyBytes))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertJSONContentType(t, recorder)

		var got world.SetBossDefeatedResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		directSession, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave direct session: %v", err)
		}
		want, err := world.SetBossDefeated(saveEngine, gameCatalog,
			directSession.SaveSessionID, 0, "boss", bossKey, true, "0")
		if err != nil {
			t.Fatalf("world.SetBossDefeated: %v", err)
		}
		want.SaveSessionID = got.SaveSessionID
		if got != want {
			t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
		}
	})

	t.Run("rejects invalid bodies", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/bosses/defeat"
		for name, body := range map[string]string{
			"missing defeated": `{"bossKind":"boss","bossKey":"` + bossKey + `","expectedRevision":"0"}`,
			"unknown field":    `{"bossKind":"boss","bossKey":"` + bossKey + `","defeated":true,"expectedRevision":"0","extra":1}`,
		} {
			t.Run(name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPut, base, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
				}
			})
		}
		info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
		if err != nil {
			t.Fatalf("GetSessionInfo: %v", err)
		}
		if info.UnsavedChanges {
			t.Errorf("session after rejected bodies = %+v, want clean", info)
		}
	})

	// The served document must carry the route, so the transport and the
	// contract cannot drift apart.
	t.Run("is described by openapi.json", func(t *testing.T) {
		recorder := do(t, newPrototypeCatalog(t), "/openapi.json")
		var document struct {
			Paths map[string]map[string]any `json:"paths"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
			t.Fatalf("decode openapi.json: %v", err)
		}
		path := "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/bosses/defeat"
		if _, exists := document.Paths[path]["put"]; !exists {
			t.Fatalf("openapi.json describes no PUT for %s", path)
		}
	})
}

func TestSetGraceVisitedRoute(t *testing.T) {
	gameCatalog := newFullCatalog(t)
	saveEngine := saveengine.New()
	visited := true
	const graceKey = "limgrave_west_gatefront"

	t.Run("conforms to endpoint mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		bodyBytes, _ := json.Marshal(setGraceVisitedRequest{
			GraceKind:        "grace",
			GraceKey:         graceKey,
			Visited:          &visited,
			ExpectedRevision: "0",
		})

		request := httptest.NewRequest(http.MethodPut,
			"/api/v1/save-sessions/"+session.SaveSessionID+"/characters/0/graces/visit",
			bytes.NewReader(bodyBytes))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertJSONContentType(t, recorder)

		var got world.SetGraceVisitedResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		directSession, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave direct session: %v", err)
		}
		want, err := world.SetGraceVisited(saveEngine, gameCatalog,
			directSession.SaveSessionID, 0, "grace", graceKey, true, "0")
		if err != nil {
			t.Fatalf("world.SetGraceVisited: %v", err)
		}
		want.SaveSessionID = got.SaveSessionID
		if got != want {
			t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
		}
	})

	t.Run("rejects invalid bodies", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/graces/visit"
		for name, body := range map[string]string{
			"missing visited": `{"graceKind":"grace","graceKey":"` + graceKey + `","expectedRevision":"0"}`,
			"unknown field":   `{"graceKind":"grace","graceKey":"` + graceKey + `","visited":true,"expectedRevision":"0","extra":1}`,
		} {
			t.Run(name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPut, base, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
				}
			})
		}
		info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
		if err != nil {
			t.Fatalf("GetSessionInfo: %v", err)
		}
		if info.UnsavedChanges {
			t.Errorf("session after rejected bodies = %+v, want clean", info)
		}
	})

	// The served document must carry the route, so the transport and the
	// contract cannot drift apart.
	t.Run("is described by openapi.json", func(t *testing.T) {
		recorder := do(t, newPrototypeCatalog(t), "/openapi.json")
		var document struct {
			Paths map[string]map[string]any `json:"paths"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
			t.Fatalf("decode openapi.json: %v", err)
		}
		path := "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/graces/visit"
		if _, exists := document.Paths[path]["put"]; !exists {
			t.Fatalf("openapi.json describes no PUT for %s", path)
		}
	})
}

func TestSetColosseumUnlockedRoute(t *testing.T) {
	gameCatalog := newFullCatalog(t)
	saveEngine := saveengine.New()
	unlocked := true
	const colosseumKey = "royal_colosseum"

	t.Run("conforms to endpoint mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		bodyBytes, _ := json.Marshal(setColosseumUnlockedRequest{
			ColosseumKind:    "colosseum",
			ColosseumKey:     colosseumKey,
			Unlocked:         &unlocked,
			ExpectedRevision: "0",
		})

		request := httptest.NewRequest(http.MethodPut,
			"/api/v1/save-sessions/"+session.SaveSessionID+"/characters/0/colosseums/unlock",
			bytes.NewReader(bodyBytes))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertJSONContentType(t, recorder)

		var got world.SetColosseumUnlockedResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		directSession, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave direct session: %v", err)
		}
		want, err := world.SetColosseumUnlocked(saveEngine, gameCatalog,
			directSession.SaveSessionID, 0, "colosseum", colosseumKey, true, "0")
		if err != nil {
			t.Fatalf("world.SetColosseumUnlocked: %v", err)
		}
		want.SaveSessionID = got.SaveSessionID
		if got != want {
			t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
		}
	})

	t.Run("rejects invalid bodies", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/colosseums/unlock"
		for name, body := range map[string]string{
			"missing unlocked": `{"colosseumKind":"colosseum","colosseumKey":"` + colosseumKey + `","expectedRevision":"0"}`,
			"unknown field":    `{"colosseumKind":"colosseum","colosseumKey":"` + colosseumKey + `","unlocked":true,"expectedRevision":"0","extra":1}`,
		} {
			t.Run(name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPut, base, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
				}
			})
		}
		info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
		if err != nil {
			t.Fatalf("GetSessionInfo: %v", err)
		}
		if info.UnsavedChanges {
			t.Errorf("session after rejected bodies = %+v, want clean", info)
		}
	})

	// The served document must carry the route, so the transport and the
	// contract cannot drift apart.
	t.Run("is described by openapi.json", func(t *testing.T) {
		recorder := do(t, newPrototypeCatalog(t), "/openapi.json")
		var document struct {
			Paths map[string]map[string]any `json:"paths"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
			t.Fatalf("decode openapi.json: %v", err)
		}
		path := "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/colosseums/unlock"
		if _, exists := document.Paths[path]["put"]; !exists {
			t.Fatalf("openapi.json describes no PUT for %s", path)
		}
	})
}

func TestSetMapRegionRevealedRoute(t *testing.T) {
	gameCatalog := newFullCatalog(t)
	saveEngine := saveengine.New()
	revealed := true
	const mapRegionKey = "limgrave_weeping_peninsula"

	t.Run("conforms to endpoint mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeWhetbladesRouteFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		body, _ := json.Marshal(setMapRegionRevealedRequest{
			MapRegionKind: "map_region", MapRegionKey: mapRegionKey,
			Revealed: &revealed, ExpectedRevision: "0",
		})
		target := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/0/map-regions/reveal"
		request := httptest.NewRequest(http.MethodPut, target, bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
		assertOK(t, recorder, target)

		var got world.SetMapRegionRevealedResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		direct, err := savesession.LoadSave(saveEngine, writeWhetbladesRouteFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave direct session: %v", err)
		}
		want, err := world.SetMapRegionRevealed(saveEngine, gameCatalog,
			direct.SaveSessionID, 0, "map_region", mapRegionKey, true, "0")
		if err != nil {
			t.Fatalf("world.SetMapRegionRevealed: %v", err)
		}
		want.SaveSessionID = got.SaveSessionID
		if got != want {
			t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
		}
	})

	t.Run("rejects invalid bodies", func(t *testing.T) {
		for name, body := range map[string]string{
			"missing revealed": `{"mapRegionKind":"map_region","mapRegionKey":"` + mapRegionKey + `","expectedRevision":"0"}`,
			"unknown field":    `{"mapRegionKind":"map_region","mapRegionKey":"` + mapRegionKey + `","revealed":true,"expectedRevision":"0","extra":1}`,
		} {
			t.Run(name, func(t *testing.T) {
				session, err := savesession.LoadSave(saveEngine, writeWhetbladesRouteFixture(t), "")
				if err != nil {
					t.Fatalf("LoadSave: %v", err)
				}
				target := "/api/v1/save-sessions/" + session.SaveSessionID +
					"/characters/0/map-regions/reveal"
				request := httptest.NewRequest(http.MethodPut, target, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				newHandler(gameCatalog, testApplicationVersion, saveEngine).
					ServeHTTP(recorder, request)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
				}
				info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
				if err != nil {
					t.Fatalf("GetSessionInfo: %v", err)
				}
				if info.UnsavedChanges {
					t.Errorf("rejected body dirtied the session: %+v", info)
				}
			})
		}
	})
}

func TestSetFogOfWarRemovedRoute(t *testing.T) {
	gameCatalog := newFullCatalog(t)
	saveEngine := saveengine.New()
	removed := true
	const target = "/characters/0/fog-of-war"

	t.Run("conforms to endpoint mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeWhetbladesRouteFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		body, _ := json.Marshal(setFogOfWarRemovedRequest{
			Removed: &removed, ExpectedRevision: "0",
		})
		route := "/api/v1/save-sessions/" + session.SaveSessionID + target
		request := httptest.NewRequest(http.MethodPut, route, bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
		assertOK(t, recorder, route)

		var got world.SetFogOfWarRemovedResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		direct, err := savesession.LoadSave(saveEngine, writeWhetbladesRouteFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave direct session: %v", err)
		}
		want, err := world.SetFogOfWarRemoved(saveEngine, direct.SaveSessionID, 0, true, "0")
		if err != nil {
			t.Fatalf("world.SetFogOfWarRemoved: %v", err)
		}
		want.SaveSessionID = got.SaveSessionID
		if got != want {
			t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
		}
	})

	t.Run("rejects invalid bodies", func(t *testing.T) {
		for name, body := range map[string]string{
			"missing removed": `{"expectedRevision":"0"}`,
			"removed false":   `{"removed":false,"expectedRevision":"0"}`,
			"unknown field":   `{"removed":true,"expectedRevision":"0","extra":1}`,
		} {
			t.Run(name, func(t *testing.T) {
				session, err := savesession.LoadSave(saveEngine, writeWhetbladesRouteFixture(t), "")
				if err != nil {
					t.Fatalf("LoadSave: %v", err)
				}
				route := "/api/v1/save-sessions/" + session.SaveSessionID + target
				request := httptest.NewRequest(http.MethodPut, route, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				newHandler(gameCatalog, testApplicationVersion, saveEngine).
					ServeHTTP(recorder, request)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
				}
				info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
				if err != nil {
					t.Fatalf("GetSessionInfo: %v", err)
				}
				if info.UnsavedChanges {
					t.Errorf("rejected body dirtied the session: %+v", info)
				}
			})
		}
	})
}

func TestSetSummoningPoolActivatedRoute(t *testing.T) {
	gameCatalog := newFullCatalog(t)
	saveEngine := saveengine.New()
	activated := true

	t.Run("conforms to endpoint mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		bodyBytes, _ := json.Marshal(setSummoningPoolActivatedRequest{
			SummoningPoolKind: "summoning_pool",
			SummoningPoolKey:  "stormveil_castle_liftside_chamber",
			Activated:         &activated,
			ExpectedRevision:  "0",
		})

		request := httptest.NewRequest(http.MethodPut,
			"/api/v1/save-sessions/"+session.SaveSessionID+"/characters/0/summoning-pools/activate",
			bytes.NewReader(bodyBytes))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertJSONContentType(t, recorder)

		var got world.SetSummoningPoolActivatedResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		directSession, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave direct session: %v", err)
		}
		want, err := world.SetSummoningPoolActivated(saveEngine, gameCatalog,
			directSession.SaveSessionID, 0, "summoning_pool", "stormveil_castle_liftside_chamber",
			true, "0")
		if err != nil {
			t.Fatalf("world.SetSummoningPoolActivated: %v", err)
		}
		want.SaveSessionID = got.SaveSessionID
		if got != want {
			t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
		}
	})

	t.Run("rejects invalid bodies", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		base := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/0/summoning-pools/activate"
		for name, body := range map[string]string{
			"missing activated": `{"summoningPoolKind":"summoning_pool","summoningPoolKey":"stormveil_castle_liftside_chamber","expectedRevision":"0"}`,
			"unknown field":     `{"summoningPoolKind":"summoning_pool","summoningPoolKey":"stormveil_castle_liftside_chamber","activated":true,"expectedRevision":"0","extra":1}`,
		} {
			t.Run(name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPut, base, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
				}
			})
		}
		info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
		if err != nil {
			t.Fatalf("GetSessionInfo: %v", err)
		}
		if info.UnsavedChanges {
			t.Errorf("session after rejected bodies = %+v, want clean", info)
		}
	})
}

func writeSetRegionRouteFixture(t *testing.T) string {
	t.Helper()

	const (
		fixtureSize                 = 0x1A00000
		pcUserData10DataOffset      = 0x19003B0
		userData10ActiveFlagsOffset = 0x1954
		slotBase                    = 0x00000310
		characterSlotDataSize       = 0x280000
		slotFixedDlcOffset          = characterSlotDataSize - 128 - 50
		slotFixedHashOffset         = characterSlotDataSize - 128
	)

	data := make([]byte, fixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[0x0C:], 12)

	data[pcUserData10DataOffset+userData10ActiveFlagsOffset] = 1

	put32 := func(at int64, v uint32) {
		binary.LittleEndian.PutUint32(data[slotBase+at:], v)
	}

	put32(0, 230) // version

	const (
		anchorAt               = int64(0x01A7)
		projectileCountOffset  = 0x93DC
		blocksBeforeStorage    = 0x1D7
		storageBoxSize         = 0x6010
		gestureSectionSize     = 0x100
		worldHeadSize          = 117
		menuProfilePayloadSize = 0x20
		trophyEquipSize        = 52
		gaItemSize             = 112008
		tutorialPayloadSize    = 0x20
		scalarsSize            = 29
		eventFlagsSize         = 0x1BF99F + 1
		coordinatesSize        = 61
		spawnPointSize         = 15
		netManSize             = 4 + 0x20000
		trailingFixedSize      = 130
		playerHashSize         = 128
	)

	anchor := []byte{
		0x00,

		0xFF, 0xFF, 0xFF, 0xFF,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

		0xFF, 0xFF, 0xFF, 0xFF,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

		0xFF, 0xFF, 0xFF, 0xFF,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

		0xFF, 0xFF, 0xFF, 0xFF,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	copy(data[slotBase+anchorAt:], anchor)

	projAt := anchorAt + projectileCountOffset
	put32(projAt, 0)

	countAt := projAt + 4 + blocksBeforeStorage + storageBoxSize + gestureSectionSize
	initialRegions := []uint32{6200000}
	put32(countAt, uint32(len(initialRegions)))
	for i, id := range initialRegions {
		put32(countAt+4+int64(i)*4, id)
	}

	pos := countAt + 4 + int64(len(initialRegions))*4
	pos += worldHeadSize
	put32(pos+4, menuProfilePayloadSize)
	pos += 8 + menuProfilePayloadSize
	pos += trophyEquipSize
	pos += gaItemSize
	put32(pos+4, tutorialPayloadSize)
	pos += 8 + tutorialPayloadSize
	pos += scalarsSize
	pos += eventFlagsSize

	// WorldGeom 5 blocks
	for i := 0; i < 5; i++ {
		put32(pos, 0x10)
		pos += 4 + 0x10
	}

	pos += coordinatesSize
	pos += spawnPointSize
	pos += netManSize
	pos += trailingFixedSize
	pos += playerHashSize

	// Fixed DLC & Hash
	for i := int64(0); i < 50; i++ {
		data[slotBase+slotFixedDlcOffset+i] = 0xAA
	}
	for i := int64(0); i < 128; i++ {
		data[slotBase+slotFixedHashOffset+i] = 0xBB
	}

	path := filepath.Join(t.TempDir(), "set-regions-route.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestSetRegionUnlockedRoute(t *testing.T) {
	gameCatalog := newFullCatalog(t)
	saveEngine := saveengine.New()
	unlocked := true
	const regionKey = "limgrave_the_first_step"

	t.Run("conforms to endpoint mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeSetRegionRouteFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		body, _ := json.Marshal(setRegionUnlockedRequest{
			RegionKind: "region", RegionKey: regionKey,
			Unlocked: &unlocked, ExpectedRevision: "0",
		})
		target := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/0/regions/unlock"
		request := httptest.NewRequest(http.MethodPut, target, bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
		assertOK(t, recorder, target)

		var got world.SetRegionUnlockedResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		direct, err := savesession.LoadSave(saveEngine, writeSetRegionRouteFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave direct session: %v", err)
		}
		want, err := world.SetRegionUnlocked(saveEngine, gameCatalog,
			direct.SaveSessionID, 0, "region", regionKey, true, "0")
		if err != nil {
			t.Fatalf("world.SetRegionUnlocked: %v", err)
		}
		want.SaveSessionID = got.SaveSessionID
		if got != want {
			t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
		}
	})

	t.Run("rejects invalid bodies", func(t *testing.T) {
		for name, body := range map[string]string{
			"missing unlocked": `{"regionKind":"region","regionKey":"` + regionKey + `","expectedRevision":"0"}`,
			"unknown field":    `{"regionKind":"region","regionKey":"` + regionKey + `","unlocked":true,"expectedRevision":"0","extra":1}`,
		} {
			t.Run(name, func(t *testing.T) {
				session, err := savesession.LoadSave(saveEngine, writeSetRegionRouteFixture(t), "")
				if err != nil {
					t.Fatalf("LoadSave: %v", err)
				}
				target := "/api/v1/save-sessions/" + session.SaveSessionID +
					"/characters/0/regions/unlock"
				request := httptest.NewRequest(http.MethodPut, target, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				newHandler(gameCatalog, testApplicationVersion, saveEngine).
					ServeHTTP(recorder, request)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
				}
				info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
				if err != nil {
					t.Fatalf("GetSessionInfo: %v", err)
				}
				if info.UnsavedChanges {
					t.Errorf("rejected body dirtied the session: %+v", info)
				}
			})
		}
	})
}

// writeSetTutorialRouteFixture is the cookbooks fixture with a TutorialData
// payload large enough to hold IDs. Raising the declared size shifts the event
// flag bitfield behind it, which this route neither reads nor writes.
func writeSetTutorialRouteFixture(t *testing.T) string {
	t.Helper()

	path := writeCookbooksFixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	tutorialAt := gesturesRouteSlotDataBase + gesturesRouteAnchorAt +
		cookbooksRouteSectionAt - cookbooksRouteScalarsSize -
		cookbooksRouteTutorialSize - cookbooksRouteDynamicHeader
	binary.LittleEndian.PutUint32(data[tutorialAt+4:], 0x20)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestSetTutorialUnlockedRoute(t *testing.T) {
	gameCatalog := newFullCatalog(t)
	saveEngine := saveengine.New()
	unlocked := true
	const tutorialKey = "2010"

	t.Run("conforms to endpoint mutation", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeSetTutorialRouteFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		body, _ := json.Marshal(setTutorialUnlockedRequest{
			TutorialKind: "tutorial", TutorialKey: tutorialKey,
			Unlocked: &unlocked, ExpectedRevision: "0",
		})
		target := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/0/tutorials/unlock"
		request := httptest.NewRequest(http.MethodPut, target, bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
		assertOK(t, recorder, target)

		var got world.SetTutorialUnlockedResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		direct, err := savesession.LoadSave(saveEngine, writeSetTutorialRouteFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave direct session: %v", err)
		}
		want, err := world.SetTutorialUnlocked(saveEngine, gameCatalog,
			direct.SaveSessionID, 0, "tutorial", tutorialKey, true, "0")
		if err != nil {
			t.Fatalf("world.SetTutorialUnlocked: %v", err)
		}
		want.SaveSessionID = got.SaveSessionID
		if got != want {
			t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
		}
	})

	t.Run("rejects invalid bodies", func(t *testing.T) {
		for name, body := range map[string]string{
			"missing unlocked": `{"tutorialKind":"tutorial","tutorialKey":"` + tutorialKey + `","expectedRevision":"0"}`,
			"unknown field":    `{"tutorialKind":"tutorial","tutorialKey":"` + tutorialKey + `","unlocked":true,"expectedRevision":"0","extra":1}`,
		} {
			t.Run(name, func(t *testing.T) {
				session, err := savesession.LoadSave(saveEngine, writeSetTutorialRouteFixture(t), "")
				if err != nil {
					t.Fatalf("LoadSave: %v", err)
				}
				target := "/api/v1/save-sessions/" + session.SaveSessionID +
					"/characters/0/tutorials/unlock"
				request := httptest.NewRequest(http.MethodPut, target, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				newHandler(gameCatalog, testApplicationVersion, saveEngine).
					ServeHTTP(recorder, request)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
				}
				info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
				if err != nil {
					t.Fatalf("GetSessionInfo: %v", err)
				}
				if info.UnsavedChanges {
					t.Errorf("rejected body dirtied the session: %+v", info)
				}
			})
		}
	})
}

func TestQuestsRouteMatchesTheGetter(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/quests"

	want, err := world.GetQuests(saveEngine, gameCatalog, session.SaveSessionID, 0, "quest", "")
	if err != nil {
		t.Fatalf("world.GetQuests: %v", err)
	}
	if !want.Active || len(want.Quests) != 36 {
		t.Fatalf("fixture result = active %t with %d quests, want true/36",
			want.Active, len(want.Quests))
	}

	target := base + "?questKind=quest"
	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	assertOK(t, recorder, target)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("quests route body differs from the GetQuests result")
	}

	// The route hands both query values through unchanged, so the endpoint's own
	// strict validation decides. An omitted questKind is a rejected request, not
	// a defaulted one.
	for name, query := range map[string]string{
		"missing questKind": "",
		"wrong questKind":   "?questKind=item",
		"unknown questKey":  "?questKind=quest&questKey=unknown_npc",
	} {
		t.Run(name, func(t *testing.T) {
			rejected := httptest.NewRecorder()
			newHandler(gameCatalog, testApplicationVersion, saveEngine).
				ServeHTTP(rejected, httptest.NewRequest(http.MethodGet, base+query, nil))
			if rejected.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %q)", rejected.Code, rejected.Body.String())
			}
		})
	}
}

func TestSetQuestStepRoute(t *testing.T) {
	saveEngine := saveengine.New()
	gameCatalog := newFullCatalog(t)

	const (
		questKey = "brother_corhyn"
		stepKey  = "legacy_000"
	)

	t.Run("applies the quest step", func(t *testing.T) {
		session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		target := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/0/quests/step"
		body := `{"questKind":"quest","questKey":"` + questKey + `","stepKind":"quest_step","stepKey":"` + stepKey + `","expectedRevision":"0"}`
		request := httptest.NewRequest(http.MethodPut, target, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).
			ServeHTTP(recorder, request)
		assertOK(t, recorder, target)

		var got world.SetQuestStepResult
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode route result: %v", err)
		}

		directSession, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
		if err != nil {
			t.Fatalf("LoadSave direct session: %v", err)
		}
		want, err := world.SetQuestStep(
			saveEngine,
			gameCatalog,
			directSession.SaveSessionID,
			0,
			"quest",
			questKey,
			"quest_step",
			stepKey,
			"0",
		)
		if err != nil {
			t.Fatalf("world.SetQuestStep: %v", err)
		}
		want.SaveSessionID = got.SaveSessionID
		if got != want {
			t.Errorf("route result = %+v, want direct endpoint result %+v", got, want)
		}
	})

	t.Run("rejects invalid bodies", func(t *testing.T) {
		for name, body := range map[string]string{
			"missing questKey": `{"questKind":"quest","stepKind":"quest_step","stepKey":"` + stepKey + `","expectedRevision":"0"}`,
			"missing stepKey":  `{"questKind":"quest","questKey":"` + questKey + `","stepKind":"quest_step","expectedRevision":"0"}`,
			"unknown field":    `{"questKind":"quest","questKey":"` + questKey + `","stepKind":"quest_step","stepKey":"` + stepKey + `","expectedRevision":"0","extra":1}`,
		} {
			t.Run(name, func(t *testing.T) {
				session, err := savesession.LoadSave(saveEngine, writeCookbooksFixture(t), "")
				if err != nil {
					t.Fatalf("LoadSave: %v", err)
				}
				target := "/api/v1/save-sessions/" + session.SaveSessionID +
					"/characters/0/quests/step"
				request := httptest.NewRequest(http.MethodPut, target, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				newHandler(gameCatalog, testApplicationVersion, saveEngine).
					ServeHTTP(recorder, request)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
				}
				info, err := saveEngine.GetSessionInfo(session.SaveSessionID)
				if err != nil {
					t.Fatalf("GetSessionInfo: %v", err)
				}
				if info.UnsavedChanges {
					t.Errorf("rejected body dirtied the session: %+v", info)
				}
			})
		}
	})
}

func writeFavoritesSwaggerFixture(t *testing.T, activeSlots map[int]bool) string {
	t.Helper()
	const (
		favPCUserData10Offset = int64(pcHeaderSize) + 10*0x280010
		favPCUserData10Size   = 0x60010
	)
	data := make([]byte, favPCUserData10Offset+favPCUserData10Size)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[0x0C:], 12)
	base := favPCUserData10Offset + 0x10

	// Character 0 appearance data
	data[base+0x1954] = 1 // active flag for slot 0
	slotBase := int64(pcHeaderSize) + 0x10
	anchor := slotBase + 0x1000
	copy(data[anchor:], []byte{
		0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	})
	data[anchor-249] = 1 // male
	data[anchor-245] = 1
	faceAt := slotBase + 0x2000
	copy(data[faceAt:], []byte{0xFF, 0xFF, 0xFF, 0xFF, 'F', 'A', 'C', 'E'})
	binary.LittleEndian.PutUint32(data[faceAt+0x08:], 4)
	binary.LittleEndian.PutUint32(data[faceAt+0x0C:], 0x120)

	for slot := 0; slot < 15; slot++ {
		if activeSlots != nil && activeSlots[slot] {
			slotAt := base + 0x154 + int64(slot)*0x130
			binary.LittleEndian.PutUint16(data[slotAt:], 0xFACE)
			binary.LittleEndian.PutUint32(data[slotAt+4:], 0x11D0)
			copy(data[slotAt+0x18:], []byte("FACE"))
			binary.LittleEndian.PutUint32(data[slotAt+0x1C:], 4)
			binary.LittleEndian.PutUint32(data[slotAt+0x20:], 0x120)
		}
	}

	path := filepath.Join(t.TempDir(), "swagger_favorites.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestGetFavoritePresetsRoute(t *testing.T) {
	saveEngine := saveengine.New()
	active := map[int]bool{
		0: true,
		3: true,
	}
	session, err := savesession.LoadSave(saveEngine, writeFavoritesSwaggerFixture(t, active), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// 1. Success and route receipt for all presets
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/favorite-presets"
	recorder := doSave(t, saveEngine, http.MethodGet, target, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	direct, err := favorites.GetFavoritePresets(saveEngine, session.SaveSessionID, nil)
	if err != nil {
		t.Fatalf("direct GetFavoritePresets: %v", err)
	}
	var routeResult favorites.GetFavoritePresetsResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &routeResult); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(routeResult, direct) {
		t.Fatalf("routeResult = %+v, direct = %+v", routeResult, direct)
	}

	// 2. Filter by favoriteSlotID
	filterTarget := "/api/v1/save-sessions/" + session.SaveSessionID + "/favorite-presets?favoriteSlotID=3"
	filterRec := doSave(t, saveEngine, http.MethodGet, filterTarget, "")
	if filterRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", filterRec.Code, filterRec.Body.String())
	}
	var filteredResult favorites.GetFavoritePresetsResult
	if err := json.Unmarshal(filterRec.Body.Bytes(), &filteredResult); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(filteredResult.Presets) != 1 || filteredResult.Presets[0].FavoriteSlotID != 3 || !filteredResult.Presets[0].Active {
		t.Fatalf("filteredResult = %+v, want slot 3 active", filteredResult)
	}

	// 3. Filter inactive slot
	filterEmptyTarget := "/api/v1/save-sessions/" + session.SaveSessionID + "/favorite-presets?favoriteSlotID=1"
	filterEmptyRec := doSave(t, saveEngine, http.MethodGet, filterEmptyTarget, "")
	if filterEmptyRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", filterEmptyRec.Code, filterEmptyRec.Body.String())
	}
	var filteredEmptyResult favorites.GetFavoritePresetsResult
	if err := json.Unmarshal(filterEmptyRec.Body.Bytes(), &filteredEmptyResult); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(filteredEmptyResult.Presets) != 1 || filteredEmptyResult.Presets[0].FavoriteSlotID != 1 || filteredEmptyResult.Presets[0].Active {
		t.Fatalf("filteredEmptyResult = %+v, want slot 1 inactive", filteredEmptyResult)
	}

	// 4. Invalid query parameter favoriteSlotID (non-integer, out of range)
	for _, invalidTarget := range []string{
		"/api/v1/save-sessions/" + session.SaveSessionID + "/favorite-presets?favoriteSlotID=abc",
		"/api/v1/save-sessions/" + session.SaveSessionID + "/favorite-presets?favoriteSlotID=-1",
		"/api/v1/save-sessions/" + session.SaveSessionID + "/favorite-presets?favoriteSlotID=15",
	} {
		rec := doSave(t, saveEngine, http.MethodGet, invalidTarget, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %q)", invalidTarget, rec.Code, rec.Body.String())
		}
	}
}

func TestDeleteFavoritePresetRoute(t *testing.T) {
	saveEngine := saveengine.New()
	active := map[int]bool{
		0: true,
		2: true,
	}
	session, err := savesession.LoadSave(saveEngine, writeFavoritesSwaggerFixture(t, active), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// 1. Success and route receipt
	reqBody := `{"expectedRevision":"0"}`
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/favorite-presets/2"
	recorder := doSave(t, saveEngine, http.MethodDelete, target, reqBody)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	var routeResult favorites.DeleteFavoritePresetResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &routeResult); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if routeResult.SaveSessionID != session.SaveSessionID || routeResult.SaveRevision != "1" || routeResult.FavoriteSlotID != 2 {
		t.Fatalf("routeResult = %+v, want slot 2 revision 1", routeResult)
	}

	// 2. Invalid path favoriteSlotID (non-integer, out of range)
	for _, invalidTarget := range []string{
		"/api/v1/save-sessions/" + session.SaveSessionID + "/favorite-presets/abc",
		"/api/v1/save-sessions/" + session.SaveSessionID + "/favorite-presets/-1",
		"/api/v1/save-sessions/" + session.SaveSessionID + "/favorite-presets/15",
	} {
		rec := doSave(t, saveEngine, http.MethodDelete, invalidTarget, `{"expectedRevision":"1"}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %q)", invalidTarget, rec.Code, rec.Body.String())
		}
	}

	// 3. Missing expectedRevision, empty body, unknown fields
	for _, badBody := range []string{
		"",
		"{}",
		`{"expectedRevision":"1","extra":123}`,
	} {
		rec := doSave(t, saveEngine, http.MethodDelete, "/api/v1/save-sessions/"+session.SaveSessionID+"/favorite-presets/1", badBody)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %q: status = %d, want 400 (body %q)", badBody, rec.Code, rec.Body.String())
		}
	}
}

func TestSetFavoritePresetRoute(t *testing.T) {
	saveEngine := saveengine.New()
	active := map[int]bool{
		0: true,
		2: true,
	}
	session, err := savesession.LoadSave(saveEngine, writeFavoritesSwaggerFixture(t, active), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// 1. Success and route receipt
	reqBody := `{"sourceCharacterID":0,"expectedRevision":"0"}`
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/favorite-presets/3"
	recorder := doSave(t, saveEngine, http.MethodPut, target, reqBody)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	var routeResult favorites.SetFavoritePresetResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &routeResult); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if routeResult.SaveSessionID != session.SaveSessionID || routeResult.SaveRevision != "1" ||
		routeResult.FavoriteSlotID != 3 || routeResult.SourceCharacterID != 0 {
		t.Fatalf("routeResult = %+v, want slot 3 character 0 revision 1", routeResult)
	}

	// 2. Invalid path favoriteSlotID (non-integer, out of range)
	for _, invalidTarget := range []string{
		"/api/v1/save-sessions/" + session.SaveSessionID + "/favorite-presets/abc",
		"/api/v1/save-sessions/" + session.SaveSessionID + "/favorite-presets/-1",
		"/api/v1/save-sessions/" + session.SaveSessionID + "/favorite-presets/15",
	} {
		rec := doSave(t, saveEngine, http.MethodPut, invalidTarget, `{"sourceCharacterID":0,"expectedRevision":"1"}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %q)", invalidTarget, rec.Code, rec.Body.String())
		}
	}

	// 3. Missing sourceCharacterID, missing expectedRevision, empty body, unknown fields
	for _, badBody := range []string{
		"",
		"{}",
		`{"expectedRevision":"1"}`,
		`{"sourceCharacterID":0}`,
		`{"sourceCharacterID":0,"expectedRevision":"1","extra":123}`,
	} {
		rec := doSave(t, saveEngine, http.MethodPut, "/api/v1/save-sessions/"+session.SaveSessionID+"/favorite-presets/1", badBody)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %q: status = %d, want 400 (body %q)", badBody, rec.Code, rec.Body.String())
		}
	}
}

func TestApplyFavoritePresetRoute(t *testing.T) {
	path := writeFavoritesSwaggerFixture(t, map[int]bool{3: true})

	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, path, "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// 1. Success apply favorite preset 3 to character 0
	reqBody := `{"favoriteSlotID":3,"expectedRevision":"0"}`
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/appearance/favorite-preset"
	recorder := doSave(t, saveEngine, http.MethodPut, target, reqBody)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	var routeResult favorites.ApplyFavoritePresetResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &routeResult); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if routeResult.SaveSessionID != session.SaveSessionID || routeResult.SaveRevision != "1" ||
		routeResult.CharacterID != 0 || routeResult.FavoriteSlotID != 3 {
		t.Fatalf("routeResult = %+v, want character 0 slot 3 revision 1", routeResult)
	}

	// 2. Invalid path characterID
	for _, invalidTarget := range []string{
		"/api/v1/save-sessions/" + session.SaveSessionID + "/characters/abc/appearance/favorite-preset",
		"/api/v1/save-sessions/" + session.SaveSessionID + "/characters/-1/appearance/favorite-preset",
		"/api/v1/save-sessions/" + session.SaveSessionID + "/characters/10/appearance/favorite-preset",
	} {
		rec := doSave(t, saveEngine, http.MethodPut, invalidTarget, `{"favoriteSlotID":3,"expectedRevision":"1"}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %q)", invalidTarget, rec.Code, rec.Body.String())
		}
	}

	// 3. Bad body (missing favoriteSlotID, missing expectedRevision, empty, extra fields)
	for _, badBody := range []string{
		"",
		"{}",
		`{"expectedRevision":"1"}`,
		`{"favoriteSlotID":3}`,
		`{"favoriteSlotID":3,"expectedRevision":"1","extra":123}`,
	} {
		rec := doSave(t, saveEngine, http.MethodPut, target, badBody)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %q: status = %d, want 400 (body %q)", badBody, rec.Code, rec.Body.String())
		}
	}
}

func TestGetBuildTemplatesRoute_EmptyLibrary(t *testing.T) {
	storeDir := filepath.Join(t.TempDir(), "templates")
	store := buildtemplates.NewStore(storeDir)
	saveEngine := saveengine.New()

	handler := newHandlerWithTemplatesStore(newPrototypeCatalog(t), testApplicationVersion, saveEngine, store)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/build-templates", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	var result struct {
		Templates []any `json:"templates"`
		Total     int   `json:"total"`
		Page      int   `json:"page"`
		PageSize  int   `json:"pageSize"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Total != 0 || len(result.Templates) != 0 || result.Page != 1 || result.PageSize != 50 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestGetBuildTemplatesRoute_FilteringAndPagination(t *testing.T) {
	storeDir := t.TempDir()
	indexJSON := `{
  "version": 1,
  "entries": [
    {
      "id": "tpl-1",
      "name": "Bleed Build",
      "description": "PvP Bleed setup",
      "tags": ["pvp", "bleed"],
      "filename": "t1.json",
      "createdAt": "2026-05-17T10:00:00Z",
      "updatedAt": "2026-05-17T12:00:00Z",
      "inventoryItems": 10,
      "storageItems": 2,
      "warnings": 0,
      "version": 2,
      "selectedSections": ["inventory.workspace"]
    },
    {
      "id": "tpl-2",
      "name": "Mage Build",
      "description": "PvE Sorcery",
      "tags": ["pve", "magic"],
      "filename": "t2.json",
      "createdAt": "2026-05-17T10:00:00Z",
      "updatedAt": "2026-05-17T11:00:00Z",
      "inventoryItems": 8,
      "storageItems": 0,
      "warnings": 0,
      "version": 1
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(storeDir, buildtemplates.IndexFileName), []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store := buildtemplates.NewStore(storeDir)
	saveEngine := saveengine.New()
	handler := newHandlerWithTemplatesStore(newPrototypeCatalog(t), testApplicationVersion, saveEngine, store)

	// 1. Search + repeated Tags query (?tags=pvp&tags=bleed)
	{
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/build-templates?search=bleed&tags=pvp&tags=bleed&page=1&pageSize=10", nil)
		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
		}
		var result struct {
			Templates []buildtemplates.TemplateMetadata `json:"templates"`
			Total     int                               `json:"total"`
			Page      int                               `json:"page"`
			PageSize  int                               `json:"pageSize"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.Total != 1 || len(result.Templates) != 1 || result.Templates[0].TemplateID != "tpl-1" {
			t.Fatalf("unexpected result: %+v", result)
		}
	}

	// 2. Comma-separated value is NOT split into multiple tags; treated as literal tag
	{
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/build-templates?tags=pvp,bleed", nil)
		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
		}
		var result struct {
			Total int `json:"total"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.Total != 0 {
			t.Fatalf("expected 0 matches for unsplit literal tag 'pvp,bleed', got %d", result.Total)
		}
	}

	// 3. Empty tag elements return HTTP 400 Bad Request
	for _, emptyTagQuery := range []string{
		"/api/v1/build-templates?tags=",
		"/api/v1/build-templates?tags=pvp&tags=",
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, emptyTagQuery, nil)
		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %q)", emptyTagQuery, recorder.Code, recorder.Body.String())
		}
	}

	// 4. Invalid paging parameters
	for _, invalidQuery := range []string{
		"/api/v1/build-templates?page=-1",
		"/api/v1/build-templates?page=abc",
		"/api/v1/build-templates?pageSize=-1",
		"/api/v1/build-templates?pageSize=xyz",
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, invalidQuery, nil)
		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %q)", invalidQuery, recorder.Code, recorder.Body.String())
		}
	}
}

func TestGetBuildTemplatesRoute_NoExplicitStoreReturns404(t *testing.T) {
	saveEngine := saveengine.New()
	handler := newHandler(newPrototypeCatalog(t), testApplicationVersion, saveEngine)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/build-templates", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 when no explicit store provided", recorder.Code)
	}
}

func TestGetBuildTemplatesRoute_ExternalBindReturns404(t *testing.T) {
	// External bind passes saveEngine = nil
	handler := newHandler(newPrototypeCatalog(t), testApplicationVersion, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/build-templates", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 on external bind", recorder.Code)
	}
}

func TestGetBuildTemplateRoute_Success(t *testing.T) {
	dir := t.TempDir()
	indexJSON := `{
  "version": 1,
  "entries": [
    {
      "id": "tpl-123",
      "name": "PvP Meta",
      "filename": "pvp.json",
      "createdAt": "2026-08-17T12:00:00Z",
      "updatedAt": "2026-08-17T12:00:00Z",
      "version": 2
    }
  ]
}`
	payloadJSON := `{
  "schema": "saveforge.build-template",
  "version": 2,
  "createdAt": "2026-08-17T12:00:00Z",
  "metadata": {
    "name": "PvP Meta"
  },
  "selection": {
    "items": true
  },
  "sections": {
    "items": {
      "entries": [
        {
          "entryID": "claymore-1",
          "itemID": 1000,
          "category": "melee_armaments",
          "quantity": 1,
          "location": "inventory"
        }
      ]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, buildtemplates.IndexFileName), []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pvp.json"), []byte(payloadJSON), 0644); err != nil {
		t.Fatalf("WriteFile payload: %v", err)
	}

	store := buildtemplates.NewStore(dir)
	saveEngine := saveengine.New()
	handler := newHandlerWithTemplatesStore(newPrototypeCatalog(t), testApplicationVersion, saveEngine, store)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/build-templates/tpl-123", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	var result templates.GetBuildTemplateResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	tpl := result.Template
	if tpl == nil {
		t.Fatal("response carries no template")
	}
	if tpl.Version != 2 || tpl.Sections.Items == nil || len(tpl.Sections.Items.Entries) != 1 {
		t.Fatalf("unexpected response payload: %+v", tpl)
	}
	// The index entry predates the persistent revision counter.
	if result.TemplateRevision != "0" {
		t.Errorf("templateRevision = %q, want \"0\"", result.TemplateRevision)
	}
}

func TestGetBuildTemplateRoute_NotFound(t *testing.T) {
	dir := t.TempDir()
	store := buildtemplates.NewStore(dir)
	saveEngine := saveengine.New()
	handler := newHandlerWithTemplatesStore(newPrototypeCatalog(t), testApplicationVersion, saveEngine, store)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/build-templates/tpl-nonexistent", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body %q)", recorder.Code, recorder.Body.String())
	}
}

func TestGetBuildTemplateRoute_CorruptPayloadReturns400(t *testing.T) {
	dir := t.TempDir()
	indexJSON := `{
  "version": 1,
  "entries": [
    {
      "id": "tpl-corrupt",
      "name": "Corrupt",
      "filename": "corrupt.json",
      "createdAt": "2026-08-17T12:00:00Z",
      "updatedAt": "2026-08-17T12:00:00Z"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, buildtemplates.IndexFileName), []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "corrupt.json"), []byte(`INVALID JSON`), 0644); err != nil {
		t.Fatalf("WriteFile payload: %v", err)
	}

	store := buildtemplates.NewStore(dir)
	saveEngine := saveengine.New()
	handler := newHandlerWithTemplatesStore(newPrototypeCatalog(t), testApplicationVersion, saveEngine, store)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/build-templates/tpl-corrupt", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
	}
}

func TestGetBuildTemplateRoute_NoExplicitStoreReturns404(t *testing.T) {
	saveEngine := saveengine.New()
	handler := newHandler(newPrototypeCatalog(t), testApplicationVersion, saveEngine)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/build-templates/tpl-123", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 when no explicit store provided", recorder.Code)
	}
}

func TestGetBuildTemplateRoute_ExternalBindReturns404(t *testing.T) {
	handler := newHandler(newPrototypeCatalog(t), testApplicationVersion, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/build-templates/tpl-123", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 on external bind", recorder.Code)
	}
}

// newDeleteBuildTemplateHandler builds a loopback explorer over a one-entry
// library at revision 7 and returns the handler plus the store directory.
func newDeleteBuildTemplateHandler(t *testing.T) (http.Handler, string) {
	t.Helper()
	dir := t.TempDir()
	indexJSON := `{
  "version": 1,
  "entries": [
    {
      "id": "tpl-123",
      "name": "PvP Meta",
      "filename": "pvp.json",
      "createdAt": "2026-08-17T12:00:00Z",
      "updatedAt": "2026-08-17T12:00:00Z",
      "revision": 7
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, buildtemplates.IndexFileName), []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pvp.json"), []byte("payload"), 0644); err != nil {
		t.Fatalf("WriteFile payload: %v", err)
	}
	handler := newHandlerWithTemplatesStore(
		newPrototypeCatalog(t), testApplicationVersion, saveengine.New(), buildtemplates.NewStore(dir),
	)
	return handler, dir
}

func deleteBuildTemplate(t *testing.T, handler http.Handler, templateID string, body string) *httptest.ResponseRecorder {
	t.Helper()
	target := "/api/v1/build-templates/" + templateID
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(http.MethodDelete, target, nil)
	} else {
		request = httptest.NewRequest(http.MethodDelete, target, bytes.NewReader([]byte(body)))
	}
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestDeleteBuildTemplateRoute_Success(t *testing.T) {
	handler, dir := newDeleteBuildTemplateHandler(t)

	recorder := deleteBuildTemplate(t, handler, "tpl-123", `{"templateRevision":"7"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	var result templates.DeleteBuildTemplateResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if result.TemplateID != "tpl-123" {
		t.Errorf("templateID = %q, want %q", result.TemplateID, "tpl-123")
	}
	if _, err := os.Stat(filepath.Join(dir, "pvp.json")); !os.IsNotExist(err) {
		t.Errorf("payload still present: %v", err)
	}
}

func TestDeleteBuildTemplateRoute_RejectsMalformedBodies(t *testing.T) {
	for name, body := range map[string]string{
		"no body":                 "",
		"no templateRevision":     `{}`,
		"unknown field":           `{"templateRevision":"7","force":true}`,
		"non-canonical revision":  `{"templateRevision":"07"}`,
		"revision as JSON number": `{"templateRevision":7}`,
		"trailing JSON document":  `{"templateRevision":"7"}{"extra":1}`,
	} {
		t.Run(name, func(t *testing.T) {
			handler, _ := newDeleteBuildTemplateHandler(t)
			recorder := deleteBuildTemplate(t, handler, "tpl-123", body)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestDeleteBuildTemplateRoute_StaleRevisionReturns409(t *testing.T) {
	handler, dir := newDeleteBuildTemplateHandler(t)

	recorder := deleteBuildTemplate(t, handler, "tpl-123", `{"templateRevision":"6"}`)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body %q)", recorder.Code, recorder.Body.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "pvp.json")); err != nil {
		t.Errorf("payload was removed on a stale revision: %v", err)
	}
}

func TestDeleteBuildTemplateRoute_NotFound(t *testing.T) {
	handler, _ := newDeleteBuildTemplateHandler(t)

	recorder := deleteBuildTemplate(t, handler, "tpl-nonexistent", `{"templateRevision":"0"}`)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body %q)", recorder.Code, recorder.Body.String())
	}
}

func TestDeleteBuildTemplateRoute_ExternalBindReturns404(t *testing.T) {
	handler := newHandler(newPrototypeCatalog(t), testApplicationVersion, nil)
	recorder := deleteBuildTemplate(t, handler, "tpl-123", `{"templateRevision":"7"}`)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 on external bind", recorder.Code)
	}
}

// newUpdateBuildTemplateHandler builds a loopback explorer over a one-entry
// library at revision 3 and returns the handler plus the store directory.
func newUpdateBuildTemplateHandler(t *testing.T) (http.Handler, string) {
	t.Helper()
	dir := t.TempDir()
	indexJSON := `{
  "version": 1,
  "entries": [
    {
      "id": "tpl-123",
      "name": "PvP Meta",
      "filename": "pvp.json",
      "createdAt": "2026-08-17T12:00:00Z",
      "updatedAt": "2026-08-17T12:00:00Z",
      "revision": 3
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, buildtemplates.IndexFileName), []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}
	payloadJSON := `{
  "schema": "saveforge.build-template",
  "version": 1,
  "createdAt": "2026-08-17T12:00:00Z",
  "metadata": {
    "name": "PvP Meta"
  },
  "sections": {
    "inventory.workspace": {
      "inventoryItems": [
        {
          "baseItemID": 100,
          "quantity": 1,
          "container": "inventory",
          "position": 0
        }
      ],
      "storageItems": []
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "pvp.json"), []byte(payloadJSON), 0644); err != nil {
		t.Fatalf("WriteFile payload: %v", err)
	}
	handler := newHandlerWithTemplatesStore(
		newPrototypeCatalog(t), testApplicationVersion, saveengine.New(), buildtemplates.NewStore(dir),
	)
	return handler, dir
}

func updateBuildTemplate(t *testing.T, handler http.Handler, templateID string, body string) *httptest.ResponseRecorder {
	t.Helper()
	target := "/api/v1/build-templates/" + templateID
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(http.MethodPut, target, nil)
	} else {
		request = httptest.NewRequest(http.MethodPut, target, bytes.NewReader([]byte(body)))
	}
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestUpdateBuildTemplateRoute_Success(t *testing.T) {
	handler, _ := newUpdateBuildTemplateHandler(t)

	body := `{
		"templateRevision": "3",
		"metadata": {
			"name": "Updated PvP Meta",
			"description": "New description",
			"tags": ["pvp", "meta", "bleed"]
		}
	}`
	recorder := updateBuildTemplate(t, handler, "tpl-123", body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	var result templates.UpdateBuildTemplateResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if result.TemplateID != "tpl-123" {
		t.Errorf("templateID = %q, want %q", result.TemplateID, "tpl-123")
	}
	if result.TemplateRevision != "4" {
		t.Errorf("templateRevision = %q, want %q", result.TemplateRevision, "4")
	}
}

func TestUpdateBuildTemplateRoute_RejectsMalformedBodies(t *testing.T) {
	for name, body := range map[string]string{
		"no body":                      "",
		"no templateRevision":          `{"metadata":{"name":"X"}}`,
		"neither metadata nor content": `{"templateRevision":"3"}`,
		"unknown field":                `{"templateRevision":"3","metadata":{"name":"X"},"extra":true}`,
		"non-canonical revision":       `{"templateRevision":"03","metadata":{"name":"X"}}`,
		"revision as JSON number":      `{"templateRevision":3,"metadata":{"name":"X"}}`,
		"trailing JSON document":       `{"templateRevision":"3","metadata":{"name":"X"}}{"extra":1}`,
		"invalid payload in content":   `{"templateRevision":"3","content":{"schema":"invalid"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			handler, _ := newUpdateBuildTemplateHandler(t)
			recorder := updateBuildTemplate(t, handler, "tpl-123", body)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestUpdateBuildTemplateRoute_StaleRevisionReturns409(t *testing.T) {
	handler, _ := newUpdateBuildTemplateHandler(t)

	body := `{"templateRevision":"2","metadata":{"name":"Stale"}}`
	recorder := updateBuildTemplate(t, handler, "tpl-123", body)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body %q)", recorder.Code, recorder.Body.String())
	}
}

func TestUpdateBuildTemplateRoute_NotFound(t *testing.T) {
	handler, _ := newUpdateBuildTemplateHandler(t)

	body := `{"templateRevision":"0","metadata":{"name":"Missing"}}`
	recorder := updateBuildTemplate(t, handler, "tpl-nonexistent", body)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body %q)", recorder.Code, recorder.Body.String())
	}
}

func TestUpdateBuildTemplateRoute_ExternalBindReturns404(t *testing.T) {
	handler := newHandler(newPrototypeCatalog(t), testApplicationVersion, nil)
	body := `{"templateRevision":"3","metadata":{"name":"X"}}`
	recorder := updateBuildTemplate(t, handler, "tpl-123", body)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 on external bind", recorder.Code)
	}
}

func createBuildTemplate(t *testing.T, handler http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	target := "/api/v1/build-templates"
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(http.MethodPost, target, nil)
	} else {
		request = httptest.NewRequest(http.MethodPost, target, bytes.NewReader([]byte(body)))
	}
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestCreateBuildTemplateRoute_Success(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	templatesDir := t.TempDir()
	handler := newHandlerWithTemplatesStore(
		newFullCatalog(t),
		testApplicationVersion,
		saveEngine,
		buildtemplates.NewStore(templatesDir),
	)

	body := fmt.Sprintf(`{
		"saveSessionID": %q,
		"sourceCharacterID": 0,
		"name": "Swagger Route Test Build",
		"description": "Created from swagger route test",
		"tags": ["swagger", "test"],
		"selection": {
			"spells": {
				"spell1": true
			}
		}
	}`, session.SaveSessionID)

	recorder := createBuildTemplate(t, handler, body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	var result templates.CreateBuildTemplateResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if result.TemplateID == "" {
		t.Errorf("expected non-empty templateID")
	}
	if result.TemplateRevision != "1" {
		t.Errorf("templateRevision = %q, want %q", result.TemplateRevision, "1")
	}
}

func TestCreateBuildTemplateRoute_RejectsMalformedBodies(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	templatesDir := t.TempDir()
	handler := newHandlerWithTemplatesStore(
		newPrototypeCatalog(t),
		testApplicationVersion,
		saveEngine,
		buildtemplates.NewStore(templatesDir),
	)

	for name, body := range map[string]string{
		"no body":                         "",
		"missing saveSessionID":           `{"sourceCharacterID":0,"name":"X","selection":{"stats":true}}`,
		"missing sourceCharacterID":       fmt.Sprintf(`{"saveSessionID":%q,"name":"X","selection":{"stats":true}}`, session.SaveSessionID),
		"missing name":                    fmt.Sprintf(`{"saveSessionID":%q,"sourceCharacterID":0,"selection":{"stats":true}}`, session.SaveSessionID),
		"empty selection":                 fmt.Sprintf(`{"saveSessionID":%q,"sourceCharacterID":0,"name":"X","selection":{}}`, session.SaveSessionID),
		"unsupported field":               fmt.Sprintf(`{"saveSessionID":%q,"sourceCharacterID":0,"name":"X","selection":{"profile":{"runes":true}}}`, session.SaveSessionID),
		"spells.All shortcut is rejected": fmt.Sprintf(`{"saveSessionID":%q,"sourceCharacterID":0,"name":"X","selection":{"spells":true}}`, session.SaveSessionID),
		"unknown json field":              fmt.Sprintf(`{"saveSessionID":%q,"sourceCharacterID":0,"name":"X","selection":{"stats":true},"extra":true}`, session.SaveSessionID),
		"trailing json document":          fmt.Sprintf(`{"saveSessionID":%q,"sourceCharacterID":0,"name":"X","selection":{"stats":true}}{"extra":1}`, session.SaveSessionID),
	} {
		t.Run(name, func(t *testing.T) {
			recorder := createBuildTemplate(t, handler, body)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestCreateBuildTemplateRoute_NotFound(t *testing.T) {
	templatesDir := t.TempDir()
	handler := newHandlerWithTemplatesStore(
		newPrototypeCatalog(t),
		testApplicationVersion,
		saveengine.New(),
		buildtemplates.NewStore(templatesDir),
	)

	body := `{"saveSessionID":"nonexistent-session","sourceCharacterID":0,"name":"Test","selection":{"stats":true}}`
	recorder := createBuildTemplate(t, handler, body)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body %q)", recorder.Code, recorder.Body.String())
	}
}

func TestCreateBuildTemplateRoute_ExternalBindReturns404(t *testing.T) {
	handler := newHandler(newPrototypeCatalog(t), testApplicationVersion, nil)
	body := `{"saveSessionID":"session-1","sourceCharacterID":0,"name":"Test","selection":{"stats":true}}`
	recorder := createBuildTemplate(t, handler, body)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 on external bind", recorder.Code)
	}
}

func getBuildTemplatePreview(t *testing.T, handler http.Handler, templateID string, body string) *httptest.ResponseRecorder {
	t.Helper()
	target := fmt.Sprintf("/api/v1/build-templates/%s/preview", templateID)
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(http.MethodPost, target, nil)
	} else {
		request = httptest.NewRequest(http.MethodPost, target, bytes.NewReader([]byte(body)))
	}
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestGetBuildTemplatePreviewRoute_Success(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	templatesDir := t.TempDir()
	store := buildtemplates.NewStore(templatesDir)
	name := "Preview Route Build"
	vig := uint32(20)
	mind := uint32(15)
	end := uint32(16)
	str := uint32(20)
	dex := uint32(18)
	intel := uint32(12)
	faith := uint32(12)
	arc := uint32(10)
	tplID, _, err := store.CreateTemplate(&buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: name,
		},
		Selection: &buildtemplates.TemplateSelection{
			Stats: &buildtemplates.SectionSelection{
				All: true,
			},
		},
		Sections: buildtemplates.TemplateSections{
			Stats: &buildtemplates.StatsSection{
				Vigor:        &vig,
				Mind:         &mind,
				Endurance:    &end,
				Strength:     &str,
				Dexterity:    &dex,
				Intelligence: &intel,
				Faith:        &faith,
				Arcane:       &arc,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	handler := newHandlerWithTemplatesStore(
		newPrototypeCatalog(t),
		testApplicationVersion,
		saveEngine,
		store,
	)

	body := fmt.Sprintf(`{
		"saveSessionID": %q,
		"characterID": 0
	}`, session.SaveSessionID)

	recorder := getBuildTemplatePreview(t, handler, tplID, body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	var result templates.GetBuildTemplatePreviewResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !result.Executable {
		t.Errorf("expected Executable=true")
	}
	if result.TemplateID != tplID {
		t.Errorf("TemplateID = %q, want %q", result.TemplateID, tplID)
	}
	if result.Plan.Stats == nil || result.Plan.Stats.Vigor == nil {
		t.Errorf("Plan.Stats.Vigor is nil")
	}
}

func TestGetBuildTemplatePreviewRoute_RejectsMalformedBodies(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	templatesDir := t.TempDir()
	handler := newHandlerWithTemplatesStore(
		newPrototypeCatalog(t),
		testApplicationVersion,
		saveEngine,
		buildtemplates.NewStore(templatesDir),
	)

	for name, body := range map[string]string{
		"no body":                "",
		"missing saveSessionID":  `{"characterID":0}`,
		"missing characterID":    fmt.Sprintf(`{"saveSessionID":%q}`, session.SaveSessionID),
		"unknown json field":     fmt.Sprintf(`{"saveSessionID":%q,"characterID":0,"extra":true}`, session.SaveSessionID),
		"trailing json document": fmt.Sprintf(`{"saveSessionID":%q,"characterID":0}{"extra":1}`, session.SaveSessionID),
	} {
		t.Run(name, func(t *testing.T) {
			recorder := getBuildTemplatePreview(t, handler, "tpl-123", body)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestGetBuildTemplatePreviewRoute_UnknownTemplate(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	templatesDir := t.TempDir()
	handler := newHandlerWithTemplatesStore(
		newPrototypeCatalog(t),
		testApplicationVersion,
		saveEngine,
		buildtemplates.NewStore(templatesDir),
	)

	body := fmt.Sprintf(`{"saveSessionID":%q,"characterID":0}`, session.SaveSessionID)
	recorder := getBuildTemplatePreview(t, handler, "tpl-nonexistent", body)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body %q)", recorder.Code, recorder.Body.String())
	}
}

func TestGetBuildTemplatePreviewRoute_UnknownSession(t *testing.T) {
	templatesDir := t.TempDir()
	store := buildtemplates.NewStore(templatesDir)
	name := "Route Build"
	vig := uint32(50)
	tplID, _, err := store.CreateTemplate(&buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: name,
		},
		Selection: &buildtemplates.TemplateSelection{
			Stats: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"vigor": true},
			},
		},
		Sections: buildtemplates.TemplateSections{
			Stats: &buildtemplates.StatsSection{
				Vigor: &vig,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	handler := newHandlerWithTemplatesStore(
		newPrototypeCatalog(t),
		testApplicationVersion,
		saveengine.New(),
		store,
	)

	body := `{"saveSessionID":"nonexistent-session","characterID":0}`
	recorder := getBuildTemplatePreview(t, handler, tplID, body)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body %q)", recorder.Code, recorder.Body.String())
	}
}

func TestGetBuildTemplatePreviewRoute_ExternalBindReturns404(t *testing.T) {
	handler := newHandler(newPrototypeCatalog(t), testApplicationVersion, nil)
	body := `{"saveSessionID":"session-1","characterID":0}`
	recorder := getBuildTemplatePreview(t, handler, "tpl-123", body)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 on external bind", recorder.Code)
	}
}

func applyBuildTemplate(t *testing.T, handler http.Handler, templateID string, body string) *httptest.ResponseRecorder {
	t.Helper()
	target := fmt.Sprintf("/api/v1/build-templates/%s/apply", templateID)
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(http.MethodPost, target, nil)
	} else {
		request = httptest.NewRequest(http.MethodPost, target, bytes.NewReader([]byte(body)))
	}
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestApplyBuildTemplateRoute_Success(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	templatesDir := t.TempDir()
	store := buildtemplates.NewStore(templatesDir)
	name := "Apply Route Build"
	vig := uint32(20)
	mind := uint32(15)
	end := uint32(16)
	str := uint32(20)
	dex := uint32(18)
	intel := uint32(12)
	faith := uint32(12)
	arc := uint32(10)
	tplID, _, err := store.CreateTemplate(&buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: name,
		},
		Selection: &buildtemplates.TemplateSelection{
			Stats: &buildtemplates.SectionSelection{
				All: true,
			},
		},
		Sections: buildtemplates.TemplateSections{
			Stats: &buildtemplates.StatsSection{
				Vigor:        &vig,
				Mind:         &mind,
				Endurance:    &end,
				Strength:     &str,
				Dexterity:    &dex,
				Intelligence: &intel,
				Faith:        &faith,
				Arcane:       &arc,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	handler := newHandlerWithTemplatesStore(
		newPrototypeCatalog(t),
		testApplicationVersion,
		saveEngine,
		store,
	)

	body := fmt.Sprintf(`{
		"saveSessionID": %q,
		"characterID": 0,
		"expectedRevision": "0"
	}`, session.SaveSessionID)

	recorder := applyBuildTemplate(t, handler, tplID, body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	var result templates.ApplyBuildTemplateResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if result.TemplateID != tplID {
		t.Errorf("TemplateID = %q, want %q", result.TemplateID, tplID)
	}
	if result.SaveRevision == "0" {
		t.Errorf("expected new SaveRevision, got %q", result.SaveRevision)
	}
	if result.Plan.Stats == nil || result.Plan.Stats.Vigor == nil {
		t.Errorf("Plan.Stats.Vigor is nil")
	}
}

func TestApplyBuildTemplateRoute_RejectsMalformedBodies(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	templatesDir := t.TempDir()
	handler := newHandlerWithTemplatesStore(
		newPrototypeCatalog(t),
		testApplicationVersion,
		saveEngine,
		buildtemplates.NewStore(templatesDir),
	)

	for name, body := range map[string]string{
		"no body":                  "",
		"missing saveSessionID":    `{"characterID":0,"expectedRevision":"0"}`,
		"missing characterID":      fmt.Sprintf(`{"saveSessionID":%q,"expectedRevision":"0"}`, session.SaveSessionID),
		"missing expectedRevision": fmt.Sprintf(`{"saveSessionID":%q,"characterID":0}`, session.SaveSessionID),
		"unknown json field":       fmt.Sprintf(`{"saveSessionID":%q,"characterID":0,"expectedRevision":"0","extra":true}`, session.SaveSessionID),
		"trailing json document":   fmt.Sprintf(`{"saveSessionID":%q,"characterID":0,"expectedRevision":"0"}{"extra":1}`, session.SaveSessionID),
	} {
		t.Run(name, func(t *testing.T) {
			recorder := applyBuildTemplate(t, handler, "tpl-123", body)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %q)", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestApplyBuildTemplateRoute_UnknownTemplate(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	templatesDir := t.TempDir()
	handler := newHandlerWithTemplatesStore(
		newPrototypeCatalog(t),
		testApplicationVersion,
		saveEngine,
		buildtemplates.NewStore(templatesDir),
	)

	body := fmt.Sprintf(`{"saveSessionID":%q,"characterID":0,"expectedRevision":"0"}`, session.SaveSessionID)
	recorder := applyBuildTemplate(t, handler, "tpl-nonexistent", body)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body %q)", recorder.Code, recorder.Body.String())
	}
}

func TestApplyBuildTemplateRoute_UnknownSession(t *testing.T) {
	templatesDir := t.TempDir()
	store := buildtemplates.NewStore(templatesDir)
	name := "Route Build"
	vig := uint32(50)
	tplID, _, err := store.CreateTemplate(&buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: name,
		},
		Selection: &buildtemplates.TemplateSelection{
			Stats: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"vigor": true},
			},
		},
		Sections: buildtemplates.TemplateSections{
			Stats: &buildtemplates.StatsSection{
				Vigor: &vig,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	handler := newHandlerWithTemplatesStore(
		newPrototypeCatalog(t),
		testApplicationVersion,
		saveengine.New(),
		store,
	)

	body := `{"saveSessionID":"nonexistent-session","characterID":0,"expectedRevision":"1"}`
	recorder := applyBuildTemplate(t, handler, tplID, body)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body %q)", recorder.Code, recorder.Body.String())
	}
}

func TestApplyBuildTemplateRoute_ExternalBindReturns404(t *testing.T) {
	handler := newHandler(newPrototypeCatalog(t), testApplicationVersion, nil)
	body := `{"saveSessionID":"session-1","characterID":0,"expectedRevision":"1"}`
	recorder := applyBuildTemplate(t, handler, "tpl-123", body)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 on external bind", recorder.Code)
	}
}

func TestApplyBuildTemplateRoute_NonCanonicalRevisionReturns400(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	templatesDir := t.TempDir()
	store := buildtemplates.NewStore(templatesDir)
	name := "Route Build"
	vig := uint32(50)
	tplID, _, err := store.CreateTemplate(&buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: name,
		},
		Selection: &buildtemplates.TemplateSelection{
			Stats: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"vigor": true},
			},
		},
		Sections: buildtemplates.TemplateSections{
			Stats: &buildtemplates.StatsSection{
				Vigor: &vig,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	handler := newHandlerWithTemplatesStore(
		newPrototypeCatalog(t),
		testApplicationVersion,
		saveEngine,
		store,
	)

	for _, badRev := range []string{"01", "+1", "rev1"} {
		body := fmt.Sprintf(`{"saveSessionID":%q,"characterID":0,"expectedRevision":%q}`, session.SaveSessionID, badRev)
		recorder := applyBuildTemplate(t, handler, tplID, body)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expectedRevision %q: status = %d, want 400 (body %q)", badRev, recorder.Code, recorder.Body.String())
		}
	}
}

func TestApplyBuildTemplateRoute_StaleRevisionReturns409(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	templatesDir := t.TempDir()
	store := buildtemplates.NewStore(templatesDir)
	name := "Route Build"
	vig := uint32(20)
	mind := uint32(15)
	end := uint32(16)
	str := uint32(20)
	dex := uint32(18)
	intel := uint32(12)
	faith := uint32(12)
	arc := uint32(10)
	tplID, _, err := store.CreateTemplate(&buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: name,
		},
		Selection: &buildtemplates.TemplateSelection{
			Stats: &buildtemplates.SectionSelection{
				All: true,
			},
		},
		Sections: buildtemplates.TemplateSections{
			Stats: &buildtemplates.StatsSection{
				Vigor:        &vig,
				Mind:         &mind,
				Endurance:    &end,
				Strength:     &str,
				Dexterity:    &dex,
				Intelligence: &intel,
				Faith:        &faith,
				Arcane:       &arc,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	handler := newHandlerWithTemplatesStore(
		newPrototypeCatalog(t),
		testApplicationVersion,
		saveEngine,
		store,
	)

	body := fmt.Sprintf(`{"saveSessionID":%q,"characterID":0,"expectedRevision":"9999"}`, session.SaveSessionID)
	recorder := applyBuildTemplate(t, handler, tplID, body)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body %q)", recorder.Code, recorder.Body.String())
	}
}

func TestSaveValidationReportRouteMatchesTheGetter(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/validation-report"

	want, err := diagnostics.GetSaveValidationReport(saveEngine, gameCatalog, session.SaveSessionID, 0, "")
	if err != nil {
		t.Fatalf("diagnostics.GetSaveValidationReport: %v", err)
	}
	if !want.Active {
		t.Fatal("the fixture slot is inactive, so the route would prove no validation at all")
	}

	// The route has to run against the same catalog, so it is served by a
	// handler built here instead of by the shared prototype-catalog helper.
	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	assertOK(t, recorder, target)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("validation report route body differs from the GetSaveValidationReport result")
	}
}

// TestSaveValidationReportRouteForwardsTheScope proves the query parameter
// reaches the endpoint unchanged: a narrowed scope must narrow the report, and
// an unknown one must be rejected rather than silently ignored.
func TestSaveValidationReportRouteForwardsTheScope(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/validation-report"

	serve := func(target string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		newHandler(gameCatalog, testApplicationVersion, saveEngine).
			ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		return recorder
	}

	recorder := serve(base + "?scope=stats")
	assertOK(t, recorder, base+"?scope=stats")
	var narrowed diagnostics.GetSaveValidationReportResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &narrowed); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(narrowed.Coverage) != 1 || narrowed.Coverage[0].Scope != "stats" {
		t.Fatalf("coverage = %+v, want the stats scope only", narrowed.Coverage)
	}

	for _, scope := range []string{"Stats", "world"} {
		if recorder := serve(base + "?scope=" + scope); recorder.Code != http.StatusBadRequest {
			t.Errorf("scope %q: status = %d, want 400 (body %q)",
				scope, recorder.Code, recorder.Body.String())
		}
	}
}

func TestSaveValidationReportRouteRejectsAMalformedCharacterID(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writePCFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}

	for _, raw := range []string{"one", " 0", "0x1"} {
		target := "/api/v1/save-sessions/" + session.SaveSessionID +
			"/characters/" + url.PathEscape(raw) + "/validation-report"
		if recorder := doSave(t, saveEngine, http.MethodGet, target, ""); recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %q)", target, recorder.Code, recorder.Body.String())
		}
	}
}

func TestSaveValidationReportRouteIsAbsentWithoutAnEngine(t *testing.T) {
	target := "/api/v1/save-sessions/any-session/characters/0/validation-report"
	recorder := doSave(t, nil, http.MethodGet, target, "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("%s: status = %d, want 404 (body %q)", target, recorder.Code, recorder.Body.String())
	}
}

func TestSaveValidationReportRouteIsDescribedInTheOpenAPIDocument(t *testing.T) {
	recorder := do(t, newPrototypeCatalog(t), "/openapi.json")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var document struct {
		Paths map[string]map[string]any `json:"paths"`
		Comps struct {
			Parameters map[string]struct {
				Name   string `json:"name"`
				In     string `json:"in"`
				Schema struct {
					Enum []string `json:"enum"`
				} `json:"schema"`
			} `json:"parameters"`
			Schemas map[string]any `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode openapi.json: %v", err)
	}

	const path = "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/validation-report"
	operation, exists := document.Paths[path]
	if !exists {
		t.Fatalf("openapi.json does not describe %s", path)
	}
	if _, hasGet := operation["get"]; !hasGet {
		t.Fatalf("openapi.json describes %s without a GET operation", path)
	}
	for _, name := range []string{"SaveValidationReport", "SaveValidationIssue", "SaveValidationScopeCoverage"} {
		if _, exists := document.Comps.Schemas[name]; !exists {
			t.Errorf("openapi.json does not describe the %s schema", name)
		}
	}

	scope, exists := document.Comps.Parameters["ValidationScope"]
	if !exists {
		t.Fatal("openapi.json does not describe the ValidationScope parameter")
	}
	if scope.Name != "scope" || scope.In != "query" {
		t.Errorf("ValidationScope = %s in %s, want scope in query", scope.Name, scope.In)
	}
	want := []string{"inventory", "storage", "stats", "equipment", "spells"}
	if !reflect.DeepEqual(scope.Schema.Enum, want) {
		t.Errorf("ValidationScope enum = %v, want %v", scope.Schema.Enum, want)
	}
}

// TestRepairPlanRouteMatchesTheGetter proves the transport is a thin shell: the
// route body must be exactly what GetRepairPlan returns for the same input, and
// the request body must reach the endpoint unchanged.
func TestRepairPlanRouteMatchesTheGetter(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/repair-plan"

	report, err := diagnostics.GetSaveValidationReport(saveEngine, gameCatalog, session.SaveSessionID, 0, "")
	if err != nil {
		t.Fatalf("diagnostics.GetSaveValidationReport: %v", err)
	}
	if len(report.Issues) == 0 {
		t.Skip("the fixture slot is clean, so it names no finding to plan for")
	}
	ids := make([]string, 0, len(report.Issues))
	for _, issue := range report.Issues {
		ids = append(ids, issue.ID)
	}

	want, err := diagnostics.GetRepairPlan(
		saveEngine, gameCatalog, session.SaveSessionID, 0, report.SaveRevision, ids)
	if err != nil {
		t.Fatalf("diagnostics.GetRepairPlan: %v", err)
	}

	body, err := json.Marshal(map[string]any{"saveRevision": report.SaveRevision, "issueIDs": ids})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
	assertOK(t, recorder, target)
	if !reflect.DeepEqual(decode(t, recorder.Body.Bytes()), marshalled(t, want)) {
		t.Fatal("repair plan route body differs from the GetRepairPlan result")
	}
}

// TestRepairPlanRouteRejectsAStaleRevision proves the route forwards the save
// revision rather than substituting the current one. A plan that could be built
// against any revision would silently address findings that have moved.
func TestRepairPlanRouteRejectsAStaleRevision(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "")
	if err != nil {
		t.Fatalf("savesession.LoadSave: %v", err)
	}
	gameCatalog := newFullCatalog(t)
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/repair-plan"

	body := []byte(`{"saveRevision":"999999","issueIDs":["stats:level_mismatch:0"]}`)
	request := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	newHandler(gameCatalog, testApplicationVersion, saveEngine).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a revision the session is not at", recorder.Code)
	}
}

// TestRepairPlanRouteIsDescribedInTheOpenAPIDocument keeps the contract the
// portal serves in step with the route.
func TestRepairPlanRouteIsDescribedInTheOpenAPIDocument(t *testing.T) {
	recorder := do(t, newPrototypeCatalog(t), "/openapi.json")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var document struct {
		Paths map[string]map[string]any `json:"paths"`
		Comps struct {
			Schemas map[string]any `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode openapi.json: %v", err)
	}

	const path = "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/repair-plan"
	operation, exists := document.Paths[path]
	if !exists {
		t.Fatalf("openapi.json does not describe %s", path)
	}
	if _, hasPost := operation["post"]; !hasPost {
		t.Fatalf("openapi.json describes %s without a POST operation", path)
	}
	for _, name := range []string{"RepairPlan", "RepairAction", "RepairRejection", "GetRepairPlanRequest"} {
		if _, exists := document.Comps.Schemas[name]; !exists {
			t.Errorf("openapi.json does not describe the %s schema", name)
		}
	}
}
