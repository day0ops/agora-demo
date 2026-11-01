import {
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type KeyboardEvent,
} from "react";
import { sendChat } from "../api";
import { IconChevronDown } from "../icons";

const DESKTOP_BREAKPOINT = "(min-width: 860px)";

interface ChatMessage {
  who: "you" | "agent";
  text: string;
}

// Today every call returns AgentInvokerStub's ErrAgentInvokerNotConfigured
// message (see console/agent_invoker.go) unless a live agent cluster is
// wired in - the panel renders that reply like any other agent message,
// which doubles as a live, honest demonstration of exactly where the real
// agent's reply appears once it's wired in.
export function ChatPanel({
  agent,
  label,
  collapsible = true,
}: {
  agent: string;
  label: string;
  collapsible?: boolean;
}) {
  // crypto.randomUUID only exists in secure contexts (HTTPS or localhost) -
  // this demo is sometimes reached over plain HTTP (e.g. a bare LAN IP or an
  // un-TLS'd load balancer), where calling it would throw.
  const [sessionId] = useState(
    () => crypto.randomUUID?.() ?? Math.random().toString(36).slice(2),
  );
  const [expanded, setExpanded] = useState(
    () => !collapsible || window.matchMedia(DESKTOP_BREAKPOINT).matches,
  );
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const logRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const collapseBodyRef = useRef<HTMLDivElement>(null);

  // `inert` isn't in @types/react 18.3's stable surface (promoted to stable
  // in React 19's types) and react-dom 18 doesn't treat it as a boolean DOM
  // property either, so a JSX prop both fails to typecheck and logs a
  // non-boolean-attribute warning at runtime. Toggle it imperatively instead.
  useEffect(() => {
    collapseBodyRef.current?.toggleAttribute("inert", !expanded);
  }, [expanded]);

  function appendMessage(who: "you" | "agent", text: string): void {
    setMessages((prev) => [...prev, { who, text }]);
    requestAnimationFrame(() => {
      if (logRef.current)
        logRef.current.scrollTop = logRef.current.scrollHeight;
    });
  }

  function handleSubmit(e: FormEvent): void {
    e.preventDefault();
    const el = inputRef.current;
    const message = el?.value.trim() ?? "";
    if (!message) return;
    appendMessage("you", message);
    if (el) el.value = "";
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
  }

  function handleInputKeyDown(e: KeyboardEvent<HTMLTextAreaElement>): void {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      e.currentTarget.form?.requestSubmit();
    }
  }

  return (
    <>
      {collapsible && (
        <div className="chat-panel-head">
          <button
            type="button"
            className="ghost chat-panel-toggle"
            aria-expanded={expanded}
            aria-controls="chat-collapse"
            aria-label={`${expanded ? "Collapse" : "Expand"} ${label}`}
            onClick={() => setExpanded((e) => !e)}
          >
            <IconChevronDown />
          </button>
        </div>
      )}
      <div
        className={`chat-panel-collapse${expanded ? "" : " collapsed"}`}
        id="chat-collapse"
      >
        {/* The collapsed panel is visually clipped to 0 height, but its form
            controls stay in the tab order and accessibility tree without
            `inert` - it removes both while collapsed. */}
        <div ref={collapseBodyRef}>
          <div className="chat-log" role="log" aria-live="polite" ref={logRef}>
            {messages.map((m, i) => (
              <div key={i} className={`chat-msg ${m.who}`}>
                <b>{m.who === "you" ? "You" : label}</b>
                <span>{m.text}</span>
              </div>
            ))}
          </div>
          <form className="chat-form" onSubmit={handleSubmit}>
            <label className="sr-only" htmlFor="chat-input">
              Message {label}
            </label>
            {/* Uncontrolled: a controlled value with no defaultValue prop
                makes React resync the textarea's defaultValue (and thus its
                childNodes/textContent) to the live value on every keystroke,
                which leaks in-progress, unsent text to text-content queries. */}
            <textarea
              id="chat-input"
              className="chat-input"
              rows={2}
              placeholder={`Ask ${label}... (Enter to send, Shift+Enter for a new line)`}
              ref={inputRef}
              onKeyDown={handleInputKeyDown}
            />
            <button type="submit">Send</button>
          </form>
        </div>
      </div>
    </>
  );
}
