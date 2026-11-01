import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import type { SellerTier } from "../types";

export type Role = "buyers" | "sellers" | "operators" | "";

export interface AccountIdentity {
  username: string;
  groups: string[];
  role: Role;
  fixtureId?: string;
  sellerTier?: SellerTier;
}

interface IdentityContextValue {
  identity: AccountIdentity | null;
  loading: boolean;
}

const IdentityContext = createContext<IdentityContextValue>({
  identity: null,
  loading: true,
});

// IdentityProvider fetches GET /account/me exactly once per page load and
// makes the result available to every descendant via useIdentity().
export function IdentityProvider({ children }: { children: ReactNode }) {
  const [identity, setIdentity] = useState<AccountIdentity | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    fetch("/account/me")
      .then((res) => (res.ok ? (res.json() as Promise<AccountIdentity>) : null))
      .catch(() => null)
      .then((resolved) => {
        if (!cancelled) {
          setIdentity(resolved);
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <IdentityContext.Provider value={{ identity, loading }}>
      {children}
    </IdentityContext.Provider>
  );
}

export function useIdentity(): IdentityContextValue {
  return useContext(IdentityContext);
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
