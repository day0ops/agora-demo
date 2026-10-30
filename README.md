# Marketplace Demo

[![ci](https://github.com/day0ops/marketplace-demo/actions/workflows/ci.yml/badge.svg)](https://github.com/day0ops/marketplace-demo/actions/workflows/ci.yml)
[![license](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

A live buy/sell + auction marketplace demo for kagent Agent Substrate + agentgateway Enterprise: buyer and seller AI concierges run as Substrate actors, with agentgateway providing multi-provider LLM failover, budget/rate-limit enforcement, PII guardrails, RFC 8693 token exchange, and zero-code MCP tool generation from a REST OpenAPI spec.

This repository holds the application tier: the catalog backend, the purpose-built MCP servers, and the marketplace console. The deployment glue that runs it on a cluster lives in [`agentic-field-kit`](https://github.com/day0ops/agentic-field-kit).

**Status: application tier complete.** `catalog-api`, `fixtures`, all four MCP servers (`bids-mcp`, `pricing-mcp`, `orders-mcp`, `trust-mcp`), and the marketplace console are built out. Wiring this into `agentic-field-kit` (the deployment profile, features, and usecase spec that actually run it on a cluster) is tracked separately.

## Components

| Path | Purpose | Status |
|---|---|---|
| `fixtures/` | Shared synthetic dataset (sellers, buyers, listings, bids, orders) | Done |
| `catalog-api/` | REST API owning all listings/bids/orders domain data. Its OpenAPI surface is wrapped into `listings-mcp` by agentgateway directly, with no MCP server code | Done |
| `mcp-servers/bids-mcp/` | MCP server: place/withdraw bids, set a standing auto-bid ceiling, bid history, current price. A thin client over `catalog-api` - holds no data of its own | Done |
| `mcp-servers/pricing-mcp/` | MCP server: `suggest_price` for sellers - real comps from `catalog-api`'s sold listings, falling back to a static baseline with no sales history yet | Done |
| `mcp-servers/orders-mcp/` | MCP server: `checkout`, `get_order`, `list_orders` - a thin client over `catalog-api`'s checkout/orders endpoints | Done |
| `mcp-servers/trust-mcp/` | MCP server: `score_bid_risk` (pluggable `RiskScorer`, currently a local heuristic - see note below) and `check_seller_reputation` | Done |
| `console/` | Marketplace console: Go backend + embedded Vite SPA with buyer (`/shop.html`), seller (`/sell.html`), and operator (`/ops.html`) views | Done |

### A note on Jev

The design calls for `trust-mcp`'s risk scoring to eventually run on Jev, TypeSafe.ai's fast "System One" structured-decision model. As of this writing there is no confirmed Jev API contract, credentials, or endpoint available to this project - only a blog post. Rather than ship an integration against an unverified API shape, `trust-mcp` defines a `RiskScorer` interface (`mcp-servers/trust-mcp/risk_scorer.go`) with a working, tested, local heuristic as the default implementation. Swapping in a real Jev-backed scorer once API access exists is a one-line change at `NewScorer()`; nothing else in the package needs to change.

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

## console

The console is a backend-for-frontend over `catalog-api` plus a live SSE feed, with no marketplace data of its own:

```
GET    /api/listings                     list listings, filterable by ?status= and ?sellerId=
GET    /api/listings/{id}                get a listing
POST   /api/listings                     create a listing
GET    /api/listings/{id}/bids           bid history for a listing
POST   /api/listings/{id}/bids           place a manual bid
DELETE /api/listings/{id}/bids/{bidID}   withdraw a specific bid
POST   /api/listings/{id}/autobid        set (or raise) a buyer's standing proxy-bid ceiling
POST   /api/listings/{id}/checkout       buy a buy-now listing, or claim a won auction
GET    /api/orders                       list orders, filterable by ?buyerId= and ?sellerId=
POST   /api/chat                         send a message to a named Substrate concierge agent
GET    /api/events                       live SSE feed of marketplace activity
GET    /api/stats                        active listings / total orders / connected clients
```

Write endpoints relay catalog-api's own status code and error message rather than flattening every failure to 400, so the frontend can tell "not found" apart from "conflict" apart from "forbidden". Every successful write also publishes an event onto the SSE feed (`listing.created`, `bid.placed`, `bid.withdrawn`, `autobid.set`, `order.created`), which all three frontend views subscribe to for live updates.

The frontend is a vanilla TypeScript + Vite SPA (no framework, matching the sibling `fraud-investigation-swarm` console's conventions) with one HTML entry point per view - `index.html` (landing page), `shop.html`, `sell.html`, `ops.html` - built to `console/web/dist` and embedded into the Go binary via `//go:embed`. There's no authentication yet: each view has a plain identity picker (`console/web/src/identities.ts`) mirroring the accounts `fixtures.Seed()` creates, so the SE picks who they're "acting as" during a demo.

### A note on agent invocation

The shop and sell views each have a chat panel against a named Substrate concierge agent (`shopping-assistant`, `listing-pricing-assistant`). The real implementation talks to kagent Agent Substrate via `CreateAgentInstance` + A2A `SendMessage` - a recipe already proven working for the sibling `fraud-ops-console` demo, but built and verified against an older kagent architecture generation. kagent v1.0.0-alpha12 (the version this project targets) replaced that generation's Harness/AgentTemplate label-selector admission model with an explicit `Agent` CRD pairing, so reusing the old recipe without re-verifying it against a live alpha12 cluster would mean shipping integration code nobody has confirmed still works. `console/agent_invoker.go` defines an `AgentInvoker` interface with a documented stub (`AgentInvokerStub`) as the default implementation, mirroring `trust-mcp`'s `RiskScorer` pattern - the chat panel renders the stub's `ErrAgentInvokerNotConfigured` message like any other agent reply, which doubles as a live demonstration of exactly where the real agent's reply will appear once it's wired in.

## Development

```bash
go work sync
go build ./fixtures/... ./catalog-api/... ./console/...
go test ./fixtures/... ./catalog-api/... ./console/...
golangci-lint run ./fixtures/... ./catalog-api/... ./console/...
```

`console/main.go` embeds the built frontend (`//go:embed all:web/dist`), so building, vetting, or testing the `console` package requires `console/web/dist` to exist first:

```bash
cd console/web && npm install && npm run build
```

Run the catalog API locally:

```bash
cd catalog-api && go run .
# listening on :8080
```

Run the console locally (after building the frontend, pointed at a local catalog-api):

```bash
cd console && CATALOG_API_URL=http://localhost:8080 go run .
# listening on :8090 - open http://localhost:8090
```

For frontend-only iteration with hot reload, `cd console/web && npm run dev` proxies `/api` to `http://localhost:8090` (see `vite.config.ts`).

## License

Apache-2.0. See [LICENSE](LICENSE).
