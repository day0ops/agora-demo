import type { Bid, Listing, ListingType, Order } from "./types";

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const res = await fetch(path, {
    method,
    headers:
      body !== undefined ? { "content-type": "application/json" } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  if (res.status === 204) return undefined as T;

  const payload: unknown = await res.json().catch(() => undefined);
  if (!res.ok) {
    const message =
      payload &&
      typeof payload === "object" &&
      typeof (payload as { error?: unknown }).error === "string"
        ? (payload as { error: string }).error
        : `request failed with status ${res.status}`;
    throw new ApiError(res.status, message);
  }
  return payload as T;
}

function query(params: Record<string, string | undefined>): string {
  const q = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value) q.set(key, value);
  }
  const qs = q.toString();
  return qs ? `?${qs}` : "";
}

export function listListings(
  params: { status?: string; sellerId?: string } = {},
): Promise<Listing[]> {
  return request("GET", `/api/listings${query(params)}`);
}

export function getListing(id: string): Promise<Listing> {
  return request("GET", `/api/listings/${encodeURIComponent(id)}`);
}

export interface CreateListingInput {
  title: string;
  description: string;
  category: string;
  type: ListingType;
  priceCents: number;
  auctionEndsAt?: string;
}

export function createListing(input: CreateListingInput): Promise<Listing> {
  return request("POST", "/account/listings", input);
}

export function bidHistory(listingId: string): Promise<Bid[]> {
  return request("GET", `/api/listings/${encodeURIComponent(listingId)}/bids`);
}

export function placeBid(listingId: string, amountCents: number): Promise<Bid> {
  return request(
    "POST",
    `/account/listings/${encodeURIComponent(listingId)}/bids`,
    { amountCents },
  );
}

export function setAutoBidMax(
  listingId: string,
  maxCents: number,
): Promise<Listing> {
  return request(
    "POST",
    `/account/listings/${encodeURIComponent(listingId)}/autobid`,
    { maxCents },
  );
}

export function withdrawBid(listingId: string, bidId: string): Promise<void> {
  return request(
    "DELETE",
    `/account/listings/${encodeURIComponent(listingId)}/bids/${encodeURIComponent(bidId)}`,
  );
}

export function checkout(listingId: string): Promise<Order> {
  return request(
    "POST",
    `/account/listings/${encodeURIComponent(listingId)}/checkout`,
  );
}

export function listOrders(): Promise<Order[]> {
  return request("GET", "/account/orders");
}

export interface ChatResponse {
  reply?: string;
  error?: string;
}

export function sendChat(
  agent: string,
  sessionId: string,
  message: string,
): Promise<ChatResponse> {
  return request("POST", "/account/chat", { agent, sessionId, message });
}

export interface Stats {
  activeListings: number;
  totalOrders: number;
  connectedClients: number;
}

export function getStats(): Promise<Stats> {
  return request("GET", "/account/stats");
}

export interface VersionInfo {
  version: string;
}

export function getVersion(): Promise<VersionInfo> {
  return request("GET", "/api/version");
}
