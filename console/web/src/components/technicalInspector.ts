import { notConnectedBanner } from "./notConnectedBanner";
import { escapeHtml } from "../html";
import type { AdapterResult } from "../adapters/types";
import type { AgentJourney } from "../adapters/executionAdapter";

const COLUMNS = ["Actor", "State", "Pool", "MCP call", "Policy"] as const;

// Real column shape always rendered; today result.status is always "empty"
// (see executionAdapter), so the body is the shared not-connected banner
// spanning every column rather than a fabricated row.
export function technicalInspector(
  result: AdapterResult<AgentJourney>,
): string {
  const header = `<tr>${COLUMNS.map((c) => `<th>${c}</th>`).join("")}</tr>`;
  if (result.status === "empty") {
    return `
      <table class="inspector-table">
        <thead>${header}</thead>
        <tbody><tr><td colspan="${COLUMNS.length}">${notConnectedBanner(result.reason)}</td></tr></tbody>
      </table>
    `;
  }
  const rows = result.data.steps
    .map(
      (s) =>
        `<tr><td>${escapeHtml(s.actor)}</td><td>${escapeHtml(s.state)}</td><td>${escapeHtml(s.pool)}</td><td>${escapeHtml(s.mcpCall)}</td><td>${escapeHtml(s.policy)}</td></tr>`,
    )
    .join("");
  return `<table class="inspector-table"><thead>${header}</thead><tbody>${rows}</tbody></table>`;
}
