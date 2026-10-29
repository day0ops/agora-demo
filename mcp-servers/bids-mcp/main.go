// Command bids-mcp is an MCP server exposing auction bidding as agent tools:
// placing and withdrawing manual bids, reading price/bid history, and
// registering a standing proxy-bid (auto-bid) ceiling. It holds no
// marketplace data of its own -- every tool call is a live HTTP round trip
// to catalog-api, the single source of truth.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "bids-mcp"

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
		Name: "place_bid", Description: "Place a manual bid on an auction listing",
	}, client.placeBidTool)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_current_price", Description: "Get a listing's current price and status",
	}, client.getCurrentPriceTool)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_bid_history", Description: "List all bids placed on a listing",
	}, client.getBidHistoryTool)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_autobid_max",
		Description: "Register a standing proxy-bid ceiling for a buyer on an auction listing",
	}, client.setAutoBidMaxTool)
	mcp.AddTool(server, &mcp.Tool{
		Name: "withdraw_bid", Description: "Withdraw a specific bid from an auction listing",
	}, client.withdrawBidTool)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)

	addr := ":" + envOr("PORT", "9120")
	log.Printf("%s MCP server listening on %s (POST /mcp), catalog-api at %s", serverName, addr, catalogAPIURL)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
