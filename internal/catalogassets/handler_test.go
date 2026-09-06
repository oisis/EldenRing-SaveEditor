package catalogassets_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/internal/catalogassets"
)

type recordingReader struct {
	path      string
	content   []byte
	mediaType string
	exists    bool
	calls     int
}

func (reader *recordingReader) ReadAssetWithMediaType(path string) ([]byte, string, bool) {
	reader.calls++
	reader.path = path
	return append([]byte(nil), reader.content...), reader.mediaType, reader.exists
}

func TestHandlerServesAValidatedEmbeddedItemIcon(t *testing.T) {
	t.Parallel()

	reader := &recordingReader{content: []byte("icon"), mediaType: "image/png", exists: true}
	request := httptest.NewRequest(http.MethodGet,
		catalogassets.URLPrefix+"assets/icons/items/melee_armaments/dagger.png", nil)
	recorder := httptest.NewRecorder()

	catalogassets.New(reader).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || !bytes.Equal(recorder.Body.Bytes(), reader.content) {
		t.Fatalf("response = %d %q", recorder.Code, recorder.Body.Bytes())
	}
	if reader.path != "assets/icons/items/melee_armaments/dagger.png" || reader.calls != 1 {
		t.Fatalf("reader path/calls = %q/%d", reader.path, reader.calls)
	}
	for name, want := range map[string]string{
		"Content-Type":           "image/png",
		"Content-Length":         "4",
		"Cache-Control":          "no-cache",
		"X-Content-Type-Options": "nosniff",
	} {
		if got := recorder.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestHandlerSupportsHeadWithoutWritingTheAsset(t *testing.T) {
	t.Parallel()

	reader := &recordingReader{content: []byte("icon"), mediaType: "image/webp", exists: true}
	request := httptest.NewRequest(http.MethodHead,
		catalogassets.URLPrefix+"assets/icons/items/melee_armaments/dagger.png", nil)
	recorder := httptest.NewRecorder()

	catalogassets.New(reader).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || recorder.Body.Len() != 0 {
		t.Fatalf("response = %d with %d body bytes", recorder.Code, recorder.Body.Len())
	}
	if recorder.Header().Get("Content-Type") != "image/webp" || recorder.Header().Get("Content-Length") != "4" {
		t.Fatalf("headers = %v", recorder.Header())
	}
}

func TestHandlerRejectsPathsOutsideValidatedItemIconsWithoutReading(t *testing.T) {
	t.Parallel()

	paths := []string{
		"/",
		catalogassets.URLPrefix,
		catalogassets.URLPrefix + "catalog.json",
		catalogassets.URLPrefix + "assets/icons/items/../catalog.json",
		catalogassets.URLPrefix + "assets/icons/items\\dagger.png",
		catalogassets.URLPrefix + "assets/icons/items/melee_armaments/dagger.webp",
		catalogassets.URLPrefix + "assets/appearance/../catalog.json",
		catalogassets.URLPrefix + "assets/appearance/geralt.png",
		catalogassets.URLPrefix + "assets/appearance/nested/geralt.jpg.json",
	}
	for _, requestPath := range paths {
		t.Run(requestPath, func(t *testing.T) {
			reader := &recordingReader{exists: true}
			recorder := httptest.NewRecorder()
			catalogassets.New(reader).ServeHTTP(
				recorder, httptest.NewRequest(http.MethodGet, requestPath, nil))
			if recorder.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", recorder.Code)
			}
			if reader.calls != 0 {
				t.Fatalf("reader calls = %d, want 0", reader.calls)
			}
		})
	}
}

func TestHandlerReturnsNotFoundForAnUnknownValidatedPath(t *testing.T) {
	t.Parallel()

	reader := &recordingReader{}
	recorder := httptest.NewRecorder()
	catalogassets.New(reader).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		catalogassets.URLPrefix+"assets/icons/items/melee_armaments/missing.png", nil))
	if recorder.Code != http.StatusNotFound || reader.calls != 1 {
		t.Fatalf("status/calls = %d/%d, want 404/1", recorder.Code, reader.calls)
	}
}

func TestHandlerFailsClosedWithoutCatalogData(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	catalogassets.New(nil).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		catalogassets.URLPrefix+"assets/icons/items/melee_armaments/dagger.png", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestHandlerRejectsMutatingMethods(t *testing.T) {
	t.Parallel()

	reader := &recordingReader{exists: true}
	recorder := httptest.NewRecorder()
	catalogassets.New(reader).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost,
		catalogassets.URLPrefix+"assets/icons/items/melee_armaments/dagger.png", nil))
	if recorder.Code != http.StatusMethodNotAllowed || recorder.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("response = %d, Allow %q", recorder.Code, recorder.Header().Get("Allow"))
	}
	if reader.calls != 0 {
		t.Fatalf("reader calls = %d, want 0", reader.calls)
	}
}

// The appearance preview of every preset is part of the same catalog asset
// contract as an item icon. Serving only item icons left the Appearance cards
// without an image, so the shipped catalog data answers both families here.
func TestHandlerServesBothAssetFamiliesFromTheShippedCatalog(t *testing.T) {
	t.Parallel()

	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("load catalog data: %v", err)
	}
	handler := catalogassets.New(data)
	for assetPath, wantMediaType := range map[string]string{
		"assets/icons/items/talismans/carian_filigreed_crest.png": "image/png",
		"assets/appearance/geralt-of-rivia-the-witcher.jpg":       "image/jpeg",
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, catalogassets.URLPrefix+assetPath, nil))
		if recorder.Code != http.StatusOK || recorder.Body.Len() == 0 {
			t.Errorf("%s = %d with %d body bytes, want 200 with content", assetPath, recorder.Code, recorder.Body.Len())
			continue
		}
		if got := recorder.Header().Get("Content-Type"); got != wantMediaType {
			t.Errorf("%s Content-Type = %q, want %q", assetPath, got, wantMediaType)
		}
	}

	// An unauthored appearance file stays unknown instead of reaching the
	// filesystem through the newly accepted prefix.
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		catalogassets.URLPrefix+"assets/appearance/not-a-preset.jpg", nil))
	if recorder.Code != http.StatusNotFound {
		t.Errorf("unknown appearance asset = %d, want 404", recorder.Code)
	}
}
