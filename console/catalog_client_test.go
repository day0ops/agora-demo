package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/day0ops/agora-demo/fixtures"
)

func TestListListingsBuildsFilterQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode([]fixtures.Listing{{ID: "l1"}})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	listings, err := c.listListings(context.Background(), "active-auction", "seller-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQuery != "sellerId=seller-1&status=active-auction" {
		t.Fatalf("unexpected query: %q", gotQuery)
	}
	if len(listings) != 1 {
		t.Fatalf("unexpected listings: %+v", listings)
	}
}

func TestGetListing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/listings/l1" {
			t.Errorf("expected /listings/l1, got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(fixtures.Listing{ID: "l1", PriceCents: 1000})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	l, err := c.getListing(context.Background(), "l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.PriceCents != 1000 {
		t.Fatalf("unexpected listing: %+v", l)
	}
}

func TestCreateListing(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(fixtures.Listing{ID: "l1"})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, err := c.createListing(context.Background(), createListingRequest{SellerID: "s1", Title: "t", PriceCents: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/listings" {
		t.Fatalf("expected POST /listings, got %s %s", gotMethod, gotPath)
	}
}

func TestPlaceBidAndWithdrawBid(t *testing.T) {
	var gotPlacePath string
	var gotWithdrawMethod, gotWithdrawPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			gotPlacePath = r.URL.Path
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(fixtures.Bid{ID: "b1", AmountCents: 1500})
		case http.MethodDelete:
			gotWithdrawMethod, gotWithdrawPath = r.Method, r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	bid, err := c.placeBid(context.Background(), "l1", "buyer-1", 1500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPlacePath != "/listings/l1/bids" || bid.AmountCents != 1500 {
		t.Fatalf("unexpected result: path=%s bid=%+v", gotPlacePath, bid)
	}

	if err := c.withdrawBid(context.Background(), "l1", "b1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotWithdrawMethod != http.MethodDelete || gotWithdrawPath != "/listings/l1/bids/b1" {
		t.Fatalf("unexpected withdraw request: %s %s", gotWithdrawMethod, gotWithdrawPath)
	}
}

func TestSetAutoBidMax(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(fixtures.Listing{ID: "l1", CurrentBidCents: 1000})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	l, err := c.setAutoBidMax(context.Background(), "l1", "buyer-1", 6000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/listings/l1/autobid" || l.CurrentBidCents != 1000 {
		t.Fatalf("unexpected result: path=%s listing=%+v", gotPath, l)
	}
}

func TestCheckoutAndListOrders(t *testing.T) {
	var gotCheckoutPath string
	var gotOrdersQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			gotCheckoutPath = r.URL.Path
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(fixtures.Order{ID: "o1", FinalPriceCents: 4500})
		default:
			gotOrdersQuery = r.URL.RawQuery
			_ = json.NewEncoder(w).Encode([]fixtures.Order{{ID: "o1"}})
		}
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	order, err := c.checkout(context.Background(), "l1", "buyer-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotCheckoutPath != "/listings/l1/checkout" || order.FinalPriceCents != 4500 {
		t.Fatalf("unexpected checkout result: path=%s order=%+v", gotCheckoutPath, order)
	}

	orders, err := c.listOrders(context.Background(), "buyer-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotOrdersQuery != "buyerId=buyer-1" || len(orders) != 1 {
		t.Fatalf("unexpected list result: query=%s orders=%+v", gotOrdersQuery, orders)
	}
}

func TestDomainErrorFor4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "bid amount is too low"})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, err := c.placeBid(context.Background(), "l1", "buyer-1", 1)
	msg, ok := domainError(err)
	if !ok || msg != "bid amount is too low" {
		t.Fatalf("expected a domain error with the catalog-api message, got ok=%v msg=%q err=%v", ok, msg, err)
	}
}

func TestNonDomainErrorFor5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, err := c.getListing(context.Background(), "l1")
	if _, ok := domainError(err); ok {
		t.Fatalf("expected a 500 to NOT be classified as a domain error, got %v", err)
	}
}
