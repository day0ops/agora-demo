import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  IdentityProvider,
  useIdentity,
  canSell,
  canOperate,
} from "./useIdentity";

beforeEach(() => {
  vi.restoreAllMocks();
});

describe("useIdentity", () => {
  it("starts loading, then resolves the fetched identity", async () => {
    vi.spyOn(global, "fetch").mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          username: "demo-buyer",
          groups: ["buyers"],
          role: "buyers",
          fixtureId: "buyer-1",
        }),
    } as Response);

    const { result } = renderHook(() => useIdentity(), {
      wrapper: IdentityProvider,
    });
    expect(result.current.loading).toBe(true);

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.identity?.username).toBe("demo-buyer");
  });

  it("resolves a null identity on a 401", async () => {
    vi.spyOn(global, "fetch").mockResolvedValue({
      ok: false,
      json: () => Promise.resolve({}),
    } as Response);

    const { result } = renderHook(() => useIdentity(), {
      wrapper: IdentityProvider,
    });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.identity).toBeNull();
  });
});

describe("canSell / canOperate", () => {
  it("canSell is true only for sellers -- operators are oversight-only, not also sellers", () => {
    expect(canSell({ username: "x", groups: [], role: "sellers" })).toBe(true);
    expect(canSell({ username: "x", groups: [], role: "operators" })).toBe(
      false,
    );
    expect(canSell(null)).toBe(false);
  });

  it("canOperate is true only for operators", () => {
    expect(canOperate({ username: "x", groups: [], role: "operators" })).toBe(
      true,
    );
    expect(canOperate({ username: "x", groups: [], role: "sellers" })).toBe(
      false,
    );
  });
});
