import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { ChatPanel } from "./ChatPanel";
import * as api from "../api";

describe("ChatPanel", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    window.matchMedia = vi.fn().mockReturnValue({
      matches: false,
    }) as unknown as typeof window.matchMedia;
  });

  it("sends a message and appends the agent's reply", async () => {
    vi.spyOn(api, "sendChat").mockResolvedValue({
      reply: "Here are three listings.",
    });
    render(
      <ChatPanel
        agent="shopping-assistant"
        label="Shopping Assistant"
        collapsible={false}
      />,
    );

    const input = screen.getByLabelText("Message Shopping Assistant");
    fireEvent.change(input, { target: { value: "Find me a watch" } });
    fireEvent.submit(
      screen.getByRole("button", { name: "Send" }).closest("form")!,
    );

    expect(
      await screen.findByText("Here are three listings."),
    ).toBeInTheDocument();
    expect(screen.getByText("Find me a watch")).toBeInTheDocument();
  });

  it("submits on Enter but not on Shift+Enter", () => {
    vi.spyOn(api, "sendChat").mockResolvedValue({ reply: "ok" });
    render(<ChatPanel agent="a" label="A" collapsible={false} />);
    const input = screen.getByLabelText("Message A") as HTMLTextAreaElement;
    fireEvent.change(input, { target: { value: "hi" } });
    fireEvent.keyDown(input, { key: "Enter", shiftKey: true });
    expect(screen.queryByText("hi")).not.toBeInTheDocument();
    fireEvent.keyDown(input, { key: "Enter" });
    expect(screen.getByText("hi")).toBeInTheDocument();
  });

  it("toggles collapse state via the header button when collapsible", () => {
    render(<ChatPanel agent="a" label="A" collapsible />);
    const toggle = screen.getByRole("button", { name: /Expand A|Collapse A/ });
    const initialExpanded = toggle.getAttribute("aria-expanded");
    fireEvent.click(toggle);
    expect(toggle.getAttribute("aria-expanded")).not.toBe(initialExpanded);
  });
});
