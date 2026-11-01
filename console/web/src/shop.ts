import "./style.css";
import {
  mountNavbar,
  mountFooter,
  renderAccountArea,
} from "./components/navbar";
import { mountChatPanel } from "./components/chatPanel";
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
import { currentIdentity, type AccountIdentity } from "./identity";
import { escapeHtml } from "./html";
import { formatCents, lotNumber } from "./format";
import { statusPill } from "./components/statusPill";
import { lotCard } from "./components/lotCard";
import { refreshCountdowns } from "./components/countdown";
import { listingImageEl } from "./components/listingImage";
import { emptyState } from "./components/emptyState";
import { icons } from "./icons";
import { openExecutionDrawer } from "./components/executionDrawer";
import type { Listing, Bid, Order, MarketplaceEvent } from "./types";

const app = document.querySelector<HTMLDivElement>("#app")!;
app.innerHTML = `
  <header id="topbar"></header>
  <main class="page">
    <div class="layout">
      <div class="panel">
        <h2>Live auctions &amp; buy-now listings</h2>
        <div class="shop-filters">
          <label class="sr-only" for="search-input">Search listings</label>
          <input type="search" id="search-input" placeholder="Search by title or category..." />
          <label class="sr-only" for="category-select">Filter by category</label>
          <select id="category-select">
            <option value="">All categories</option>
          </select>
          <label class="sr-only" for="status-filter">Filter by status</label>
          <select id="status-filter">
            <option value="active">Active listings</option>
            <option value="sold">Sold</option>
          </select>
        </div>
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
  </main>
  <footer id="footer"></footer>
`;

mountNavbar(document.querySelector("#topbar")!, "shop");
mountFooter(document.querySelector("#footer")!);
mountChatPanel(document.querySelector<HTMLDivElement>("#chat-mount")!, {
  agent: "shopping-assistant",
  label: "Shopping Assistant",
});

let identity: AccountIdentity | null = null;
function currentBuyerId(): string | null {
  return identity?.fixtureId ?? null;
}
function requireLogin(): boolean {
  if (currentBuyerId()) return true;
  window.location.href = "/account/login";
  return false;
}

const listings = new Map<string, Listing>();
let selectedId: string | null = null;
let searchQuery = "";
let categoryFilter = "";
let statusFilter: "active" | "sold" = "active";
const expiryRefreshRequested = new Set<string>();

const grid = document.querySelector<HTMLDivElement>("#listing-grid")!;
const detailPanel = document.querySelector<HTMLDivElement>("#detail-panel")!;
const ordersLedger =
  document.querySelector<HTMLUListElement>("#orders-ledger")!;
const searchInput = document.querySelector<HTMLInputElement>("#search-input")!;
const categorySelect =
  document.querySelector<HTMLSelectElement>("#category-select")!;
const statusSelect =
  document.querySelector<HTMLSelectElement>("#status-filter")!;

function highestBid(bids: Bid[]): Bid | undefined {
  return bids.reduce<Bid | undefined>(
    (max, b) => (!max || b.amountCents > max.amountCents ? b : max),
    undefined,
  );
}

function priceFor(l: Listing): number {
  return l.type === "auction"
    ? l.currentBidCents || l.priceCents
    : l.priceCents;
}

function matchesFilters(l: Listing): boolean {
  const q = searchQuery.trim().toLowerCase();
  const matchesSearch =
    !q ||
    l.title.toLowerCase().includes(q) ||
    l.category.toLowerCase().includes(q);
  const matchesCategory = !categoryFilter || l.category === categoryFilter;
  return matchesSearch && matchesCategory;
}

// renderCategoryOptions rebuilds the filter dropdown from the listings
// actually visible in the grid, so it never offers a category with zero
// matches; if the previously-selected category drops out it falls back to
// "All categories" rather than silently filtering on a stale value.
function renderCategoryOptions(active: Listing[]): void {
  const categories = [...new Set(active.map((l) => l.category))].sort();
  if (categoryFilter && !categories.includes(categoryFilter)) {
    categoryFilter = "";
  }
  categorySelect.innerHTML =
    '<option value="">All categories</option>' +
    categories
      .map((c) => `<option value="${escapeHtml(c)}">${escapeHtml(c)}</option>`)
      .join("");
  categorySelect.value = categoryFilter;
}

