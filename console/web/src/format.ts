export function formatCents(cents: number): string {
  return (cents / 100).toLocaleString("en-US", {
    style: "currency",
    currency: "USD",
  });
}

export function formatCountdown(
  endsAt: string | undefined,
  now: number,
): string {
  if (!endsAt) return "";
  const ms = new Date(endsAt).getTime() - now;
  if (ms <= 0) return "ending...";
  const totalSeconds = Math.floor(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes >= 60) {
    const hours = Math.floor(minutes / 60);
    return `${hours}h ${minutes % 60}m left`;
  }
  return `${minutes}m ${seconds.toString().padStart(2, "0")}s left`;
}

const STATUS_LABELS: Record<string, string> = {
  draft: "Draft",
  "active-auction": "Live auction",
  "active-buy-now": "Buy now",
  sold: "Sold",
  "unsold-expired": "Unsold",
  withdrawn: "Withdrawn",
};

export function statusLabel(status: string): string {
  return STATUS_LABELS[status] ?? status;
}

const STATUS_PILL_CLASSES: Record<string, string> = {
  draft: "idle",
  "active-auction": "warn",
  "active-buy-now": "info",
  sold: "ok",
  "unsold-expired": "idle",
  withdrawn: "bad",
};

export function statusPillClass(status: string): string {
  return STATUS_PILL_CLASSES[status] ?? "idle";
}

// lotNumber renders a listing's ID as an auction-catalog lot number (fixture
// IDs are "listing-N"; real-world auction lots are always numbered, so this
// is the one place numbering carries genuine information rather than
// decorating the card).
export function lotNumber(id: string): string {
  const match = /(\d+)$/.exec(id);
  const n = match ? Number(match[1]) : 0;
  return `LOT ${String(n).padStart(3, "0")}`;
}
