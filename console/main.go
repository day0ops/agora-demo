// Command console serves the marketplace console: a REST API backed by
// catalog-api plus the embedded React SPA (buyer/seller/operator views all
// served from one index.html, routed client-side). It holds no marketplace
// data of its own.
package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed all:web/dist
var webDist embed.FS

// version is set at build time via -ldflags "-X main.version=<tag>" (see
// Dockerfile and release.yml) -- "dev" when built without that flag.
var version = "dev"

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// spaHandler serves a real static file when one exists at the requested
// path (JS bundles, CSS, images, favicon), otherwise serves index.html so
// client-side routing resolves a direct link or page refresh on any SPA
// route (e.g. /shop, /sell, /ops).
func spaHandler(dist fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path != "" {
			if _, err := fs.Stat(dist, path); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
}

func main() {
	catalogAPIURL := envOr("CATALOG_API_URL", "http://catalog-api:8080")

	var invoker AgentInvoker = AgentInvokerStub{}
	if base := os.Getenv("AGENTGATEWAY_BASE_URL"); base != "" {
		invoker = newAgentInvokerHTTP(base)
	}

	srv := newConsoleServer(newCatalogClient(catalogAPIURL), newEventHub(), invoker)

	dist, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		log.Fatalf("console: load embedded web/dist: %v", err)
	}
	mux := srv.routes()
	mux.Handle("/", spaHandler(dist))

	addr := ":" + envOr("PORT", "8090")
	log.Printf("console listening on %s, catalog-api at %s", addr, catalogAPIURL)
	httpServer := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
