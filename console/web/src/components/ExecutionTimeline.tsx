import { Fragment } from "react";
import { NotConnectedBanner } from "./NotConnectedBanner";
import type { AdapterResult } from "../adapters/types";
import type { AgentJourney } from "../adapters/executionAdapter";

// Renders the horizontal-timeline shape when real data exists; today
// result.status is always "empty" (see executionAdapter), so this always
// renders the shared not-connected banner instead of timeline nodes.
export function ExecutionTimeline({
  result,
}: {
  result: AdapterResult<AgentJourney>;
}) {
  if (result.status === "empty")
    return <NotConnectedBanner reason={result.reason} />;
  return (
    <div className="execution-timeline">
      {result.data.steps.map((s, i) => (
        <Fragment key={`${s.actor}-${i}`}>
          {i > 0 && <span className="execution-timeline-arrow">&rarr;</span>}
          <span className="execution-timeline-step">{s.actor}</span>
        </Fragment>
      ))}
    </div>
  );
}
