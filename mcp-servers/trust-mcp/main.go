// Command trust-mcp is an MCP server exposing fraud/shill-bid risk scoring
// (score_bid_risk) and seller activity summaries (check_seller_reputation)
// as agent tools. It holds no marketplace data of its own -- every tool call
// reads live from catalog-api.
//
// score_bid_risk's scoring is pluggable (see risk_scorer.go): today it is a
// local heuristic, deliberately not a fabricated integration with an
// unverified third-party API.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "trust-mcp"

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	catalogAPIURL := envOr("CATALOG_API_URL", "http://catalog-api:8080")
	s := &server{catalog: newCatalogClient(catalogAPIURL), scorer: NewScorer()}

	mcpServer := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: "0.1.0"}, nil)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "score_bid_risk",
		Description: "Score the fraud/shill-bid risk of a candidate bid, fast",
	}, s.scoreBidRiskTool)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "check_seller_reputation",
		Description: "Summarize a seller's listing activity (sold rate, tenure)",
	}, s.checkSellerReputationTool)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return mcpServer }, nil)
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)

	addr := ":" + envOr("PORT", "9123")
	log.Printf("%s MCP server listening on %s (POST /mcp), catalog-api at %s", serverName, addr, catalogAPIURL)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
