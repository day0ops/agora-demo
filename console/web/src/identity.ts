import type { SellerTier } from "./types";

export type Role = "buyers" | "sellers" | "operators" | "";

export interface AccountIdentity {
  username: string;
  groups: string[];
  role: Role;
  fixtureId?: string;
  sellerTier?: SellerTier;
}

let identityPromise: Promise<AccountIdentity | null> | null = null;

// currentIdentity() is memoized for the life of the page load: GET
// /account/me is the single source of truth for who's logged in, and every
// view (navbar, shop, sell, ops) needs the same answer without each
// triggering its own redundant fetch.
export function currentIdentity(): Promise<AccountIdentity | null> {
  if (!identityPromise) {
    identityPromise = fetch("/account/me")
      .then((res) => (res.ok ? (res.json() as Promise<AccountIdentity>) : null))
      .catch(() => null);
  }
  return identityPromise;
}

// Operators are a pure oversight role -- they have no seller fixture of
// their own, so they don't get the Sell experience even though they rank
// above sellers in role priority.
export function canSell(identity: AccountIdentity | null): boolean {
  return identity?.role === "sellers";
}

export function canOperate(identity: AccountIdentity | null): boolean {
  return identity?.role === "operators";
}
