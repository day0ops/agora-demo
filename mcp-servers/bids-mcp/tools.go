package main

import (
	"context"
	"net/http"
	"net/url"

	"github.com/day0ops/marketplace-demo/fixtures"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type placeBidBody struct {
	BuyerID     string `json:"buyerId"`
	AmountCents int64  `json:"amountCents"`
}

type setAutoBidMaxBody struct {
	BuyerID  string `json:"buyerId"`
	MaxCents int64  `json:"maxCents"`
}

// listingPath builds a URL path under /listings/{id}, escaping the ID so an
// agent-supplied value containing a slash or other reserved character can
// never be misrouted to a different endpoint.
func listingPath(listingID string, suffix string) string {
	return "/listings/" + url.PathEscape(listingID) + suffix
}

type PlaceBidInput struct {
	ListingID   string `json:"listing_id" jsonschema:"the listing to bid on"`
	BuyerID     string `json:"buyer_id" jsonschema:"the bidding buyer's id"`
	AmountCents int64  `json:"amount_cents" jsonschema:"the bid amount, in cents"`
}
type PlaceBidOutput struct {
	Bid   fixtures.Bid `json:"bid,omitzero" jsonschema:"the recorded bid, if it was accepted"`
	Error string       `json:"error,omitempty" jsonschema:"why the bid was rejected, if it was"`
}

func (c *catalogClient) placeBidTool(
	ctx context.Context, _ *mcp.CallToolRequest, in PlaceBidInput,
) (*mcp.CallToolResult, PlaceBidOutput, error) {
	var bid fixtures.Bid
	body := placeBidBody{BuyerID: in.BuyerID, AmountCents: in.AmountCents}
	if err := c.do(ctx, http.MethodPost, listingPath(in.ListingID, "/bids"), body, &bid); err != nil {
		if msg, ok := domainError(err); ok {
			return nil, PlaceBidOutput{Error: msg}, nil
		}
		return nil, PlaceBidOutput{}, err
	}
	return nil, PlaceBidOutput{Bid: bid}, nil
}

type GetCurrentPriceInput struct {
	ListingID string `json:"listing_id" jsonschema:"the listing to check"`
}
type GetCurrentPriceOutput struct {
	Listing fixtures.Listing `json:"listing,omitzero" jsonschema:"the listing, including its current price and status"`
	Error   string           `json:"error,omitempty" jsonschema:"why the lookup failed, if it did"`
}

func (c *catalogClient) getCurrentPriceTool(
	ctx context.Context, _ *mcp.CallToolRequest, in GetCurrentPriceInput,
) (*mcp.CallToolResult, GetCurrentPriceOutput, error) {
	var l fixtures.Listing
	if err := c.do(ctx, http.MethodGet, listingPath(in.ListingID, ""), nil, &l); err != nil {
		if msg, ok := domainError(err); ok {
			return nil, GetCurrentPriceOutput{Error: msg}, nil
		}
		return nil, GetCurrentPriceOutput{}, err
	}
	return nil, GetCurrentPriceOutput{Listing: l}, nil
}

type GetBidHistoryInput struct {
	ListingID string `json:"listing_id" jsonschema:"the listing whose bids to list"`
}
type GetBidHistoryOutput struct {
	Bids  []fixtures.Bid `json:"bids,omitempty" jsonschema:"bids for the listing, oldest first"`
	Error string         `json:"error,omitempty" jsonschema:"why the lookup failed, if it did"`
}

func (c *catalogClient) getBidHistoryTool(
	ctx context.Context, _ *mcp.CallToolRequest, in GetBidHistoryInput,
) (*mcp.CallToolResult, GetBidHistoryOutput, error) {
	var bids []fixtures.Bid
	if err := c.do(ctx, http.MethodGet, listingPath(in.ListingID, "/bids"), nil, &bids); err != nil {
		if msg, ok := domainError(err); ok {
			return nil, GetBidHistoryOutput{Error: msg}, nil
		}
		return nil, GetBidHistoryOutput{}, err
	}
	return nil, GetBidHistoryOutput{Bids: bids}, nil
}

type SetAutoBidMaxInput struct {
	ListingID string `json:"listing_id" jsonschema:"the listing to bid on"`
	BuyerID   string `json:"buyer_id" jsonschema:"the buyer registering the ceiling"`
	MaxCents  int64  `json:"max_cents" jsonschema:"the most the buyer is willing to pay, in cents"`
}
type SetAutoBidMaxOutput struct {
	Listing fixtures.Listing `json:"listing,omitzero" jsonschema:"the listing after proxy bidding resolved"`
	Error   string           `json:"error,omitempty" jsonschema:"why the request was rejected, if it was"`
}

func (c *catalogClient) setAutoBidMaxTool(
	ctx context.Context, _ *mcp.CallToolRequest, in SetAutoBidMaxInput,
) (*mcp.CallToolResult, SetAutoBidMaxOutput, error) {
	var l fixtures.Listing
	body := setAutoBidMaxBody{BuyerID: in.BuyerID, MaxCents: in.MaxCents}
	if err := c.do(ctx, http.MethodPost, listingPath(in.ListingID, "/autobid"), body, &l); err != nil {
		if msg, ok := domainError(err); ok {
			return nil, SetAutoBidMaxOutput{Error: msg}, nil
		}
		return nil, SetAutoBidMaxOutput{}, err
	}
	return nil, SetAutoBidMaxOutput{Listing: l}, nil
}

type WithdrawBidInput struct {
	ListingID string `json:"listing_id" jsonschema:"the listing the bid was placed on"`
	BidID     string `json:"bid_id" jsonschema:"the id of the bid to withdraw"`
}
type WithdrawBidOutput struct {
	Success bool   `json:"success" jsonschema:"whether the bid was withdrawn"`
	Error   string `json:"error,omitempty" jsonschema:"why the withdrawal failed, if it did"`
}

func (c *catalogClient) withdrawBidTool(
	ctx context.Context, _ *mcp.CallToolRequest, in WithdrawBidInput,
) (*mcp.CallToolResult, WithdrawBidOutput, error) {
	path := listingPath(in.ListingID, "/bids/"+url.PathEscape(in.BidID))
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		if msg, ok := domainError(err); ok {
			return nil, WithdrawBidOutput{Error: msg}, nil
		}
		return nil, WithdrawBidOutput{}, err
	}
	return nil, WithdrawBidOutput{Success: true}, nil
}
