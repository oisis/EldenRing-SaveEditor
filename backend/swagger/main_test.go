package main

import (
	"bytes"
	"compress/flate"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/json"
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
	"testing"
	"unicode/utf16"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/appearance"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/application"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/character"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/equipment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/inventory"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/network"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/savesession"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
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

func newPrototypeCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	gameCatalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("gamecatalog.NewPrototype: %v", err)
	}
	return gameCatalog
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
// save-session route is registered.
func doSave(
	t *testing.T,
	saveEngine *saveengine.Engine,
	method string,
	target string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	recorder := httptest.NewRecorder()
	newHandler(newPrototypeCatalog(t), testApplicationVersion, saveEngine).
		ServeHTTP(recorder, httptest.NewRequest(method, target, reader))
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
		{http.MethodDelete, "/api/v1/save-sessions/any-session", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/profile", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/stats", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/appearance", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/equipment", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/quick-items", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/pouch-items", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/physick-mixture", ""},
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
			Parameters map[string]any `json:"parameters"`
			Schemas    map[string]any `json:"schemas"`
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
		"/api/v1/save-sessions":                                                          "post",
		"/api/v1/save-sessions/{saveSessionID}":                                          "get",
		"/api/v1/save-sessions/{saveSessionID}/characters":                               "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/profile":         "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/stats":           "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance":      "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipment":       "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/quick-items":     "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/pouch-items":     "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/physick-mixture": "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory":       "get",
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
	assertLoopbackOnlySaveSessionRoutes(t, document.Paths)

	for _, name := range []string{"ResourceKind", "ResourceKey", "RelationType", "RelationDirection"} {
		if _, exists := document.Comps.Parameters[name]; !exists {
			t.Fatalf("openapi.json is missing the shared %s parameter", name)
		}
	}
	if _, exists := document.Comps.Parameters["ResourceID"]; exists {
		t.Fatal("openapi.json still declares the removed ResourceID parameter")
	}
	for _, name := range []string{
		"Error",
		"SupportedSchema",
		"GetApplicationInfoResult",
		"GetCatalogInfoResult",
		"GetResourceResult",
		"GetItemVariantsResult",
		"GetResourceRelationsResult",
		"GetResourcesResult",
		"Relation",
		"ResourceRef",
		"NetworkParamValues",
		"NetworkPreset",
		"GetNetworkPresetsResult",
		"AppearancePresetSummary",
		"GetAppearancePresetsResult",
		"LoadSaveRequest",
		"CloseSaveResult",
		"SessionInfo",
		"SaveCharacters",
		"CharacterProfile",
		"CharacterStats",
		"CharacterAppearance",
		"CharacterEquipment",
		"QuickItemSlot",
		"CharacterQuickItems",
		"PouchItemSlot",
		"CharacterPouchItems",
		"CharacterPhysickMixture",
		"InventoryRecord",
		"CharacterInventory",
	} {
		if _, exists := document.Comps.Schemas[name]; !exists {
			t.Fatalf("openapi.json is missing the %s schema", name)
		}
	}
	if _, exists := document.Comps.Schemas["ResourceID"]; exists {
		t.Fatal("openapi.json still declares the removed ResourceID schema")
	}
	assertRelationEndpointsAreResourceRefs(t, document.Comps.Schemas)
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
	if found != 14 {
		t.Fatalf("openapi.json describes %d save-session operations, want 14", found)
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

// LoadSave is sent as JSON, which makes the browser preflight it. The mux routes
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
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions} {
		if !strings.Contains(methods, method) {
			t.Fatalf("Access-Control-Allow-Methods = %q, want it to allow %s", methods, method)
		}
	}
	// Content-Type is what turns the LoadSave body into a preflighted request.
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
	for _, name := range []string{"EquippedSpellSlot", "CharacterEquippedSpells"} {
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

	absent := doSave(t, nil, http.MethodGet, target, "")
	if absent.Code != http.StatusNotFound {
		t.Fatalf("%s without an engine: status = %d, want 404 (body %q)",
			target, absent.Code, absent.Body.String())
	}
}

// The inventory route is the one save-session route with a section filter and
// paging, so its fixture carries real records: two common and one key. The
// offsets are restated here instead of reused from SaveEngine.
const (
	inventoryRouteSlotDataBase = int64(pcHeaderSize) + 0x10
	inventoryRouteUserData10   = int64(pcHeaderSize) + 10*0x280010 + 0x10
	inventoryRouteFlagsOffset  = 0x1954
	inventoryRouteAnchorAt     = 0x0640
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

	want, err := inventory.GetInventory(saveEngine, session.SaveSessionID, 0, "", 0, 0)
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
	wantKey, err := inventory.GetInventory(saveEngine, session.SaveSessionID, 0, "key", 1, 1)
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
