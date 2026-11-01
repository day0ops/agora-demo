import { NotConnectedBanner } from "./NotConnectedBanner";
import type { AdapterResult } from "../adapters/types";
import type { AgentJourney } from "../adapters/executionAdapter";

const COLUMNS = ["Actor", "State", "Pool", "MCP call", "Policy"] as const;

// Real column shape always rendered; today result.status is always "empty"
// (see executionAdapter), so the body is the shared not-connected banner
// spanning every column rather than a fabricated row.
export function TechnicalInspector({
  result,
}: {
  result: AdapterResult<AgentJourney>;
}) {
  const header = (
    <tr>
      {COLUMNS.map((c) => (
        <th key={c}>{c}</th>
      ))}
    </tr>
  );
  if (result.status === "empty") {
    return (
      <div className="inspector-table-wrap">
        <table className="inspector-table">
          <thead>{header}</thead>
          <tbody>
            <tr>
              <td colSpan={COLUMNS.length}>
                <NotConnectedBanner reason={result.reason} />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    );
  }
  return (
    <div className="inspector-table-wrap">
      <table className="inspector-table">
        <thead>{header}</thead>
        <tbody>
          {result.data.steps.map((s, i) => (
            <tr key={i}>
              <td>{s.actor}</td>
              <td>{s.state}</td>
              <td>{s.pool}</td>
              <td>{s.mcpCall}</td>
              <td>{s.policy}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
