import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { Ops } from "./Ops";
import { IdentityProvider } from "../hooks/useIdentity";
import * as api from "../api";
import * as substrateAdapter from "../adapters/substrateAdapter";

function renderOps(identityResponse: unknown) {
  vi.spyOn(global, "fetch").mockResolvedValue({
    ok: identityResponse !== null,
    json: () => Promise.resolve(identityResponse ?? {}),
  } as Response);
  return render(
    <MemoryRouter>
      <IdentityProvider>
        <Ops />
      </IdentityProvider>
    </MemoryRouter>,
  );
}

class MockEventSource {
  close() {}
  constructor(public url: string) {}
}

describe("Ops", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    // @ts-expect-error -- test double
    global.EventSource = MockEventSource;
  });

  it("shows a denial state for a non-operator", async () => {
    renderOps({
      username: "demo-buyer",
      groups: ["buyers"],
      role: "buyers",
      fixtureId: "buyer-1",
    });
    expect(
      await screen.findByText("Operator access required"),
    ).toBeInTheDocument();
  });

  it("shows a denial state when logged out", async () => {
    renderOps(null);
    expect(
      await screen.findByText("Operator access required"),
    ).toBeInTheDocument();
  });

  it("shows live stats and substrate topology for an operator", async () => {
    vi.spyOn(api, "getStats").mockResolvedValue({
      activeListings: 2,
      totalOrders: 1,
      connectedClients: 3,
    });
    vi.spyOn(api, "listListings").mockResolvedValue([]);
    vi.spyOn(substrateAdapter, "getSubstrateStatus").mockResolvedValue({
      status: "empty",
      reason: "Not connected to a live substrate.",
    });

    renderOps({
      username: "demo-operator",
      groups: ["operators"],
      role: "operators",
    });
    expect(await screen.findByText("2")).toBeInTheDocument();
    expect(
      screen.getByText("Not connected to a live substrate."),
    ).toBeInTheDocument();
  });
});
