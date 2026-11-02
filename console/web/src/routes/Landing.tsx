import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Navbar } from "../components/Navbar";
import { Footer } from "../components/Footer";
import { ListingImage } from "../components/ListingImage";
import { StatusPill } from "../components/StatusPill";
import { EmptyState } from "../components/EmptyState";
import { IconShop } from "../icons";
import { useIdentity } from "../hooks/useIdentity";
import { listListings } from "../api";
import { formatCents } from "../format";
import type { Listing } from "../types";

type ListingsState =
  | { status: "loading" }
  | { status: "unreachable" }
  | { status: "ready"; listings: Listing[] };

function priceFor(l: Listing): number {
  return l.type === "auction"
    ? l.currentBidCents || l.priceCents
    : l.priceCents;
}

function RailCard({ listing }: { listing: Listing }) {
  return (
    <Link className="rail-card" to="/shop">
      <ListingImage listing={listing} />
      <div className="rail-card-title">{listing.title}</div>
      <div className="rail-card-foot">
        <span className="price">{formatCents(priceFor(listing))}</span>
        <StatusPill status={listing.status} />
      </div>
    </Link>
  );
}

function RailSection({
  title,
  state,
  select,
  emptyTitle,
  emptyBody,
}: {
  title: string;
  state: ListingsState;
  select: (listings: Listing[]) => Listing[];
  emptyTitle: string;
  emptyBody: string;
}) {
  const selected = state.status === "ready" ? select(state.listings) : [];
  return (
    <section className="rail">
      <h2>{title}</h2>
      <div className="rail-track">
        {state.status === "loading" && <div className="empty">Loading...</div>}
        {state.status === "unreachable" && (
          <EmptyState
            title="Catalog unavailable"
            body="Could not reach catalog-api. Try refreshing in a moment."
            icon={<IconShop />}
          />
        )}
        {state.status === "ready" &&
          (selected.length ? (
            selected.map((l) => <RailCard key={l.id} listing={l} />)
          ) : (
            <EmptyState
              title={emptyTitle}
              body={emptyBody}
              icon={<IconShop />}
            />
          ))}
      </div>
    </section>
  );
}

export function Landing() {
  const { identity } = useIdentity();
  const [state, setState] = useState<ListingsState>({ status: "loading" });

  useEffect(() => {
    listListings()
      .then((listings) => setState({ status: "ready", listings }))
      .catch(() => setState({ status: "unreachable" }));
  }, []);

  return (
    <>
      <Navbar active="index" />
      <main>
        <div className="hero">
          <h1>A marketplace where agents do the work.</h1>
          <p>
            Discover, negotiate, buy, and sell with intelligent agents working
            together on your behalf.
          </p>
          <div className="hero-ctas">
            <Link className="primary" to="/shop">
              Explore marketplace
            </Link>
            {!identity && (
              <a className="secondary" href="/account/login">
                Login
              </a>
            )}
          </div>
        </div>
        <div className="rails">
          <RailSection
            title="Featured"
            state={state}
            select={(listings) =>
              listings
                .filter(
                  (l) =>
                    l.status === "active-auction" ||
                    l.status === "active-buy-now",
                )
                .sort((a, b) => priceFor(b) - priceFor(a))
                .slice(0, 4)
            }
            emptyTitle="No listings yet"
            emptyBody="Check back soon as sellers list items."
          />
          <RailSection
            title="Closing soon"
            state={state}
            select={(listings) =>
              listings
                .filter((l) => l.status === "active-auction")
                .sort(
                  (a, b) =>
                    new Date(a.auctionEndsAt ?? 0).getTime() -
                    new Date(b.auctionEndsAt ?? 0).getTime(),
                )
                .slice(0, 4)
            }
            emptyTitle="No auctions closing soon"
            emptyBody="Live auctions will appear here as they near their end."
          />
          <RailSection
            title="Just listed"
            state={state}
            select={(listings) =>
              listings
                .filter(
                  (l) =>
                    l.status === "active-auction" ||
                    l.status === "active-buy-now",
                )
                .sort(
                  (a, b) =>
                    new Date(b.createdAt).getTime() -
                    new Date(a.createdAt).getTime(),
                )
                .slice(0, 4)
            }
            emptyTitle="No listings yet"
            emptyBody="New listings will appear here as sellers add them."
          />
        </div>
      </main>
      <Footer />
    </>
  );
}
