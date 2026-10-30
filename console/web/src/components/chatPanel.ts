import { sendChat } from "../api";
import { escapeHtml } from "../html";
import { icons } from "../icons";

const DESKTOP_BREAKPOINT = "(min-width: 860px)";

interface ChatPanelOptions {
  agent: string;
  label: string;
  collapsible?: boolean;
}

// mountChatPanel wires a minimal chat panel against the named Substrate
// concierge agent. Today every call returns AgentInvokerStub's
// ErrAgentInvokerNotConfigured message (see console/agent_invoker.go) - the
// panel renders that reply like any other agent message, which doubles as a
// live, honest demonstration of exactly where the real agent's reply will
// appear once it's wired in.
export function mountChatPanel(
  root: HTMLElement,
  { agent, label, collapsible = true }: ChatPanelOptions,
): void {
  // crypto.randomUUID only exists in secure contexts (HTTPS or localhost) -
  // this demo is sometimes reached over plain HTTP (e.g. a bare LAN IP or an
  // un-TLS'd load balancer), where calling it would throw and silently kill
  // the whole page before chat, listings, or live events ever render.
  const sessionId =
    crypto.randomUUID?.() ?? Math.random().toString(36).slice(2);
  const expanded =
    !collapsible || window.matchMedia(DESKTOP_BREAKPOINT).matches;

  root.innerHTML = `
    ${
      collapsible
        ? `<div class="chat-panel-head">
      <button type="button" class="ghost chat-panel-toggle" aria-expanded="${expanded}" aria-controls="chat-collapse" aria-label="${expanded ? "Collapse" : "Expand"} ${escapeHtml(label)}">${icons.chevronDown}</button>
    </div>`
        : ""
    }
    <div class="chat-panel-collapse${expanded ? "" : " collapsed"}" id="chat-collapse">
      <div${expanded ? "" : " inert"}>
        <div class="chat-log" id="chat-log" role="log" aria-live="polite"></div>
        <form id="chat-form" class="chat-form">
          <label class="sr-only" for="chat-input">Message ${escapeHtml(label)}</label>
          <textarea id="chat-input" class="chat-input" rows="2" placeholder="Ask ${escapeHtml(label)}... (Enter to send, Shift+Enter for a new line)"></textarea>
          <button type="submit">Send</button>
        </form>
      </div>
    </div>
  `;

  const toggle = root.querySelector<HTMLButtonElement>(".chat-panel-toggle");
  const collapse = root.querySelector<HTMLDivElement>("#chat-collapse")!;
  const collapseInner = collapse.querySelector<HTMLDivElement>(":scope > div")!;
  toggle?.addEventListener("click", () => {
    const next = toggle.getAttribute("aria-expanded") !== "true";
    toggle.setAttribute("aria-expanded", String(next));
    toggle.setAttribute(
      "aria-label",
      `${next ? "Collapse" : "Expand"} ${label}`,
    );
    collapse.classList.toggle("collapsed", !next);
    // The collapsed panel is visually clipped to 0 height, but its form
    // controls stay in the tab order and accessibility tree without this -
    // inert removes both while collapsed.
    collapseInner.toggleAttribute("inert", !next);
  });

  const log = root.querySelector<HTMLDivElement>("#chat-log")!;
  const form = root.querySelector<HTMLFormElement>("#chat-form")!;
  const input = root.querySelector<HTMLTextAreaElement>("#chat-input")!;

  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      form.requestSubmit();
    }
  });

  function appendMessage(who: "you" | "agent", text: string): void {
    const line = document.createElement("div");
    line.className = `chat-msg ${who}`;
    line.innerHTML = `<b>${who === "you" ? "You" : escapeHtml(label)}</b><span>${escapeHtml(text)}</span>`;
    log.appendChild(line);
    log.scrollTop = log.scrollHeight;
  }

  form.addEventListener("submit", (e) => {
    e.preventDefault();
    const message = input.value.trim();
    if (!message) return;
    appendMessage("you", message);
    input.value = "";
    sendChat(agent, sessionId, message)
      .then((resp) =>
        appendMessage("agent", resp.reply || resp.error || "No response."),
      )
      .catch((err: unknown) =>
        appendMessage(
          "agent",
          `Error: ${err instanceof Error ? err.message : String(err)}`,
        ),
      );
  });
}
