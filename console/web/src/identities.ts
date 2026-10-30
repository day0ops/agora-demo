export interface Identity {
  id: string;
  name: string;
}

// Mirrors fixtures.SellerTier (fixtures/types.go) - a Pro-tier seller gets a
// stronger model behind their pricing assistant; the routing itself is only
// surfaced in Phase 3's Operator Studio, not here.
export type SellerTier = "basic" | "pro";

export interface Seller extends Identity {
  tier: SellerTier;
}

// The console has no authentication yet (see console/agent_invoker.go for
// the equivalent honest-stub pattern on the agent-invocation side) - these
// mirror the accounts fixtures.Seed() (fixtures/seed.go) creates, so the SE
// can pick who they are "acting as" for the demo.
export const BUYERS: Identity[] = [
  { id: "buyer-1", name: "Alex Chen" },
  { id: "buyer-2", name: "Jordan Patel" },
  { id: "buyer-3", name: "Sam Rivera" },
];

export const SELLERS: Seller[] = [
  { id: "seller-1", name: "Marina's Vintage Finds", tier: "pro" },
  { id: "seller-2", name: "Dave's Garage Sale", tier: "basic" },
  { id: "seller-3", name: "Collector's Corner", tier: "pro" },
];
