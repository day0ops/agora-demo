// Command orders-mcp is an MCP server exposing order checkout and status as
// agent tools: checkout, get_order, and list_orders. It holds no marketplace
// data of its own -- every tool call is a live HTTP round trip to
// catalog-api, the single source of truth.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "orders-mcp"

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
		Name:        "checkout",
		Description: "Buy an active buy-now listing outright, or claim a closed auction you won",
	}, client.checkoutTool)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_order", Description: "Look up an order by id",
	}, client.getOrderTool)
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_orders", Description: "List orders, optionally filtered by buyer or seller",
	}, client.listOrdersTool)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)

	addr := ":" + envOr("PORT", "9122")
	log.Printf("%s MCP server listening on %s (POST /mcp), catalog-api at %s", serverName, addr, catalogAPIURL)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
