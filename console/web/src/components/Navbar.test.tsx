import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { Navbar } from "./Navbar";
import { IdentityProvider } from "../hooks/useIdentity";

function renderNavbar(identityResponse: unknown) {
  vi.spyOn(global, "fetch").mockResolvedValue({
    ok: identityResponse !== null,
    json: () => Promise.resolve(identityResponse ?? {}),
  } as Response);
  return render(
    <MemoryRouter>
      <IdentityProvider>
        <Navbar active="shop" />
      </IdentityProvider>
    </MemoryRouter>,
  );
}

describe("Navbar", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("shows a Login link and hides Sell/Operator when logged out", async () => {
    renderNavbar(null);
    expect(await screen.findByText("Login")).toBeInTheDocument();
    expect(screen.getByText("Sell")).not.toBeVisible();
    expect(screen.getByText("Operator")).not.toBeVisible();
    expect(screen.getByText("Shop")).toBeVisible();
  });

  it("shows the account menu and reveals Sell for a seller, but not Operator", async () => {
    renderNavbar({
      username: "demo-seller",
      groups: ["sellers"],
      role: "sellers",
      fixtureId: "seller-1",
    });
    expect(await screen.findByText("demo-seller")).toBeInTheDocument();
    expect(screen.getByText("Sell")).toBeVisible();
    expect(screen.getByText("Operator")).not.toBeVisible();
  });

  it("opens the account menu on click and closes it on an outside click", async () => {
    const user = userEvent.setup();
    renderNavbar({
      username: "demo-buyer",
      groups: ["buyers"],
      role: "buyers",
      fixtureId: "buyer-1",
    });
    await screen.findByText("demo-buyer");

    expect(screen.queryByText("Log out")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /demo-buyer/i }));
    expect(screen.getByText("Log out")).toBeInTheDocument();

    fireEvent.click(document.body);
    expect(screen.queryByText("Log out")).not.toBeInTheDocument();
  });
});
