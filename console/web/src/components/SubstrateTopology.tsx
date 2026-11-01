import { NotConnectedBanner } from "./NotConnectedBanner";
import type { AdapterResult } from "../adapters/types";
import type { SubstrateStatus } from "../adapters/substrateAdapter";

const COLUMNS = ["Actor", "State", "Pool", "Last activity"] as const;

// Real column shape always rendered; today getSubstrateStatus always
// resolves "empty" (no live Agent Substrate cluster is wired to this
// console), so the body is the shared not-connected banner spanning every
// column rather than a fabricated row.
export function SubstrateTopology({
  result,
}: {
  result: AdapterResult<SubstrateStatus>;
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
          {result.data.actors.map((a) => (
            <tr key={a.id}>
              <td>{a.id}</td>
              <td>{a.state}</td>
              <td>{a.pool}</td>
              <td>{a.lastActivity}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
