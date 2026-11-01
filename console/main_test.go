package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestSpaHandlerServesRealFilesDirectly(t *testing.T) {
	dist := fstest.MapFS{
		"index.html":    {Data: []byte("<html>spa shell</html>")},
		"assets/app.js": {Data: []byte("console.log('app')")},
	}
	handler := spaHandler(dist)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "console.log('app')" {
		t.Fatalf("expected the real file's content, got %q", rec.Body.String())
	}
}

func TestSpaHandlerFallsBackToIndexForUnknownPaths(t *testing.T) {
	dist := fstest.MapFS{
		"index.html": {Data: []byte("<html>spa shell</html>")},
	}
	handler := spaHandler(dist)

	for _, path := range []string{"/", "/shop", "/sell", "/ops", "/shop/deep/link"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("path %q: expected 200, got %d", path, rec.Code)
		}
		if rec.Body.String() != "<html>spa shell</html>" {
			t.Fatalf("path %q: expected the SPA shell, got %q", path, rec.Body.String())
		}
	}
}
