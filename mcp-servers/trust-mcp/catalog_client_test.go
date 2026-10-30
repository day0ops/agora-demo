package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/day0ops/marketplace-demo/fixtures"
)

func TestListingFetchesByID(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(fixtures.Listing{ID: "l1", PriceCents: 1000})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	l, err := c.listing(context.Background(), "l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/listings/l1" || l.PriceCents != 1000 {
		t.Fatalf("unexpected result: path=%s listing=%+v", gotPath, l)
	}
}

func TestBidHistoryFetchesByListingID(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode([]fixtures.Bid{{ID: "b1"}})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	bids, err := c.bidHistory(context.Background(), "l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/listings/l1/bids" || len(bids) != 1 {
		t.Fatalf("unexpected result: path=%s bids=%+v", gotPath, bids)
	}
}

func TestListingsBySellerBuildsQuery(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		_ = json.NewEncoder(w).Encode([]fixtures.Listing{{ID: "l1"}, {ID: "l2"}})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	listings, err := c.listingsBySeller(context.Background(), "seller-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/listings?sellerId=seller-1" || len(listings) != 2 {
		t.Fatalf("unexpected result: path=%s listings=%+v", gotPath, listings)
	}
}

func TestGetReturnsNonDomainErrorOn5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, err := c.listing(context.Background(), "l1")
	if err == nil {
		t.Fatal("expected an error for a 500 response")
	}
	if _, ok := domainError(err); ok {
		t.Fatalf("expected a 500 to NOT be classified as a domain error, got ok=true for %v", err)
	}
}

func TestGetReturnsDomainErrorOn404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "listing not found"})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, err := c.listing(context.Background(), "missing")
	msg, ok := domainError(err)
	if !ok {
		t.Fatalf("expected a domain error, got %v", err)
	}
	if msg != "listing not found" {
		t.Fatalf("expected the catalog-api error message, got %q", msg)
	}
}
