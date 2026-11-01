package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/day0ops/agora-demo/fixtures"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	_, _, listings, bids, orders := fixtures.Seed(time.Now())
	store := NewStore(listings, bids)
	store.SeedOrders(orders)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           NewMux(store),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("catalog-api listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("catalog-api: %v", err)
	}
}
