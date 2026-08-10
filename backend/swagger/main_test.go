package main

import (
	"encoding/binary"
	"encoding/json"
	"io"
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

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/appearance"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/application"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/character"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/network"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/savesession"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
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
		"/api/v1/save-sessions":                                                     "post",
		"/api/v1/save-sessions/{saveSessionID}":                                     "get",
		"/api/v1/save-sessions/{saveSessionID}/characters":                          "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/profile":    "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/stats":      "get",
		"/api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance": "get",
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
	if found != 7 {
		t.Fatalf("openapi.json describes %d save-session operations, want 7", found)
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
