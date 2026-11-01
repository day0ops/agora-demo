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

func TestHandleAccountMeReturns401WithNoToken(t *testing.T) {
	srv := newTestServer("http://unused")
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/account/me", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleAccountMeReturnsResolvedIdentity(t *testing.T) {
	srv := newTestServer("http://unused")
	req := httptest.NewRequest(http.MethodGet, "/account/me", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-seller", []string{"sellers"}))
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var id identity
	if err := json.Unmarshal(rec.Body.Bytes(), &id); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if id.Username != "demo-seller" || id.Role != "sellers" || id.FixtureID != "seller-1" {
		t.Fatalf("unexpected identity: %+v", id)
	}
}

func TestHandleAccountLoginRedirectsToReturnTo(t *testing.T) {
	srv := newTestServer("http://unused")
	req := httptest.NewRequest(http.MethodGet, "/account/login?returnTo=/shop.html", nil)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/shop.html" {
		t.Fatalf("expected redirect to /shop.html, got %q", loc)
	}
}

func TestHandleAccountLoginRejectsAnOffSiteReturnTo(t *testing.T) {
	srv := newTestServer("http://unused")
	req := httptest.NewRequest(http.MethodGet, "/account/login?returnTo=//evil.example.com", nil)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Fatalf("expected an off-site returnTo to fall back to /, got %q", loc)
	}
}

func TestHandleCreateListingDerivesSellerIDFromIdentityNotRequestBody(t *testing.T) {
	var gotBody createListingRequest
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(fixtures.Listing{ID: "l1", Title: "Widget"})
	}))
	t.Cleanup(catalog.Close)

	srv := newTestServer(catalog.URL)
	body, _ := json.Marshal(map[string]any{"sellerId": "someone-elses-id", "title": "Widget", "priceCents": 1000})
	req := httptest.NewRequest(http.MethodPost, "/account/listings", bytes.NewReader(body))
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-seller", []string{"sellers"}))
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotBody.SellerID != "seller-1" {
		t.Fatalf("expected the identity-derived seller-1, got %q (client-supplied sellerId must be ignored)", gotBody.SellerID)
	}
}

func TestHandleCreateListingRequires401WithNoToken(t *testing.T) {
	srv := newTestServer("http://unused")
	body, _ := json.Marshal(map[string]any{"title": "Widget", "priceCents": 1000})
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/account/listings", bytes.NewReader(body)))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandlePlaceBidDerivesBuyerIDFromIdentityAndPublishesEvent(t *testing.T) {
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(fixtures.Bid{ID: "b1", BuyerID: "buyer-1", AmountCents: 1500})
	}))
	t.Cleanup(catalog.Close)

	srv := newTestServer(catalog.URL)
	ch, unsubscribe := srv.hub.subscribe()
	t.Cleanup(unsubscribe)

	body, _ := json.Marshal(map[string]any{"amountCents": 1500})
	req := httptest.NewRequest(http.MethodPost, "/account/listings/l1/bids", bytes.NewReader(body))
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-buyer", []string{"buyers"}))
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

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

func TestHandleWithdrawBidRequiresAuthentication(t *testing.T) {
	srv := newTestServer("http://unused")
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/account/listings/l1/bids/b1", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleCheckoutDerivesBuyerIDFromIdentityWithNoRequestBody(t *testing.T) {
	var gotPath string
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var body struct {
			BuyerID string `json:"buyerId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.BuyerID != "buyer-1" {
			t.Errorf("expected catalog-api to receive buyerId=buyer-1, got %q", body.BuyerID)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(fixtures.Order{ID: "o1", FinalPriceCents: 4500})
	}))
	t.Cleanup(catalog.Close)

	srv := newTestServer(catalog.URL)
	req := httptest.NewRequest(http.MethodPost, "/account/listings/l1/checkout", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-buyer", []string{"buyers"}))
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotPath != "/listings/l1/checkout" {
		t.Fatalf("unexpected catalog-api path: %q", gotPath)
	}
}

func TestHandleAccountOrdersUsesIdentityNotQueryParam(t *testing.T) {
	var gotQuery string
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode([]fixtures.Order{})
	}))
	t.Cleanup(catalog.Close)

	srv := newTestServer(catalog.URL)
	req := httptest.NewRequest(http.MethodGet, "/account/orders?buyerId=someone-elses-id", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-buyer", []string{"buyers"}))
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if gotQuery != "buyerId=buyer-1" {
		t.Fatalf("expected catalog-api to be queried with the identity-derived buyer-1, got %q", gotQuery)
	}
}

func TestHandleAccountStatsRejectsNonOperators(t *testing.T) {
	srv := newTestServer("http://unused")
	req := httptest.NewRequest(http.MethodGet, "/account/stats", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-buyer", []string{"buyers"}))
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a non-operator, got %d", rec.Code)
	}
}

func TestHandleAccountStatsAllowsOperators(t *testing.T) {
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/listings":
			_ = json.NewEncoder(w).Encode([]fixtures.Listing{{ID: "l1", Status: fixtures.ListingStatusActiveAuction}})
		case "/orders":
			_ = json.NewEncoder(w).Encode([]fixtures.Order{{ID: "o1"}})
		}
	}))
	t.Cleanup(catalog.Close)

	srv := newTestServer(catalog.URL)
	req := httptest.NewRequest(http.MethodGet, "/account/stats", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-operator", []string{"operators"}))
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for an operator, got %d: %s", rec.Code, rec.Body.String())
	}
	var stats statsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if stats.ActiveListings != 1 || stats.TotalOrders != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
