import { statusLabel, statusPillClass } from "../format";

export function statusPill(status: string): string {
  return `<span class="pill ${statusPillClass(status)}">${statusLabel(status)}</span>`;
}
