package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/day0ops/agora-demo/fixtures"
)

func newTestServer(catalogURL string) *consoleServer {
	return newConsoleServer(newCatalogClient(catalogURL), newEventHub(), AgentInvokerStub{})
}

func TestHandleListListingsRelaysCatalogData(t *testing.T) {
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]fixtures.Listing{{ID: "l1"}})
	}))
	t.Cleanup(catalog.Close)

	srv := newTestServer(catalog.URL)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/listings", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var listings []fixtures.Listing
	if err := json.Unmarshal(rec.Body.Bytes(), &listings); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(listings) != 1 || listings[0].ID != "l1" {
		t.Fatalf("unexpected listings: %+v", listings)
	}
}

func TestHandleGetListingRelaysCatalogErrorStatusAndMessage(t *testing.T) {
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "listing not found"})
	}))
	t.Cleanup(catalog.Close)

	srv := newTestServer(catalog.URL)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/listings/missing", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected the console to relay catalog-api's 404 verbatim, got %d", rec.Code)
	}
	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != "listing not found" {
		t.Fatalf("expected catalog-api's message to be relayed, got %q", body.Error)
	}
}

func TestHandleGetListingReports502WhenCatalogUnreachable(t *testing.T) {
	// A closed server guarantees connection failure, not any particular HTTP
	// status: this exercises the non-apiError branch of writeCatalogError.
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	catalog.Close()

	srv := newTestServer(catalog.URL)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/listings/l1", nil))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 when catalog-api is unreachable, got %d", rec.Code)
	}
}

func TestHandleCreateListingPublishesEvent(t *testing.T) {
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(fixtures.Listing{ID: "l1", Title: "Widget"})
	}))
	t.Cleanup(catalog.Close)

	srv := newTestServer(catalog.URL)
	ch, unsubscribe := srv.hub.subscribe()
	t.Cleanup(unsubscribe)

	body, _ := json.Marshal(createListingRequest{SellerID: "s1", Title: "Widget", PriceCents: 1000})
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/listings", bytes.NewReader(body)))

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	select {
	case evt := <-ch:
		if evt.Type != "listing.created" {
			t.Fatalf("unexpected event type: %q", evt.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("expected a listing.created event to be published")
	}
}

func TestHandlePlaceBidPublishesEvent(t *testing.T) {
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(fixtures.Bid{ID: "b1", AmountCents: 1500})
	}))
	t.Cleanup(catalog.Close)

	srv := newTestServer(catalog.URL)
	ch, unsubscribe := srv.hub.subscribe()
	t.Cleanup(unsubscribe)

	body, _ := json.Marshal(placeBidRequest{BuyerID: "buyer-1", AmountCents: 1500})
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/listings/l1/bids", bytes.NewReader(body)))

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	select {
	case evt := <-ch:
		if evt.Type != "bid.placed" {
			t.Fatalf("unexpected event type: %q", evt.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("expected a bid.placed event to be published")
	}
}

func TestHandleWithdrawBidPublishesEventAndReturnsNoContent(t *testing.T) {
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(catalog.Close)

	srv := newTestServer(catalog.URL)
	ch, unsubscribe := srv.hub.subscribe()
	t.Cleanup(unsubscribe)

	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/listings/l1/bids/b1", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	select {
	case evt := <-ch:
		if evt.Type != "bid.withdrawn" {
			t.Fatalf("unexpected event type: %q", evt.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("expected a bid.withdrawn event to be published")
	}
}

func TestHandleCheckoutPublishesEvent(t *testing.T) {
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(fixtures.Order{ID: "o1", FinalPriceCents: 4500})
	}))
	t.Cleanup(catalog.Close)

	srv := newTestServer(catalog.URL)
	ch, unsubscribe := srv.hub.subscribe()
	t.Cleanup(unsubscribe)

	body, _ := json.Marshal(checkoutRequest{BuyerID: "buyer-1"})
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/listings/l1/checkout", bytes.NewReader(body)))

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	select {
	case evt := <-ch:
		if evt.Type != "order.created" {
			t.Fatalf("unexpected event type: %q", evt.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("expected an order.created event to be published")
	}
}

func TestHandleChatReturns200WithErrorWhenInvokerFails(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})

	body, _ := json.Marshal(chatRequest{Agent: "shopping-assistant", SessionID: "s1", Message: "hi"})
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (agent unavailability is a normal chat response, not a transport error), got %d", rec.Code)
	}
	var resp chatResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error == "" || resp.Reply != "" {
		t.Fatalf("expected an Error and no Reply, got %+v", resp)
	}
}

