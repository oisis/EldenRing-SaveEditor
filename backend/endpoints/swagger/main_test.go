package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
)

// The prototype catalog holds real, schema-valid resources, so every route is
// exercised against the same data the getter tests use instead of a mock.
const daggerResourceKey = "item:000F4240"

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
	newHandler(gameCatalog).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
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

	want, err := catalog.GetResource(gameCatalog, daggerResourceKey)
	if err != nil {
		t.Fatalf("catalog.GetResource: %v", err)
	}

	recorder := do(t, gameCatalog, "/api/v1/catalog/resource?resourceID="+daggerResourceKey)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	got := decode(t, recorder.Body.Bytes())
	if !reflect.DeepEqual(got, marshalled(t, want)) {
		t.Fatal("resource route body differs from the GetResource result")
	}
}

func TestResourceRouteRejectsMissingResourceID(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	_, wantErr := catalog.GetResource(gameCatalog, "")
	assertErrorMessage(t, do(t, gameCatalog, "/api/v1/catalog/resource"), http.StatusBadRequest, wantErr)
}

func TestResourceRouteReportsUnknownResourceID(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	const unknownKey = "item:FFFFFFFF"
	_, wantErr := catalog.GetResource(gameCatalog, unknownKey)
	assertErrorMessage(
		t,
		do(t, gameCatalog, "/api/v1/catalog/resource?resourceID="+unknownKey),
		http.StatusBadRequest,
		wantErr,
	)
}

func TestItemVariantsRouteMatchesGetter(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	want, err := catalog.GetItemVariants(gameCatalog, daggerResourceKey)
	if err != nil {
		t.Fatalf("catalog.GetItemVariants: %v", err)
	}

	recorder := do(t, gameCatalog, "/api/v1/catalog/item-variants?resourceID="+daggerResourceKey)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", recorder.Code, recorder.Body.String())
	}
	assertJSONContentType(t, recorder)

	got := decode(t, recorder.Body.Bytes())
	if !reflect.DeepEqual(got, marshalled(t, want)) {
		t.Fatal("item variants route body differs from the GetItemVariants result")
	}
}

func TestItemVariantsRouteRejectsMissingResourceID(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	_, wantErr := catalog.GetItemVariants(gameCatalog, "")
	assertErrorMessage(t, do(t, gameCatalog, "/api/v1/catalog/item-variants"), http.StatusBadRequest, wantErr)
}

func TestItemVariantsRouteReportsUnknownResourceID(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	const unknownKey = "item:FFFFFFFF"
	_, wantErr := catalog.GetItemVariants(gameCatalog, unknownKey)
	assertErrorMessage(
		t,
		do(t, gameCatalog, "/api/v1/catalog/item-variants?resourceID="+unknownKey),
		http.StatusBadRequest,
		wantErr,
	)
}

func TestUnimplementedGettersAreNotExposed(t *testing.T) {
	gameCatalog := newPrototypeCatalog(t)

	// GetResources and GetResourceRelations are contract definition only, so no
	// route may answer for them.
	for _, target := range []string{"/api/v1/catalog/resources", "/api/v1/catalog/resource-relations"} {
		if recorder := do(t, gameCatalog, target); recorder.Code != http.StatusNotFound {
			t.Fatalf("%s: status = %d, want 404", target, recorder.Code)
		}
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
		"/api/v1/catalog/info",
		"/api/v1/catalog/resource",
		"/api/v1/catalog/item-variants",
		"/healthz",
		"/openapi.json",
		"/docs",
	} {
		operation, exists := document.Paths[path]
		if !exists {
			t.Fatalf("openapi.json does not describe %s", path)
		}
		if _, hasGet := operation["get"]; !hasGet {
			t.Fatalf("openapi.json describes %s without a GET operation", path)
		}
	}
	for _, path := range []string{"/api/v1/catalog/resources", "/api/v1/catalog/resource-relations"} {
		if _, exists := document.Paths[path]; exists {
			t.Fatalf("openapi.json describes %s, which has no runtime handler", path)
		}
	}
	if _, exists := document.Comps.Parameters["ResourceID"]; !exists {
		t.Fatal("openapi.json is missing the shared ResourceID parameter")
	}
	for _, name := range []string{"Error", "GetCatalogInfoResult", "GetResourceResult", "GetItemVariantsResult"} {
		if _, exists := document.Comps.Schemas[name]; !exists {
			t.Fatalf("openapi.json is missing the %s schema", name)
		}
	}
}

func TestDocsServesTheEmbeddedExplorer(t *testing.T) {
	recorder := do(t, newPrototypeCatalog(t), "/docs")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", contentType)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "/openapi.json") {
		t.Fatal("the explorer does not read /openapi.json")
	}
	// The response view must offer both the table and the untouched payload.
	for _, marker := range []string{`textContent: "Tabela"`, `textContent: "Raw JSON"`, "function tableNode("} {
		if !strings.Contains(body, marker) {
			t.Fatalf("the explorer is missing the response view part %q", marker)
		}
	}
	// The UI must be fully local: no external script, style or font host.
	for _, forbidden := range []string{"http://", "https://", "//cdn", "unpkg", "jsdelivr"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("the explorer references an external resource %q", forbidden)
		}
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
