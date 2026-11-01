import { useCallback, useEffect, useRef, useState } from "react";
import { Navbar } from "../components/Navbar";
import { Footer } from "../components/Footer";
import { EmptyState } from "../components/EmptyState";
import { SubstrateTopology } from "../components/SubstrateTopology";
import { useExecutionDrawer } from "../components/ExecutionDrawer";
import { IconScan } from "../icons";
import { useIdentity, canOperate } from "../hooks/useIdentity";
import { useEvents } from "../hooks/useEvents";
import { getStats, listListings, type Stats } from "../api";
import { getSubstrateStatus } from "../adapters/substrateAdapter";
import { formatCents } from "../format";
import type { AdapterResult } from "../adapters/types";
import type { SubstrateStatus } from "../adapters/substrateAdapter";
import type { MarketplaceEvent } from "../types";

const MAX_LOG_ENTRIES = 50;

interface LogEntry {
  id: number;
  ts: string;
  kind: string;
  text: string;
  trigger?: () => void;
}

function describe(
  evt: MarketplaceEvent,
  titleFor: (id: string) => string,
): string {
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

function OperatorContent() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [substrate, setSubstrate] =
    useState<AdapterResult<SubstrateStatus> | null>(null);
  const listingTitlesRef = useRef<Map<string, string>>(new Map());
  const nextIdRef = useRef(0);
  const { open: openExecutionDrawer, drawer } = useExecutionDrawer();

  const refreshStats = useCallback(async () => {
    try {
      setStats(await getStats());
    } catch {
      // best-effort; the next poll or event-triggered refresh will retry
    }
  }, []);

  useEffect(() => {
    listListings({})
      .then((all) => {
        all.forEach((l) => listingTitlesRef.current.set(l.id, l.title));
      })
      .catch(() => {
        // titles are a display nicety; log lines fall back to the raw listing ID
      });
    void refreshStats();
    void getSubstrateStatus().then(setSubstrate);
    const id = setInterval(() => void refreshStats(), 10000);
    return () => clearInterval(id);
  }, [refreshStats]);

  function titleFor(listingId: string): string {
    return listingTitlesRef.current.get(listingId) ?? listingId;
  }

  function triggerFor(evt: MarketplaceEvent): (() => void) | undefined {
    if (evt.type === "bid.placed") {
      const b = evt.data.bid;
      return () =>
        openExecutionDrawer({
          summary: {
            title: `${b.buyerId} bid ${formatCents(b.amountCents)} on "${titleFor(evt.data.listingId)}"`,
            body: b.isAutoBid
              ? "Placed automatically by the proxy-bidding engine."
              : "Placed manually.",
            trustCheck: b.trustCheck
              ? { score: b.trustCheck.score, reasons: b.trustCheck.reasons }
              : "unavailable",
          },
          correlationId: b.correlationId,
        });
    }
    if (evt.type === "order.created") {
      return () =>
        openExecutionDrawer({
          summary: {
            title: `${evt.data.buyerId} purchased "${titleFor(evt.data.listingId)}"`,
            body: `Final price ${formatCents(evt.data.finalPriceCents)}.`,
            trustCheck: "unavailable",
          },
          correlationId: evt.data.correlationId,
        });
    }
    return undefined;
  }

  useEvents((evt) => {
    if (evt.type === "listing.created") {
      listingTitlesRef.current.set(evt.data.id, evt.data.title);
    }
    const entry: LogEntry = {
      id: nextIdRef.current++,
      ts: new Date().toLocaleTimeString(),
      kind: evt.type,
      text: describe(evt, titleFor),
      trigger: triggerFor(evt),
    };
    setEntries((prev) => [...prev, entry].slice(-MAX_LOG_ENTRIES));
    void refreshStats();
  });

  return (
    <>
      <div className="stats">
        <div className="stat">
          <b>{stats ? stats.activeListings : "-"}</b>
          <span>Active listings</span>
        </div>
        <div className="stat">
          <b>{stats ? stats.totalOrders : "-"}</b>
          <span>Total orders</span>
        </div>
        <div className="stat">
          <b>{stats ? stats.connectedClients : "-"}</b>
          <span>Connected clients</span>
        </div>
      </div>
      <div className="note">
        Budget enforcement, rate limiting, and PII guardrail telemetry come from
        agentgateway Enterprise policies running against the live kagent Agent
        Substrate cluster backing this console. See the Agent Substrate panel
        below for live worker-pool topology.
      </div>
      <div className="panel activity-panel">
        <h2>Live activity</h2>
        <div className="event-log">
          {entries.map((e) => (
            <div key={e.id} className="entry">
              <span className="ts">{e.ts}</span>
              <span className="kind">{e.kind}</span>
              <span>{e.text}</span>
              {e.trigger && (
                <button
                  type="button"
                  className="link behind-the-scenes-btn"
                  aria-label="Behind the scenes"
                  onClick={e.trigger}
                >
                  <IconScan />
                </button>
              )}
            </div>
          ))}
        </div>
      </div>
      <div className="panel">
        <h2>Agent Substrate</h2>
        {substrate ? (
          <SubstrateTopology result={substrate} />
        ) : (
          <div className="empty">Loading...</div>
        )}
      </div>
      {drawer}
    </>
  );
}

export function Ops() {
  const { identity, loading } = useIdentity();

  return (
    <>
      <Navbar active="ops" />
      <main>
        {loading ? (
          <div className="empty">Loading...</div>
        ) : identity && canOperate(identity) ? (
          <OperatorContent />
        ) : (
          <EmptyState
            title="Operator access required"
            body="Log in with an operator account to see marketplace oversight."
            icon={<IconScan />}
          />
        )}
      </main>
      <Footer />
    </>
  );
}
