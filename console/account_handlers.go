package main

import (
	"net/http"
	"strings"
)

// accountRoutes registers every handler gated behind the /account path
// prefix. oidc-gateway's AuthConfig enforces a real Keycloak session for
// anything under this prefix before a request ever reaches these handlers
// (see agentic-field-kit's agora-real-identity-ux design doc), so these do
// not re-check "is there a session" -- resolveIdentity returning
// ErrNotAuthenticated here means this specific request bypassed the gate,
// not that login is needed.
func (s *consoleServer) accountRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /account/me", s.handleAccountMe)
	mux.HandleFunc("GET /account/login", s.handleAccountLogin)
	mux.HandleFunc("POST /account/listings", s.handleCreateListing)
	mux.HandleFunc("POST /account/listings/{id}/bids", s.handlePlaceBid)
	mux.HandleFunc("DELETE /account/listings/{id}/bids/{bidID}", s.handleWithdrawBid)
	mux.HandleFunc("POST /account/listings/{id}/autobid", s.handleSetAutoBidMax)
	mux.HandleFunc("POST /account/listings/{id}/checkout", s.handleCheckout)
	mux.HandleFunc("GET /account/orders", s.handleAccountOrders)
	mux.HandleFunc("GET /account/stats", s.handleAccountStats)
}

func (s *consoleServer) handleAccountMe(w http.ResponseWriter, r *http.Request) {
	id, err := s.resolveIdentity(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, id)
}

// handleAccountLogin only ever runs once oidc-gateway's ext-auth-service has
// already established a session for this path prefix -- it exists purely so
// a logged-out visit to it triggers that gate's redirect-to-Keycloak flow,
// then bounces back to returnTo (or "/") once login completes. returnTo is
// restricted to a same-site path to avoid being used as an open redirect.
func (s *consoleServer) handleAccountLogin(w http.ResponseWriter, r *http.Request) {
	returnTo := r.URL.Query().Get("returnTo")
	if returnTo == "" || !strings.HasPrefix(returnTo, "/") || strings.HasPrefix(returnTo, "//") {
		returnTo = "/"
	}
	http.Redirect(w, r, returnTo, http.StatusFound)
}

func (s *consoleServer) handleCreateListing(w http.ResponseWriter, r *http.Request) {
	id, err := s.resolveIdentity(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}
	var req createListingRequest
	if err := decodeBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	// SellerID always comes from the resolved identity, never the request
	// body -- a caller cannot list on another seller's behalf by passing a
	// different sellerId.
	req.SellerID = id.FixtureID
	l, err := s.catalog.createListing(r.Context(), req)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	s.hub.publish(Event{Type: "listing.created", Data: l})
	writeJSON(w, http.StatusCreated, l)
}

type placeBidRequest struct {
	AmountCents int64 `json:"amountCents"`
}

func (s *consoleServer) handlePlaceBid(w http.ResponseWriter, r *http.Request) {
	id, err := s.resolveIdentity(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}
	var req placeBidRequest
	if err := decodeBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	listingID := r.PathValue("id")
	bid, err := s.catalog.placeBid(r.Context(), listingID, id.FixtureID, req.AmountCents)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	s.hub.publish(Event{Type: "bid.placed", Data: map[string]any{"listingId": listingID, "bid": bid}})
	writeJSON(w, http.StatusCreated, bid)
}

type setAutoBidMaxRequest struct {
	MaxCents int64 `json:"maxCents"`
}

func (s *consoleServer) handleSetAutoBidMax(w http.ResponseWriter, r *http.Request) {
	id, err := s.resolveIdentity(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}
	var req setAutoBidMaxRequest
	if err := decodeBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	listingID := r.PathValue("id")
	l, err := s.catalog.setAutoBidMax(r.Context(), listingID, id.FixtureID, req.MaxCents)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	s.hub.publish(Event{Type: "autobid.set", Data: map[string]any{"listingId": listingID, "listing": l}})
	writeJSON(w, http.StatusOK, l)
}

func (s *consoleServer) handleWithdrawBid(w http.ResponseWriter, r *http.Request) {
	if _, err := s.resolveIdentity(r); err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}
	listingID := r.PathValue("id")
	bidID := r.PathValue("bidID")
	if err := s.catalog.withdrawBid(r.Context(), listingID, bidID); err != nil {
		writeCatalogError(w, err)
		return
	}
	s.hub.publish(Event{Type: "bid.withdrawn", Data: map[string]any{"listingId": listingID, "bidId": bidID}})
	w.WriteHeader(http.StatusNoContent)
}

func (s *consoleServer) handleCheckout(w http.ResponseWriter, r *http.Request) {
	id, err := s.resolveIdentity(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}
	listingID := r.PathValue("id")
	order, err := s.catalog.checkout(r.Context(), listingID, id.FixtureID)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	s.hub.publish(Event{Type: "order.created", Data: order})
	writeJSON(w, http.StatusCreated, order)
}

func (s *consoleServer) handleAccountOrders(w http.ResponseWriter, r *http.Request) {
	id, err := s.resolveIdentity(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}
	orders, err := s.catalog.listOrders(r.Context(), id.FixtureID, "")
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

// handleAccountStats is the only /account/* data here that isn't the
// caller's own (every buyer's/seller's activity, not just theirs), so a
// plain session isn't enough -- it additionally requires the operators role
// before delegating to handlers.go's handleStats.
func (s *consoleServer) handleAccountStats(w http.ResponseWriter, r *http.Request) {
	id, err := s.resolveIdentity(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}
	if id.Role != "operators" {
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "operator role required"})
		return
	}
	s.handleStats(w, r)
}
