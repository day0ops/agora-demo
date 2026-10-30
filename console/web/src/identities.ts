export interface Identity {
  id: string;
  name: string;
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

export const SELLERS: Identity[] = [
  { id: "seller-1", name: "Marina's Vintage Finds" },
  { id: "seller-2", name: "Dave's Garage Sale" },
  { id: "seller-3", name: "Collector's Corner" },
];
