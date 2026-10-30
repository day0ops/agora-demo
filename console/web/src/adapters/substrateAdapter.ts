import type { AdapterResult } from "./types";

export interface SubstrateActor {
  id: string;
  state: string;
  pool: string;
  lastActivity: string;
}

export interface SubstrateStatus {
  actors: SubstrateActor[];
}

// Always empty today: no live Agent Substrate cluster is wired to this
// console. Swapping in a real client here requires no change to any caller.
export function getSubstrateStatus(): Promise<AdapterResult<SubstrateStatus>> {
  return Promise.resolve({
    status: "empty",
    reason: "Not connected to a live substrate.",
  });
}
