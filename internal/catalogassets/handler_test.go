package catalogassets_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

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
