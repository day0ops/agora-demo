import "./style.css";
import { mountTopbar, mountFooter } from "./layout";
import { mountChat } from "./chat";
import { connectEvents } from "./events";
import {
  listListings,
  getListing,
  bidHistory,
  placeBid,
  setAutoBidMax,
  withdrawBid,
  checkout,
  listOrders,
  ApiError,
} from "./api";
import { BUYERS } from "./identities";
import { escapeHtml } from "./html";
import { getStored, setStored } from "./storage";
import {
  formatCents,
  formatCountdown,
  lotNumber,
  statusLabel,
  statusPillClass,
} from "./format";
import type { Listing, Bid, Order, MarketplaceEvent } from "./types";

const URGENT_THRESHOLD_MS = 60_000;

const app = document.querySelector<HTMLDivElement>("#app")!;
app.innerHTML = `
  <div id="topbar"></div>
  <div class="page">
    <div class="identity-picker">
      Shopping as
      <select id="buyer-select"></select>
    </div>
    <div class="layout">
      <div class="panel">
        <h2>Live auctions &amp; buy-now listings</h2>
        <div class="grid" id="listing-grid"><div class="empty">Loading listings...</div></div>
      </div>
      <div class="panel-stack">
        <div class="panel detail" id="detail-panel">
          <h2>Listing</h2>
          <div class="empty">Select a listing to view details and place a bid.</div>
        </div>
        <div class="panel chat-panel">
          <h2>Shopping Assistant</h2>
          <div id="chat-mount" class="chat-mount"></div>
        </div>
      </div>
    </div>
    <div class="panel" id="orders-panel">
      <h2>Your Orders</h2>
      <ul class="orders-ledger" id="orders-ledger"><li class="empty">No orders yet.</li></ul>
    </div>
  </div>
  <div id="footer"></div>
`;

mountTopbar(document.querySelector("#topbar")!, "shop");
mountFooter(document.querySelector("#footer")!);
mountChat(
  document.querySelector("#chat-mount")!,
  "shopping-assistant",
  "Shopping Assistant",
);

const buyerSelect = document.querySelector<HTMLSelectElement>("#buyer-select")!;
buyerSelect.innerHTML = BUYERS.map(
  (b) => `<option value="${b.id}">${escapeHtml(b.name)}</option>`,
).join("");
buyerSelect.value = getStored("mktBuyerId") || BUYERS[0].id;
buyerSelect.addEventListener("change", () => {
  setStored("mktBuyerId", buyerSelect.value);
  void loadOrders();
});
function currentBuyerId(): string {
  return buyerSelect.value;
}

const listings = new Map<string, Listing>();
let selectedId: string | null = null;
const expiryRefreshRequested = new Set<string>();

const grid = document.querySelector<HTMLDivElement>("#listing-grid")!;
const detailPanel = document.querySelector<HTMLDivElement>("#detail-panel")!;
const ordersLedger =
  document.querySelector<HTMLUListElement>("#orders-ledger")!;

function priceFor(l: Listing): number {
  return l.type === "auction"
    ? l.currentBidCents || l.priceCents
    : l.priceCents;
}

function renderCard(l: Listing): string {
  const remaining = l.auctionEndsAt
    ? new Date(l.auctionEndsAt).getTime() - Date.now()
    : Infinity;
  const countdown =
    l.type === "auction"
      ? `<span class="countdown ${remaining < URGENT_THRESHOLD_MS ? "urgent" : ""}" id="cd-${l.id}">${formatCountdown(l.auctionEndsAt, Date.now())}</span>`
      : "";
  return `
    <div class="card ${l.id === selectedId ? "selected" : ""}" data-id="${l.id}">
      <span class="lot-plate">${lotNumber(l.id)}</span>
      <div class="title">${escapeHtml(l.title)}</div>
      <div class="meta">${escapeHtml(l.category)}</div>
      <div class="price">${formatCents(priceFor(l))}</div>
      <div class="foot">
        <span class="pill ${statusPillClass(l.status)}">${statusLabel(l.status)}</span>
        ${countdown}
      </div>
    </div>
  `;
}

function renderGrid(): void {
  const visible = [...listings.values()]
    .filter(
      (l) => l.status === "active-auction" || l.status === "active-buy-now",
    )
    .sort((a, b) => (a.createdAt < b.createdAt ? 1 : -1));
  grid.innerHTML = visible.length
    ? visible.map(renderCard).join("")
    : '<div class="empty">No active listings right now.</div>';
}

