package main

import (
	"context"
	"fmt"

	"github.com/day0ops/marketplace-demo/fixtures"
)

// RiskScore is a calibrated-style risk assessment for a candidate bid.
type RiskScore struct {
	Score    float64  `json:"score"`     // 0.0 (no concern) to 1.0 (high concern)
	Reasons  []string `json:"reasons"`   // human-readable signals that contributed; empty when Score is 0
	ScoredBy string   `json:"scored_by"` // which scorer produced this -- "heuristic" today
}

// BidRiskInput is the context a RiskScorer needs to assess one candidate
// bid. BidHistory is every bid already on the listing, oldest first -- it
// does not include the candidate bid itself.
type BidRiskInput struct {
	ListingID         string
	BuyerID           string
	AmountCents       int64
	ListingPriceCents int64
	BidHistory        []fixtures.Bid
}

// RiskScorer assesses the fraud/shill-bid risk of a candidate bid.
//
// The production default (NewHeuristicScorer) is a local, rule-based scorer.
// A fast external structured-decision model -- e.g. TypeSafe.ai's Jev
// "System One" model, built for exactly this kind of low-latency calibrated
// classification rather than open-ended generation -- is a natural fit for
// this interface. It is deliberately NOT wired in here: this project has no
// confirmed Jev API contract, credentials, or endpoint, only a blog post, so
// building against it now would mean shipping an integration nobody has
// verified actually works. Swap NewScorer's return value for a real
// Jev-backed implementation once that access exists; nothing else in this
// package needs to change.
type RiskScorer interface {
	ScoreBidRisk(ctx context.Context, in BidRiskInput) (RiskScore, error)
}

// NewScorer returns the risk scorer this server uses. Currently always the
// heuristic scorer -- see the RiskScorer doc comment for the intended
// Jev-backed extension point.
func NewScorer() RiskScorer {
	return NewHeuristicScorer()
}

// heuristicScorer flags two specific, explainable patterns computable from a
// single listing's own bid history, and nothing more. It deliberately does
// not attempt deeper fraud modeling (e.g. cross-listing account behavior,
// device/IP signals) that this demo's dataset has no way to support
// honestly; a modest, correct heuristic beats a sophisticated-looking one
// built on data that doesn't exist.
type heuristicScorer struct{}

// NewHeuristicScorer returns the default, local RiskScorer.
func NewHeuristicScorer() RiskScorer {
	return heuristicScorer{}
}

func (heuristicScorer) ScoreBidRisk(_ context.Context, in BidRiskInput) (RiskScore, error) {
	var reasons []string
	var score float64

	if leader, ok := currentLeader(in.BidHistory); ok && leader == in.BuyerID {
		// No legitimate reason for a buyer to manually re-bid over their own
		// standing lead -- raising a ceiling is what set_autobid_max is for.
		score += 0.5
		reasons = append(reasons, "buyer already holds the current highest bid on this listing")
	}

	if flagged, manualCount, distinctBuyers := isManualPingPong(in.BidHistory, in.BuyerID); flagged {
		score += 0.3
		reasons = append(reasons, fmt.Sprintf(
			"all %d manual bids on this listing come from just %d buyer(s)", manualCount, distinctBuyers,
		))
	}

	if score > 1.0 {
		score = 1.0
	}

	return RiskScore{Score: score, Reasons: reasons, ScoredBy: "heuristic"}, nil
}

// currentLeader returns the buyer with the highest amount in history, or
// ("", false) if history is empty.
func currentLeader(history []fixtures.Bid) (string, bool) {
	var leader string
	var highest int64
	for _, b := range history {
		if b.AmountCents > highest {
			highest = b.AmountCents
			leader = b.BuyerID
		}
	}
	return leader, leader != ""
}

// manualPingPongThreshold is deliberately well above a normal two-party
// bidding war (bid, counter-bid, counter-counter-bid is 3 manual bids from 2
// buyers and is completely ordinary) -- the signal is meant to catch a
// sustained, unusually long exchange confined to the same two buyers, not
// the first counter-offer.
const manualPingPongThreshold = 5

// isManualPingPong reports whether at least manualPingPongThreshold manual
// (non-auto-generated) bids -- counting the candidate bid itself -- involve
// 2 or fewer distinct buyers, along with those counts for an accurate reason
// string. Auto-generated proxy bids are excluded: the proxy-bidding engine
// legitimately produces rapid two-party back-and-forth bidding as part of
// normal operation, and that must not be mistaken for shill behavior.
func isManualPingPong(history []fixtures.Bid, candidateBuyerID string) (flagged bool, manualCount, distinctBuyers int) {
	manualBidders := map[string]bool{candidateBuyerID: true}
	manualCount = 1 // the candidate bid is manual by definition (this scorer is for manual bids)
	for _, b := range history {
		if b.IsAutoBid {
			continue
		}
		manualBidders[b.BuyerID] = true
		manualCount++
	}
	distinctBuyers = len(manualBidders)
	return manualCount >= manualPingPongThreshold && distinctBuyers <= 2, manualCount, distinctBuyers
}
