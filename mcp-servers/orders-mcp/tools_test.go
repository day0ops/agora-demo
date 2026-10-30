package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/day0ops/marketplace-demo/fixtures"
)

func TestCheckoutToolSuccess(t *testing.T) {
	var gotPath string
	var gotBody checkoutBody
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(fixtures.Order{ID: "order-1", ListingID: "l1", BuyerID: "b1", FinalPriceCents: 4500})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.checkoutTool(context.Background(), nil, CheckoutInput{ListingID: "l1", BuyerID: "b1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/listings/l1/checkout" {
		t.Fatalf("expected path /listings/l1/checkout, got %s", gotPath)
	}
	if gotBody.BuyerID != "b1" {
		t.Fatalf("unexpected request body: %+v", gotBody)
	}
	if out.Error != "" || out.Order.ID != "order-1" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestCheckoutToolDomainRejection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "only the winning bidder can check out this auction"})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.checkoutTool(context.Background(), nil, CheckoutInput{ListingID: "l1", BuyerID: "b1"})
	if err != nil {
		t.Fatalf("expected a domain rejection, not a tool error: %v", err)
	}
	if out.Error != "only the winning bidder can check out this auction" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestCheckoutToolRejectionOmitsFakeOrderFromJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "listing is not available for purchase"})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.checkoutTool(context.Background(), nil, CheckoutInput{ListingID: "l1", BuyerID: "b1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	marshaled, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}
	var asMap map[string]json.RawMessage
	if err := json.Unmarshal(marshaled, &asMap); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if _, present := asMap["order"]; present {
		t.Fatalf("expected a rejected checkout to omit the zero-valued \"order\" field entirely, got %s", marshaled)
	}
}

func TestCheckoutToolInfraFailureReturnsRealError(t *testing.T) {
	c := newCatalogClient("http://127.0.0.1:1")
	_, _, err := c.checkoutTool(context.Background(), nil, CheckoutInput{ListingID: "l1", BuyerID: "b1"})
	if err == nil {
		t.Fatal("expected a real tool error for an unreachable catalog-api")
	}
}

func TestGetOrderTool(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orders/order-1" {
			t.Errorf("expected path /orders/order-1, got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(fixtures.Order{ID: "order-1", Status: fixtures.OrderStatusPending})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.getOrderTool(context.Background(), nil, GetOrderInput{OrderID: "order-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Order.Status != fixtures.OrderStatusPending {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestGetOrderToolNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order not found"})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.getOrderTool(context.Background(), nil, GetOrderInput{OrderID: "missing"})
	if err != nil {
		t.Fatalf("expected a domain rejection, not a tool error: %v", err)
	}
	if out.Error != "order not found" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestListOrdersToolBuildsQueryParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode([]fixtures.Order{{ID: "order-1"}})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.listOrdersTool(context.Background(), nil, ListOrdersInput{BuyerID: "b1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQuery != "buyerId=b1" {
		t.Fatalf("expected query buyerId=b1, got %q", gotQuery)
	}
	if len(out.Orders) != 1 {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestListOrdersToolBuildsQueryParamsWithBothFilters(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode([]fixtures.Order{})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	if _, _, err := c.listOrdersTool(context.Background(), nil, ListOrdersInput{BuyerID: "b1", SellerID: "s1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQuery != "buyerId=b1&sellerId=s1" {
		t.Fatalf("expected query buyerId=b1&sellerId=s1, got %q", gotQuery)
	}
}

func TestListOrdersToolWithNoFilter(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode([]fixtures.Order{})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	if _, _, err := c.listOrdersTool(context.Background(), nil, ListOrdersInput{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQuery != "" {
		t.Fatalf("expected no query string with no filters, got %q", gotQuery)
	}
}

func TestListingPathEscapesAgentSuppliedIDs(t *testing.T) {
	got := listingPath("l1/evil", "/checkout")
	want := "/listings/l1%2Fevil/checkout"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
