package main

import (
	"context"
	"net/http"
	"net/url"

	"github.com/day0ops/marketplace-demo/fixtures"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type checkoutBody struct {
	BuyerID string `json:"buyerId"`
}

// listingPath builds a URL path under /listings/{id}, escaping the ID so an
// agent-supplied value containing a slash or other reserved character can
// never be misrouted to a different endpoint.
func listingPath(listingID string, suffix string) string {
	return "/listings/" + url.PathEscape(listingID) + suffix
}

type CheckoutInput struct {
	ListingID string `json:"listing_id" jsonschema:"the listing to buy - an active buy-now listing, or a closed auction you won"`
	BuyerID   string `json:"buyer_id" jsonschema:"the buyer checking out"`
}
type CheckoutOutput struct {
	Order fixtures.Order `json:"order,omitzero" jsonschema:"the created order, if checkout succeeded"`
	Error string         `json:"error,omitempty" jsonschema:"why checkout was rejected, if it was - e.g. not the winning bidder, or already sold"`
}

func (c *catalogClient) checkoutTool(
	ctx context.Context, _ *mcp.CallToolRequest, in CheckoutInput,
) (*mcp.CallToolResult, CheckoutOutput, error) {
	var order fixtures.Order
	body := checkoutBody{BuyerID: in.BuyerID}
	if err := c.do(ctx, http.MethodPost, listingPath(in.ListingID, "/checkout"), body, &order); err != nil {
		if msg, ok := domainError(err); ok {
			return nil, CheckoutOutput{Error: msg}, nil
		}
		return nil, CheckoutOutput{}, err
	}
	return nil, CheckoutOutput{Order: order}, nil
}

type GetOrderInput struct {
	OrderID string `json:"order_id" jsonschema:"the order to look up"`
}
type GetOrderOutput struct {
	Order fixtures.Order `json:"order,omitzero" jsonschema:"the order, if found"`
	Error string         `json:"error,omitempty" jsonschema:"why the lookup failed, if it did"`
}

func (c *catalogClient) getOrderTool(
	ctx context.Context, _ *mcp.CallToolRequest, in GetOrderInput,
) (*mcp.CallToolResult, GetOrderOutput, error) {
	var order fixtures.Order
	path := "/orders/" + url.PathEscape(in.OrderID)
	if err := c.do(ctx, http.MethodGet, path, nil, &order); err != nil {
		if msg, ok := domainError(err); ok {
			return nil, GetOrderOutput{Error: msg}, nil
		}
		return nil, GetOrderOutput{}, err
	}
	return nil, GetOrderOutput{Order: order}, nil
}

type ListOrdersInput struct {
	BuyerID  string `json:"buyer_id,omitempty" jsonschema:"filter to orders for this buyer"`
	SellerID string `json:"seller_id,omitempty" jsonschema:"filter to orders for this seller"`
}
type ListOrdersOutput struct {
	Orders []fixtures.Order `json:"orders,omitempty" jsonschema:"matching orders, oldest first"`
	Error  string           `json:"error,omitempty" jsonschema:"why the lookup failed, if it did"`
}

func (c *catalogClient) listOrdersTool(
	ctx context.Context, _ *mcp.CallToolRequest, in ListOrdersInput,
) (*mcp.CallToolResult, ListOrdersOutput, error) {
	q := url.Values{}
	if in.BuyerID != "" {
		q.Set("buyerId", in.BuyerID)
	}
	if in.SellerID != "" {
		q.Set("sellerId", in.SellerID)
	}
	path := "/orders"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var orders []fixtures.Order
	if err := c.do(ctx, http.MethodGet, path, nil, &orders); err != nil {
		if msg, ok := domainError(err); ok {
			return nil, ListOrdersOutput{Error: msg}, nil
		}
		return nil, ListOrdersOutput{}, err
	}
	return nil, ListOrdersOutput{Orders: orders}, nil
}
