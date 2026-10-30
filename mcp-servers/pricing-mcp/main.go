// Command pricing-mcp is an MCP server exposing a single seller-facing tool,
// suggest_price, which recommends a price range for a listing category. It
// prefers real comps computed from catalog-api's sold listings, falling back
// to a static baseline when there is no sales history yet for that category.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "pricing-mcp"

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
		Name:        "suggest_price",
		Description: "Suggest a price range for a listing category, based on real recent sales when available",
	}, client.suggestPriceTool)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)

	addr := ":" + envOr("PORT", "9121")
	log.Printf("%s MCP server listening on %s (POST /mcp), catalog-api at %s", serverName, addr, catalogAPIURL)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
