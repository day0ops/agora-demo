package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
)

func fakeToken(t *testing.T, username string, groups []string) string {
	t.Helper()
	payload, err := json.Marshal(forwardedClaims{PreferredUsername: username, Groups: groups})
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func TestResolveIdentityReturnsErrNotAuthenticatedWithNoHeader(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	req := httptest.NewRequest("GET", "/account/me", nil)

	if _, err := srv.resolveIdentity(req); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestResolveIdentityMapsDemoBuyerToBuyerOneFixture(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	req := httptest.NewRequest("GET", "/account/me", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-buyer", []string{"buyers"}))

	id, err := srv.resolveIdentity(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Username != "demo-buyer" || id.Role != "buyers" || id.FixtureID != "buyer-1" {
		t.Fatalf("unexpected identity: %+v", id)
	}
}

func TestResolveIdentityPicksHighestPriorityRoleWhenMultipleGroupsPresent(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	req := httptest.NewRequest("GET", "/account/me", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "someone", []string{"buyers", "operators"}))

	id, err := srv.resolveIdentity(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Role != "operators" {
		t.Fatalf("expected operators to outrank buyers, got role %q", id.Role)
	}
}

func TestResolveIdentityLeavesFixtureIDEmptyForUnmappedUsername(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	req := httptest.NewRequest("GET", "/account/me", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-operator", []string{"operators"}))

	id, err := srv.resolveIdentity(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.FixtureID != "" {
		t.Fatalf("expected no fixture mapping for demo-operator, got %q", id.FixtureID)
	}
}

func TestResolveIdentityIncludesSellerTierForSellers(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	req := httptest.NewRequest("GET", "/account/me", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-seller", []string{"sellers"}))

	id, err := srv.resolveIdentity(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.SellerTier != "pro" {
		t.Fatalf("expected demo-seller to have sellerTier pro, got %q", id.SellerTier)
	}
}

func TestResolveIdentityLeavesSellerTierEmptyForBuyers(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	req := httptest.NewRequest("GET", "/account/me", nil)
	req.Header.Set(forwardedTokenHeader, fakeToken(t, "demo-buyer", []string{"buyers"}))

	id, err := srv.resolveIdentity(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.SellerTier != "" {
		t.Fatalf("expected no sellerTier for a buyer, got %q", id.SellerTier)
	}
}

func TestResolveIdentityRejectsAMalformedToken(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})
	req := httptest.NewRequest("GET", "/account/me", nil)
	req.Header.Set(forwardedTokenHeader, "not-a-jwt")

	if _, err := srv.resolveIdentity(req); err == nil {
		t.Fatal("expected an error for a malformed token")
	}
}
