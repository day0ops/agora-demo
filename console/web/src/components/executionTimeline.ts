import { notConnectedBanner } from "./notConnectedBanner";
import { escapeHtml } from "../html";
import type { AdapterResult } from "../adapters/types";
import type { AgentJourney } from "../adapters/executionAdapter";

// Renders the section 5 horizontal-timeline shape when real data exists;
// today result.status is always "empty" (see executionAdapter), so this
// always renders the shared not-connected banner instead of timeline nodes.
export function executionTimeline(result: AdapterResult<AgentJourney>): string {
  if (result.status === "empty") {
    return notConnectedBanner(result.reason);
  }
  return `
    <div class="execution-timeline">
      ${result.data.steps
        .map(
          (s) =>
            `<span class="execution-timeline-step">${escapeHtml(s.actor)}</span>`,
        )
        .join('<span class="execution-timeline-arrow">&rarr;</span>')}
    </div>
  `;
}
