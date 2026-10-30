import type { AdapterResult } from "./types";

export interface AgentJourneyStep {
  actor: string;
  state: string;
  pool: string;
  mcpCall: string;
  policy: string;
}

export interface AgentJourney {
  steps: AgentJourneyStep[];
}

// Always empty today: nothing populates a Bid/Order's correlationId yet, so
// there is never a real execution to assemble. Swapping in a real client
// here requires no change to any caller.
export function getAgentJourney(
  correlationId: string | undefined,
): Promise<AdapterResult<AgentJourney>> {
  return Promise.resolve({
    status: "empty",
    reason: correlationId
      ? "Agent participation appears here when running against a live substrate."
      : "No correlation id recorded for this activity.",
  });
}
