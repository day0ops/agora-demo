import { statusLabel, statusPillClass } from "../format";

export function StatusPill({ status }: { status: string }) {
  return (
    <span className={`pill ${statusPillClass(status)}`}>
      {statusLabel(status)}
    </span>
  );
}
