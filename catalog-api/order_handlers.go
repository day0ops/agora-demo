package main

import "net/http"

type checkoutRequest struct {
	BuyerID string `json:"buyerId"`
}

func handleCheckout(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req checkoutRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeBadRequest(w, err)
			return
		}
		order, err := store.Checkout(r.PathValue("id"), req.BuyerID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		w.Header().Set("Location", "/orders/"+order.ID)
		writeJSON(w, http.StatusCreated, order)
	}
}

func handleGetOrder(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		order, err := store.GetOrder(r.PathValue("id"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, order)
	}
}

func handleListOrders(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := OrderFilter{
			BuyerID:  r.URL.Query().Get("buyerId"),
			SellerID: r.URL.Query().Get("sellerId"),
		}
		writeJSON(w, http.StatusOK, store.ListOrders(filter))
	}
}
