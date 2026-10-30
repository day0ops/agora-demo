package main

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// server bundles the catalog client and risk scorer each tool needs.
type server struct {
	catalog *catalogClient
	scorer  RiskScorer
}

type ScoreBidRiskInput struct {
	ListingID   string `json:"listing_id" jsonschema:"the listing the bid is being placed on"`
	BuyerID     string `json:"buyer_id" jsonschema:"the bidder"`
	AmountCents int64  `json:"amount_cents" jsonschema:"the candidate bid amount, in cents"`
}
type ScoreBidRiskOutput struct {
	Risk  RiskScore `json:"risk,omitzero" jsonschema:"the risk assessment, if the lookup succeeded"`
	Error string    `json:"error,omitempty" jsonschema:"set instead of risk if the listing or its bid history could not be looked up"`
}

func (s *server) scoreBidRiskTool(
	ctx context.Context, _ *mcp.CallToolRequest, in ScoreBidRiskInput,
) (*mcp.CallToolResult, ScoreBidRiskOutput, error) {
	listing, err := s.catalog.listing(ctx, in.ListingID)
	if err != nil {
		if msg, ok := domainError(err); ok {
			return nil, ScoreBidRiskOutput{Error: msg}, nil
		}
		return nil, ScoreBidRiskOutput{}, err
	}

	history, err := s.catalog.bidHistory(ctx, in.ListingID)
	if err != nil {
		if msg, ok := domainError(err); ok {
			return nil, ScoreBidRiskOutput{Error: msg}, nil
		}
		return nil, ScoreBidRiskOutput{}, err
	}

	risk, err := s.scorer.ScoreBidRisk(ctx, BidRiskInput{
		ListingID:         in.ListingID,
		BuyerID:           in.BuyerID,
		AmountCents:       in.AmountCents,
		ListingPriceCents: listing.PriceCents,
		BidHistory:        history,
	})
	if err != nil {
		// The scorer itself failing (e.g. a future Jev-backed implementation's
		// own API call failing) is always an infrastructure problem under the
		// current RiskScorer contract (a plain error, with no domain/infra
		// distinction available to it) -- there is no catalog-api response to
		// classify here either way.
		return nil, ScoreBidRiskOutput{}, err
	}

	return nil, ScoreBidRiskOutput{Risk: risk}, nil
}

type CheckSellerReputationInput struct {
	SellerID string `json:"seller_id" jsonschema:"the seller to check"`
}
type CheckSellerReputationOutput struct {
	Reputation SellerReputation `json:"reputation,omitzero" jsonschema:"a summary of the seller's listing activity"`
	Error      string           `json:"error,omitempty" jsonschema:"set instead of a reputation if the lookup failed"`
}

func (s *server) checkSellerReputationTool(
	ctx context.Context, _ *mcp.CallToolRequest, in CheckSellerReputationInput,
) (*mcp.CallToolResult, CheckSellerReputationOutput, error) {
	// seller_id="" is not a lookup catalog-api can reject for us: GET
	// /listings?sellerId= is a filter, and an empty filter value matches
	// every listing in the marketplace, not "no seller". Guard locally
	// rather than silently attributing the whole catalog to seller "".
	if in.SellerID == "" {
		return nil, CheckSellerReputationOutput{Error: "seller_id is required"}, nil
	}

	listings, err := s.catalog.listingsBySeller(ctx, in.SellerID)
	if err != nil {
		if msg, ok := domainError(err); ok {
			return nil, CheckSellerReputationOutput{Error: msg}, nil
		}
		return nil, CheckSellerReputationOutput{}, err
	}
	return nil, CheckSellerReputationOutput{Reputation: computeReputation(in.SellerID, listings)}, nil
}