grid.addEventListener("click", (e) => {
  const card = (e.target as HTMLElement).closest<HTMLElement>("[data-id]");
  if (card) selectListing(card.dataset.id!);
});

function selectListing(id: string): void {
  selectedId = id;
  renderGrid();
  void renderDetail();
}

async function renderDetail(): Promise<void> {
  const listing = selectedId ? listings.get(selectedId) : undefined;
  if (!listing) return;

  detailPanel.innerHTML = '<h2>Listing</h2><div class="empty">Loading...</div>';
  let bids: Bid[] = [];
  if (listing.type === "auction") {
    try {
      bids = await bidHistory(listing.id);
    } catch {
      // bid history is best-effort; the detail panel still renders without it
    }
  }
  if (selectedId !== listing.id) return; // selection changed while awaiting

  const price = priceFor(listing);
  const isAuction = listing.type === "auction";
  const isOpen =
    listing.status === "active-auction" || listing.status === "active-buy-now";
  const isSold = listing.status === "sold";
  const isWon = isAuction && isSold;
  const isPassed =
    listing.status === "unsold-expired" || listing.status === "withdrawn";

  detailPanel.innerHTML = `
    <h2>Listing</h2>
    <div class="detail">
      ${isSold ? '<span class="sold-stamp">Sold</span>' : ""}
      <span class="lot-plate">${lotNumber(listing.id)}</span>
      <h3>${escapeHtml(listing.title)}</h3>
      <p class="desc">${escapeHtml(listing.description)}</p>
      <div class="price-row">
        <span class="price">${formatCents(price)}</span>
        <span class="pill ${statusPillClass(listing.status)}">${statusLabel(listing.status)}</span>
      </div>
      ${
        isAuction
          ? `<ul class="bid-history">${
              bids.length
                ? bids
                    .slice()
                    .reverse()
                    .map(
                      (b) => `
                      <li data-bid-id="${b.id}">
                        <span class="who">${escapeHtml(b.buyerId)}${b.isAutoBid ? " (auto)" : ""}</span>
                        <span class="amt">${formatCents(b.amountCents)}</span>
                        ${b.buyerId === currentBuyerId() && isOpen ? `<button type="button" class="link withdraw-bid-btn" data-bid-id="${b.id}">Withdraw</button>` : ""}
                      </li>`,
                    )
                    .join("")
                : '<li><span class="empty">No bids yet</span></li>'
            }</ul>`
          : ""
      }
      ${
        isWon
          ? `<button id="claim-win-btn" class="buy-now-btn">Claim Your Win - ${formatCents(price)}</button>`
          : isPassed
            ? '<div class="note">This lot passed with no winning bid.</div>'
            : isSold
              ? '<div class="note">This listing has sold.</div>'
              : isAuction
                ? `
        <form class="inline" id="bid-form">
          <input type="number" id="bid-amount" min="0.01" step="0.01" placeholder="Bid amount ($)" required />
          <button type="submit">Place Bid</button>
        </form>
        <form class="inline" id="autobid-form">
          <input type="number" id="autobid-amount" min="0.01" step="0.01" placeholder="Auto-bid max ($)" required />
          <button type="submit" class="secondary">Set Auto-Bid</button>
        </form>
      `
                : `<button id="buy-now-btn" class="buy-now-btn">Buy Now - ${formatCents(price)}</button>`
      }
      <div class="note" id="detail-error"></div>
    </div>
  `;

  const errorBox = detailPanel.querySelector<HTMLDivElement>("#detail-error")!;
  errorBox.style.display = "none";
  function showError(err: unknown): void {
    errorBox.textContent =
      err instanceof ApiError
        ? err.message
        : "Something went wrong. Please try again.";
    errorBox.classList.add("error");
    errorBox.style.display = "block";
  }

  detailPanel
    .querySelector<HTMLFormElement>("#bid-form")
    ?.addEventListener("submit", async (e) => {
      e.preventDefault();
      const dollars = Number(
        detailPanel.querySelector<HTMLInputElement>("#bid-amount")!.value,
      );
      errorBox.style.display = "none";
      try {
        await placeBid(listing.id, currentBuyerId(), Math.round(dollars * 100));
      } catch (err) {
        showError(err);
      }
    });

  detailPanel
    .querySelector<HTMLFormElement>("#autobid-form")
    ?.addEventListener("submit", async (e) => {
      e.preventDefault();
      const dollars = Number(
        detailPanel.querySelector<HTMLInputElement>("#autobid-amount")!.value,
      );
      errorBox.style.display = "none";
      try {
        await setAutoBidMax(
          listing.id,
          currentBuyerId(),
          Math.round(dollars * 100),
        );
      } catch (err) {
        showError(err);
      }
    });

  detailPanel
    .querySelector<HTMLButtonElement>("#buy-now-btn")
    ?.addEventListener("click", async () => {
      errorBox.style.display = "none";
      try {
        await checkout(listing.id, currentBuyerId());
        void loadOrders();
      } catch (err) {
        showError(err);
      }
    });

  detailPanel
    .querySelector<HTMLButtonElement>("#claim-win-btn")
    ?.addEventListener("click", async () => {
      errorBox.style.display = "none";
      try {
        await checkout(listing.id, currentBuyerId());
        void loadOrders();
      } catch (err) {
        showError(err);
      }
    });

  detailPanel
    .querySelectorAll<HTMLButtonElement>(".withdraw-bid-btn")
    .forEach((btn) => {
      btn.addEventListener("click", async () => {
        errorBox.style.display = "none";
        try {
          await withdrawBid(listing.id, btn.dataset.bidId!);
        } catch (err) {
          showError(err);
        }
      });
    });
}

