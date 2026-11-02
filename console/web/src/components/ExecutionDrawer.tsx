import {
  useEffect,
  useRef,
  useState,
  type ReactNode,
  type KeyboardEvent,
} from "react";
import { IconClose } from "../icons";
import { getAgentJourney } from "../adapters/executionAdapter";
import { ExecutionTimeline } from "./ExecutionTimeline";
import { TechnicalInspector } from "./TechnicalInspector";
import type { AdapterResult } from "../adapters/types";
import type { AgentJourney } from "../adapters/executionAdapter";

export interface ExecutionTrustCheck {
  score: number;
  reasons: string[];
}

export interface ExecutionSummary {
  title: string;
  body: string;
  trustCheck: ExecutionTrustCheck | "unavailable";
}

export interface OpenExecutionDrawerOptions {
  summary: ExecutionSummary;
  correlationId?: string;
}

function trustCheckLine(
  trustCheck: ExecutionTrustCheck | "unavailable",
): string {
  if (trustCheck === "unavailable") {
    return "Trust check: not available for this bid.";
  }
  if (trustCheck.reasons.length === 0) {
    return `Trust check: cleared (score ${trustCheck.score.toFixed(2)}).`;
  }
  return `Trust check: flagged (score ${trustCheck.score.toFixed(2)}) - ${trustCheck.reasons.join("; ")}`;
}

// useExecutionDrawer gives every caller the original's "open it from
// anywhere" ergonomics without a global DOM singleton: call open(opts) from
// a click handler, render {drawer} once near the root of the page.
export function useExecutionDrawer(): {
  open: (opts: OpenExecutionDrawerOptions) => void;
  drawer: ReactNode;
} {
  const [isOpen, setIsOpen] = useState(false);
  const [opts, setOpts] = useState<OpenExecutionDrawerOptions | null>(null);
  const [journey, setJourney] = useState<AdapterResult<AgentJourney> | null>(
    null,
  );
  const lastFocused = useRef<HTMLElement | null>(null);

  function open(next: OpenExecutionDrawerOptions): void {
    lastFocused.current = document.activeElement as HTMLElement | null;
    setOpts(next);
    setJourney(null);
    setIsOpen(true);
    void getAgentJourney(next.correlationId).then(setJourney);
  }

  function close(): void {
    setIsOpen(false);
    lastFocused.current?.focus();
  }

  return {
    open,
    drawer:
      isOpen && opts ? (
        <ExecutionDrawerDialog opts={opts} journey={journey} onClose={close} />
      ) : null,
  };
}

function ExecutionDrawerDialog({
  opts,
  journey,
  onClose,
}: {
  opts: OpenExecutionDrawerOptions;
  journey: AdapterResult<AgentJourney> | null;
  onClose: () => void;
}) {
  const closeButtonRef = useRef<HTMLButtonElement>(null);
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    closeButtonRef.current?.focus();
  }, []);

  function handleKeyDown(e: KeyboardEvent): void {
    if (e.key === "Escape") {
      onClose();
      return;
    }
    if (e.key !== "Tab" || !dialogRef.current) return;
    const focusable = dialogRef.current.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
    );
    if (focusable.length === 0) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  }

  return (
    <div
      className="execution-drawer-root"
      ref={dialogRef}
      onKeyDown={handleKeyDown}
    >
      <div className="execution-drawer-backdrop" onClick={onClose} />
      <div
        className="execution-drawer"
        role="dialog"
        aria-modal="true"
        aria-labelledby="execution-drawer-title"
      >
        <div className="execution-drawer-head">
          <h2 id="execution-drawer-title">Behind the scenes</h2>
          <button
            type="button"
            className="ghost"
            aria-label="Close"
            onClick={onClose}
            ref={closeButtonRef}
          >
            <IconClose />
          </button>
        </div>
        <div className="execution-drawer-body">
          <section className="execution-section">
            <h3>Summary</h3>
            <div>
              <p>{opts.summary.title}</p>
              <p>{opts.summary.body}</p>
              <p className="trust-check-line">
                {trustCheckLine(opts.summary.trustCheck)}
              </p>
            </div>
          </section>
          <section className="execution-section">
            <h3>Agent Journey</h3>
            <div>{journey && <ExecutionTimeline result={journey} />}</div>
          </section>
          <section className="execution-section">
            <h3>Technical Inspector</h3>
            <div>{journey && <TechnicalInspector result={journey} />}</div>
          </section>
        </div>
      </div>
    </div>
  );
}