function renderGrid(): void {
  const matchingStatus = [...listings.values()]
    .filter((l) =>
      statusFilter === "active"
        ? l.status === "active-auction" || l.status === "active-buy-now"
        : l.status === "sold",
    )
    .sort((a, b) => (a.createdAt < b.createdAt ? 1 : -1));
  renderCategoryOptions(matchingStatus);

  const visible = matchingStatus.filter(matchesFilters);
  if (visible.length) {
    grid.innerHTML = visible
      .map((l) =>
        lotCard(l, { selected: l.id === selectedId, interactive: true }),
      )
      .join("");
    return;
  }
  grid.innerHTML = matchingStatus.length
    ? emptyState({
        title: "No matching listings",
        body: "Try a different search or category.",
        icon: icons.shop,
      })
    : emptyState(
        statusFilter === "active"
          ? {
              title: "No active listings",
              body: "Check back soon as sellers list items.",
              icon: icons.shop,
            }
          : {
              title: "No sold listings yet",
              body: "Sold listings will appear here once something sells.",
              icon: icons.shop,
            },
      );
}

searchInput.addEventListener("input", () => {
  searchQuery = searchInput.value;
  renderGrid();
});
categorySelect.addEventListener("change", () => {
  categoryFilter = categorySelect.value;
  renderGrid();
});
statusSelect.addEventListener("change", () => {
  statusFilter = statusSelect.value as "active" | "sold";
  selectedId = null;
  detailPanel.innerHTML =
    '<h2>Listing</h2><div class="empty">Select a listing to view details and place a bid.</div>';
  void loadListings();
});

grid.addEventListener("click", (e) => {
  const card = (e.target as HTMLElement).closest<HTMLElement>("[data-id]");
  if (card) selectListing(card.dataset.id!);
});
grid.addEventListener("keydown", (e) => {
  if (e.key !== "Enter" && e.key !== " ") return;
  const card = (e.target as HTMLElement).closest<HTMLElement>("[data-id]");
  if (!card) return;
  e.preventDefault();
  selectListing(card.dataset.id!);
});