function renderOrders(orders: Order[]): void {
  ordersLedger.innerHTML = orders.length
    ? orders
        .slice()
        .reverse()
        .map((o) => {
          const l = listings.get(o.listingId);
          return `<li><span class="item">${escapeHtml(l?.title ?? o.listingId)}</span><span class="amt">${formatCents(o.finalPriceCents)}</span></li>`;
        })
        .join("")
    : '<li class="empty">No orders yet.</li>';
}

async function loadOrders(): Promise<void> {
  try {
    const orders = await listOrders({ buyerId: currentBuyerId() });
    renderOrders(orders);
  } catch {
    // best-effort; the ledger just keeps its last known state
  }
}

async function loadListings(): Promise<void> {
  try {
    const [auctions, buyNow] = await Promise.all([
      listListings({ status: "active-auction" }),
      listListings({ status: "active-buy-now" }),
    ]);
    listings.clear();
    [...auctions, ...buyNow].forEach((l) => listings.set(l.id, l));
    renderGrid();
  } catch {
    grid.innerHTML = '<div class="empty">Could not reach catalog-api.</div>';
  }
}

function refreshListing(listingId: string): void {
  getListing(listingId)
    .then((l) => {
      listings.set(l.id, l);
      if (l.status === "active-auction") expiryRefreshRequested.delete(l.id);
      renderGrid();
      if (selectedId === l.id) void renderDetail();
    })
    .catch(() => {
      // best-effort refresh; the next SSE event or poll will retry
    });
}

connectEvents((evt: MarketplaceEvent) => {
  switch (evt.type) {
    case "listing.created":
      listings.set(evt.data.id, evt.data);
      renderGrid();
      break;
    case "bid.placed":
    case "bid.withdrawn":
      refreshListing(evt.data.listingId);
      break;
    case "autobid.set":
      listings.set(evt.data.listing.id, evt.data.listing);
      renderGrid();
      if (selectedId === evt.data.listing.id) void renderDetail();
      break;
    case "order.created":
      refreshListing(evt.data.listingId);
      if (evt.data.buyerId === currentBuyerId()) void loadOrders();
      break;
  }
});

setInterval(() => {
  const now = Date.now();
  for (const l of listings.values()) {
    if (l.type !== "auction" || l.status !== "active-auction") continue;
    const remaining = l.auctionEndsAt
      ? new Date(l.auctionEndsAt).getTime() - now
      : Infinity;
    const el = document.getElementById(`cd-${l.id}`);
    if (el) {
      el.textContent = formatCountdown(l.auctionEndsAt, now);
      el.classList.toggle("urgent", remaining < URGENT_THRESHOLD_MS);
    }
    // The server is the source of truth for when an auction actually closes;
    // this just prompts one refetch shortly after the clock hits zero so a
    // closed auction doesn't sit showing "ending..." until some other event
    // happens to touch it.
    if (remaining <= 0 && !expiryRefreshRequested.has(l.id)) {
      expiryRefreshRequested.add(l.id);
      refreshListing(l.id);
    }
  }
}, 1000);

void loadListings();
void loadOrders();
