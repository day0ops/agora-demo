# Marketplace Demo

[![ci](https://github.com/day0ops/marketplace-demo/actions/workflows/ci.yml/badge.svg)](https://github.com/day0ops/marketplace-demo/actions/workflows/ci.yml)
[![license](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

A live buy/sell + auction marketplace demo for kagent Agent Substrate + agentgateway Enterprise: buyer and seller AI concierges run as Substrate actors, with agentgateway providing multi-provider LLM failover, budget/rate-limit enforcement, PII guardrails, RFC 8693 token exchange, and zero-code MCP tool generation from a REST OpenAPI spec.

This repository holds the application tier: the catalog backend, the purpose-built MCP servers, and the marketplace console. The deployment glue that runs it on a cluster lives in [`agentic-field-kit`](https://github.com/day0ops/agentic-field-kit).

**Status: work in progress.** The design is fully specified; `catalog-api`, `fixtures`, `bids-mcp`, `pricing-mcp`, and `orders-mcp` are built out. One MCP server adapter and the marketplace console are not yet implemented.

## Components

| Path | Purpose | Status |
|---|---|---|
| `fixtures/` | Shared synthetic dataset (sellers, buyers, listings, bids, orders) | Done |
| `catalog-api/` | REST API owning all listings/bids/orders domain data. Its OpenAPI surface is wrapped into `listings-mcp` by agentgateway directly, with no MCP server code | Done |
| `mcp-servers/bids-mcp/` | MCP server: place/withdraw bids, set a standing auto-bid ceiling, bid history, current price. A thin client over `catalog-api` - holds no data of its own | Done |
| `mcp-servers/pricing-mcp/` | MCP server: `suggest_price` for sellers - real comps from `catalog-api`'s sold listings, falling back to a static baseline with no sales history yet | Done |
| `mcp-servers/orders-mcp/` | MCP server: `checkout`, `get_order`, `list_orders` - a thin client over `catalog-api`'s checkout/orders endpoints | Done |
| `mcp-servers/trust-mcp/` | MCP server: fraud/counterfeit/shill-bid signals, backed by a fast Jev risk score | Not started |
| `console/` | Marketplace console: Go backend + web SPA with buyer (`/shop`), seller (`/sell`), and operator (`/ops`) views | Not started |

## catalog-api

```
GET    /listings                     list listings, filterable by ?status= and ?sellerId=
GET    /listings/{id}                get a listing
POST   /listings                     create a listing
GET    /listings/{id}/bids           bid history for a listing
POST   /listings/{id}/bids           place a manual bid
DELETE /listings/{id}/bids/{bidID}   withdraw a specific bid
POST   /listings/{id}/autobid        set (or raise) a buyer's standing proxy-bid ceiling
POST   /listings/{id}/checkout       buy a buy-now listing, or claim a won auction
GET    /orders/{id}                  get an order
GET    /orders                       list orders, filterable by ?buyerId= and ?sellerId=
```

Auctions close automatically once their end time passes (to `sold` if they received a bid, `unsold-expired` otherwise). Proxy bidding is classic eBay-style: the price settles one increment above the second-highest standing ceiling, capped at the top ceiling; a manual bid also counts as a ceiling of exactly its own amount, so a stronger standing auto-bid immediately outbids it. Checking out a won auction requires the caller to be the actual winning bidder (403 otherwise); each listing can only be checked out once.

## Development

```bash
go work sync
go build ./fixtures/... ./catalog-api/...
go test ./fixtures/... ./catalog-api/...
golangci-lint run ./fixtures/... ./catalog-api/...
```

Run the catalog API locally:

```bash
cd catalog-api && go run .
# listening on :8080
```

## License

Apache-2.0. See [LICENSE](LICENSE).
