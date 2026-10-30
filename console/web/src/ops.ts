import "./style.css";
import { mountNavbar, mountFooter } from "./components/navbar";
import { connectEvents } from "./events";
import { getStats, listListings } from "./api";
import { escapeHtml } from "./html";
import { formatCents } from "./format";
import type { MarketplaceEvent } from "./types";

const app = document.querySelector<HTMLDivElement>("#app")!;
app.innerHTML = `
  <header id="topbar"></header>
  <main class="page">
    <div class="stats" id="stats">
      <div class="stat"><b id="stat-active">-</b><span>Active listings</span></div>
      <div class="stat"><b id="stat-orders">-</b><span>Total orders</span></div>
      <div class="stat"><b id="stat-clients">-</b><span>Connected clients</span></div>
    </div>
    <div class="note">
      Budget enforcement, rate limiting, and PII guardrail telemetry come from agentgateway Enterprise
      policies running against a live kagent Agent Substrate cluster. This console has no cluster behind
      it yet, so that telemetry isn't shown here rather than shown as fabricated numbers.
    </div>
    <div class="panel activity-panel">
      <h2>Live activity</h2>
      <div class="event-log" id="event-log"></div>
    </div>
  </main>
  <footer id="footer"></footer>
`;

mountNavbar(document.querySelector("#topbar")!, "ops");
mountFooter(document.querySelector("#footer")!);

const listingTitles = new Map<string, string>();

const statActive = document.querySelector<HTMLElement>("#stat-active")!;
const statOrders = document.querySelector<HTMLElement>("#stat-orders")!;
const statClients = document.querySelector<HTMLElement>("#stat-clients")!;

async function refreshStats(): Promise<void> {
  try {
    const stats = await getStats();
    statActive.textContent = String(stats.activeListings);
    statOrders.textContent = String(stats.totalOrders);
    statClients.textContent = String(stats.connectedClients);
  } catch {
    // best-effort; the next poll or event-triggered refresh will retry
  }
}

async function loadListingTitles(): Promise<void> {
  try {
    const all = await listListings({});
    all.forEach((l) => listingTitles.set(l.id, l.title));
  } catch {
    // titles are a display nicety; log lines fall back to the raw listing ID
  }
}

const MAX_LOG_ENTRIES = 50;
const eventLog = document.querySelector<HTMLDivElement>("#event-log")!;

function titleFor(listingId: string): string {
  return listingTitles.get(listingId) ?? listingId;
}

function describe(evt: MarketplaceEvent): string {
  switch (evt.type) {
    case "listing.created":
      return `"${evt.data.title}" listed by ${evt.data.sellerId}`;
    case "bid.placed":
      return `${evt.data.bid.buyerId} bid ${formatCents(evt.data.bid.amountCents)} on "${titleFor(evt.data.listingId)}"${evt.data.bid.isAutoBid ? " (auto-bid)" : ""}`;
    case "bid.withdrawn":
      return `A bid was withdrawn from "${titleFor(evt.data.listingId)}"`;
    case "autobid.set":
      return `Auto-bid ceiling updated on "${evt.data.listing.title}"`;
    case "order.created":
      return `${evt.data.buyerId} purchased "${titleFor(evt.data.listingId)}" for ${formatCents(evt.data.finalPriceCents)}`;
  }
}

function logEvent(evt: MarketplaceEvent): void {
  const entry = document.createElement("div");
  entry.className = "entry";
  const ts = new Date().toLocaleTimeString();
  entry.innerHTML = `<span class="ts">${ts}</span><span class="kind">${escapeHtml(evt.type)}</span><span>${escapeHtml(describe(evt))}</span>`;
  eventLog.appendChild(entry);
  while (eventLog.children.length > MAX_LOG_ENTRIES) {
    eventLog.removeChild(eventLog.firstChild!);
  }
}

connectEvents((evt: MarketplaceEvent) => {
  if (evt.type === "listing.created")
    listingTitles.set(evt.data.id, evt.data.title);
  logEvent(evt);
  void refreshStats();
});

void loadListingTitles();
void refreshStats();
setInterval(() => void refreshStats(), 10000);
