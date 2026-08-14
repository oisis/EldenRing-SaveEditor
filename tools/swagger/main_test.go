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
		{http.MethodDelete, "/api/v1/save-sessions/any-session", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/profile", ""},
		{http.MethodPatch, "/api/v1/save-sessions/any-session/characters/0/gender", `{"gender":0,"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/stats", ""},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/appearance", ""},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/appearance", `{"appearance":{},"expectedRevision":"0"}`},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/appearance/preset", `{"presetID":"geralt-of-rivia-the-witcher","expectedRevision":"0"}`},
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
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/inventory/order", `{"orderedOwnedItemIDs":["any-token"],"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/owned-items/any-token", ""},
		{http.MethodDelete, "/api/v1/save-sessions/any-session/characters/0/owned-items/any-token", `{"expectedRevision":"0"}`},
		{http.MethodPatch, "/api/v1/save-sessions/any-session/characters/0/owned-items/any-token/quantity", `{"quantity":1,"expectedRevision":"0"}`},
		{http.MethodPost, "/api/v1/save-sessions/any-session/characters/0/owned-items/any-token/move-to-inventory", `{"targetPosition":0,"expectedRevision":"0"}`},
		{http.MethodPost, "/api/v1/save-sessions/any-session/characters/0/owned-items/any-token/move-to-storage", `{"targetPosition":0,"expectedRevision":"0"}`},
		{http.MethodPatch, "/api/v1/save-sessions/any-session/characters/0/owned-items/any-token/upgrade-level", `{"upgradeLevel":1,"expectedRevision":"0"}`},
		{http.MethodPost, "/api/v1/save-sessions/any-session/characters/0/inventory/items", `{"kind":"item","key":"400006A4","quantity":1,"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/gestures", ""},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/gestures/unlock", `{"gestureKind":"item","gestureKey":"401EA7AB","unlocked":true,"expectedRevision":"0"}`},
		{http.MethodGet, "/api/v1/save-sessions/any-session/characters/0/cookbooks", ""},
		{http.MethodPut, "/api/v1/save-sessions/any-session/characters/0/cookbooks/unlock", `{"cookbookKind":"item","cookbookKey":"40002455","unlocked":true,"expectedRevision":"0"}`},
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
		"/api/v1/save-sessions":                                                                             "post",
		"/api/v1/save-sessions/{saveSessionID}":                                                             "get",
		"/api/v1/save-sessions/{saveSessionID}/write":                                                       "post",
		"/api/v1/save-sessions/{saveSessionID}/characters":                                                  "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/profile":                            "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/name":                               "patch",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/gender":                             "patch",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/runes":                              "patch",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/stats":                              "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance":                         "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance/preset":                  "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipment":                          "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-armaments":                 "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-armor":                     "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/quick-items":                        "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/pouch-items":                        "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/physick-mixture":                    "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory":                          "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory/order":                    "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory/items":                    "post",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/storage":                            "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}":          "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/quantity": "patch",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/gestures":                           "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/gestures/unlock":                    "put",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/cookbooks":                          "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/cookbooks/unlock":                   "put",
		"/api/v1/save-sessions/{saveSessionID}/network-settings":                                            "get",
		"/api/v1/save-sessions/{saveSessionID}/network-settings/preset":                                     "put",
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
	spiritAshUpgrade := ownedItem + "/spirit-ash-upgrade-level"
	if _, hasPatch := document.Paths[spiritAshUpgrade]["patch"]; !hasPatch {
		t.Fatalf("openapi.json describes no PATCH for %s", spiritAshUpgrade)
	}
	characterAppearance := "/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance"
	if _, hasPut := document.Paths[characterAppearance]["put"]; !hasPut {
		t.Fatalf("openapi.json describes no PUT for %s", characterAppearance)
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
		"ResourceKind", "ResourceKey", "RelationType", "RelationDirection", "AvailabilityFilter",
	} {
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
		"SetCharacterNameRequest",
		"SetCharacterNameResult",
		"SetCharacterGenderRequest",
		"SetCharacterGenderResult",
		"SetCharacterRunesRequest",
		"SetCharacterRunesResult",
		"CharacterStats",
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
		"SetWeaponUpgradeLevelRequest",
		"SetWeaponUpgradeLevelResult",
		"SetSpiritAshUpgradeLevelRequest",
		"SetSpiritAshUpgradeLevelResult",
		"SetWeaponInfusionRequest",
		"SetWeaponInfusionResult",
		"GestureEntry",
		"GetGesturesResult",
		"SetGestureUnlockedRequest",
		"SetGestureUnlockedResult",
		"CookbookEntry",
		"GetCookbooksResult",
		"SetCookbookUnlockedRequest",
		"SetCookbookUnlockedResult",
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

// SaveEngine stores a quantity in 31 bits because 0x80000000 is a preserved
// record flag, so a document promising the full uint32 range would advertise
// values SetOwnedItemQuantity rejects.
func assertQuantityFitsTheRecord(t *testing.T, schemas map[string]any) {
	t.Helper()

	for _, name := range []string{
		"SetOwnedItemQuantityRequest", "SetOwnedItemQuantityResult",
		"AddItemToInventoryRequest", "AddItemToInventoryResult",
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
	if found != 45 {
		t.Fatalf("openapi.json describes %d save-session operations, want 45", found)
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
		!reflect.DeepEqual(result.AcquisitionIndices, []uint32{434, 436}) {
		t.Fatalf("SetStorageOrder result = %+v", result)
	}

	updated, err := saveEngine.GetStorage(
		session.SaveSessionID, 0, saveengine.StorageSectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetStorage after reorder: %v", err)
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