type fakeInvoker struct{ reply string }

func (f fakeInvoker) Send(context.Context, string, string, string) (string, error) {
	return f.reply, nil
}

func TestHandleChatReturnsReplyOnSuccess(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), fakeInvoker{reply: "Here are three listings you might like."})

	body, _ := json.Marshal(chatRequest{Agent: "shopping-assistant", SessionID: "s1", Message: "hi"})
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body)))

	var resp chatResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Reply != "Here are three listings you might like." || resp.Error != "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestHandleStatsCountsActiveListingsAndSubscribers(t *testing.T) {
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/listings":
			_ = json.NewEncoder(w).Encode([]fixtures.Listing{
				{ID: "l1", Status: fixtures.ListingStatusActiveAuction},
				{ID: "l2", Status: fixtures.ListingStatusActiveBuyNow},
				{ID: "l3", Status: fixtures.ListingStatusSold},
			})
		case "/orders":
			_ = json.NewEncoder(w).Encode([]fixtures.Order{{ID: "o1"}, {ID: "o2"}})
		}
	}))
	t.Cleanup(catalog.Close)

	srv := newTestServer(catalog.URL)
	_, unsubscribe := srv.hub.subscribe()
	t.Cleanup(unsubscribe)

	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stats", nil))

	var stats statsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if stats.ActiveListings != 2 {
		t.Fatalf("expected 2 active listings, got %d", stats.ActiveListings)
	}
	if stats.TotalOrders != 2 {
		t.Fatalf("expected 2 orders, got %d", stats.TotalOrders)
	}
	if stats.ConnectedClients != 1 {
		t.Fatalf("expected 1 connected client, got %d", stats.ConnectedClients)
	}
}

func TestHandleEventsStreamsPublishedEvents(t *testing.T) {
	srv := newTestServer("http://unused")
	httpServer := httptest.NewServer(srv.routes())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, httpServer.URL+"/api/events", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect to event stream: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	// Give the handler a moment to register its subscription before
	// publishing, since subscription happens after headers are already
	// flushed to the client.
	deadline := time.Now().Add(time.Second)
	for srv.hub.subscriberCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if srv.hub.subscriberCount() != 1 {
		t.Fatalf("expected the SSE client to be subscribed, got %d subscribers", srv.hub.subscriberCount())
	}

	srv.hub.publish(Event{Type: "bid.placed", Data: "hello"})

	reader := bufio.NewReader(resp.Body)
	dataLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read data line: %v", err)
	}
	trimmed := strings.TrimSpace(dataLine)
	if !strings.HasPrefix(trimmed, "data: ") {
		t.Fatalf("unexpected SSE line: %q", dataLine)
	}
	var evt Event
	if err := json.Unmarshal([]byte(strings.TrimPrefix(trimmed, "data: ")), &evt); err != nil {
		t.Fatalf("decode event payload: %v", err)
	}
	if evt.Type != "bid.placed" || evt.Data != "hello" {
		t.Fatalf("unexpected event: %+v", evt)
	}

	// The SSE spec terminates a frame with a blank line - without it, a real
	// browser EventSource buffers forever and never dispatches the event.
	terminator, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read frame terminator: %v", err)
	}
	if strings.TrimRight(terminator, "\n") != "" {
		t.Fatalf("expected a blank line terminating the SSE frame, got %q", terminator)
	}
}
