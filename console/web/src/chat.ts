import { sendChat } from "./api";
import { escapeHtml } from "./html";

// mountChat wires a minimal chat panel against the named Substrate concierge
// agent. Today every call returns AgentInvokerStub's ErrAgentInvokerNotConfigured
// message (see console/agent_invoker.go) - the panel renders that reply like
// any other agent message, which doubles as a live, honest demonstration of
// exactly where the real agent's reply will appear once it's wired in.
export function mountChat(
  root: HTMLElement,
  agentName: string,
  agentLabel: string,
): void {
  // crypto.randomUUID only exists in secure contexts (HTTPS or localhost) -
  // this demo is sometimes reached over plain HTTP (e.g. a bare LAN IP or an
  // un-TLS'd load balancer), where calling it would throw and silently kill
  // the whole page before chat, listings, or live events ever render.
  const sessionId =
    crypto.randomUUID?.() ?? Math.random().toString(36).slice(2);
  root.innerHTML = `
    <div class="chat-log" id="chat-log"></div>
    <form id="chat-form" class="chat-form">
      <textarea id="chat-input" class="chat-input" rows="2" placeholder="Ask ${escapeHtml(agentLabel)}... (Enter to send, Shift+Enter for a new line)"></textarea>
      <button type="submit">Send</button>
    </form>
  `;
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
    line.innerHTML = `<b>${who === "you" ? "You" : escapeHtml(agentLabel)}</b><span>${escapeHtml(text)}</span>`;
    log.appendChild(line);
    log.scrollTop = log.scrollHeight;
  }

  form.addEventListener("submit", (e) => {
    e.preventDefault();
    const message = input.value.trim();
    if (!message) return;
    appendMessage("you", message);
    input.value = "";
    sendChat(agentName, sessionId, message)
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
