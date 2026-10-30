import { escapeHtml } from "../html";

// The shared "honest empty" visual language for every substrate/agent
// surface that has no live data source yet - reuses the existing `.note`
// style already established on Operator Studio's stats page rather than
// inventing a new visual pattern.
export function notConnectedBanner(reason: string): string {
  return `<div class="note not-connected-banner">${escapeHtml(reason)}</div>`;
}
