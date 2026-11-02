package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/day0ops/agora-demo/fixtures"
)

// consoleServer bundles everything the console's HTTP handlers need.
type consoleServer struct {
	catalog *catalogClient
	hub     *eventHub
	invoker AgentInvoker
}

func newConsoleServer(catalog *catalogClient, hub *eventHub, invoker AgentInvoker) *consoleServer {
	return &consoleServer{catalog: catalog, hub: hub, invoker: invoker}
}

func (s *consoleServer) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/listings", s.handleListListings)
	mux.HandleFunc("GET /api/listings/{id}", s.handleGetListing)
	mux.HandleFunc("GET /api/listings/{id}/bids", s.handleBidHistory)
	mux.HandleFunc("GET /api/events", s.handleEvents)
	mux.HandleFunc("GET /api/version", s.handleVersion)
	s.accountRoutes(mux)
	return mux
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeCatalogError relays catalog-api's own status code and message
// whenever it produced one (so the frontend can tell "not found" apart from
// "conflict" apart from "forbidden"), and otherwise reports a 502, since
// anything else means catalog-api or the network itself failed, not that
// the request was understood and declined.
func writeCatalogError(w http.ResponseWriter, err error) {
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		writeJSON(w, apiErr.Status, errorResponse{Error: apiErr.Message})
		return
	}
	writeJSON(w, http.StatusBadGateway, errorResponse{Error: "catalog-api unavailable: " + err.Error()})
}

// maxRequestBodyBytes bounds every request body this API decodes, so a
// misbehaving or malicious client cannot exhaust memory with an unbounded
// upload. Mirrors catalog-api's own limit (catalog-api/handlers.go) - the
// console is the more exposed of the two services, since it's the one a
// browser (and potentially the internet) reaches directly.
const maxRequestBodyBytes = 1 << 20 // 1 MiB

func decodeBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

type versionResponse struct {
	Version string `json:"version"`
}

func (s *consoleServer) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, versionResponse{Version: version})
}

func (s *consoleServer) handleListListings(w http.ResponseWriter, r *http.Request) {
	listings, err := s.catalog.listListings(r.Context(), r.URL.Query().Get("status"), r.URL.Query().Get("sellerId"))
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listings)
}

func (s *consoleServer) handleGetListing(w http.ResponseWriter, r *http.Request) {
	l, err := s.catalog.getListing(r.Context(), r.PathValue("id"))
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func (s *consoleServer) handleBidHistory(w http.ResponseWriter, r *http.Request) {
	bids, err := s.catalog.bidHistory(r.Context(), r.PathValue("id"))
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bids)
}

type chatRequest struct {
	Agent     string `json:"agent"`
	SessionID string `json:"sessionId"`
	Message   string `json:"message"`
}
type chatResponse struct {
	Reply string `json:"reply,omitempty"`
	Error string `json:"error,omitempty"`
}

// forwardedTokenHeader is the header oidc-gateway's AuthConfig
// (forwardAccessTokenHeader) sets on every request once the visitor has a
// real Keycloak session -- see the agora-oidc-agentgateway-a2a design doc.
const forwardedTokenHeader = "X-Agora-User-Token"

// handleChat always responds 200: an agent being unavailable, or the caller
// not being authenticated, are normal states the frontend renders as a chat
// message, not network failures.
func (s *consoleServer) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := decodeBody(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	token := r.Header.Get(forwardedTokenHeader)
	if token == "" {
		writeJSON(w, http.StatusOK, chatResponse{Error: ErrNotAuthenticated.Error()})
		return
	}
	reply, err := s.invoker.Send(r.Context(), req.Agent, req.SessionID, req.Message, token)
	if err != nil {
		writeJSON(w, http.StatusOK, chatResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, chatResponse{Reply: reply})
}

// handleEvents streams the live event feed over SSE. Every operator/seller/
// buyer view subscribes here for real-time updates.
func (s *consoleServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ch, unsubscribe := s.hub.subscribe()
	defer unsubscribe()

	// An idle stream gets reaped by intermediate proxies/load balancers
	// (this demo runs behind one); a comment line keeps the connection
	// visibly alive without the client treating it as a real event.
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case evt, open := <-ch:
			if !open {
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

type statsResponse struct {
	ActiveListings   int `json:"activeListings"`
	TotalOrders      int `json:"totalOrders"`
	ConnectedClients int `json:"connectedClients"`
}

// handleStats feeds the operator console's live metrics strip from real
// catalog-api data plus the hub's own connected-client count. Gated to the
// operators role by handleAccountStats, its only caller -- see
// account_handlers.go.
func (s *consoleServer) handleStats(w http.ResponseWriter, r *http.Request) {
	listings, err := s.catalog.listListings(r.Context(), "", "")
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	orders, err := s.catalog.listOrders(r.Context(), "", "")
	if err != nil {
		writeCatalogError(w, err)
		return
	}

	active := 0
	for _, l := range listings {
		if l.Status == fixtures.ListingStatusActiveAuction || l.Status == fixtures.ListingStatusActiveBuyNow {
			active++
		}
	}

	writeJSON(w, http.StatusOK, statsResponse{
		ActiveListings:   active,
		TotalOrders:      len(orders),
		ConnectedClients: s.hub.subscriberCount(),
	})
}
