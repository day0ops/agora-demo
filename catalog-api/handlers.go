package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/day0ops/agora-demo/fixtures"
)

// maxRequestBodyBytes bounds every request body this API decodes, so a
// misbehaving or malicious client cannot exhaust memory with an unbounded
// upload.
const maxRequestBodyBytes = 1 << 20 // 1 MiB

// NewMux wires the catalog API's HTTP routes against the given store.
func NewMux(store *Store) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /listings", handleListListings(store))
	mux.HandleFunc("POST /listings", handleCreateListing(store))
	mux.HandleFunc("GET /listings/{id}", handleGetListing(store))
	mux.HandleFunc("GET /listings/{id}/bids", handleBidHistory(store))
	mux.HandleFunc("POST /listings/{id}/bids", handlePlaceBid(store))
	mux.HandleFunc("DELETE /listings/{id}/bids/{bidID}", handleWithdrawBid(store))
	mux.HandleFunc("POST /listings/{id}/autobid", handleSetAutoBidMax(store))
	mux.HandleFunc("POST /listings/{id}/checkout", handleCheckout(store))
	mux.HandleFunc("GET /orders/{id}", handleGetOrder(store))
	mux.HandleFunc("GET /orders", handleListOrders(store))
	return mux
}

func handleListListings(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := ListingFilter{
			Status:   fixtures.ListingStatus(r.URL.Query().Get("status")),
			SellerID: r.URL.Query().Get("sellerId"),
		}
		writeJSON(w, http.StatusOK, store.ListListings(filter))
	}
}

func handleGetListing(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l, err := store.GetListing(r.PathValue("id"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, l)
	}
}

type createListingRequest struct {
	SellerID      string               `json:"sellerId"`
	Title         string               `json:"title"`
	Description   string               `json:"description"`
	Category      string               `json:"category"`
	Type          fixtures.ListingType `json:"type"`
	PriceCents    int64                `json:"priceCents"`
	AuctionEndsAt *time.Time           `json:"auctionEndsAt,omitempty"`
}

func handleCreateListing(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createListingRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeBadRequest(w, err)
			return
		}
		l, err := store.CreateListing(
			req.SellerID, req.Title, req.Description, req.Category,
			req.Type, req.PriceCents, req.AuctionEndsAt,
		)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		w.Header().Set("Location", "/listings/"+l.ID)
		writeJSON(w, http.StatusCreated, l)
	}
}

// placeBidRequest intentionally has no auto-bid fields: PlaceBid is a manual
// bid only. Standing auto-bid ceilings are a separate concern, implemented
// alongside bids-mcp's set_autobid_max tool.
type placeBidRequest struct {
	BuyerID     string `json:"buyerId"`
	AmountCents int64  `json:"amountCents"`
	// CorrelationID is empty for every caller today (no agent is wired to
	// this API yet) - accepted now so a future real agent-driven caller can
	// set it without an API change once that wiring lands.
	CorrelationID string `json:"correlationId,omitempty"`
}

func handlePlaceBid(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req placeBidRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeBadRequest(w, err)
			return
		}
		bid, err := store.PlaceBidWithCorrelation(r.PathValue("id"), req.BuyerID, req.AmountCents, req.CorrelationID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, bid)
	}
}

type setAutoBidMaxRequest struct {
	BuyerID  string `json:"buyerId"`
	MaxCents int64  `json:"maxCents"`
}

func handleSetAutoBidMax(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req setAutoBidMaxRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeBadRequest(w, err)
			return
		}
		l, err := store.SetAutoBidMax(r.PathValue("id"), req.BuyerID, req.MaxCents)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, l)
	}
}

func handleWithdrawBid(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := store.WithdrawBid(r.PathValue("id"), r.PathValue("bidID")); err != nil {
			writeStoreError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleBidHistory(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		history, err := store.BidHistory(r.PathValue("id"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, history)
	}
}

// decodeJSONBody bounds and strictly decodes a JSON request body, rejecting
// unknown fields so a caller's typo (e.g. an agent misspelling a field name)
// fails loudly with a 400 instead of silently producing a zero-valued field.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("decode request body: %w", err)
	}
	return nil
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func writeBadRequest(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
}

// writeStoreError maps a Store sentinel error to the right HTTP status: 404
// for not-found, 400 for invalid input, 403 when the request is denied
// because of who the caller is rather than the listing's current state
// (e.g. checking out an auction you didn't win), 409 for a valid request
// that conflicts with the listing's current state.
func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrListingNotFound), errors.Is(err, ErrBidNotFound), errors.Is(err, ErrOrderNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	case errors.Is(err, ErrBuyerIDRequired),
		errors.Is(err, ErrSellerIDRequired),
		errors.Is(err, ErrTitleRequired),
		errors.Is(err, ErrInvalidPrice),
		errors.Is(err, ErrAuctionEndRequired),
		errors.Is(err, ErrAuctionEndInPast),
		errors.Is(err, ErrInvalidListingType):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, ErrNotTheWinningBidder):
		writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
	case errors.Is(err, ErrListingNotAuction),
		errors.Is(err, ErrAuctionClosed),
		errors.Is(err, ErrBidTooLow),
		errors.Is(err, ErrListingNotForSale):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})
	default:
		log.Printf("unmapped store error: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
	}
}