function selectListing(id: string): void {
  // renderGrid() replaces the grid's innerHTML, which would otherwise drop
  // keyboard focus back to <body>; re-focus the same card by id afterward.
  const hadGridFocus = grid.contains(document.activeElement);
  selectedId = id;
  renderGrid();
  if (hadGridFocus) {
    grid.querySelector<HTMLElement>(`[data-id="${id}"]`)?.focus();
  }
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
  const winningBid = isSold ? highestBid(bids) : undefined;
  const isPassed =
    listing.status === "unsold-expired" || listing.status === "withdrawn";

  detailPanel.innerHTML = `
    <h2>Listing</h2>
    <div class="detail">
      ${listingImageEl(listing, { eager: true })}
      ${isSold ? '<span class="sold-stamp">Sold</span>' : ""}
      ${isSold ? `<button type="button" class="link behind-the-scenes-btn" id="detail-behind-the-scenes-btn">${icons.scan} Behind the scenes</button>` : ""}
      <span class="lot-plate">${lotNumber(listing.id)}</span>
      <h3>${escapeHtml(listing.title)}</h3>
      <p class="desc">${escapeHtml(listing.description)}</p>
      <div class="price-row">
        <span class="price">${formatCents(price)}</span>
        ${statusPill(listing.status)}
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
          <label class="sr-only" for="bid-amount">Bid amount ($)</label>
          <input type="number" id="bid-amount" min="0.01" step="0.01" placeholder="Bid amount ($)" required />
          <button type="submit">Place Bid</button>
        </form>
        <form class="inline" id="autobid-form">
          <label class="sr-only" for="autobid-amount">Auto-bid max ($)</label>
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
      if (!requireLogin()) return;
      const dollars = Number(
        detailPanel.querySelector<HTMLInputElement>("#bid-amount")!.value,
      );
      errorBox.style.display = "none";
      try {
        await placeBid(listing.id, Math.round(dollars * 100));
      } catch (err) {
        showError(err);
      }
    });

  detailPanel
    .querySelector<HTMLFormElement>("#autobid-form")
    ?.addEventListener("submit", async (e) => {
      e.preventDefault();
      if (!requireLogin()) return;
      const dollars = Number(
        detailPanel.querySelector<HTMLInputElement>("#autobid-amount")!.value,
      );
      errorBox.style.display = "none";
      try {
        await setAutoBidMax(listing.id, Math.round(dollars * 100));
      } catch (err) {
        showError(err);
      }
    });

  detailPanel
    .querySelector<HTMLButtonElement>("#buy-now-btn")
    ?.addEventListener("click", async () => {
      if (!requireLogin()) return;
      errorBox.style.display = "none";
      try {
        await checkout(listing.id);
        void loadOrders();
      } catch (err) {
        showError(err);
      }
    });

  detailPanel
    .querySelector<HTMLButtonElement>("#claim-win-btn")
    ?.addEventListener("click", async () => {
      if (!requireLogin()) return;
      errorBox.style.display = "none";
      try {
        await checkout(listing.id);
        void loadOrders();
      } catch (err) {
        showError(err);
      }
    });

  detailPanel
    .querySelectorAll<HTMLButtonElement>(".withdraw-bid-btn")
    .forEach((btn) => {
      btn.addEventListener("click", async () => {
        if (!requireLogin()) return;
        errorBox.style.display = "none";
        try {
          await withdrawBid(listing.id, btn.dataset.bidId!);
        } catch (err) {
          showError(err);
        }
      });
    });

  detailPanel
    .querySelector<HTMLButtonElement>("#detail-behind-the-scenes-btn")
    ?.addEventListener("click", () => {
      openExecutionDrawer({
        summary: {
          title: winningBid
            ? `${winningBid.buyerId} won "${listing.title}"`
            : `"${listing.title}" sold`,
          body: `Final price ${formatCents(price)}.`,
          trustCheck: winningBid?.trustCheck
            ? {
                score: winningBid.trustCheck.score,
                reasons: winningBid.trustCheck.reasons,
              }
            : "unavailable",
        },
        correlationId: winningBid?.correlationId,
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
          const title = l?.title ?? o.listingId;
          return `
            <li>
              <span class="item">${escapeHtml(title)}</span>
              <span class="amt">${formatCents(o.finalPriceCents)}</span>
              <button type="button" class="link behind-the-scenes-btn" data-listing-id="${escapeHtml(o.listingId)}" data-buyer-id="${escapeHtml(o.buyerId)}" data-title="${escapeHtml(title)}" data-price-cents="${escapeHtml(String(o.finalPriceCents))}" data-correlation-id="${escapeHtml(o.correlationId ?? "")}">${icons.scan} Behind the scenes</button>
            </li>`;
        })
        .join("")
    : '<li class="empty">No orders yet.</li>';
}

ordersLedger.addEventListener("click", async (e) => {
  const btn = (e.target as HTMLElement).closest<HTMLButtonElement>(
    ".behind-the-scenes-btn",
  );
  if (!btn) return;
  const { listingId, buyerId, title, priceCents, correlationId } = btn.dataset;
  let winningBid: Bid | undefined;
  try {
    const history = await bidHistory(listingId!);
    winningBid = highestBid(history);
  } catch {
    // best-effort; the drawer still opens with an order-only summary
  }
  openExecutionDrawer({
    summary: {
      title: winningBid
        ? `${buyerId} won "${title}"`
        : `${buyerId} purchased "${title}"`,
      body: `Final price ${formatCents(Number(priceCents))}.`,
      trustCheck: winningBid?.trustCheck
        ? {
            score: winningBid.trustCheck.score,
            reasons: winningBid.trustCheck.reasons,
          }
        : "unavailable",
    },
    correlationId: winningBid?.correlationId || correlationId || undefined,
  });
});

async function loadOrders(): Promise<void> {
  if (!currentBuyerId()) {
    ordersLedger.innerHTML =
      '<li class="empty"><a href="/account/login">Log in</a> to see your orders.</li>';
    return;
  }
  try {
    const orders = await listOrders();
    renderOrders(orders);
  } catch {
    // best-effort; the ledger just keeps its last known state
  }
}

async function loadListings(): Promise<void> {
  try {
    const fetched =
      statusFilter === "active"
        ? await Promise.all([
            listListings({ status: "active-auction" }),
            listListings({ status: "active-buy-now" }),
          ]).then(([auctions, buyNow]) => [...auctions, ...buyNow])
        : await listListings({ status: "sold" });
    listings.clear();
    fetched.forEach((l) => listings.set(l.id, l));
    renderGrid();
  } catch {
    grid.innerHTML = emptyState({
      title: "Catalog unavailable",
      body: "Could not reach catalog-api. Try refreshing in a moment.",
      icon: icons.shop,
    });
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
  // The server is the source of truth for when an auction actually closes;
  // this just prompts one refetch shortly after the clock hits zero so a
  // closed auction doesn't sit showing "ending..." until some other event
  // happens to touch it.
  for (const id of refreshCountdowns(listings.values(), now)) {
    if (!expiryRefreshRequested.has(id)) {
      expiryRefreshRequested.add(id);
      refreshListing(id);
    }
  }
}, 1000);

void currentIdentity().then((resolved) => {
  identity = resolved;
  renderAccountArea(document.querySelector("#topbar")!, identity);
  void loadOrders();
});
void loadListings();
