package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/day0ops/agora-demo/fixtures"
)

func newTestServer(t *testing.T) (*httptest.Server, *Store) {
	t.Helper()
	end := time.Now().Add(time.Hour)
	store := newStoreWithIDFunc(
		[]fixtures.Listing{
			{
				ID: "l1", SellerID: "seller-1", Title: "Test Item", Type: fixtures.ListingTypeAuction,
				Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, CurrentBidCents: 1000,
				AuctionEndsAt: &end,
			},
		},
		nil,
		func(prefix string) string { return prefix + "-generated" },
	)
	srv := httptest.NewServer(NewMux(store))
	t.Cleanup(srv.Close)
	return srv, store
}

func decodeJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return v
}

func TestHandleListListings(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, err := http.Get(srv.URL + "/listings")
	if err != nil {
		t.Fatalf("GET /listings: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	listings := decodeJSON[[]fixtures.Listing](t, resp)
	if len(listings) != 1 || listings[0].ID != "l1" {
		t.Fatalf("expected [l1], got %+v", listings)
	}
}

func TestHandleGetListingNotFound(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, err := http.Get(srv.URL + "/listings/missing")
	if err != nil {
		t.Fatalf("GET /listings/missing: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	body := decodeJSON[errorResponse](t, resp)
	if body.Error == "" {
		t.Fatal("expected a non-empty JSON error message")
	}
}

func TestHandleCreateListing(t *testing.T) {
	srv, _ := newTestServer(t)

	body, err := json.Marshal(createListingRequest{
		SellerID: "seller-2", Title: "New Item", Category: "misc",
		Type: fixtures.ListingTypeBuyNow, PriceCents: 2500,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	resp, err := http.Post(srv.URL+"/listings", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /listings: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc == "" {
		t.Fatal("expected a Location header on 201 Created")
	}
	created := decodeJSON[fixtures.Listing](t, resp)
	if created.Title != "New Item" || created.Status != fixtures.ListingStatusActiveBuyNow {
		t.Fatalf("unexpected created listing: %+v", created)
	}
}

func TestHandleCreateListingValidationError(t *testing.T) {
	srv, _ := newTestServer(t)

	body, err := json.Marshal(createListingRequest{Title: "No Seller", PriceCents: 100, Type: fixtures.ListingTypeBuyNow})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	resp, err := http.Post(srv.URL+"/listings", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /listings: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHandleCreateListingRejectsUnknownFields(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, err := http.Post(srv.URL+"/listings", "application/json", bytes.NewReader(
		[]byte(`{"sellerId":"s1","title":"t","priceCents":100,"type":"buy-now","typoField":true}`),
	))
	if err != nil {
		t.Fatalf("POST /listings: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for an unknown field, got %d", resp.StatusCode)
	}
}

func TestHandlePlaceBidAndWithdraw(t *testing.T) {
	srv, _ := newTestServer(t)

	bidBody, err := json.Marshal(placeBidRequest{BuyerID: "buyer-1", AmountCents: 1500})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	resp, err := http.Post(srv.URL+"/listings/l1/bids", "application/json", bytes.NewReader(bidBody))
	if err != nil {
		t.Fatalf("POST /listings/l1/bids: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	bid := decodeJSON[fixtures.Bid](t, resp)
	if bid.AmountCents != 1500 {
		t.Fatalf("expected bid amount 1500, got %d", bid.AmountCents)
	}

	historyResp, err := http.Get(srv.URL + "/listings/l1/bids")
	if err != nil {
		t.Fatalf("GET /listings/l1/bids: %v", err)
	}
	defer func() { _ = historyResp.Body.Close() }()
	history := decodeJSON[[]fixtures.Bid](t, historyResp)
	if len(history) != 1 {
		t.Fatalf("expected 1 bid in history, got %d", len(history))
	}

	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/listings/l1/bids/"+bid.ID, nil)
	if err != nil {
		t.Fatalf("build delete request: %v", err)
	}
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /listings/l1/bids/%s: %v", bid.ID, err)
	}
	defer func() { _ = delResp.Body.Close() }()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", delResp.StatusCode)
	}
}

func TestHandlePlaceBidConflict(t *testing.T) {
	srv, _ := newTestServer(t)

	bidBody, err := json.Marshal(placeBidRequest{BuyerID: "buyer-1", AmountCents: 500})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	resp, err := http.Post(srv.URL+"/listings/l1/bids", "application/json", bytes.NewReader(bidBody))
	if err != nil {
		t.Fatalf("POST /listings/l1/bids: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for a bid below the current price, got %d", resp.StatusCode)
	}
}

func TestHandleSetAutoBidMax(t *testing.T) {
	end := time.Now().Add(time.Hour)
	store := newStoreWithIDFunc(
		[]fixtures.Listing{
			{
				ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction,
				PriceCents: 1000, CurrentBidCents: 0, AuctionEndsAt: &end,
			},
		},
		nil,
		func(prefix string) string { return prefix + "-generated" },
	)
	srv := httptest.NewServer(NewMux(store))
	t.Cleanup(srv.Close)

	body, err := json.Marshal(setAutoBidMaxRequest{BuyerID: "buyer-1", MaxCents: 5000})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	resp, err := http.Post(srv.URL+"/listings/l1/autobid", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /listings/l1/autobid: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	l := decodeJSON[fixtures.Listing](t, resp)
	if l.CurrentBidCents != 1000 {
		t.Fatalf("expected the lone ceiling to settle at the starting price 1000, got %d", l.CurrentBidCents)
	}

	historyResp, err := http.Get(srv.URL + "/listings/l1/bids")
	if err != nil {
		t.Fatalf("GET /listings/l1/bids: %v", err)
	}
	defer func() { _ = historyResp.Body.Close() }()
	history := decodeJSON[[]fixtures.Bid](t, historyResp)
	if len(history) != 1 || history[0].BuyerID != "buyer-1" || !history[0].IsAutoBid {
		t.Fatalf("expected one auto-generated bid for buyer-1 to actually be recorded, got %+v", history)
	}
}

func TestHandleSetAutoBidMaxValidationError(t *testing.T) {
	srv, _ := newTestServer(t)

	body, err := json.Marshal(setAutoBidMaxRequest{BuyerID: "buyer-1", MaxCents: 0})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	resp, err := http.Post(srv.URL+"/listings/l1/autobid", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /listings/l1/autobid: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for a non-positive maxCents, got %d", resp.StatusCode)
	}
}

func TestHandleCheckoutBuyNowAndGetOrder(t *testing.T) {
	store := newStoreWithIDFunc(
		[]fixtures.Listing{
			{ID: "l2", SellerID: "seller-1", Type: fixtures.ListingTypeBuyNow, Status: fixtures.ListingStatusActiveBuyNow, PriceCents: 4500},
		},
		nil,
		func(prefix string) string { return prefix + "-generated" },
	)
	srv := httptest.NewServer(NewMux(store))
	t.Cleanup(srv.Close)

	body, err := json.Marshal(checkoutRequest{BuyerID: "buyer-1"})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	resp, err := http.Post(srv.URL+"/listings/l2/checkout", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /listings/l2/checkout: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	order := decodeJSON[fixtures.Order](t, resp)
	if order.FinalPriceCents != 4500 {
		t.Fatalf("unexpected order: %+v", order)
	}
	if loc := resp.Header.Get("Location"); loc != "/orders/"+order.ID {
		t.Fatalf("expected Location /orders/%s, got %q", order.ID, loc)
	}

	getResp, err := http.Get(srv.URL + "/orders/" + order.ID)
	if err != nil {
		t.Fatalf("GET /orders/%s: %v", order.ID, err)
	}
	defer func() { _ = getResp.Body.Close() }()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResp.StatusCode)
	}
	fetched := decodeJSON[fixtures.Order](t, getResp)
	if fetched.ID != order.ID {
		t.Fatalf("expected to fetch the same order, got %+v", fetched)
	}
}

func TestHandleCheckoutConflict(t *testing.T) {
	store := newStoreWithIDFunc(
		[]fixtures.Listing{{ID: "l2", Type: fixtures.ListingTypeBuyNow, Status: fixtures.ListingStatusSold, PriceCents: 4500}},
		nil,
		func(prefix string) string { return prefix + "-generated" },
	)
	srv := httptest.NewServer(NewMux(store))
	t.Cleanup(srv.Close)

	body, err := json.Marshal(checkoutRequest{BuyerID: "buyer-1"})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	resp, err := http.Post(srv.URL+"/listings/l2/checkout", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /listings/l2/checkout: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for an already-sold listing, got %d", resp.StatusCode)
	}
}

func TestHandleListOrders(t *testing.T) {
	store := newStoreWithIDFunc(
		[]fixtures.Listing{{ID: "l2", SellerID: "seller-1", Type: fixtures.ListingTypeBuyNow, Status: fixtures.ListingStatusActiveBuyNow, PriceCents: 4500}},
		nil,
		func(prefix string) string { return prefix + "-generated" },
	)
	srv := httptest.NewServer(NewMux(store))
	t.Cleanup(srv.Close)

	body, err := json.Marshal(checkoutRequest{BuyerID: "buyer-1"})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	postResp, err := http.Post(srv.URL+"/listings/l2/checkout", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /listings/l2/checkout: %v", err)
	}
	_ = postResp.Body.Close()

	listResp, err := http.Get(srv.URL + "/orders?buyerId=buyer-1")
	if err != nil {
		t.Fatalf("GET /orders?buyerId=buyer-1: %v", err)
	}
	defer func() { _ = listResp.Body.Close() }()
	orders := decodeJSON[[]fixtures.Order](t, listResp)
	if len(orders) != 1 || orders[0].BuyerID != "buyer-1" {
		t.Fatalf("expected 1 order for buyer-1, got %+v", orders)
	}
}
