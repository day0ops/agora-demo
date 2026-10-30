import { notConnectedBanner } from "./notConnectedBanner";
import { escapeHtml } from "../html";
import type { AdapterResult } from "../adapters/types";
import type { SubstrateStatus } from "../adapters/substrateAdapter";

const COLUMNS = ["Actor", "State", "Pool", "Last activity"] as const;

// Real column shape always rendered; today getSubstrateStatus always
// resolves "empty" (no live Agent Substrate cluster is wired to this
// console), so the body is the shared not-connected banner spanning every
// column rather than a fabricated row. Worker-pool density is the Pool
// column grouped client-side once real rows exist - no separate adapter
// call or data shape is needed for that.
export function substrateTopology(
  result: AdapterResult<SubstrateStatus>,
): string {
  const header = `<tr>${COLUMNS.map((c) => `<th>${c}</th>`).join("")}</tr>`;
  if (result.status === "empty") {
    return `
      <div class="inspector-table-wrap">
        <table class="inspector-table">
          <thead>${header}</thead>
          <tbody><tr><td colspan="${COLUMNS.length}">${notConnectedBanner(result.reason)}</td></tr></tbody>
        </table>
      </div>
    `;
  }
  const rows = result.data.actors
    .map(
      (a) =>
        `<tr><td>${escapeHtml(a.id)}</td><td>${escapeHtml(a.state)}</td><td>${escapeHtml(a.pool)}</td><td>${escapeHtml(a.lastActivity)}</td></tr>`,
    )
    .join("");
  return `<div class="inspector-table-wrap"><table class="inspector-table"><thead>${header}</thead><tbody>${rows}</tbody></table></div>`;
}
