// Command catalog-mcp is an MCP server exposing a single buyer-facing tool,
// search_listings, which finds active listings (open auctions and live
// buy-now items) by category, price range, and keyword.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "catalog-mcp"

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	catalogAPIURL := envOr("CATALOG_API_URL", "http://catalog-api:8080")
	client := newCatalogClient(catalogAPIURL)

	server := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: "0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_listings",
		Description: "Search active listings (open auctions and live buy-now items) by category, price range, and keyword",
	}, client.searchListingsTool)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)

	addr := ":" + envOr("PORT", "9124")
	log.Printf("%s MCP server listening on %s (POST /mcp), catalog-api at %s", serverName, addr, catalogAPIURL)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
