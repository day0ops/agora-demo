package main

import (
	"testing"

	"github.com/day0ops/agora-demo/fixtures"
)

func TestComputeReputationCountsByStatus(t *testing.T) {
	listings := []fixtures.Listing{
		{Status: fixtures.ListingStatusSold},
		{Status: fixtures.ListingStatusSold},
		{Status: fixtures.ListingStatusUnsoldExpired},
		{Status: fixtures.ListingStatusWithdrawn},
		{Status: fixtures.ListingStatusActiveAuction},
	}

	got := computeReputation("seller-1", listings)
	if got.SellerID != "seller-1" {
		t.Fatalf("unexpected seller ID: %+v", got)
	}
	if got.TotalListings != 5 || got.SoldCount != 2 || got.UnsoldExpiredCount != 1 ||
		got.WithdrawnCount != 1 || got.ActiveCount != 1 {
		t.Fatalf("unexpected counts: %+v", got)
	}
}

func TestComputeReputationSoldRateExcludesStillActiveListings(t *testing.T) {
	listings := []fixtures.Listing{
		{Status: fixtures.ListingStatusSold},
		{Status: fixtures.ListingStatusSold},
		{Status: fixtures.ListingStatusSold},
		{Status: fixtures.ListingStatusUnsoldExpired},
		{Status: fixtures.ListingStatusActiveAuction}, // not yet closed, must not count either way
	}

	got := computeReputation("seller-1", listings)
	want := 3.0 / 4.0 // 3 sold out of 4 CLOSED listings (3 sold + 1 unsold-expired)
	if got.SoldRate != want {
		t.Fatalf("expected sold rate %f, got %f", want, got.SoldRate)
	}
}

func TestComputeReputationSoldRateZeroWithNoClosedListings(t *testing.T) {
	listings := []fixtures.Listing{{Status: fixtures.ListingStatusActiveAuction}}

	got := computeReputation("seller-1", listings)
	if got.SoldRate != 0 {
		t.Fatalf("expected sold rate 0 with no closed listings, got %f", got.SoldRate)
	}
}

func TestComputeReputationTenure(t *testing.T) {
	newSeller := computeReputation("s1", []fixtures.Listing{{}, {}})
	if newSeller.Tenure != "new" {
		t.Fatalf("expected a seller with 2 listings to be tenure=new, got %q", newSeller.Tenure)
	}

	established := computeReputation("s1", []fixtures.Listing{{}, {}, {}})
	if established.Tenure != "established" {
		t.Fatalf("expected a seller with 3 listings to be tenure=established, got %q", established.Tenure)
	}
}

func TestComputeReputationNoListings(t *testing.T) {
	got := computeReputation("s1", nil)
	if got.TotalListings != 0 || got.SoldRate != 0 || got.Tenure != "new" {
		t.Fatalf("unexpected result for a seller with no listings: %+v", got)
	}
}
